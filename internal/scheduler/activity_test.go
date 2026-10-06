package scheduler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aeon022/missionctl-core/activity"
	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/models"
	"github.com/aeon022/postctl/internal/store"
)

func isolate(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("MISSIONCTL_DATA_DIR", t.TempDir())
	t.Setenv("MISSIONCTL_ACTIVITY", "")
}

func today(t *testing.T) []activity.Event {
	t.Helper()
	from, to := activity.Day(time.Now())
	evs, err := activity.Read(from, to)
	if err != nil {
		t.Fatal(err)
	}
	return evs
}

// discordHook serves a fake Discord webhook: GET validates, POST answers with
// a message id (what ?wait=true returns).
func discordHook(t *testing.T) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"4711"}`))
	}))
	t.Cleanup(srv.Close)
	old := config.ActiveConfig.Discord.WebhookURL
	config.ActiveConfig.Discord.WebhookURL = srv.URL
	t.Cleanup(func() { config.ActiveConfig.Discord.WebhookURL = old })
}

func newStore(t *testing.T) *store.SQLiteStore {
	t.Helper()
	s, err := store.NewSQLiteStore(filepath.Join(t.TempDir(), "postctl.db"), false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func discordPost() *models.Post {
	return &models.Post{ID: "p1", Platform: models.PlatformDiscord, Type: "single", Title: "Launch day",
		Body: "secret body text that must not be logged", Status: models.StatusDraft, CreatedAt: time.Now(), UpdatedAt: time.Now()}
}

func TestPublishLogsOnePublishedEvent(t *testing.T) {
	isolate(t)
	discordHook(t)
	if _, err := PublishPost(context.Background(), newStore(t), discordPost(), false); err != nil {
		t.Fatalf("PublishPost: %v", err)
	}
	evs := today(t)
	if len(evs) != 1 || evs[0].Tool != "postctl" || evs[0].Action != "published" || evs[0].Title != "Launch day → discord" {
		t.Fatalf("events = %+v, want one 'postctl published Launch day → discord'", evs)
	}
	raw, _ := os.ReadFile(activity.Path())
	if strings.Contains(string(raw), "secret body") {
		t.Error("post body leaked into the log when a title exists")
	}
}

func TestDryRunLogsNothing(t *testing.T) {
	isolate(t)
	if _, err := PublishPost(context.Background(), newStore(t), discordPost(), true); err != nil {
		t.Fatal(err)
	}
	if evs := today(t); len(evs) != 0 {
		t.Errorf("a dry run published nothing, must log nothing: %+v", evs)
	}
}

func TestPublishSucceedsWithActivityOff(t *testing.T) {
	isolate(t)
	t.Setenv("MISSIONCTL_ACTIVITY", "off")
	discordHook(t)
	if _, err := PublishPost(context.Background(), newStore(t), discordPost(), false); err != nil {
		t.Fatalf("publishing must not depend on logging: %v", err)
	}
	if _, err := os.Stat(activity.Path()); err == nil {
		t.Error("logging off must not create the log")
	}
}

func TestLogPostTitles(t *testing.T) {
	isolate(t)
	at := time.Date(2026, 12, 24, 18, 30, 0, 0, time.Local)
	long := strings.Repeat("wort ", 30)
	LogPost("published", &models.Post{Platform: "bluesky", Body: long})
	LogPost("published", &models.Post{Platform: "twitter", Tweets: []models.Tweet{{Index: 1, Content: "erster Tweet"}, {Index: 2, Content: "zweiter"}}})
	LogPost("scheduled", &models.Post{Platform: "mastodon", Title: "Weihnachten", ScheduledAt: &at})
	evs := today(t)
	if len(evs) != 3 {
		t.Fatalf("events = %+v", evs)
	}
	if !strings.HasSuffix(evs[0].Title, "… → bluesky") || len([]rune(evs[0].Title)) > 60+len(" → bluesky")+1 {
		t.Errorf("body fallback must be cut at 60 runes: %q", evs[0].Title)
	}
	if evs[1].Title != "erster Tweet → twitter" {
		t.Errorf("thread fallback = %q", evs[1].Title)
	}
	if evs[2].Action != "scheduled" || evs[2].Title != "Weihnachten → mastodon (2026-12-24 18:30)" {
		t.Errorf("scheduled = %+v", evs[2])
	}
}
