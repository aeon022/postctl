package tui

import (
	"fmt"
	"strings"

	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/models"
	"github.com/charmbracelet/lipgloss"
)

// settingKind classifies a settingsOptions() row for cursor movement and
// key handling — see settingsOptions' doc comment for why this replaced
// hardcoded index literals scattered across app.go and this file.
type settingKind int

const (
	settingCyclable     settingKind = iota // left/right cycles its value (cycleSetting)
	settingDisplay                         // not selectable at all (License)
	settingPlatformAuth                    // enter authenticates, delete resets
	settingAction                          // enter runs a one-off action
)

// settingActionID identifies which action an settingAction row runs — an
// index into this instead of the row's raw cursor position, so inserting a
// new action row doesn't require renumbering the others.
type settingActionID int

const (
	actionNone settingActionID = iota
	actionExport
	actionImport
	actionEditSlots
)

// settingOption is one row of the Settings tab (row 5). settingKey
// identifies a settingCyclable row for cycleSetting — reusing its i18n
// label key (settings_ai_provider, settings_dry_run, ...) since that's
// already a stable, unique-per-row identifier, rather than inventing a
// second one.
type settingOption struct {
	label           string
	value           string
	kind            settingKind
	settingKey      string // set when kind == settingCyclable
	platform        string // set when kind == settingPlatformAuth
	action          settingActionID
	separatorBefore string // section header printed above this row, if any
}

// settingsOptions is the single source of truth for the Settings tab's row
// order and behavior. renderSettings, maxCursorItems, cursor Up/Down
// (skipping the non-selectable License row), cycleSetting (left/right), and
// the Enter/Delete key handlers (platform auth range, action rows) all
// derive from this instead of separately hardcoded cursor-index literals —
// which is what the Auto-Publish row addition needed touching seven
// different places for, each a silent way to break navigation if missed.
// Inserting a new row now only ever means adding one entry here.
func (m Model) settingsOptions() []settingOption {
	licenseStatus := Tr("license_core")
	if config.IsPro() {
		licenseStatus = Tr("license_pro")
	}
	platformStatus := func(p string) string {
		if m.platforms[p] {
			return Tr("dash_connected")
		}
		return Tr("dash_not_auth")
	}
	return []settingOption{
		{label: Tr("settings_ai_provider"), value: config.ActiveConfig.AI.Provider, kind: settingCyclable, settingKey: "settings_ai_provider"},
		{label: Tr("settings_ai_model"), value: config.ActiveConfig.AI.Model, kind: settingCyclable, settingKey: "settings_ai_model"},
		{label: Tr("settings_dry_run"), value: fmt.Sprintf("%t", config.ActiveConfig.Defaults.DryRun), kind: settingCyclable, settingKey: "settings_dry_run"},
		{label: Tr("settings_auto_publish"), value: fmt.Sprintf("%t", config.ActiveConfig.Scheduler.AutoPublish), kind: settingCyclable, settingKey: "settings_auto_publish"},
		{label: Tr("settings_language"), value: config.ActiveConfig.Defaults.Language, kind: settingCyclable, settingKey: "settings_language"},
		{label: Tr("settings_license"), value: licenseStatus, kind: settingDisplay},
		{label: Tr("settings_auth_twitter"), value: platformStatus(models.PlatformTwitter), kind: settingPlatformAuth, platform: models.PlatformTwitter, separatorBefore: "PLATFORM ACCOUNTS"},
		{label: Tr("settings_auth_linkedin"), value: platformStatus(models.PlatformLinkedIn), kind: settingPlatformAuth, platform: models.PlatformLinkedIn},
		{label: Tr("settings_auth_threads"), value: platformStatus(models.PlatformThreads), kind: settingPlatformAuth, platform: models.PlatformThreads},
		{label: Tr("settings_auth_mastodon"), value: platformStatus(models.PlatformMastodon), kind: settingPlatformAuth, platform: models.PlatformMastodon},
		{label: Tr("settings_auth_bluesky"), value: platformStatus(models.PlatformBluesky), kind: settingPlatformAuth, platform: models.PlatformBluesky},
		{label: Tr("settings_auth_facebook"), value: platformStatus(models.PlatformFacebook), kind: settingPlatformAuth, platform: models.PlatformFacebook},
		{label: Tr("settings_auth_telegram"), value: platformStatus(models.PlatformTelegram), kind: settingPlatformAuth, platform: models.PlatformTelegram},
		{label: Tr("settings_auth_discord"), value: platformStatus(models.PlatformDiscord), kind: settingPlatformAuth, platform: models.PlatformDiscord},
		{label: Tr("settings_auth_devto"), value: platformStatus(models.PlatformDevTo), kind: settingPlatformAuth, platform: models.PlatformDevTo},
		{label: Tr("settings_auth_reddit"), value: platformStatus(models.PlatformReddit), kind: settingPlatformAuth, platform: models.PlatformReddit},
		{label: Tr("settings_auth_hashnode"), value: platformStatus(models.PlatformHashnode), kind: settingPlatformAuth, platform: models.PlatformHashnode},
		{label: Tr("settings_auth_medium"), value: platformStatus(models.PlatformMedium), kind: settingPlatformAuth, platform: models.PlatformMedium},
		{label: Tr("settings_config_export"), value: Tr("settings_run_action"), kind: settingAction, action: actionExport, separatorBefore: "BACKUP & SYNC"},
		{label: Tr("settings_config_import"), value: Tr("settings_run_action"), kind: settingAction, action: actionImport},
		{label: Tr("settings_edit_slots"), value: Tr("settings_run_action"), kind: settingAction, action: actionEditSlots},
	}
}

// renderSettings zeichnet das Einstellungsmenü im Terminal
func (m Model) renderSettings() string {
	if m.editingQueueSlots {
		var builder strings.Builder
		builder.WriteString(StyleHeader.Render("QUEUE-SLOTS BEARBEITEN (EDIT QUEUE SLOTS)") + "\n\n")

		var helpText string
		if config.ActiveConfig.Defaults.Language == "de" {
			helpText = "Gib die Slots kommagetrennt ein (Format: 'Tag HH:MM', z.B. 'Mon 09:00, Wed 14:00'):"
		} else {
			helpText = "Enter slots comma-separated (format: 'Day HH:MM', e.g. 'Mon 09:00, Wed 14:00'):"
		}
		builder.WriteString(lipgloss.NewStyle().Foreground(ColorLightGray).Render(helpText) + "\n\n")
		builder.WriteString(m.queueSlotsInput.View() + "\n\n\n\n\n\n\n\n")

		var keysText string
		if config.ActiveConfig.Defaults.Language == "de" {
			keysText = "Enter: Speichern  ·  Esc: Abbrechen"
		} else {
			keysText = "Enter: Save  ·  Esc: Cancel"
		}
		builder.WriteString(StyleHelp.Render(keysText))
		return StyleBox.Width(78).Height(21).Render(builder.String())
	}

	var builder strings.Builder

	builder.WriteString(StyleHeader.Render(Tr("header_settings")) + "\n\n")

	options := m.settingsOptions()

	for i, opt := range options {
		selectable := opt.kind != settingDisplay

		cursorStr := "  "
		if i == m.cursor && selectable {
			cursorStr = "> "
		}

		labelStyle := lipgloss.NewStyle().Foreground(ColorLightGray)
		if i == m.cursor && selectable {
			labelStyle = lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true)
		}

		valStyle := lipgloss.NewStyle().Foreground(ColorLightGray)
		switch {
		case opt.kind == settingDisplay: // Lizenztyp
			if config.IsPro() {
				valStyle = lipgloss.NewStyle().Foreground(ColorPosted).Bold(true)
			}
		case opt.kind == settingPlatformAuth || opt.kind == settingAction:
			if strings.Contains(opt.value, "✓") || strings.Contains(opt.value, "Verbunden") || strings.Contains(opt.value, "Connected") {
				valStyle = lipgloss.NewStyle().Foreground(ColorPosted).Bold(true)
			} else if opt.kind == settingPlatformAuth {
				valStyle = lipgloss.NewStyle().Foreground(ColorFailed)
			} else {
				valStyle = lipgloss.NewStyle().Foreground(ColorSecondary)
			}
		case opt.value == "true":
			valStyle = lipgloss.NewStyle().Foreground(ColorPosted).Bold(true)
		case opt.value == "false":
			valStyle = lipgloss.NewStyle().Foreground(ColorFailed).Bold(true)
		default:
			valStyle = lipgloss.NewStyle().Foreground(ColorSecondary)
		}

		if opt.separatorBefore != "" {
			builder.WriteString("\n" + StyleHeader.Render(opt.separatorBefore) + "\n")
		}
		builder.WriteString(fmt.Sprintf("%s%s: %s\n", cursorStr, labelStyle.Render(opt.label), valStyle.Render(opt.value)))
	}

	// 3. Scheduler Slots anzeigen
	slotsStr := strings.Join(config.ActiveConfig.Scheduler.Slots, ", ")
	if slotsStr == "" {
		slotsStr = "None"
	}
	builder.WriteString("\n" + StyleHeader.Render("SCHEDULER QUEUE SLOTS") + "\n")
	builder.WriteString(lipgloss.NewStyle().Foreground(ColorLightGray).Render("  "+slotsStr) + "\n\n")

	if m.statusMessage != "" {
		builder.WriteString(lipgloss.NewStyle().Foreground(ColorSecondary).Render(m.statusMessage) + "\n")
	}
	builder.WriteString(StyleHelp.Render(Tr("settings_help_footer")))

	return StyleBox.Width(78).Height(21).Render(builder.String())
}
