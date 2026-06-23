package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderHistory rendert den History-Tab (Tab 3)
func (m Model) renderHistory() string {
	var builder strings.Builder

	builder.WriteString(StyleHeader.Render("POSTING HISTORY") + "\n")
	if len(m.history) == 0 {
		builder.WriteString("No posting history found.\n")
		return StyleBox.Width(78).Height(12).Render(builder.String())
	}

	for i, entry := range m.history {
		cursor := "  "
		selected := false
		if m.activeTab == 3 && i == m.cursor {
			cursor = "> "
			selected = true
		}

		timeStr := entry.CreatedAt.Format("02.01.2006 15:04:05")
		
		// Status Aktion formatieren (posted = grün, failed = rot)
		actionStr := entry.Action
		if entry.Action == "posted" {
			actionStr = lipgloss.NewStyle().Foreground(ColorPosted).Render(entry.Action)
		} else if entry.Action == "failed" {
			actionStr = lipgloss.NewStyle().Foreground(ColorFailed).Render(entry.Action)
		}

		infoText := fmt.Sprintf("Post: %s", entry.PostID)
		if entry.PlatformID != "" {
			infoText += fmt.Sprintf(" (ID: %s)", entry.PlatformID)
		}
		if entry.Error != "" {
			infoText += fmt.Sprintf(" - Error: %s", entry.Error)
		}

		itemStyle := lipgloss.NewStyle().Foreground(ColorLightGray)
		if selected {
			itemStyle = lipgloss.NewStyle().Foreground(ColorSecondary)
		}

		builder.WriteString(fmt.Sprintf("%s%s  %-8s %s\n", 
			cursor,
			timeStr,
			actionStr,
			itemStyle.Render(infoText),
		))
	}

	return StyleBox.Width(78).Height(12).Render(builder.String())
}

// renderHelp rendert die Tastaturbefehle am unteren Bildschirmrand
func (m Model) renderHelp() string {
	var sb strings.Builder

	if m.statusMessage != "" {
		sb.WriteString(lipgloss.NewStyle().Foreground(ColorSecondary).Render("💡 "+m.statusMessage) + "\n\n")
	}

	if m.showHelp {
		var builder strings.Builder
		builder.WriteString(StyleHeader.Render("KEYBOARD HELP") + "\n")
		builder.WriteString("  tab        Next Tab\n")
		builder.WriteString("  shift+tab  Previous Tab\n")
		builder.WriteString("  ↑/k        Move Up\n")
		builder.WriteString("  ↓/j        Move Down\n")
		builder.WriteString("  enter      Select / Open Preview (on Posts list) / Filter by campaign (on Dashboard)\n")
		builder.WriteString("  n          Create a new post draft\n")
		builder.WriteString("  e          Edit selected post draft\n")
		builder.WriteString("  i          Import posts from Markdown files/folders (pauses TUI)\n")
		builder.WriteString("  d          Delete selected post\n")
		builder.WriteString("  r          Repurpose selected post via AI to other platforms\n")
		builder.WriteString("  esc        Close Preview / Clear filter\n")
		builder.WriteString("  f1/R       Open complete README documentation with TOC\n")
		builder.WriteString("  ?          Toggle Quick Help\n")
		builder.WriteString("  q/ctrl+c   Quit application\n")
		sb.WriteString(StyleHelp.Render(builder.String()))
	} else {
		// Standard Kurzhilfe (zweizeilig)
		line1 := "tab: next tab  ·  ↑↓: navigate  ·  enter: select  ·  n: new  ·  e: edit  ·  i: import  ·  d: delete  ·  r: repurpose  ·  q: quit"
		if m.activeTab == 1 && m.filterCampaign != "" {
			line1 = "esc: clear filter  ·  " + line1
		}
		helpText := line1 + "\n" + "f1/R: readme  ·  ?: quick help"
		sb.WriteString(StyleHelp.Render(helpText))
	}
	return sb.String()
}
