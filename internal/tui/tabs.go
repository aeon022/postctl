package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// RenderTabs zeichnet die Tab-Leiste oben auf dem Bildschirm
func RenderTabs(activeTab int) string {
	tabs := []string{
		Tr("tab_dashboard"),
		Tr("tab_posts"),
		Tr("tab_schedule"),
		Tr("tab_history"),
		Tr("tab_settings"),
	}
	
	var renderedTabs []string
	for i, name := range tabs {
		if i == activeTab {
			renderedTabs = append(renderedTabs, StyleTabActive.Render(name))
		} else {
			renderedTabs = append(renderedTabs, StyleTabInactive.Render(name))
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Bottom, renderedTabs...)
}
