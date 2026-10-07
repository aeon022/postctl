package tui

import (
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/aeon022/missionctl-core/ui"
	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/models"
	"github.com/charmbracelet/x/ansi"
)

// renderSchedule renders the queue tab (tab 2): scheduled posts grouped by
// campaign, selectable rows with platform pill + relative time, and the same
// Preview panel as the posts tab from wideBreak columns.
func (m Model) renderSchedule(w, h int) string {
	if len(m.nextUp) == 0 {
		return emptyBody(w, h, "No posts currently scheduled.", Tr("queue_empty_hint"))
	}
	now := time.Now()

	listW := w
	if w >= wideBreak {
		listW = (w - 1) * 58 / 100
	}
	rowW := listW
	if w >= wideBreak {
		rowW = panelRowW(listW)
	}

	type lineItem struct {
		text      string
		postIndex int // index in m.nextUp, -1 for headers/blank lines
	}
	var items []lineItem
	var lastCampaign string
	for idx, p := range m.nextUp {
		if p.Campaign != lastCampaign || idx == 0 {
			lastCampaign = p.Campaign
			name := lastCampaign
			if name == "" {
				name = "INDIVIDUAL POSTS"
				if strings.ToLower(config.ActiveConfig.Defaults.Language) == "de" {
					name = "EINZELPOSTS"
				}
			}
			if idx > 0 {
				items = append(items, lineItem{"", -1})
			}
			items = append(items, lineItem{ui.Divider(rowW, "📁 "+strings.ToUpper(name)), -1})
		}
		items = append(items, lineItem{postIndex: idx})
	}

	selected := -1
	for i, it := range items {
		if it.postIndex == m.cursor {
			selected = i
		}
	}
	capacity := h - 1
	if w >= wideBreak {
		capacity = h - 2
	}
	start, end := window(len(items), selected, capacity)

	var rows []string
	for _, it := range items[start:end] {
		if it.postIndex < 0 {
			rows = append(rows, it.text)
			continue
		}
		rows = append(rows, m.queueRow(m.nextUp[it.postIndex], rowW, m.activeTab == 2 && it.postIndex == m.cursor, now))
	}

	if w >= wideBreak {
		var sel *models.Post
		if m.cursor >= 0 && m.cursor < len(m.nextUp) {
			sel = &m.nextUp[m.cursor]
		}
		left := ui.Panel(listW, h, "SCHEDULED POSTS", strings.Join(rows, "\n"), true)
		return joinColumns(left, m.previewPanel(sel, w-1-listW, h), listW, 1)
	}
	return strings.Join(append([]string{ui.Divider(w, "SCHEDULED POSTS")}, rows...), "\n")
}

func (m Model) queueRow(p models.Post, w int, selected bool, now time.Time) string {
	when := ""
	if p.ScheduledAt != nil {
		when = ui.RelTime(*p.ScheduledAt, now) + " " + p.ScheduledAt.Format("15:04")
	}
	left := dimStyle.Render(padRight(when, 16)) + padRight(platformPill(p.Platform), 11)
	title := ansi.Truncate(stripEmojis(p.Title), max(w-2-ansiW(left), 4), "…")
	return ui.Row(w, selected, left+title)
}

func ansiW(s string) int { return lipgloss.Width(s) }
