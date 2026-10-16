package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/aeon022/missionctl-core/theme"
	"github.com/aeon022/missionctl-core/ui"
	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/models"
	"github.com/charmbracelet/x/ansi"
)

// The calendar tab: a month grid where every post is a dot in its platform's brand
// color (● scheduled, ○ posted, ✗ failed, · draft), and the posts of the selected
// day listed below. Arrow keys move, [ ] change month, t jumps to today.

func dayStart(t time.Time) time.Time {
	y, mo, d := t.Date()
	return time.Date(y, mo, d, 0, 0, 0, 0, time.Local)
}

func dayKey(t time.Time) string { return t.Format("2006-01-02") }

// postDay is the day a post belongs to: when it went out, else when it is planned.
func postDay(p models.Post) (time.Time, bool) {
	if p.Status == models.StatusPosted && p.PostedAt != nil {
		return dayStart(p.PostedAt.Local()), true
	}
	if p.ScheduledAt != nil {
		return dayStart(p.ScheduledAt.Local()), true
	}
	return time.Time{}, false
}

func postTime(p models.Post) time.Time {
	if p.Status == models.StatusPosted && p.PostedAt != nil {
		return p.PostedAt.Local()
	}
	if p.ScheduledAt != nil {
		return p.ScheduledAt.Local()
	}
	return time.Time{}
}

func (m Model) postsByDay() map[string][]models.Post {
	out := map[string][]models.Post{}
	for _, p := range m.posts {
		if d, ok := postDay(p); ok {
			out[dayKey(d)] = append(out[dayKey(d)], p)
		}
	}
	for _, ps := range out {
		sort.Slice(ps, func(i, j int) bool { return postTime(ps[i]).Before(postTime(ps[j])) })
	}
	return out
}

func (m Model) calSel() time.Time {
	if m.calDay.IsZero() {
		return dayStart(time.Now())
	}
	return m.calDay
}

var (
	monthsDE = []string{"Januar", "Februar", "März", "April", "Mai", "Juni", "Juli", "August", "September", "Oktober", "November", "Dezember"}
	monthsEN = []string{"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"}
	wdDE     = []string{"Mo", "Di", "Mi", "Do", "Fr", "Sa", "So"}
	wdEN     = []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
)

func calGerman() bool { return strings.EqualFold(config.ActiveConfig.Defaults.Language, "de") }

func monthName(t time.Time) string {
	if calGerman() {
		return monthsDE[t.Month()-1]
	}
	return monthsEN[t.Month()-1]
}

func weekdayNames() []string {
	if calGerman() {
		return wdDE
	}
	return wdEN
}

// mondayOnOrBefore returns the Monday of t's week.
func mondayOnOrBefore(t time.Time) time.Time {
	off := (int(t.Weekday()) + 6) % 7
	return t.AddDate(0, 0, -off)
}

// dots renders up to room symbols for the posts of one day, +n when there are more.
func dots(ps []models.Post, room int) string {
	if room < 1 {
		return ""
	}
	limit := len(ps)
	if limit > room {
		limit = max(room-2, 1) // leave space for "+n"
	}
	var b strings.Builder
	shown := 0
	for _, p := range ps[:limit] {
		sym := "●"
		switch p.Status {
		case models.StatusPosted:
			sym = "○"
		case models.StatusFailed:
			sym = "✗"
		case models.StatusDraft:
			sym = "·"
		}
		st := lipgloss.NewStyle().Foreground(theme.MutedV2)
		if c, ok := platformBrand[p.Platform]; ok {
			st = lipgloss.NewStyle().Foreground(lipgloss.Color(c))
		}
		if p.Status == models.StatusFailed {
			st = lipgloss.NewStyle().Foreground(theme.RedV2)
		}
		b.WriteString(st.Render(sym))
		shown++
	}
	if rest := len(ps) - shown; rest > 0 {
		b.WriteString(dimStyle.Render(fmt.Sprintf("+%d", rest)))
	}
	return b.String()
}

// calendarKey handles keys on the calendar tab; ok=false lets the generic handlers run.
func (m Model) calendarKey(msg tea.KeyPressMsg) (Model, tea.Cmd, bool) {
	sel := m.calSel()
	move := func(d time.Time) (Model, tea.Cmd, bool) {
		m.calDay = dayStart(d)
		return m, nil, true
	}
	switch msg.String() {
	case "left", "h":
		return move(sel.AddDate(0, 0, -1))
	case "right", "l":
		return move(sel.AddDate(0, 0, 1))
	case "up", "k":
		return move(sel.AddDate(0, 0, -7))
	case "down", "j":
		return move(sel.AddDate(0, 0, 7))
	case "[", "pgup":
		return move(sel.AddDate(0, -1, 0))
	case "]", "pgdown":
		return move(sel.AddDate(0, 1, 0))
	case "t", "home":
		return move(time.Now())
	case "enter":
		if ps := m.postsByDay()[dayKey(sel)]; len(ps) > 0 {
			p := ps[min(m.calCursor, len(ps)-1)]
			m.selectedPost = &p
		}
		return m, nil, true
	case "tab", "shift+tab":
		return m, nil, false
	case ",":
		m.calCursor = max(m.calCursor-1, 0)
		return m, nil, true
	case ".":
		if ps := m.postsByDay()[dayKey(sel)]; m.calCursor < len(ps)-1 {
			m.calCursor++
		}
		return m, nil, true
	}
	return m, nil, false
}

// calGeom is the calendar tab's layout, shared by drawing and mouse hit-testing.
type calGeom struct {
	first, last, start time.Time // month bounds and the Monday the shown grid starts on
	weeks, showWeeks   int
	rowH, gridH, cellW int
}

func calLayout(w, h int, sel time.Time) calGeom {
	g := calGeom{}
	g.first = time.Date(sel.Year(), sel.Month(), 1, 0, 0, 0, 0, time.Local)
	g.last = g.first.AddDate(0, 1, -1)
	gridStart := mondayOnOrBefore(g.first)
	g.weeks = int(mondayOnOrBefore(g.last).Sub(gridStart).Hours()/24/7) + 1

	// Month grid needs 2 border rows, 1 weekday row and rowH rows per week; the day list gets the rest.
	// When that leaves too little, show only the selected week.
	g.showWeeks, g.start, g.rowH = g.weeks, gridStart, 2
	if h-(3+3*g.weeks) >= 9 { // roomy: a third line per week shows the first post of the day
		g.rowH = 3
	}
	if h-(3+g.rowH*g.weeks) < 6 {
		g.showWeeks, g.start, g.rowH = 1, mondayOnOrBefore(sel), 2
	}
	g.gridH = 3 + g.rowH*g.showWeeks
	g.cellW = max((w-4-6)/7, 5)
	return g
}

// calHit maps a click inside the body (x, y relative to the body's top-left) to a grid day or a
// post row of the selected day's list. day is zero when the click is not on a day cell.
func calHit(g calGeom, x, y int) (day time.Time, postRow int) {
	postRow = -1
	cx := x - 4 // 2 indent + border + space
	if y >= 2 && y < g.gridH-1 {
		wk := (y - 2) / g.rowH
		c := cx / (g.cellW + 1)
		if cx >= 0 && c < 7 && wk < g.showWeeks && cx%(g.cellW+1) < g.cellW {
			return g.start.AddDate(0, 0, wk*7+c), -1
		}
		return time.Time{}, -1
	}
	if y >= g.gridH+1 {
		return time.Time{}, y - (g.gridH + 1)
	}
	return time.Time{}, -1
}

// calendarClick selects the clicked day; a click on the already selected day or post opens it.
func (m Model) calendarClick(x, y int) (Model, bool) {
	w, h := m.bodyDims()
	g := calLayout(w, h, m.calSel())
	day, row := calHit(g, x, y-len(m.headerLines()))
	sel := m.calSel()
	switch {
	case !day.IsZero():
		d := dayStart(day)
		if d.Equal(sel) {
			if ps := m.postsByDay()[dayKey(sel)]; len(ps) > 0 {
				p := ps[min(m.calCursor, len(ps)-1)]
				m.selectedPost = &p
			}
		} else {
			m.calDay, m.calCursor = d, 0
		}
		return m, true
	case row >= 0:
		ps := m.postsByDay()[dayKey(sel)]
		if row < len(ps) {
			if row == m.calCursor {
				p := ps[row]
				m.selectedPost = &p
			}
			m.calCursor = row
			return m, true
		}
	}
	return m, false
}

func (m Model) renderCalendarTab(w, h int) string {
	sel := m.calSel()
	byDay := m.postsByDay()
	today := dayStart(time.Now())
	g := calLayout(w, h, sel)
	first, last, start := g.first, g.last, g.start
	showWeeks, rowH, gridH, cellW := g.showWeeks, g.rowH, g.gridH, g.cellW

	posts := 0
	for d := first; !d.After(last); d = d.AddDate(0, 0, 1) {
		posts += len(byDay[dayKey(d)])
	}
	title := fmt.Sprintf("%s %d", monthName(sel), sel.Year())
	if posts > 0 {
		title += fmt.Sprintf("  ·  %d Posts", posts)
	}

	var rows []string
	var hdr []string
	for _, n := range weekdayNames() {
		hdr = append(hdr, lipgloss.NewStyle().Foreground(theme.MutedV2).Width(cellW).Render(ansi.Truncate(n, cellW, "")))
	}
	rows = append(rows, strings.Join(hdr, " "))
	for wk := 0; wk < showWeeks; wk++ {
		var l1, l2, l3 []string
		for dd := 0; dd < 7; dd++ {
			d := start.AddDate(0, 0, wk*7+dd)
			ps := byDay[dayKey(d)]
			num := fmt.Sprintf("%2d", d.Day())
			n1 := lipgloss.NewStyle().Width(cellW)
			switch {
			case d.Equal(sel):
				n1 = n1.Reverse(true).Bold(true)
			case d.Equal(today):
				n1 = n1.Bold(true).Underline(true)
			case d.Month() != sel.Month():
				n1 = n1.Foreground(theme.SubtleV2)
			}
			cnt := ""
			if len(ps) > 0 {
				cnt = fmt.Sprintf(" %d", len(ps))
			}
			l1 = append(l1, n1.Render(num+cnt))
			l2 = append(l2, lipgloss.NewStyle().Width(cellW).Render(dots(ps, cellW-1)))
			first := ""
			if rowH == 3 && len(ps) > 0 && cellW >= 12 {
				first = dimStyle.Render(ansi.Truncate(postTime(ps[0]).Format("15:04")+" "+stripEmojis(ps[0].Title), cellW, "…"))
			}
			l3 = append(l3, lipgloss.NewStyle().Width(cellW).Render(first))
		}
		rows = append(rows, strings.Join(l1, " "), strings.Join(l2, " "))
		if rowH == 3 {
			rows = append(rows, strings.Join(l3, " "))
		}
	}
	grid := ui.Panel(w, gridH, title, strings.Join(rows, "\n"), true)

	// selected day
	ps := byDay[dayKey(sel)]
	dayTitle := sel.Format("Mon 02 Jan")
	if calGerman() {
		dayTitle = fmt.Sprintf("%s, %d. %s", weekdayNames()[(int(sel.Weekday())+6)%7], sel.Day(), monthName(sel))
	}
	dh := max(h-gridH, 3)
	var lines []string
	for i, p := range ps {
		ts := postTime(p).Format("15:04")
		title := ansi.Truncate(stripEmojis(p.Title), max(w-34, 8), "…")
		pill := platformPill(p.Platform)
		pill += strings.Repeat(" ", max(12-lipgloss.Width(pill), 0)) // align the columns
		lines = append(lines, ui.Row(w-4, i == m.calCursor, fmt.Sprintf("%s  %s %s  %s", ts, pill, statusPill(p.Status), title)))
	}
	if len(lines) == 0 {
		lines = []string{dimStyle.Render(Tr("cal_empty_day"))}
	}
	day := ui.Panel(w, dh, dayTitle, strings.Join(lines[:min(len(lines), max(dh-2, 1))], "\n"), false)
	return grid + "\n" + day
}
