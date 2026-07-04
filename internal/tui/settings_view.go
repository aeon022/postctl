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

	builder.WriteString(StyleHeader.Render(Tr("header_settings")) + "\n\n")

	licenseStatus := Tr("license_core")
	if config.IsPro() {
		licenseStatus = Tr("license_pro")
	}

	getPlatformStatus := func(p string) string {
		if m.platforms[p] {
			return Tr("dash_connected")
		}
		return Tr("dash_not_auth")
	}

	options := []struct {
		label    string
		value    string
		isAction bool
	}{
		{Tr("settings_ai_provider"), config.ActiveConfig.AI.Provider, false},
		{Tr("settings_ai_model"), config.ActiveConfig.AI.Model, false},
		{Tr("settings_dry_run"), fmt.Sprintf("%t", config.ActiveConfig.Defaults.DryRun), false},
		{Tr("settings_language"), config.ActiveConfig.Defaults.Language, false},
		{Tr("settings_license"), licenseStatus, false},
		{Tr("settings_auth_twitter"), getPlatformStatus(models.PlatformTwitter), true},
		{Tr("settings_auth_linkedin"), getPlatformStatus(models.PlatformLinkedIn), true},
		{Tr("settings_auth_threads"), getPlatformStatus(models.PlatformThreads), true},
		{Tr("settings_auth_mastodon"), getPlatformStatus(models.PlatformMastodon), true},
		{Tr("settings_auth_bluesky"), getPlatformStatus(models.PlatformBluesky), true},
		{Tr("settings_auth_facebook"), getPlatformStatus(models.PlatformFacebook), true},
		{Tr("settings_config_export"), Tr("settings_run_action"), true},
		{Tr("settings_config_import"), Tr("settings_run_action"), true},
	}

	for i, opt := range options {
		cursorStr := "  "
		// i == 4 ist Lizenztyp (nicht auswählbar)
		if i == m.cursor && i != 4 {
			cursorStr = "> "
		}

		labelStyle := lipgloss.NewStyle().Foreground(ColorLightGray)
		if i == m.cursor && i != 4 {
			labelStyle = lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true)
		}

		valStyle := lipgloss.NewStyle().Foreground(ColorLightGray)
		if i == 4 { // Lizenztyp
			if config.IsPro() {
				valStyle = lipgloss.NewStyle().Foreground(ColorPosted).Bold(true)
			} else {
				valStyle = lipgloss.NewStyle().Foreground(ColorLightGray)
			}
		} else if opt.isAction {
			if strings.Contains(opt.value, "✓") || strings.Contains(opt.value, "Verbunden") || strings.Contains(opt.value, "Connected") {
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
		if i == 4 {
			builder.WriteString("\n" + StyleHeader.Render("PLATFORM ACCOUNTS") + "\n")
		}
		// Einen kleinen visuellen Trenner vor Backup & Sync einfügen
		if i == 10 {
			builder.WriteString("\n" + StyleHeader.Render("BACKUP & SYNC") + "\n")
		}
	}

	builder.WriteString("\n")
	if m.statusMessage != "" {
		builder.WriteString(lipgloss.NewStyle().Foreground(ColorSecondary).Render(m.statusMessage) + "\n")
	}
	builder.WriteString(StyleHelp.Render(Tr("settings_help_footer")))

	return StyleBox.Width(78).Height(17).Render(builder.String())
}
