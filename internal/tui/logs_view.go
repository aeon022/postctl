package tui

import (
	"github.com/aeon022/postctl/internal/platforms"
)

// renderLogs is the logs tab: the newest background log lines in one panel.
func (m Model) renderLogs(w, h int) string {
	platforms.LogMu.Lock()
	logs := make([]string, len(platforms.LogBuffer))
	copy(logs, platforms.LogBuffer)
	platforms.LogMu.Unlock()

	if len(logs) == 0 {
		return emptyBody(w, h, Tr("panel_logs"), Tr("logs_empty"))
	}
	room := max(h-2, 1)
	if len(logs) > room {
		logs = logs[len(logs)-room:]
	}
	return scrollPanel(w, h, Tr("panel_logs"), logs, 0, true)
}
