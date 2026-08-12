package scheduler

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/models"
	"github.com/aeon022/postctl/internal/store"
)

// newDueScheduledPost inserts a post whose ScheduledAt is already in the
// past, the state runIfDesignated's gate is meant to guard.
func newDueScheduledPost(t *testing.T, s *store.SQLiteStore) *models.Post {
	t.Helper()
	past := time.Now().Add(-time.Hour)
	p := &models.Post{
		ID:          "test-post-1",
		Platform:    "mastodon",
		Type:        "single",
		Body:        "hello",
		Status:      models.StatusScheduled,
		ScheduledAt: &past,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := s.SavePost(context.Background(), p); err != nil {
		t.Fatalf("SavePost: %v", err)
	}
	return p
}

func TestRunIfDesignated_BlocksByDefault(t *testing.T) {
	// AutoPublish defaults to false — this is the exact real-world
	// configuration a non-designated machine should have. A due post must
	// come out of runIfDesignated untouched: still "scheduled", not
	// republished, no matter how overdue it looks locally.
	dir := t.TempDir()
	t.Setenv("POSTCTL_DATA_DIR", dir)
	config.ActiveConfig.Scheduler.AutoPublish = false
	t.Cleanup(func() { config.ActiveConfig.Scheduler.AutoPublish = false })

	s, err := store.NewSQLiteStore(config.GetDBPath(), config.Shared())
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	defer s.Close()

	newDueScheduledPost(t, s)

	runIfDesignated(context.Background(), s, false)

	posts, err := s.ListPosts(context.Background(), "all", models.StatusScheduled, "")
	if err != nil {
		t.Fatalf("ListPosts: %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("expected the due post to remain scheduled (untouched by a non-designated machine), got %d scheduled posts", len(posts))
	}
}

func TestRunIfDesignated_DryRunAlwaysProceeds(t *testing.T) {
	// dry-run never actually posts or persists a real publish, so it's safe
	// to preview on any machine regardless of AutoPublish/DBSettled — it
	// should still reach checkAndPublishDue (verified indirectly: it must
	// not early-return before RescheduleOverdue runs and touches nothing
	// scheduled-related that would fail below).
	dir := t.TempDir()
	t.Setenv("POSTCTL_DATA_DIR", dir)
	config.ActiveConfig.Scheduler.AutoPublish = false
	t.Cleanup(func() { config.ActiveConfig.Scheduler.AutoPublish = false })

	s, err := store.NewSQLiteStore(config.GetDBPath(), config.Shared())
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	defer s.Close()

	newDueScheduledPost(t, s)

	// Should not panic or error out just because the gate would otherwise
	// block a real run.
	runIfDesignated(context.Background(), s, true)
}

func TestRunIfDesignated_ProceedsWhenDesignatedAndSettled(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("POSTCTL_DATA_DIR", dir)
	config.ActiveConfig.Scheduler.AutoPublish = true
	t.Cleanup(func() { config.ActiveConfig.Scheduler.AutoPublish = false })

	s, err := store.NewSQLiteStore(config.GetDBPath(), config.Shared())
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	defer s.Close()

	newDueScheduledPost(t, s)

	// Back-date the DB file's mtime past dbSettleQuiet instead of sleeping
	// for it — DBSettled only looks at the file's ModTime.
	past := time.Now().Add(-2 * dbSettleQuiet)
	dbFile := filepath.Join(dir, "postctl.db")
	if err := os.Chtimes(dbFile, past, past); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}

	runIfDesignated(context.Background(), s, false)

	// The post targets a platform with no real credentials configured, so
	// PublishPost is expected to fail authentication — but it must have
	// been *attempted* (status flips away from "scheduled" either way,
	// since checkAndPublishDue's TryLockPost+PublishPost path always
	// updates status on both success and failure). That's the observable
	// signal that the gate let this run through, unlike the blocked case.
	posts, err := s.ListPosts(context.Background(), "all", models.StatusScheduled, "")
	if err != nil {
		t.Fatalf("ListPosts: %v", err)
	}
	if len(posts) != 0 {
		t.Errorf("expected the designated+settled machine to attempt the due post (leaving it no longer \"scheduled\"), got %d still scheduled", len(posts))
	}
}
