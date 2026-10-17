package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/aeon022/postctl/internal/models"
	"github.com/aeon022/postctl/internal/store"
)

// A token written under another key must not turn a scheduled post into a failed one.
func TestUndecryptableTokenKeepsPostScheduled(t *testing.T) {
	t.Setenv("POSTCTL_ENCRYPTION_KEY", "key-of-the-other-device")
	s, err := store.NewSQLiteStore(":memory:", false)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	if err := s.SaveToken(ctx, "threads", "secret", "", nil); err != nil {
		t.Fatal(err)
	}
	t.Setenv("POSTCTL_ENCRYPTION_KEY", "a-different-key") // this process has another key

	now := time.Now()
	post := &models.Post{ID: "p", Platform: "threads", Type: "single", Body: "hi", Status: "posting", ScheduledAt: &now, CreatedAt: now, UpdatedAt: now}
	if err := s.SavePost(ctx, post); err != nil {
		t.Fatal(err)
	}
	if _, err := PublishPost(ctx, s, post, false); err == nil {
		t.Fatal("publishing must report the problem")
	}
	got, _ := s.GetPost(ctx, "p")
	if got.Status != models.StatusScheduled {
		t.Fatalf("post must stay scheduled, got %q", got.Status)
	}
}
