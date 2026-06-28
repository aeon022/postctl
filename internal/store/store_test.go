package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/aeon022/postctl/internal/models"
)

func TestStoreWorkflow(t *testing.T) {
	ctx := context.Background()

	// In-Memory SQLite Instanz erstellen
	s, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create in-memory store: %v", err)
	}
	defer s.Close()

	// Test 1: Save & Get Post (Single)
	now := time.Now().Round(time.Second) // Runden, da DB Millisekunden evtl. abschneidet
	p1 := &models.Post{
		ID:          "test-single-post",
		Platform:    models.PlatformLinkedIn,
		Type:        "single",
		Language:    "en",
		Campaign:    "test-camp",
		Title:       "Title 1",
		Body:        "Hello LinkedIn!",
		Images:      []string{"img1.png"},
		Status:      models.StatusDraft,
		ScheduledAt: &now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.SavePost(ctx, p1); err != nil {
		t.Fatalf("failed to save single post: %v", err)
	}

	gotP1, err := s.GetPost(ctx, p1.ID)
	if err != nil {
		t.Fatalf("failed to get single post: %v", err)
	}

	if gotP1.ID != p1.ID || gotP1.Body != p1.Body || gotP1.Platform != p1.Platform {
		t.Errorf("got single post mismatch: %+v", gotP1)
	}
	if len(gotP1.Images) != 1 || gotP1.Images[0] != "img1.png" {
		t.Errorf("got images mismatch: %v", gotP1.Images)
	}
	if gotP1.ScheduledAt == nil || !gotP1.ScheduledAt.Equal(*p1.ScheduledAt) {
		t.Errorf("scheduled_at mismatch: got %v, want %v", gotP1.ScheduledAt, p1.ScheduledAt)
	}

	// Test 2: Save & Get Post (Thread)
	p2 := &models.Post{
		ID:       "test-thread-post",
		Platform: models.PlatformTwitter,
		Type:     "thread",
		Language: "de",
		Campaign: "test-camp",
		Title:    "Title 2",
		Tweets: []models.Tweet{
			{Index: 1, Content: "Tweet 1", Image: "img_tweet1.png"},
			{Index: 2, Content: "Tweet 2", IsReply: true},
		},
		Status:    models.StatusScheduled,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.SavePost(ctx, p2); err != nil {
		t.Fatalf("failed to save thread post: %v", err)
	}

	gotP2, err := s.GetPost(ctx, p2.ID)
	if err != nil {
		t.Fatalf("failed to get thread post: %v", err)
	}

	if gotP2.ID != p2.ID || len(gotP2.Tweets) != 2 || gotP2.Tweets[0].Content != "Tweet 1" {
		t.Errorf("got thread post mismatch: %+v", gotP2)
	}
	if gotP2.Tweets[0].Image != "img_tweet1.png" || !gotP2.Tweets[1].IsReply {
		t.Errorf("got tweets details mismatch: %+v", gotP2.Tweets)
	}

	// Test 3: List Posts & Filters
	posts, err := s.ListPosts(ctx, "all", "all", "")
	if err != nil {
		t.Fatalf("failed to list posts: %v", err)
	}
	if len(posts) != 2 {
		t.Errorf("expected 2 posts, got %d", len(posts))
	}

	// Filter auf linkedin
	postsFiltered, err := s.ListPosts(ctx, models.PlatformLinkedIn, "all", "")
	if err != nil {
		t.Fatalf("failed to list filtered posts: %v", err)
	}
	if len(postsFiltered) != 1 || postsFiltered[0].ID != p1.ID {
		t.Errorf("expected 1 linkedin post, got %d", len(postsFiltered))
	}

	// Filter auf draft
	postsStatus, err := s.ListPosts(ctx, "all", models.StatusDraft, "")
	if err != nil {
		t.Fatalf("failed to list filtered status posts: %v", err)
	}
	if len(postsStatus) != 1 || postsStatus[0].ID != p1.ID {
		t.Errorf("expected 1 draft post, got %d", len(postsStatus))
	}

	// Test 4: Auth Tokens
	expires := now.Add(1 * time.Hour)
	err = s.SaveToken(ctx, models.PlatformTwitter, "secret-token-val", "refresh-token-val", &expires)
	if err != nil {
		t.Fatalf("failed to save token: %v", err)
	}

	// 2. Plattform hinzufügen (sollte auf Core erfolgreich sein)
	err = s.SaveToken(ctx, models.PlatformLinkedIn, "linkedin-token-val", "", nil)
	if err != nil {
		t.Fatalf("failed to save 2nd platform token: %v", err)
	}

	// 3. Plattform hinzufügen (sollte auf Core fehlschlagen)
	err = s.SaveToken(ctx, models.PlatformThreads, "threads-token-val", "", nil)
	if err == nil {
		t.Fatal("expected error when saving 3rd platform token in Core tier, got nil")
	}
	if !strings.Contains(err.Error(), "Pro Feature") {
		t.Errorf("expected Pro limit error message, got: %v", err)
	}

	// Holen und prüfen (Entschlüsselung testen)
	tok, ref, exp, err := s.GetToken(ctx, models.PlatformTwitter)
	if err != nil {
		t.Fatalf("failed to get token: %v", err)
	}

	if tok != "secret-token-val" || ref != "refresh-token-val" {
		t.Errorf("token decryption mismatch: token=%q, refresh=%q", tok, ref)
	}
	if exp == nil || !exp.Equal(expires) {
		t.Errorf("token expires mismatch: got %v, want %v", exp, expires)
	}

	// Test 5: History Entries
	h1 := &models.HistoryEntry{
		PostID:     p1.ID,
		Action:     "posted",
		PlatformID: "1234567890",
	}
	if err := s.AddHistoryEntry(ctx, h1); err != nil {
		t.Fatalf("failed to add history entry: %v", err)
	}

	history, err := s.GetHistory(ctx, 10)
	if err != nil {
		t.Fatalf("failed to get history: %v", err)
	}

	if len(history) != 1 || history[0].PostID != p1.ID || history[0].PlatformID != "1234567890" {
		t.Errorf("history mismatch: %+v", history)
	}

	// Test 6: Delete Post (kaskadiert die History)
	if err := s.DeletePost(ctx, p1.ID); err != nil {
		t.Fatalf("failed to delete post: %v", err)
	}

	_, err = s.GetPost(ctx, p1.ID)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("expected sql.ErrNoRows for deleted post, got %v", err)
	}

	// History sollte kaskadiert gelöscht sein
	historyAfterDelete, err := s.GetHistory(ctx, 10)
	if err != nil {
		t.Fatalf("failed to get history after delete: %v", err)
	}
	if len(historyAfterDelete) != 0 {
		t.Errorf("expected history to be empty after cascading delete, got %d", len(historyAfterDelete))
	}
}
