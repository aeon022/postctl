package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aeon022/missionctl-core/tuitest"
	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/platforms"
	"github.com/aeon022/postctl/internal/service"
)

func TestLogSeverity(t *testing.T) {
	for line, want := range map[string]int{
		"[SCHEDULER FEHLER] Kann geplante Posts nicht lesen": 2,
		"Veröffentlichung fehlgeschlagen: 401":               2,
		"17:30:36 Check übersprungen: locked":                1,
		"[SCHEDULER] Post x erfolgreich veröffentlicht.":     0,
	} {
		if got := logSeverity(line); got != want {
			t.Errorf("logSeverity(%q) = %d, want %d", line, got, want)
		}
	}
}

func TestLogsHeaderCountsAndSourceSwitchReadsTheServiceLog(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	prev := config.ActiveProfile
	config.ActiveProfile = "bruecke"
	t.Cleanup(func() { config.ActiveProfile = prev })

	platforms.LogMu.Lock()
	platforms.LogBuffer = []string{"[10:00:00] alles gut", "[10:00:01] Veröffentlichung fehlgeschlagen: 401", "[10:00:02] Check übersprungen: locked"}
	platforms.LogMu.Unlock()

	m := layoutModel(t, 120, 36)
	mm, _ := m.switchTab(tabLogs)
	text := tuitest.Text(mm)
	for _, want := range []string{"Status", "TUI", "✗ 1 errors", "⚠ 1 notices", "3 lines", "alles gut"} {
		if !strings.Contains(text, want) {
			t.Errorf("TUI log view should contain %q:\n%s", want, text)
		}
	}

	logPath := service.LogPath("bruecke")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(logPath, []byte("[SCHEDULER] Veröffentliche fälligen Post a\n[SCHEDULER] Post a erfolgreich veröffentlicht.\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	sw, _ := tuitest.Keys(mm, "s")
	dt := tuitest.Text(sw)
	for _, want := range []string{"Scheduler service", "erfolgreich veröffentlicht", "postctl-bruecke.log", "✓ 0 errors"} {
		if !strings.Contains(dt, want) {
			t.Errorf("service log view should contain %q:\n%s", want, dt)
		}
	}
	back, _ := tuitest.Keys(sw, "s")
	if back.(Model).logSource != logSourceTUI {
		t.Error("s toggles back to the TUI log")
	}
}
