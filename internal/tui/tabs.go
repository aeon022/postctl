package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// RenderTabs zeichnet die Tab-Leiste oben auf dem Bildschirm
func RenderTabs(activeTab int) string {
	tabs := []string{
		Tr("tab_dashboard"),
		Tr("tab_posts"),
		Tr("tab_schedule"),
		Tr("tab_history"),
		Tr("tab_analytics"),
		Tr("tab_settings"),
		Tr("tab_logs"),
	}
	
	var renderedTabs []string
	for i, name := range tabs {
		label := " " + strings.ToUpper(name) + " "
		if i == activeTab {
			styled := lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorSecondary).
				Render(label)
			renderedTabs = append(renderedTabs, styled)
		} else {
			styled := lipgloss.NewStyle().
				Foreground(ColorLightGray).
				Render(label)
			renderedTabs = append(renderedTabs, styled)
		}
	}

	// Trennzeichen
	divider := lipgloss.NewStyle().Foreground(ColorDarkGray).Render("│")
	tabRow := strings.Join(renderedTabs, divider)
	
	// Horizontale Trennlinie passend zur Boxbreite (78 Zeichen)
	bottomLine := lipgloss.NewStyle().Foreground(ColorPrimary).Render(strings.Repeat("─", 78))

	return tabRow + "\n" + bottomLine
}
