package scheduler

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aeon022/postctl/internal/models"
	"github.com/aeon022/postctl/internal/platforms"
	"github.com/aeon022/postctl/internal/store"
)

// PublishPost veröffentlicht einen Post und aktualisiert den DB-Status sowie die Historie
func PublishPost(ctx context.Context, s store.Store, post *models.Post, dryRun bool) (string, error) {
	// Plattform holen
	plat, err := platforms.GetPlatform(post.Platform, s, dryRun)
	if err != nil {
		return "", err
	}

	// Authentifizierung prüfen (nicht im dry-run)
	if !dryRun && !plat.IsAuthenticated(ctx) {
		err := fmt.Errorf("not authenticated with %s", post.Platform)
		post.Status = models.StatusFailed
		post.Error = err.Error()
		_ = s.SavePost(ctx, post)
		
		_ = s.AddHistoryEntry(ctx, &models.HistoryEntry{
			PostID: post.ID,
			Action: "failed",
			Error:  err.Error(),
		})
		return "", err
	}

	// Post veröffentlichen
	platformID, err := plat.Post(ctx, post)
	if err != nil {
		t := time.Now()
		post.Status = models.StatusFailed
		post.Error = err.Error()
		post.UpdatedAt = t
		_ = s.SavePost(ctx, post)

		_ = s.AddHistoryEntry(ctx, &models.HistoryEntry{
			PostID: post.ID,
			Action: "failed",
			Error:  err.Error(),
		})
		return "", err
	}

	// Erfolgs-Status eintragen
	if dryRun {
		return platformID, nil
	}

	t := time.Now()
	post.Status = models.StatusPosted
	post.PostedAt = &t
	post.PlatformID = platformID
	post.Error = ""
	post.UpdatedAt = t
	_ = s.SavePost(ctx, post)

	_ = s.AddHistoryEntry(ctx, &models.HistoryEntry{
		PostID:     post.ID,
		Action:     "posted",
		PlatformID: platformID,
	})

	return platformID, nil
}

// RunDaemon startet den Scheduler-Daemon im Headless-Modus (Endlosschleife)
func RunDaemon(ctx context.Context, s store.Store, checkInterval time.Duration, dryRun bool) error {
	fmt.Fprintf(os.Stderr, "Starte postctl Scheduler-Daemon (Intervall: %v, Dry-Run: %v)...\n", checkInterval, dryRun)
	fmt.Fprintln(os.Stderr, "Drücke Ctrl+C zum Beenden.")

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	// Initialer Check beim Start
	checkAndPublishDue(ctx, s, dryRun)

	for {
		select {
		case <-ctx.Done():
			fmt.Fprintln(os.Stderr, "Scheduler-Daemon wird heruntergefahren...")
			return nil
		case <-ticker.C:
			checkAndPublishDue(ctx, s, dryRun)
		}
	}
}

// checkAndPublishDue prüft die DB auf fällige geplante Posts und veröffentlicht sie
func checkAndPublishDue(ctx context.Context, s store.Store, dryRun bool) {
	now := time.Now()
	
	// Hole alle geplanten Posts
	posts, err := s.ListPosts(ctx, "all", models.StatusScheduled, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "[SCHEDULER FEHLER] Kann geplante Posts nicht lesen: %v\n", err)
		return
	}

	for _, p := range posts {
		if p.ScheduledAt != nil && p.ScheduledAt.Before(now) {
			fmt.Fprintf(os.Stderr, "[SCHEDULER] Veröffentliche fälligen Post %s (%s)...\n", p.ID, p.Platform)
			
			_, err := PublishPost(ctx, s, &p, dryRun)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[SCHEDULER FEHLER] Posten von %s fehlgeschlagen: %v\n", p.ID, err)
			} else {
				fmt.Fprintf(os.Stderr, "[SCHEDULER] Post %s erfolgreich veröffentlicht.\n", p.ID)
			}
		}
	}
}
