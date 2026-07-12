package tui

import (
	"strings"

	"github.com/aeon022/postctl/internal/platforms"
)

// renderLogs rendert die Log-Ansicht (Tab 6)
func (m Model) renderLogs() string {
	var builder strings.Builder

	builder.WriteString(StyleHeader.Render(Tr("header_logs")) + "\n")

	platforms.LogMu.Lock()
	logs := make([]string, len(platforms.LogBuffer))
	copy(logs, platforms.LogBuffer)
	platforms.LogMu.Unlock()

	boxHeight := m.getBoxHeight()
	maxLogLines := boxHeight - 6
	if maxLogLines < 5 {
		maxLogLines = 5
	}

	if len(logs) == 0 {
		builder.WriteString("  Keine Logs vorhanden. Hintergrund-Aktivitäten werden hier protokolliert.\n")
	} else {
		startIdx := len(logs) - maxLogLines
		if startIdx < 0 {
			startIdx = 0
		}
		for i := startIdx; i < len(logs); i++ {
			builder.WriteString("  " + logs[i] + "\n")
		}
	}

	return StyleBox.Width(84).Height(boxHeight).Render(builder.String())
}
