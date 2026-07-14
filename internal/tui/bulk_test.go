package tui

import (
	"context"
	"testing"
	"time"

	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/models"
	"github.com/aeon022/postctl/internal/store"
	tea "github.com/charmbracelet/bubbletea"
)

func TestTUIBulkActions(t *testing.T) {
	s, err := store.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("sqlite memory store error: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	// 1. Create mock posts in DB
	post1 := &models.Post{
		ID:        "post-1",
		Platform:  "twitter",
		Status:    models.StatusDraft,
		Title:     "First Post",
		Body:      "Content of first",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	post2 := &models.Post{
		ID:        "post-2",
		Platform:  "twitter",
		Status:    models.StatusDraft,
		Title:     "Second Post",
		Body:      "Content of second",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.SavePost(ctx, post1); err != nil {
		t.Fatalf("failed to save post1: %v", err)
	}
	if err := s.SavePost(ctx, post2); err != nil {
		t.Fatalf("failed to save post2: %v", err)
	}

	// 2. Initialize Model
	m := NewModel(s)
	m.activeTab = 1 // Posts tab
	m.loading = false

	// Load data manually for the test
	msg := m.loadDataCmd()
	resModel, _ := m.Update(msg)
	m = resModel.(Model)

	if len(m.posts) != 2 {
		t.Fatalf("expected 2 loaded posts, got %d", len(m.posts))
	}

	// 3. Select first post (Space key)
	m.cursor = 0
	resModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	m = resModel.(Model)

	if !m.selectedPosts["post-1"] {
		t.Errorf("expected post-1 to be selected, but was not")
	}

	// 4. Select second post (Space key)
	m.cursor = 1
	resModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	m = resModel.(Model)

	if !m.selectedPosts["post-2"] {
		t.Errorf("expected post-2 to be selected, but was not")
	}

	// 5. Test Bulk Schedule
	config.ActiveConfig.Scheduler.Slots = []string{"Mon 09:00", "Wed 14:00"}
	resModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	m = resModel.(Model)

	if cmd == nil {
		t.Fatalf("expected schedule cmd to be returned, got nil")
	}

	// Run schedule cmd
	resMsg := cmd()
	resModel, _ = m.Update(resMsg)
	m = resModel.(Model)

	// Verify posts are scheduled
	p1, err := s.GetPost(ctx, "post-1")
	if err != nil || p1.Status != models.StatusScheduled || p1.ScheduledAt == nil {
		t.Errorf("expected post-1 to be scheduled, got status: %s", p1.Status)
	}

	p2, err := s.GetPost(ctx, "post-2")
	if err != nil || p2.Status != models.StatusScheduled || p2.ScheduledAt == nil {
		t.Errorf("expected post-2 to be scheduled, got status: %s", p2.Status)
	}

	// 6. Test Bulk Delete (mark again)
	// Reset status to draft for delete test
	p1.Status = models.StatusDraft
	p2.Status = models.StatusDraft
	s.SavePost(ctx, p1)
	s.SavePost(ctx, p2)

	// Reload data
	msg = m.loadDataCmd()
	resModel, _ = m.Update(msg)
	m = resModel.(Model)

	m.selectedPosts["post-1"] = true
	m.selectedPosts["post-2"] = true

	resModel, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	m = resModel.(Model)

	if cmd == nil {
		t.Fatalf("expected delete cmd to be returned, got nil")
	}

	// Run delete cmd
	resMsg = cmd()
	resModel, _ = m.Update(resMsg)
	m = resModel.(Model)

	// Verify posts are deleted
	_, err = s.GetPost(ctx, "post-1")
	if err == nil {
		t.Errorf("expected post-1 to be deleted, but it still exists")
	}
	_, err = s.GetPost(ctx, "post-2")
	if err == nil {
		t.Errorf("expected post-2 to be deleted, but it still exists")
	}
}
