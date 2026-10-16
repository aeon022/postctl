package tui

import (
	"fmt"

	"charm.land/lipgloss/v2"
	"github.com/aeon022/missionctl-core/theme"
)

// First-run checklist: until the user has connected a platform and created a post, the
// dashboard's campaign panel shows the four steps with ✓ for what is already done.

type onboardingStep struct {
	done bool
	text string // Tr key
	hint string // Tr key
}

func (m Model) onboardingSteps() []onboardingStep {
	connected := false
	for _, ok := range m.platforms {
		if ok {
			connected = true
			break
		}
	}
	return []onboardingStep{
		{connected, "onb_connect", "onb_connect_hint"},
		{len(m.posts) > 0, "onb_create", "onb_create_hint"},
		{m.stats.scheduled > 0 || m.stats.posted > 0, "onb_schedule", "onb_schedule_hint"},
		{m.health.schedulerOn || m.health.autoPublish, "onb_scheduler", "onb_scheduler_hint"},
	}
}

// needsOnboarding: show the checklist while the first steps are open (nothing connected or no post yet).
func (m Model) needsOnboarding() bool {
	s := m.onboardingSteps()
	return !s[0].done || !s[1].done
}

func (m Model) onboardingLines() []string {
	green := lipgloss.NewStyle().Foreground(theme.GreenV2)
	out := []string{dimStyle.Render(Tr("onb_intro")), ""}
	next := true
	for i, st := range m.onboardingSteps() {
		mark := dimStyle.Render("○")
		if st.done {
			mark = green.Render("✓")
		}
		line := fmt.Sprintf("%s %d. %s", mark, i+1, Tr(st.text))
		if !st.done && next {
			line = lipgloss.NewStyle().Bold(true).Render(line) + "  " + dimStyle.Render("→ "+Tr(st.hint))
			next = false
		}
		out = append(out, line)
	}
	return out
}
