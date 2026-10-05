package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/aeon022/missionctl-core/tuitest"
	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/models"
	"github.com/aeon022/postctl/internal/store"
)

// smokeModel builds the real root model on an in-memory store. HOME is a temp
// dir and the global config is restored afterwards: the settings tab calls
// config.SaveConfig() synchronously from Update.
func smokeModel(t *testing.T, withPosts bool) Model {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	prevCfg, prevProfile := config.ActiveConfig, config.ActiveProfile
	t.Cleanup(func() { config.ActiveConfig, config.ActiveProfile = prevCfg, prevProfile })

	s, err := store.NewSQLiteStore(":memory:", false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	if withPosts {
		for _, p := range []*models.Post{
			{ID: "p1", Platform: "twitter", Status: models.StatusDraft, Title: "First", Body: "first body", CreatedAt: time.Now(), UpdatedAt: time.Now()},
			{ID: "p2", Platform: "bluesky", Status: models.StatusScheduled, Title: "Second", Body: "second body", CreatedAt: time.Now(), UpdatedAt: time.Now(), ScheduledAt: ptr(time.Now().Add(time.Hour))},
		} {
			if err := s.SavePost(context.Background(), p); err != nil {
				t.Fatal(err)
			}
		}
	}
	m := NewModel(s)
	m.loading = false
	m.width, m.height = 100, 30
	res, _ := m.Update(m.loadDataCmd())
	return res.(Model)
}

func ptr[T any](v T) *T { return &v }

// every tab and the views reachable from them, each left via esc
var smokeFlows = [][]string{
	{"tab", "j", "k", "tab", "j", "k", "tab", "j", "k", "tab", "tab", "tab", "tab"}, // all 7 tabs, wrapping
	{"shift+tab", "shift+tab", "tab"},
	{"tab", "j", "enter", "j", "k", "esc"},       // post detail
	{"tab", "n", "x", "esc"},                     // new post editor
	{"tab", "e", "tab", "esc"},                   // edit
	{"?", "esc"},                                 // help
	{"f1", "j", "k", "esc"},                      // readme
	{"R", "esc"},                                 //
	{"tab", "f", "esc"},                          // campaign filter
	{"tab", "space", "j", "space", "d", "esc"},   // bulk select + delete confirm
	{"tab", "tab", "enter", "esc"},               // schedule
	{"tab", "tab", "tab", "enter", "j", "esc"},   // history detail
	{"tab", "tab", "tab", "tab", "tab", "enter"}, // settings
}

func TestSmokeAllViews(t *testing.T) {
	for _, withPosts := range []bool{true, false} {
		for _, f := range smokeFlows {
			t.Run(strings.Join(f, " "), func(t *testing.T) {
				tuitest.Smoke(t, smokeModel(t, withPosts), f...)
			})
		}
	}
}

func TestSmokeSmallTerminal(t *testing.T) {
	for _, f := range smokeFlows {
		mm, _ := tuitest.Send(smokeModel(t, true), tuitest.Resize(60, 15))
		mm, _ = tuitest.Keys(mm, f...)
		if strings.TrimSpace(tuitest.Text(mm)) == "" {
			t.Errorf("empty view at 60x15 after %v", f)
		}
	}
}

// guard against the flows above silently doing nothing: the key sequences
// must actually reach the views they claim to visit.
func TestSmokeFlowsReachTheirViews(t *testing.T) {
	for _, c := range []struct {
		name string
		keys []string
		ok   func(Model) bool
	}{
		{"posts tab", []string{"tab"}, func(m Model) bool { return m.activeTab == 1 }},
		{"detail", []string{"tab", "enter"}, func(m Model) bool { return m.selectedPost != nil }},
		{"editor", []string{"tab", "n"}, func(m Model) bool { return m.isEditing }},
		{"help", []string{"?"}, func(m Model) bool { return m.showHelp }},
		{"readme", []string{"f1"}, func(m Model) bool { return m.showReadme }},
		{"bulk select", []string{"tab", "space"}, func(m Model) bool { return len(m.selectedPosts) == 1 }},
		{"settings tab", []string{"tab", "tab", "tab", "tab", "tab"}, func(m Model) bool { return m.activeTab == 5 }},
	} {
		mm, _ := tuitest.Keys(smokeModel(t, true), c.keys...)
		if !c.ok(mm.(Model)) {
			t.Errorf("%s: keys %v did not reach the view", c.name, c.keys)
		}
	}
}
