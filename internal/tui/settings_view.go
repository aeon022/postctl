package tui

import (
	"fmt"
	"strings"

	"github.com/aeon022/missionctl-core/ui"
	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/models"
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
		{label: Tr("settings_auth_twitter"), value: platformStatus(models.PlatformTwitter), kind: settingPlatformAuth, platform: models.PlatformTwitter, separatorBefore: Tr("settings_sec_platforms")},
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
		{label: Tr("settings_config_export"), value: Tr("settings_run_action"), kind: settingAction, action: actionExport, separatorBefore: Tr("settings_sec_backup")},
		{label: Tr("settings_config_import"), value: Tr("settings_run_action"), kind: settingAction, action: actionImport},
		{label: Tr("settings_edit_slots"), value: Tr("settings_run_action"), kind: settingAction, action: actionEditSlots},
	}
}

// renderSettings is the settings tab: one panel with selectable rows grouped
// by dividers, the scheduler slots and the support/license notes. While the
// slots are edited it shows the input instead.
func (m Model) renderSettings(w, h int) string {
	rw := panelRowW(w)
	if m.editingQueueSlots {
		lines := append(wrapLines(dimStyle.Render(Tr("slots_help")), rw), "", m.queueSlotsInput.View())
		return ui.Panel(w, h, Tr("panel_slots"), strings.Join(lines, "\n"), true)
	}

	var lines []string
	selLine := 0
	for i, opt := range m.settingsOptions() {
		if opt.separatorBefore != "" {
			lines = append(lines, "", ui.Divider(rw, opt.separatorBefore))
		}
		sel := i == m.cursor && opt.kind != settingDisplay
		if sel {
			selLine = len(lines)
		}
		lines = append(lines, ui.Row(rw, sel, padRight(opt.label, 26)+m.settingValue(opt)))
	}

	slots := strings.Join(config.ActiveConfig.Scheduler.Slots, ", ")
	if slots == "" {
		slots = Tr("settings_none")
	}
	lines = append(lines, "", ui.Divider(rw, Tr("settings_sec_slots")), "  "+slots, "")
	_, notes, _ := strings.Cut(Tr("settings_help_footer"), "\n") // first line = key hints, now in the footer
	for _, l := range strings.Split(notes, "\n") {
		lines = append(lines, wrapLines(dimStyle.Render(l), rw)...)
	}
	return scrollPanel(w, h, Tr("panel_settings"), lines, aroundLine(len(lines), selLine, max(h-2, 1)), true)
}

// settingValue colors a row's value: connected/true/pro green, off/not
// connected red, everything else plain.
func (m Model) settingValue(opt settingOption) string {
	switch {
	case opt.kind == settingDisplay:
		if config.IsPro() {
			return ui.Pill(opt.value, ui.OK)
		}
		return dimStyle.Render(opt.value)
	case opt.kind == settingPlatformAuth:
		if m.platforms[opt.platform] {
			return ui.Dot(ui.OK) + " " + opt.value
		}
		return ui.Dot(ui.Muted) + " " + dimStyle.Render(opt.value)
	case opt.kind == settingAction:
		return dimStyle.Render(opt.value)
	case opt.value == "true":
		return ui.Dot(ui.OK) + " " + opt.value
	case opt.value == "false":
		return ui.Dot(ui.Err) + " " + opt.value
	}
	return opt.value
}
