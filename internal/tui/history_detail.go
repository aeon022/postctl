package tui

import (
	"strings"

	"github.com/aeon022/missionctl-core/ui"
	"github.com/aeon022/postctl/internal/models"
)

// renderHistoryDetailView is the detail of one history entry: a scrollable
// focused panel with the metadata and the full output / error text.
func (m Model) renderHistoryDetailView(w, h int) string {
	if m.selectedHistory == nil {
		return ""
	}
	e := m.selectedHistory
	lines := historyDetailLines(e, panelRowW(w))
	return scrollPanel(w, h, Tr("panel_hdetail")+" · "+strings.ToUpper(e.Action), lines, m.detailScrollOffset, true)
}

// historyDetailLines is the metadata plus the full output / error text of one entry.
func historyDetailLines(e *models.HistoryEntry, iw int) []string {
	lines := []string{actionPill(e.Action), ""}
	lines = append(lines, kv(Tr("hd_timestamp"), e.CreatedAt.Format("02.01.2006 15:04:05")), kv(Tr("hd_post_id"), e.PostID))
	if e.PlatformID != "" {
		lines = append(lines, kv(Tr("hd_platform_id"), e.PlatformID))
	}
	lines = append(lines, "", ui.Divider(iw, Tr("panel_output")))
	text := e.Error
	if text == "" {
		text = Tr("hd_no_error")
	}
	for _, l := range wrapLines(text, iw-2) {
		lines = append(lines, "  "+l)
	}
	return lines
}
