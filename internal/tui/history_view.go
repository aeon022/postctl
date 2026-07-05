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

	windowSize := 9
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

	return StyleBox.Width(84).Height(14).Render(builder.String())
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
		// Standard Kurzhilfe (zweizeilig)
		var line1 string
		if strings.ToLower(config.ActiveConfig.Defaults.Language) == "de" {
			line1 = "tab: Nächster Tab  ·  ↑↓: Navigieren  ·  enter: Wählen  ·  n: Neu  ·  e: Bearbeiten  ·  i: Import  ·  d: Löschen  ·  r: Umschreiben  ·  q: Beenden"
			if m.activeTab == 1 {
				if m.filterCampaign != "" {
					line1 = "esc: Filter löschen  ·  f: Filter wechseln  ·  " + line1
				} else {
					line1 = "f: Filter (Kampagne)  ·  " + line1
				}
			} else if m.activeTab == 3 {
				line1 = "tab: Nächster Tab  ·  ↑↓: Navigieren  ·  enter: Details öffnen  ·  x: Exportieren  ·  q: Beenden"
			}
			helpText := line1 + "\n" + "f1/R: Handbuch  ·  ?: Schnellhilfe"
			sb.WriteString(StyleHelp.Render(helpText))
		} else {
			line1 = "tab: next tab  ·  ↑↓: navigate  ·  enter: select  ·  n: new  ·  e: edit  ·  i: import  ·  d: delete  ·  r: repurpose  ·  q: quit"
			if m.activeTab == 1 {
				if m.filterCampaign != "" {
					line1 = "esc: clear filter  ·  f: change filter  ·  " + line1
				} else {
					line1 = "f: filter campaign  ·  " + line1
				}
			} else if m.activeTab == 3 {
				line1 = "tab: next tab  ·  ↑↓: navigate  ·  enter: view details  ·  x: export  ·  q: quit"
			}
			helpText := line1 + "\n" + "f1/R: readme  ·  ?: quick help"
			sb.WriteString(StyleHelp.Render(helpText))
		}
	}
	return sb.String()
}
