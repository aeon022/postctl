package models

import (
	"regexp"
	"strings"
	"time"
)

// Status-Konstanten
const (
	StatusDraft     = "draft"
	StatusScheduled = "scheduled"
	StatusPosted    = "posted"
	StatusFailed    = "failed"
	StatusPartial   = "partial"
)

// Platform-Konstanten
const (
	PlatformTwitter  = "twitter"
	PlatformLinkedIn = "linkedin"
	PlatformThreads  = "threads"
	PlatformMastodon = "mastodon"
	PlatformBluesky  = "bluesky"
	PlatformFacebook = "facebook"
)

// Post repräsentiert einen Social-Media-Post
type Post struct {
	ID          string     `json:"id" yaml:"id"`
	Platform    string     `json:"platform" yaml:"platform"`
	Type        string     `json:"type" yaml:"type"` // thread, single, article
	Language    string     `json:"language" yaml:"language"`
	Campaign    string     `json:"campaign" yaml:"campaign"`
	Title       string     `json:"title" yaml:"title"`
	Tweets      []Tweet    `json:"tweets" yaml:"tweets"` // Für Threads
	Body        string     `json:"body" yaml:"body"`     // Für Singles
	Images      []string   `json:"images" yaml:"images"`
	Tags        []string   `json:"tags" yaml:"tags"`
	Status      string     `json:"status" yaml:"status"`
	ScheduledAt *time.Time `json:"scheduled_at" yaml:"schedule"`
	PostedAt    *time.Time `json:"posted_at" yaml:"posted_at"`
	PlatformID  string     `json:"platform_id" yaml:"platform_id"`
	Error       string     `json:"error" yaml:"error"`
	SourceFile  string     `json:"source_file" yaml:"source_file"`
	CreatedAt   time.Time  `json:"created_at" yaml:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" yaml:"updated_at"`
}

// Tweet ist ein einzelner Tweet in einem Thread
type Tweet struct {
	Index   int    `json:"index"`
	Content string `json:"content"`
	Image   string `json:"image"`    // Optionaler Bild-Pfad
	IsReply bool   `json:"is_reply"` // Letzter Tweet = Reply mit Links
}

var urlRegex = regexp.MustCompile(`https?://[^\s]+`)

// CharCount gibt die Zeichenanzahl zurück (URLs = 23 Zeichen)
func (t Tweet) CharCount() int {
	// Twitter zählt jede URL als genau 23 Zeichen (t.co)
	processed := urlRegex.ReplaceAllString(t.Content, "12345678901234567890123")
	return len([]rune(processed))
}

// IsValid prüft ob der Tweet innerhalb des Limits ist
func (t Tweet) IsValid() bool {
	return t.CharCount() <= 280
}

// Campaign gruppiert Posts
type Campaign struct {
	Slug      string `json:"slug"`
	Posts     []Post `json:"posts"`
	Posted    int    `json:"posted"`
	Drafts    int    `json:"drafts"`
	Scheduled int    `json:"scheduled"`
}

// HistoryEntry repräsentiert einen Eintrag in der Post-Historie
type HistoryEntry struct {
	ID         string    `json:"id"`
	PostID     string    `json:"post_id"`
	Action     string    `json:"action"` // posted, failed, retried, edited
	PlatformID string    `json:"platform_id"`
	Error      string    `json:"error"`
	CreatedAt  time.Time `json:"created_at"`
}

// DeriveTitle generiert einen Titel aus dem Post-Inhalt
func DeriveTitle(body string) string {
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "##") && !strings.HasPrefix(trimmed, "---") {
			// Markdown-Formatierung entfernen
			trimmed = strings.ReplaceAll(trimmed, "*", "")
			trimmed = strings.ReplaceAll(trimmed, "_", "")
			trimmed = strings.ReplaceAll(trimmed, "`", "")
			trimmed = strings.ReplaceAll(trimmed, "#", "")
			trimmed = strings.TrimSpace(trimmed)
			if len(trimmed) > 40 {
				return trimmed[:37] + "..."
			}
			return trimmed
		}
	}
	return "Unbenannter Beitrag"
}

// PrepareTweets stellt sicher, dass bei einem Thread-Post die Bilder aus Images auf die Tweets verteilt werden,
// falls alle Tweets bisher keine Bilder zugewiesen haben.
func (p *Post) PrepareTweets() {
	if p.Type != "thread" || len(p.Images) == 0 {
		return
	}

	// Prüfen, ob bereits irgendein Tweet ein Bild zugewiesen hat
	hasAnyTweetImage := false
	for _, t := range p.Tweets {
		if t.Image != "" {
			hasAnyTweetImage = true
			break
		}
	}

	// Falls kein Tweet ein Bild hat, verteilen wir die globalen Bilder
	if !hasAnyTweetImage {
		for i := range p.Tweets {
			if i < len(p.Images) {
				p.Tweets[i].Image = p.Images[i]
			}
		}
	}
}

