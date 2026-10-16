package tui

import (
	"github.com/charmbracelet/x/ansi"
	"strings"
	"testing"
	"time"

	"github.com/aeon022/missionctl-core/tuitest"
	"github.com/aeon022/postctl/internal/models"
)

func TestOnboardingChecklistShowsUntilFirstPostAndMarksDoneSteps(t *testing.T) {
	m := layoutModel(t, 120, 40)
	m.posts, m.campaigns, m.platforms = nil, nil, map[string]bool{}
	text := tuitest.Text(m)
	for _, want := range []string{"Getting started", "1. Connect a platform", "→ tab to Settings"} {
		if !strings.Contains(text, want) {
			t.Errorf("fresh install should show %q:\n%s", want, text)
		}
	}

	m.platforms["bluesky"] = true
	steps := m.onboardingSteps()
	if !steps[0].done || steps[1].done {
		t.Errorf("connecting a platform completes step 1 only: %+v", steps)
	}
	if text := tuitest.Text(m); !strings.Contains(text, "→ n") {
		t.Errorf("the next open step carries the hint:\n%s", text)
	}

	m.posts = []models.Post{{ID: "p", Platform: "bluesky", Status: models.StatusDraft}}
	if m.needsOnboarding() {
		t.Error("with a platform and a post the checklist goes away")
	}
}

func TestCampaignRowShowsProgressFailuresAndNext(t *testing.T) {
	now := time.Now()
	soon := now.Add(48 * time.Hour)
	c := models.Campaign{Slug: "bruecke-events", Posted: 3, Posts: []models.Post{
		{Status: models.StatusPosted}, {Status: models.StatusPosted}, {Status: models.StatusPosted},
		{Status: models.StatusFailed}, {Status: models.StatusScheduled, ScheduledAt: &soon}, {Status: models.StatusDraft},
	}}
	row := ansi.Strip(campaignRow(c, 80, now))
	for _, want := range []string{"bruecke-events", "3/6", "✗1", "in 2d"} {
		if !strings.Contains(row, want) {
			t.Errorf("campaign row %q should contain %q", row, want)
		}
	}
}
