package platforms

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/aeon022/postctl/internal/models"
)

// DryRunPlatform ist eine Mock-Plattform für Test- und Simulationszwecke
type DryRunPlatform struct {
	platformName string
}

// NewDryRunPlatform erstellt eine neue DryRun-Instanz für eine Zielplattform
func NewDryRunPlatform(name string) *DryRunPlatform {
	return &DryRunPlatform{platformName: name}
}

// Name gibt den Namen der simulierten Plattform zurück
func (d *DryRunPlatform) Name() string {
	return d.platformName
}

// Auth simuliert eine erfolgreiche Authentifizierung
func (d *DryRunPlatform) Auth(ctx context.Context) error {
	fmt.Fprintf(os.Stderr, "[DRY RUN] Authentifizierung für %s wird simuliert...\n", d.platformName)
	time.Sleep(200 * time.Millisecond)
	return nil
}

// Post simuliert die Veröffentlichung eines Beitrags (Ausgabe auf Stderr, damit Stdout frei für JSON bleibt)
func (d *DryRunPlatform) Post(ctx context.Context, post *models.Post) (string, error) {
	fmt.Fprintf(os.Stderr, "[DRY RUN] Veröffentliche Beitrag auf %s...\n", d.platformName)
	time.Sleep(500 * time.Millisecond)

	if post.Type == "thread" {
		fmt.Fprintf(os.Stderr, "[DRY RUN] Thread gepostet (%d Tweets):\n", len(post.Tweets))
		for _, tweet := range post.Tweets {
			imgInfo := ""
			if tweet.Image != "" {
				imgInfo = fmt.Sprintf(" [Bild: %s]", tweet.Image)
			}
			fmt.Fprintf(os.Stderr, "  - Tweet %d: %q%s\n", tweet.Index, tweet.Content, imgInfo)
		}
	} else {
		imgInfo := ""
		if len(post.Images) > 0 {
			imgInfo = fmt.Sprintf(" [Bilder: %v]", post.Images)
		}
		fmt.Fprintf(os.Stderr, "[DRY RUN] Single Post: %q%s\n", post.Body, imgInfo)
	}

	// Zufällige ID generieren
	fakeID := fmt.Sprintf("dryrun-%d", rand.Int63n(100000000))
	return fakeID, nil
}

// UploadImage simuliert den Upload eines Bildes
func (d *DryRunPlatform) UploadImage(ctx context.Context, path string) (string, error) {
	fmt.Fprintf(os.Stderr, "[DRY RUN] Lade Bild hoch: %s...\n", path)
	time.Sleep(300 * time.Millisecond)
	fakeMediaID := fmt.Sprintf("dryrun-media-%d", rand.Int63n(100000))
	return fakeMediaID, nil
}

// IsAuthenticated gibt immer true zurück
func (d *DryRunPlatform) IsAuthenticated(ctx context.Context) bool {
	return true
}

// FetchAnalytics liefert simulierte Interaktionsdaten
func (d *DryRunPlatform) FetchAnalytics(ctx context.Context, platformID string) (models.AnalyticsData, error) {
	// Erzeuge deterministisch wirkende Pseudozufallsdaten für die Vorschau
	likes := 12 + rand.Intn(180)
	shares := 2 + rand.Intn(45)
	comments := rand.Intn(15)
	impressions := (likes + shares + 10) * (8 + rand.Intn(15))

	return models.AnalyticsData{
		PlatformID:  platformID,
		Likes:       likes,
		Shares:      shares,
		Comments:    comments,
		Impressions: impressions,
		FetchedAt:   time.Now(),
	}, nil
}

// Delete simuliert das Löschen eines Beitrags
func (d *DryRunPlatform) Delete(ctx context.Context, platformID string) error {
	fmt.Fprintf(os.Stderr, "[DRY RUN] Lösche Beitrag auf %s (ID: %s)...\n", d.platformName, platformID)
	time.Sleep(200 * time.Millisecond)
	return nil
}

