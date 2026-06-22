package tui

import (
	"fmt"
	"strings"

	"github.com/aeon022/postctl/internal/config"
	"github.com/charmbracelet/lipgloss"
)

// renderSettings zeichnet das Einstellungsmenü im Terminal
func (m Model) renderSettings() string {
	var builder strings.Builder

	builder.WriteString(StyleHeader.Render("EINSTELLUNGEN / CONFIGURATION") + "\n\n")

	licenseStatus := "Core (Gratis)"
	if config.IsPro() {
		licenseStatus = "Pro (Aktiv ✅)"
	}

	options := []struct {
		label string
		value string
	}{
		{"AI Provider", config.ActiveConfig.AI.Provider},
		{"AI Model   ", config.ActiveConfig.AI.Model},
		{"Dry Run    ", fmt.Sprintf("%t", config.ActiveConfig.Defaults.DryRun)},
		{"Lizenztyp  ", licenseStatus},
	}

	for i, opt := range options {
		cursorStr := "  "
		// Lizenztyp ist nicht auswählbar, daher Cursor unterdrücken, falls er darauf zeigen würde
		if i == m.cursor && i < 3 {
			cursorStr = "> "
		}

		labelStyle := lipgloss.NewStyle().Foreground(ColorText)
		if i == m.cursor && i < 3 {
			labelStyle = lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true)
		}

		valStyle := lipgloss.NewStyle().Foreground(ColorLightGray)
		if opt.label == "Lizenztyp  " {
			if config.IsPro() {
				valStyle = lipgloss.NewStyle().Foreground(ColorPosted).Bold(true)
			} else {
				valStyle = lipgloss.NewStyle().Foreground(ColorLightGray)
			}
		} else if opt.value == "true" {
			valStyle = lipgloss.NewStyle().Foreground(ColorPosted).Bold(true)
		} else if opt.value == "false" {
			valStyle = lipgloss.NewStyle().Foreground(ColorFailed).Bold(true)
		} else {
			valStyle = lipgloss.NewStyle().Foreground(ColorSecondary)
		}

		builder.WriteString(fmt.Sprintf("%s%s: %s\n", cursorStr, labelStyle.Render(opt.label), valStyle.Render(opt.value)))
	}

	builder.WriteString("\n")
	builder.WriteString(StyleHelp.Render("←/→ / enter: Werte ändern  ·  Änderungen werden sofort gespeichert.\nPro-Lizenz über CLI aktivieren:\n  postctl config set license_key <key>"))

	return StyleBox.Width(78).Height(14).Render(builder.String())
}
