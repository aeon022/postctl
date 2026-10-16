package tui

import (
	"io"
	"os"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/aeon022/missionctl-core/theme"
	"github.com/aeon022/missionctl-core/ui"
	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/platforms"
	"github.com/aeon022/postctl/internal/service"
	"github.com/charmbracelet/x/ansi"
)

// Logs tab: a header with the source switch, counts and the scheduler's state, and below it the
// newest lines — either this TUI's own log or the tail of the background scheduler's log file
// (s switches). Errors are red, skipped/warning lines amber.

const (
	logSourceTUI = iota
	logSourceDaemon
)

func logSeverity(line string) int {
	l := strings.ToLower(line)
	switch {
	case strings.Contains(l, "fehler") || strings.Contains(l, "fehlgeschlagen") || strings.Contains(l, "failed") || strings.Contains(l, "error"):
		return 2
	case strings.Contains(l, "übersprungen") || strings.Contains(l, "skip") || strings.Contains(l, "warn") || strings.Contains(l, "⚠"):
		return 1
	}
	return 0
}

// tailFile returns the last n lines of a file (reading at most the last 64 KB) and its mtime.
func tailFile(path string, n int) ([]string, time.Time) {
	f, err := os.Open(path)
	if err != nil {
		return nil, time.Time{}
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, time.Time{}
	}
	if off := st.Size() - 64*1024; off > 0 {
		_, _ = f.Seek(off, io.SeekStart)
	}
	b, _ := io.ReadAll(f)
	lines := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines, st.ModTime()
}

func (m Model) logLines(limit int) (lines []string, source string, mtime time.Time) {
	if m.logSource == logSourceDaemon {
		path := service.LogPath(config.ActiveProfile)
		lines, mtime = tailFile(path, limit)
		return lines, path, mtime
	}
	platforms.LogMu.Lock()
	lines = append([]string(nil), platforms.LogBuffer...)
	platforms.LogMu.Unlock()
	if len(lines) > limit {
		lines = lines[len(lines)-limit:]
	}
	return lines, "TUI", time.Time{}
}

func (m Model) logsHeader(w int, lines []string, source string, mtime time.Time) string {
	errs, warns := 0, 0
	for _, l := range lines {
		switch logSeverity(l) {
		case 2:
			errs++
		case 1:
			warns++
		}
	}
	tab := func(label string, on bool) string {
		if on {
			return ui.Pill(label, ui.Info)
		}
		return dimStyle.Render(" " + label + " ")
	}
	row1 := tab(Tr("logs_src_tui"), m.logSource == logSourceTUI) + " " + tab(Tr("logs_src_daemon"), m.logSource == logSourceDaemon) + "   "
	if errs > 0 {
		row1 += lipgloss.NewStyle().Foreground(theme.RedV2).Render("✗ "+itoa(errs)+" "+Tr("logs_errors")) + "  "
	} else {
		row1 += dimStyle.Render("✓ 0 " + Tr("logs_errors") + "  ")
	}
	if warns > 0 {
		row1 += amberStyle().Render("⚠ "+itoa(warns)+" "+Tr("logs_warnings")) + "  "
	}
	row1 += dimStyle.Render(itoa(len(lines)) + " " + Tr("logs_lines"))

	state := dimStyle.Render("○ " + Tr("health_sched_off"))
	if m.health.schedulerOn {
		state = lipgloss.NewStyle().Foreground(theme.GreenV2).Render("● ") + dimStyle.Render(Tr("health_sched_on"))
	}
	row2 := state
	if m.logSource == logSourceDaemon {
		src := source
		if home, err := os.UserHomeDir(); err == nil {
			src = strings.Replace(src, home, "~", 1)
		}
		row2 += dimStyle.Render("  ·  " + ansi.Truncate(src, max(w-40, 12), "…"))
		if !mtime.IsZero() {
			row2 += dimStyle.Render("  ·  " + ui.RelTime(mtime, time.Now()))
		}
	}
	return ui.Panel(w, 4, Tr("panel_logs_status"), row1+"\n"+row2, false)
}

// renderLogs is the logs tab.
func (m Model) renderLogs(w, h int) string {
	showHeader := h >= 10
	room := max(h-2, 1)
	if showHeader {
		room = max(h-4-2, 1)
	}
	lines, source, mtime := m.logLines(max(room*4, 200)) // header counts cover more than the visible lines
	header := ""
	if showHeader {
		header = m.logsHeader(w, lines, source, mtime) + "\n"
	}
	body := lines
	if len(body) > room {
		body = body[len(body)-room:]
	}
	if len(lines) == 0 {
		empty := Tr("logs_empty")
		if m.logSource == logSourceDaemon {
			empty = Tr("logs_daemon_empty")
		}
		if showHeader {
			return header + ui.Panel(w, max(h-4, 3), Tr("panel_logs"), dimStyle.Render(empty), true)
		}
		return emptyBody(w, h, Tr("panel_logs"), empty)
	}
	styled := make([]string, len(body))
	for i, l := range body {
		l = ansi.Truncate(l, max(w-4, 8), "…")
		switch logSeverity(l) {
		case 2:
			l = lipgloss.NewStyle().Foreground(theme.RedV2).Render(l)
		case 1:
			l = amberStyle().Render(l)
		}
		styled[i] = l
	}
	ph := h
	if showHeader {
		ph = h - 4
	}
	return header + scrollPanel(w, ph, Tr("panel_logs"), styled, 0, true)
}
