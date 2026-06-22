package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderSchedule rendert den Schedule-Tab (Tab 2)
func (m Model) renderSchedule() string {
	var builder strings.Builder

	builder.WriteString(StyleHeader.Render("SCHEDULED POSTS") + "\n")
	if len(m.nextUp) == 0 {
		builder.WriteString("No posts currently scheduled.\n")
		return StyleBox.Width(78).Height(12).Render(builder.String())
	}

	for i, p := range m.nextUp {
		cursor := "  "
		selected := false
		if m.activeTab == 2 && i == m.cursor {
			cursor = "> "
			selected = true
		}

		timeStr := ""
		if p.ScheduledAt != nil {
			timeStr = p.ScheduledAt.Format("02.01.2006 15:04")
		}

		titlePreview := p.Title
		if len(titlePreview) > 40 {
			titlePreview = titlePreview[:37] + "..."
		}

		itemStyle := lipgloss.NewStyle().Foreground(ColorLightGray)
		if selected {
			itemStyle = lipgloss.NewStyle().Foreground(ColorSecondary)
		}

		builder.WriteString(fmt.Sprintf("%s◷ %-17s %-8s %-2s  %s\n", 
			cursor,
			timeStr,
			itemStyle.Render(strings.ToUpper(p.Platform)),
			strings.ToUpper(p.Language),
			titlePreview,
		))
	}

	return StyleBox.Width(78).Height(12).Render(builder.String())
}
