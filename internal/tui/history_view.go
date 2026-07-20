package tui

import (
	"fmt"
	"strings"

	"github.com/aeon022/postctl/internal/config"
	"github.com/charmbracelet/lipgloss"
)

// renderHistory rendert den History-Tab (Tab 3)
func (m Model) renderHistory() string {
	var builder strings.Builder

	builder.WriteString(StyleHeader.Render(Tr("header_history")) + "\n")
	if len(m.history) == 0 {
		builder.WriteString(Tr("history_none_found"))
		return StyleBox.Width(84).Height(14).Render(builder.String())
	}

	boxHeight := m.getBoxHeight()
	windowSize := boxHeight - 6
	if windowSize < 5 {
		windowSize = 5
	}
	startIdx := 0
	endIdx := len(m.history)

	if len(m.history) > windowSize {
		startIdx = m.cursor - windowSize/2
		if startIdx < 0 {
			startIdx = 0
		}
		if startIdx+windowSize > len(m.history) {
			startIdx = len(m.history) - windowSize
		}
		endIdx = startIdx + windowSize
	}

	for i := startIdx; i < endIdx; i++ {
		entry := m.history[i]
		cursor := "  "
		selected := false
		if m.activeTab == 3 && i == m.cursor {
			cursor = "> "
			selected = true
		}

		timeStr := entry.CreatedAt.Format("02.01.2006 15:04:05")
		
		// Status Aktion formatieren (posted = grün, failed = rot)
		actionFormatted := fmt.Sprintf("%-8s", entry.Action)
		actionStr := actionFormatted
		if entry.Action == "posted" {
			actionStr = lipgloss.NewStyle().Foreground(ColorPosted).Render(actionFormatted)
		} else if entry.Action == "failed" {
			actionStr = lipgloss.NewStyle().Foreground(ColorFailed).Render(actionFormatted)
		}

		infoText := fmt.Sprintf("Post: %s", entry.PostID)
		if entry.PlatformID != "" {
			infoText += fmt.Sprintf(" (ID: %s)", entry.PlatformID)
		}
		if entry.Error != "" {
			errText := entry.Error
			if idx := strings.Index(errText, "\n"); idx != -1 {
				errText = errText[:idx]
			}
			infoText += fmt.Sprintf(" - Error: %s", errText)
		}

		// Truncate infoText so the entire entry fits on a single line
		maxInfoLen := 38
		if len(infoText) > maxInfoLen {
			infoText = infoText[:maxInfoLen-3] + "..."
		}

		itemStyle := lipgloss.NewStyle().Foreground(ColorLightGray)
		if selected {
			itemStyle = lipgloss.NewStyle().Foreground(ColorSecondary)
		}

		builder.WriteString(fmt.Sprintf("%s%s  %s %s\n", 
			cursor,
			timeStr,
			actionStr,
			itemStyle.Render(infoText),
		))
	}

	return StyleBox.Width(84).Height(boxHeight).Render(builder.String())
}

// renderHelp rendert die Tastaturbefehle am unteren Bildschirmrand
func (m Model) renderHelp() string {
	var sb strings.Builder

	if m.statusMessage != "" {
		sb.WriteString(lipgloss.NewStyle().Foreground(ColorSecondary).Render("💡 "+m.statusMessage) + "\n\n")
	}

	if m.showHelp {
		var builder strings.Builder
		builder.WriteString(StyleHeader.Render(Tr("help_title")) + "\n")
		builder.WriteString("  tab        " + Tr("help_tab") + "\n")
		builder.WriteString("  shift+tab  " + Tr("help_shifttab") + "\n")
		builder.WriteString("  ↑/k        " + Tr("help_up") + "\n")
		builder.WriteString("  ↓/j        " + Tr("help_down") + "\n")
		builder.WriteString("  enter      " + Tr("help_enter") + "\n")
		builder.WriteString("  n          " + Tr("help_new_post") + "\n")
		builder.WriteString("  e          " + Tr("help_edit_post") + "\n")
		builder.WriteString("  s          " + Tr("help_schedule") + "\n")
		builder.WriteString("  p          " + Tr("help_post") + "\n")
		builder.WriteString("  i          " + Tr("help_import") + "\n")
		builder.WriteString("  d          " + Tr("help_delete") + "\n")
		builder.WriteString("  r          " + Tr("help_repurpose") + "\n")
		builder.WriteString("  f          " + Tr("help_filter") + "\n")
		builder.WriteString("  esc        " + Tr("help_esc") + "\n")
		builder.WriteString("  f1/R       " + Tr("help_readme") + "\n")
		builder.WriteString("  ?          " + Tr("help_toggle") + "\n")
		builder.WriteString("  q/ctrl+c   " + Tr("help_quit") + "\n")
		sb.WriteString(StyleHelp.Render(builder.String()))
	} else {
		// Standard Kurzhilfe (zweizeilig & ausgewogen)
		var line1, line2 string
		if strings.ToLower(config.ActiveConfig.Defaults.Language) == "de" {
			if m.activeTab == 1 { // Posts
				if m.filterCampaign != "" {
					line1 = "esc: Filter löschen  ·  f: Filter wechseln  ·  tab: Nächster Tab  ·  ↑↓: Navigieren  ·  enter: Wählen"
				} else {
					line1 = "f: Filter (Kampagne)  ·  tab: Nächster Tab  ·  ↑↓: Navigieren  ·  enter: Wählen"
				}
				line2 = "n: Neu  ·  e: Bearbeiten  ·  s: Einplanen  ·  p: Sofort posten  ·  d: Löschen  ·  r: Umschreiben  ·  i: Import  ·  f1/R: Handbuch  ·  ?: Hilfe  ·  q: Beenden"
			} else if m.activeTab == 3 { // History
				line1 = "tab: Nächster Tab  ·  ↑↓: Navigieren  ·  enter: Details öffnen  ·  x: Exportieren"
				line2 = "f1/R: Handbuch  ·  ?: Schnellhilfe  ·  q: Beenden"
			} else { // Default
				line1 = "tab: Nächster Tab  ·  ↑↓: Navigieren  ·  enter: Wählen  ·  n: Neu  ·  e: Bearbeiten  ·  d: Löschen"
				line2 = "r: Umschreiben  ·  i: Import  ·  f1/R: Handbuch  ·  ?: Schnellhilfe  ·  q: Beenden"
			}
			helpText := line1 + "\n" + line2
			sb.WriteString(StyleHelp.Render(helpText))
		} else {
			if m.activeTab == 1 { // Posts
				if m.filterCampaign != "" {
					line1 = "esc: clear filter  ·  f: change filter  ·  tab: next tab  ·  ↑↓: navigate  ·  enter: select"
				} else {
					line1 = "f: filter campaign  ·  tab: next tab  ·  ↑↓: navigate  ·  enter: select"
				}
				line2 = "n: new  ·  e: edit  ·  s: schedule  ·  p: post now  ·  d: delete  ·  r: repurpose  ·  i: import  ·  f1/R: readme  ·  ?: help  ·  q: quit"
			} else if m.activeTab == 3 { // History
				line1 = "tab: next tab  ·  ↑↓: navigate  ·  enter: view details  ·  x: export"
				line2 = "f1/R: readme  ·  ?: quick help  ·  q: quit"
			} else { // Default
				line1 = "tab: next tab  ·  ↑↓: navigate  ·  enter: select  ·  n: new  ·  e: edit  ·  d: delete"
				line2 = "r: repurpose  ·  i: import  ·  f1/R: readme  ·  ?: quick help  ·  q: quit"
			}
			helpText := line1 + "\n" + line2
			sb.WriteString(StyleHelp.Render(helpText))
		}
	}
	return sb.String()
}
