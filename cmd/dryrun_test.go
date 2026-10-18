package cmd

import (
	"context"
	"testing"
	"time"

	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/models"
	"github.com/aeon022/postctl/internal/store"
)

// --dry-run must never change a post: delete, cancel and schedule used to ignore it (a "dry run" of delete really deleted).
func TestDryRunDoesNotDeleteCancelOrSchedule(t *testing.T) {
	ctx := context.Background()
	s, err := store.NewSQLiteStore(config.GetDBPath(), config.Shared())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	when := time.Now().Add(24 * time.Hour)
	for _, p := range []*models.Post{
		{ID: "dry-draft", Platform: "threads", Type: "single", Body: "x", Status: models.StatusDraft},
		{ID: "dry-sched", Platform: "threads", Type: "single", Body: "y", Status: models.StatusScheduled, ScheduledAt: &when},
	} {
		if err := s.SavePost(ctx, p); err != nil {
			t.Fatal(err)
		}
	}
	DryRunFlag = true
	t.Cleanup(func() { DryRunFlag = false })

	for _, args := range [][]string{
		{"delete", "dry-draft"},
		{"cancel", "dry-sched"},
		{"schedule", "dry-draft", "2030-01-01 10:00"},
	} {
		rootCmd.SetArgs(append([]string{"--profile", ""}, args...))
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}

	if p, err := s.GetPost(ctx, "dry-draft"); err != nil || p.Status != models.StatusDraft {
		t.Errorf("dry-run delete/schedule changed the draft: %v %+v", err, p)
	}
	if p, err := s.GetPost(ctx, "dry-sched"); err != nil || p.Status != models.StatusScheduled || p.ScheduledAt == nil {
		t.Errorf("dry-run cancel changed the scheduled post: %v %+v", err, p)
	}
}

func TestInstagramTokenCheck(t *testing.T) {
	now := time.Date(2026, 12, 1, 12, 0, 0, 0, time.Local)
	for _, tc := range []struct {
		issued string
		ok     bool
	}{
		{"2026-11-20", true},  // 11 days old
		{"2026-10-16", false}, // 46 days old: the daemon should have refreshed it
		{"2026-09-01", false}, // expired
	} {
		c, found := instagramTokenCheck("instagram", true, tc.issued, now)
		if !found || c.OK != tc.ok {
			t.Errorf("issued %s: found=%v ok=%v (%s), want ok=%v", tc.issued, found, c.OK, c.Detail, tc.ok)
		}
	}
	if _, found := instagramTokenCheck("facebook", true, "2026-11-20", now); found {
		t.Error("only the Instagram-Login token is checked")
	}
}
