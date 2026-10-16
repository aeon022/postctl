package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/aeon022/missionctl-core/ui"
	"github.com/aeon022/postctl/internal/models"
	"github.com/charmbracelet/x/ansi"
)

// actionPill is a history action as a fixed-width pill: posted ok, failed err.
func actionPill(action string) string {
	k := ui.Muted
	switch action {
	case "posted":
		k = ui.OK
	case "failed":
		k = ui.Err
	}
	return ui.Pill(padRight(strings.ToUpper(action), 8), k)
}

// historyInfo is the one-line "Post: id (ID: x) – error" text of an entry.
func historyInfo(e models.HistoryEntry) string {
	info := "Post: " + e.PostID
	if e.PlatformID != "" {
		info += " (ID: " + e.PlatformID + ")"
	}
	if e.Error != "" {
		errText, _, _ := strings.Cut(e.Error, "\n")
		info += " – " + errText
	}
	return info
}

// renderHistory is the history tab: one panel, a windowed list of selectable rows.
func (m Model) renderHistory(w, h int) string {
	if len(m.history) == 0 {
		return emptyBody(w, h, Tr("history_none_found"), "")
	}
	if w >= wideBreak { // list on the left, the selected entry's details on the right
		lw := w * 11 / 20
		rw := w - lw - 1
		sel := m.history[min(m.cursor, len(m.history)-1)]
		detail := ui.Panel(rw, h, Tr("panel_hdetail")+" · "+strings.ToUpper(sel.Action),
			strings.Join(historyDetailLines(&sel, panelRowW(rw))[:min(len(historyDetailLines(&sel, panelRowW(rw))), max(h-2, 1))], "\n"), false)
		return joinColumns(m.historyList(lw, h), detail, lw, 1)
	}
	return m.historyList(w, h)
}

// historyList is the windowed list of selectable history rows in one panel.
func (m Model) historyList(w, h int) string {
	rw := panelRowW(w)
	start, end := window(len(m.history), m.cursor, max(h-2, 1))
	var rows []string
	for i := start; i < end; i++ {
		e := m.history[i]
		sel := m.activeTab == tabHistory && i == m.cursor
		pre := dimStyle.Render(e.CreatedAt.Format("02.01.2006 15:04")) + " " + actionPill(e.Action) + " "
		info := ansi.Truncate(historyInfo(e), max(rw-lipgloss.Width(pre)-2, 4), "…")
		rows = append(rows, ui.Row(rw, sel, pre+info))
	}
	return ui.Panel(w, h, Tr("panel_history"), strings.Join(rows, "\n"), true)
}
