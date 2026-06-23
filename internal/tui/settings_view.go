package tui

import (
	"fmt"
	"strings"

	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/models"
	"github.com/charmbracelet/lipgloss"
)

// renderSettings zeichnet das Einstellungsmenü im Terminal
func (m Model) renderSettings() string {
	var builder strings.Builder

	builder.WriteString(StyleHeader.Render("EINSTELLUNGEN & VERBINDUNGEN") + "\n\n")

	licenseStatus := "Core (Gratis)"
	if config.IsPro() {
		licenseStatus = "Pro (Aktiv ✅)"
	}

	getPlatformStatus := func(p string) string {
		if m.platforms[p] {
			return "Verbunden ✓"
		}
		return "Nicht verbunden (Enter drücken)"
	}

	options := []struct {
		label    string
		value    string
		isAction bool
	}{
		{"AI Provider  ", config.ActiveConfig.AI.Provider, false},
		{"AI Model     ", config.ActiveConfig.AI.Model, false},
		{"Dry Run      ", fmt.Sprintf("%t", config.ActiveConfig.Defaults.DryRun), false},
		{"Lizenztyp    ", licenseStatus, false},
		{"Twitter/X    ", getPlatformStatus(models.PlatformTwitter), true},
		{"LinkedIn     ", getPlatformStatus(models.PlatformLinkedIn), true},
		{"Threads      ", getPlatformStatus(models.PlatformThreads), true},
		{"Backup Exp.  ", "Ausführen (Enter drücken)", true},
		{"Backup Imp.  ", "Ausführen (Enter drücken)", true},
	}

	for i, opt := range options {
		cursorStr := "  "
		// i == 3 ist Lizenztyp (nicht auswählbar)
		if i == m.cursor && i != 3 {
			cursorStr = "> "
		}

		labelStyle := lipgloss.NewStyle().Foreground(ColorText)
		if i == m.cursor && i != 3 {
			labelStyle = lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true)
		}

		valStyle := lipgloss.NewStyle().Foreground(ColorLightGray)
		if opt.label == "Lizenztyp    " {
			if config.IsPro() {
				valStyle = lipgloss.NewStyle().Foreground(ColorPosted).Bold(true)
			} else {
				valStyle = lipgloss.NewStyle().Foreground(ColorLightGray)
			}
		} else if opt.isAction {
			if strings.Contains(opt.value, "✓") {
				valStyle = lipgloss.NewStyle().Foreground(ColorPosted).Bold(true)
			} else {
				valStyle = lipgloss.NewStyle().Foreground(ColorFailed)
			}
		} else if opt.value == "true" {
			valStyle = lipgloss.NewStyle().Foreground(ColorPosted).Bold(true)
		} else if opt.value == "false" {
			valStyle = lipgloss.NewStyle().Foreground(ColorFailed).Bold(true)
		} else {
			valStyle = lipgloss.NewStyle().Foreground(ColorSecondary)
		}

		builder.WriteString(fmt.Sprintf("%s%s: %s\n", cursorStr, labelStyle.Render(opt.label), valStyle.Render(opt.value)))

		// Einen kleinen visuellen Trenner vor den Plattformen einfügen
		if i == 3 {
			builder.WriteString("\n" + StyleHeader.Render("PLATFORM ACCOUNTS") + "\n")
		}
		// Einen kleinen visuellen Trenner vor Backup & Sync einfügen
		if i == 6 {
			builder.WriteString("\n" + StyleHeader.Render("BACKUP & SYNC") + "\n")
		}
	}

	builder.WriteString("\n")
	if m.statusMessage != "" {
		builder.WriteString(lipgloss.NewStyle().Foreground(ColorSecondary).Render(m.statusMessage) + "\n")
	}
	builder.WriteString(StyleHelp.Render("←/→ / enter: Werte ändern / Verbinden  ·  Änderungen werden sofort gespeichert.\nPro-Lizenz über CLI aktivieren: postctl config set license_key <key>"))

	return StyleBox.Width(78).Height(17).Render(builder.String())
}
