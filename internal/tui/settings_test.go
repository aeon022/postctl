package tui

import (
	"fmt"
	"github.com/aeon022/missionctl-core/theme"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/models"
	"github.com/aeon022/postctl/internal/store"
)

func TestSettingsEnterKeyWithConfig(t *testing.T) {
	s, err := store.NewSQLiteStore(":memory:", false)
	if err != nil {
		t.Fatalf("sqlite memory store error: %v", err)
	}
	defer s.Close()

	// Vorbereitung der Credentials, damit direkt der Auth-Flow gestartet wird
	config.ActiveConfig.Bluesky.Handle = "test-handle"
	config.ActiveConfig.Bluesky.AppPassword = "test-password"

	m := NewModel(s)
	m.activeTab = tabSettings
	m.loading = false

	// Test case for Bluesky
	m.cursor = settingsIndex(t, m, models.PlatformBluesky)
	msg := tea.KeyPressMsg{Code: tea.KeyEnter}

	newModel, cmd := m.Update(msg)
	updatedModel := newModel.(Model)

	if !updatedModel.loading {
		t.Errorf("expected loading to be true after hitting Enter on Bluesky, got false")
	}

	if cmd == nil {
		t.Errorf("expected cmd to be returned, got nil")
	}

	// Trigger the command to see if it executes without crashing
	// (it should run in background and return authResultMsg)
	authRes, ok := firstMsg[authResultMsg](cmd)
	if !ok {
		t.Fatalf("expected the command (or a batched one) to produce an authResultMsg")
	}

	if authRes.platform != models.PlatformBluesky {
		t.Errorf("expected platform to be %s, got %s", models.PlatformBluesky, authRes.platform)
	}

	if authRes.err == nil {
		t.Errorf("expected error since handle/password are fake, got nil")
	} else {
		fmt.Printf("Success: Got expected error for Bluesky Auth: %v\n", authRes.err)
	}
}

func TestSettingsEnterKeyNeedsSetup(t *testing.T) {
	s, err := store.NewSQLiteStore(":memory:", false)
	if err != nil {
		t.Fatalf("sqlite memory store error: %v", err)
	}
	defer s.Close()

	// Credentials leeren, damit der Setup-Wizard gestartet wird
	config.ActiveConfig.Bluesky.Handle = ""
	config.ActiveConfig.Bluesky.AppPassword = ""

	m := NewModel(s)
	m.activeTab = tabSettings
	m.loading = false

	// Test case for Bluesky
	m.cursor = settingsIndex(t, m, models.PlatformBluesky)
	msg := tea.KeyPressMsg{Code: tea.KeyEnter}

	newModel, cmd := m.Update(msg)
	updatedModel := newModel.(Model)

	// Beim Setup-Wizard wird loading NICHT auf true gesetzt (die TUI wird stattdessen pausiert/suspendiert)
	if updatedModel.loading {
		t.Errorf("expected loading to remain false when starting setup wizard, got true")
	}

	if cmd == nil {
		t.Errorf("expected cmd to be returned, got nil")
	}

	// Das Kommando sollte vom Typ tea.execMsg sein, da tea.ExecProcess zurückgegeben wird
	// Da tea.execMsg im bubbletea-Paket nicht exportiert ist, können wir den Typ nicht direkt prüfen,
	// aber wir können sicherstellen, dass cmd nicht nil ist und das Modell im korrekten Zustand bleibt.
}

// TestSettingsAutoPublishToggle covers the new Scheduler.AutoPublish entry
// (cursor 3, between Dry Run and Language) inserted into the settings list —
// pins that Left/Right actually flips it and that every hardcoded cursor
// index shifted around it (License at 5, the platform-auth range 6-17,
// export/import/edit-slots at 18/19/20) still lines up; those are plain
// integer literals in app.go/settings_view.go with nothing enforcing they
// stay in sync with the options slice's real length, so a future insertion
// there could silently break navigation without this failing loudly.
//
// cycleSetting() calls config.SaveConfig() unconditionally, which writes to
// the real ~/.config/postctl/config.yaml for the default profile — routed
// through an isolated ActiveProfile here instead, cleaned up after, so this
// never touches the user's actual config (their real platform tokens).
func TestSettingsAutoPublishToggle(t *testing.T) {
	prevProfile := config.ActiveProfile
	config.ActiveProfile = "test-auto-publish-toggle"
	home, _ := os.UserHomeDir()
	testProfileDir := filepath.Join(home, ".config", "postctl", "profiles", config.ActiveProfile)
	t.Cleanup(func() {
		config.ActiveProfile = prevProfile
		os.RemoveAll(testProfileDir)
	})

	s, err := store.NewSQLiteStore(":memory:", false)
	if err != nil {
		t.Fatalf("sqlite memory store error: %v", err)
	}
	defer s.Close()

	config.ActiveConfig.Scheduler.AutoPublish = false

	m := NewModel(s)
	m.activeTab = tabSettings
	m.loading = false
	m.cursor = 3 // Auto-Publish

	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	m = newModel.(Model)

	if !config.ActiveConfig.Scheduler.AutoPublish {
		t.Error("expected Scheduler.AutoPublish to flip to true after Right on cursor 3, stayed false")
	}

	newModel, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	m = newModel.(Model)

	if config.ActiveConfig.Scheduler.AutoPublish {
		t.Error("expected Scheduler.AutoPublish to flip back to false after Left on cursor 3, stayed true")
	}

	// The cursor-5-is-License skip (Up/Down) and the platform-auth range
	// starting at 6 (Enter) are the two other things that shift when an
	// item is inserted before them — verify both landed correctly rather
	// than trusting the arithmetic.
	m.cursor = 5 // Theme
	newModel, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m = newModel.(Model)
	if m.cursor != 7 {
		t.Errorf("Down from Theme (cursor 5) = %d, want 7 (License at 6 must be skipped)", m.cursor)
	}
}

// TestSettingsOptions_Shape pins settingsOptions' row order and kind/
// platform/action assignments — the single source of truth Up/Down/Left/
// Right/Enter/Delete and the render loop all now derive from (see its doc
// comment for why: this replaced seven separately hardcoded cursor-index
// literals across app.go/settings_view.go that the Auto-Publish row
// addition had to update by hand, any one of which silently breaking
// navigation if missed). A regression here — a reordered or
// wrongly-classified row — would silently misroute a key press to the
// wrong platform/action instead of failing loudly, so it's worth pinning
// directly rather than only through the existing cursor-position tests.
func TestSettingsOptions_Shape(t *testing.T) {
	s, err := store.NewSQLiteStore(":memory:", false)
	if err != nil {
		t.Fatalf("sqlite memory store error: %v", err)
	}
	defer s.Close()
	m := NewModel(s)

	opts := m.settingsOptions()
	if len(opts) != 25 {
		t.Fatalf("len(settingsOptions()) = %d, want 25", len(opts))
	}

	wantCyclable := []string{"settings_ai_provider", "settings_ai_model", "settings_dry_run", "settings_auto_publish", "settings_language", "settings_theme"}
	for i, key := range wantCyclable {
		if opts[i].kind != settingCyclable || opts[i].settingKey != key {
			t.Errorf("opts[%d] = {kind:%v settingKey:%q}, want {settingCyclable %q}", i, opts[i].kind, opts[i].settingKey, key)
		}
	}

	if opts[6].kind != settingDisplay {
		t.Errorf("opts[6] (License) kind = %v, want settingDisplay", opts[6].kind)
	}

	wantPlatforms := []string{
		models.PlatformTwitter, models.PlatformLinkedIn, models.PlatformThreads, models.PlatformMastodon,
		models.PlatformBluesky, models.PlatformFacebook, models.PlatformInstagram, models.PlatformTikTok, models.PlatformTelegram, models.PlatformDiscord,
		models.PlatformDevTo, models.PlatformReddit, models.PlatformHashnode, models.PlatformMedium,
	}
	for i, p := range wantPlatforms {
		idx := 7 + i
		if opts[idx].kind != settingPlatformAuth || opts[idx].platform != p {
			t.Errorf("opts[%d] = {kind:%v platform:%q}, want {settingPlatformAuth %q}", idx, opts[idx].kind, opts[idx].platform, p)
		}
	}
	if opts[7].separatorBefore == "" {
		t.Error(`opts[7] (first platform row) missing separatorBefore ("PLATFORM ACCOUNTS")`)
	}

	wantActions := []settingActionID{actionExport, actionImport, actionEditSlots, actionScheduler}
	for i, a := range wantActions {
		idx := 21 + i
		if opts[idx].kind != settingAction || opts[idx].action != a {
			t.Errorf("opts[%d] = {kind:%v action:%v}, want {settingAction %v}", idx, opts[idx].kind, opts[idx].action, a)
		}
	}
	if opts[21].separatorBefore == "" {
		t.Error(`opts[21] (Export) missing separatorBefore ("BACKUP & SYNC")`)
	}
}

// firstMsg runs cmd (unpacking a batch, skipping the spinner tick) and returns the first message of type T.
func firstMsg[T any](cmd tea.Cmd) (T, bool) {
	var zero T
	if cmd == nil {
		return zero, false
	}
	switch msg := cmd().(type) {
	case tea.BatchMsg:
		for _, c := range msg {
			if c == nil {
				continue
			}
			if v, ok := firstMsg[T](c); ok {
				return v, true
			}
		}
	case T:
		return msg, true
	}
	return zero, false
}

// settingsIndex finds a platform's row by what it is, not by position: rows get added, and an
// Enter on the wrong row would run a real login flow (network, browser, config writes).
func settingsIndex(t *testing.T, m Model, platform string) int {
	t.Helper()
	for i, o := range m.settingsOptions() {
		if o.kind == settingPlatformAuth && o.platform == platform {
			return i
		}
	}
	t.Fatalf("no settings row for %s", platform)
	return -1
}

func TestThemeRowCyclesLiveAndSavesForTheSuite(t *testing.T) {
	s, err := store.NewSQLiteStore(":memory:", false)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	orig := theme.Current()
	t.Cleanup(func() { _ = theme.Apply(orig) })

	m := NewModel(s)
	m.activeTab = tabSettings
	for i, o := range m.settingsOptions() {
		if o.settingKey == "settings_theme" {
			m.cursor = i
		}
	}
	before := theme.Current()
	right, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	after := theme.Current()
	if after == before {
		t.Fatalf("→ must switch to the next preset (still %s)", after)
	}
	if got := right.(Model).statusMessage; !strings.Contains(got, after) {
		t.Errorf("status message should name the new theme, got %q", got)
	}
	if b, err := os.ReadFile(filepath.Join(os.Getenv("HOME"), ".config", "missionctl", "theme.yaml")); err != nil || !strings.Contains(string(b), "preset: "+after) {
		t.Errorf("the choice must be saved for every tool (err=%v file=%q)", err, b)
	}
	if _, _ = right.(Model).Update(tea.KeyPressMsg{Code: tea.KeyLeft}); theme.Current() != before {
		t.Errorf("← goes back to %s, got %s", before, theme.Current())
	}
	// the colors cached by this package follow the switch
	if ColorPrimary != theme.BlueV2 {
		t.Error("ColorPrimary must be refreshed by the theme change")
	}
}
