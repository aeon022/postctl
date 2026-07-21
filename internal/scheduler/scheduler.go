package scheduler

import (
	"context"
	"fmt"
	"net"
	"os"
	"sort"
	"time"

	"github.com/aeon022/postctl/internal/models"
	"github.com/aeon022/postctl/internal/platforms"
	"github.com/aeon022/postctl/internal/store"
)

// isOnline prüft, ob eine Internetverbindung besteht.
func isOnline() bool {
	_, err := net.LookupHost("one.one.one.one")
	return err == nil
}

// PublishPost veröffentlicht einen Post und aktualisiert den DB-Status sowie die Historie
func PublishPost(ctx context.Context, s *store.SQLiteStore, post *models.Post, dryRun bool) (string, error) {
	// Falls es ein Thread-Post ist, verteile die globalen Bilder auf die einzelnen Tweets
	post.PrepareTweets()

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

	// Post veröffentlichen (mit zentralem Retry bei transienten Fehlern: Netzwerk, 429, 5xx)
	platformID, err := platforms.WithRetry(ctx, platforms.DefaultRetryConfig, post.Platform, func() (string, error) {
		return plat.Post(ctx, post)
	})
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
func RunDaemon(ctx context.Context, s *store.SQLiteStore, checkInterval time.Duration, dryRun bool) error {
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

// RescheduleOverdue prüft, ob mehrere überfällige Beiträge für dieselbe Plattform vorhanden sind.
// Falls ja, bleibt der älteste Beitrag wie geplant (sofortige Veröffentlichung), während die
// nachfolgenden Beiträge in 20-Minuten-Abständen ab time.Now() neu geplant werden.
func RescheduleOverdue(ctx context.Context, s *store.SQLiteStore) error {
	now := time.Now()
	posts, err := s.ListPosts(ctx, "all", models.StatusScheduled, "")
	if err != nil {
		return err
	}

	// Überfällige Beiträge nach Plattform gruppieren
	overdueByPlatform := make(map[string][]models.Post)
	for _, p := range posts {
		if p.ScheduledAt != nil && p.ScheduledAt.Before(now) {
			overdueByPlatform[p.Platform] = append(overdueByPlatform[p.Platform], p)
		}
	}

	for platform, overdueList := range overdueByPlatform {
		if len(overdueList) <= 1 {
			continue
		}

		// Nach ursprünglicher geplanter Zeit sortieren (älteste zuerst)
		sort.Slice(overdueList, func(i, j int) bool {
			if overdueList[i].ScheduledAt == nil {
				return true
			}
			if overdueList[j].ScheduledAt == nil {
				return false
			}
			return overdueList[i].ScheduledAt.Before(*overdueList[j].ScheduledAt)
		})

		// Der erste (Index 0) bleibt unverändert (geht sofort raus).
		// Die nachfolgenden werden in 20-Minuten-Schritten neu geplant.
		for i := 1; i < len(overdueList); i++ {
			p := overdueList[i]
			newScheduled := now.Add(time.Duration(i) * 20 * time.Minute)
			p.ScheduledAt = &newScheduled
			p.UpdatedAt = now
			
			if err := s.SavePost(ctx, &p); err != nil {
				return fmt.Errorf("reschedule post %s failed: %w", p.ID, err)
			}
			platforms.Log("[SAFETY] Überfälliger Post %s (%s) wurde auf %s verschoben, um Spam/Sperren zu vermeiden.", 
				p.ID, platform, newScheduled.Format("15:04:05"))
		}
	}

	return nil
}

// checkAndPublishDue prüft die DB auf fällige geplante Posts und veröffentlicht sie
func checkAndPublishDue(ctx context.Context, s *store.SQLiteStore, dryRun bool) {
	// Sicherheits-Rescheduling für überfällige Beiträge durchführen
	if err := RescheduleOverdue(ctx, s); err != nil {
		platforms.Log("[SCHEDULER FEHLER] Sicherheits-Rescheduling fehlgeschlagen: %v", err)
	}

	now := time.Now()
	
	// Hole alle geplanten Posts erneut (nach potentiellem Rescheduling)
	posts, err := s.ListPosts(ctx, "all", models.StatusScheduled, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "[SCHEDULER FEHLER] Kann geplante Posts nicht lesen: %v\n", err)
		return
	}

	// Prüfe, ob fällige Posts existieren
	hasDue := false
	for _, p := range posts {
		if p.ScheduledAt != nil && p.ScheduledAt.Before(now) {
			hasDue = true
			break
		}
	}

	// Falls offline und fällige Posts da sind, abbrechen und im nächsten Tick erneut versuchen
	if hasDue && !dryRun && !isOnline() {
		fmt.Fprintln(os.Stderr, "[SCHEDULER] Maschine ist offline (DNS-Lookup fehlgeschlagen). Verschiebe Veröffentlichung fälliger Posts, bis Verbindung hergestellt ist.")
		return
	}

	for _, p := range posts {
		if p.ScheduledAt != nil && p.ScheduledAt.Before(now) {
			if !dryRun {
				// Versuche den Post atomar zu sperren, um doppeltes Posten zu verhindern
				locked, err := s.TryLockPost(ctx, p.ID)
				if err != nil {
					fmt.Fprintf(os.Stderr, "[SCHEDULER FEHLER] Fehler beim Sperren von %s: %v\n", p.ID, err)
					continue
				}
				if !locked {
					// Post wurde bereits von einem anderen Prozess gesperrt oder gepostet
					continue
				}
			}

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
