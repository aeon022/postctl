package tui

import (
	"strings"
	"testing"

	"github.com/aeon022/missionctl-core/tuitest"
	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/models"
)

func TestProfileColors(t *testing.T) {
	if profileColor("") != "" || profileColor("default") != "" || profileChip("") != "" {
		t.Error("the default profile has no color")
	}
	if profileColor("bruecke") != "#E8142B" {
		t.Error("bruecke is the Brücke's red")
	}
	a, b := profileColor("work"), profileColor("work")
	if a == "" || a != b {
		t.Errorf("named profiles get a stable color, got %q/%q", a, b)
	}
}

func TestManualPublishAsksFirstAndOnlyYesSends(t *testing.T) {
	prev := config.ActiveProfile
	config.ActiveProfile = "bruecke"
	t.Cleanup(func() { config.ActiveProfile = prev })

	m := layoutModel(t, 120, 40)
	m.posts = []models.Post{{ID: "p1", Platform: "instagram", Title: "ABBA", Status: models.StatusScheduled}}
	m.cursor = 0
	mm, _ := m.switchTab(tabPosts)
	mm.cursor = 0

	asked, _ := tuitest.Keys(mm, "p")
	am := asked.(Model)
	if am.ask == nil || am.loading {
		t.Fatalf("p must open the question, not publish (ask=%v loading=%v)", am.ask, am.loading)
	}
	text := tuitest.Text(am)
	if !strings.Contains(text, "Publish now?") || !strings.Contains(text, "bruecke") || !strings.Contains(text, "ABBA") {
		t.Errorf("confirmation should name profile and post:\n%s", text)
	}

	canceled, _ := tuitest.Keys(am, "n")
	if cm := canceled.(Model); cm.ask != nil || cm.loading {
		t.Error("n must cancel without publishing")
	}
	sent, _ := tuitest.Keys(am, "y")
	if sm := sent.(Model); sm.ask != nil || !sm.loading {
		t.Error("y must start publishing")
	}
}

func TestDefaultProfileIsLabelledPrivate(t *testing.T) {
	if got := profileLabel(""); got != "Privat" && got != "Private" {
		t.Errorf("the default profile should read Privat/Private in the picker, got %q", got)
	}
}
