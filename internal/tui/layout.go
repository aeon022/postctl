package tui

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/aeon022/missionctl-core/emptystate"
	"github.com/aeon022/missionctl-core/statusbar"
	"github.com/aeon022/missionctl-core/theme"
	"github.com/aeon022/missionctl-core/ui"
	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/models"
	"github.com/charmbracelet/x/ansi"
)

// Suite layout: header · divider · tabs · blank · body · one-line footer, all
// inside a 2-column indent and exactly the terminal height (ui.Frame).
const (
	indent     = 2
	wideBreak  = 120 // posts/queue get a Preview panel from here on
	dashTwoCol = 100 // dashboard uses two panel columns from here on
	tabsRow    = 2   // screen row of the tab bar (header, divider, tabs)
)

var dimStyle = lipgloss.NewStyle().Foreground(theme.MutedV2)

func (m Model) termW() int {
	if m.width > 0 {
		return m.width
	}
	return 100
}

// innerW is the width available inside the 2-column indent.
func (m Model) innerW() int { return max(m.termW()-2*indent, 20) }

// boxW is the width of a legacy fixed-size box (want = its old inner size),
// shrunk to the terminal so nothing overflows narrow windows.
func (m Model) boxW(want int) int { return min(want+2, max(m.termW()-2*indent, 10)) }

func (m Model) scopeText() string {
	parts := []string{}
	if p := config.ActiveProfile; p != "" && p != "default" {
		parts = append(parts, "profile: "+p)
	}
	if m.filterCampaign != "" {
		parts = append(parts, "campaign: "+m.filterCampaign)
	}
	return strings.Join(parts, " · ")
}

func (m Model) tabLabels() []string {
	return []string{Tr("tab_dashboard"), Tr("tab_posts"), Tr("tab_schedule"), Tr("tab_history"),
		Tr("tab_analytics"), Tr("tab_settings"), Tr("tab_logs")}
}

func (m Model) tabCounts() []int { return []int{0, len(m.posts), len(m.nextUp)} }

// headerLines is header + divider + tabs, plus a blank line when there is room.
func (m Model) headerLines() []string {
	w := m.innerW()
	lines := []string{
		ui.Header(w, "postctl · Social Media", m.scopeText(), time.Now().Format("Mon 02 Jan")),
		ui.Divider(w, ""),
		ui.Tabs(w, m.tabLabels(), m.activeTab, m.tabCounts()),
	}
	if m.height <= 0 || m.height >= 22 {
		lines = append(lines, "")
	}
	return lines
}

// tabAt maps a screen column on the tab row to a tab index (-1 for none) by
// locating each label in the text that was actually drawn.
func (m Model) tabAt(x int) int {
	plain := ansi.Strip(ui.Tabs(m.innerW(), m.tabLabels(), m.activeTab, m.tabCounts()))
	counts := m.tabCounts()
	off := 0
	for i, l := range m.tabLabels() {
		text := l
		if i < len(counts) && counts[i] > 0 {
			text += fmt.Sprintf(" %d", counts[i])
		}
		idx := strings.Index(plain[off:], text)
		if idx < 0 {
			continue
		}
		start := off + idx
		col := lipgloss.Width(plain[:start]) + indent
		if x >= col-1 && x < col+lipgloss.Width(text)+1 {
			return i
		}
		off = start + len(text)
	}
	return -1
}

// switchTab is the shared tab change used by keys and clicks.
func (m Model) switchTab(i int) (Model, tea.Cmd) {
	m.activeTab = ((i % 7) + 7) % 7
	m.cursor = 0
	if m.activeTab == 4 {
		m.analyticsLoading = true
		return m, m.loadAnalyticsCmd
	}
	return m, nil
}

// ── footer ────────────────────────────────────────────────────────────────────

func hint(key, trKey string) [2]string { return [2]string{key, Tr(trKey)} }

// hintsFor lists the keys valid in the current state in priority order: `?` and
// `q` come early so they are the last to be dropped on a narrow terminal.
func (m Model) hintsFor() [][2]string {
	if m.showHelp {
		return [][2]string{hint("?", "hint_close"), hint("q", "hint_quit")}
	}
	switch m.activeTab {
	case 1: // posts
		h := [][2]string{hint("enter", "hint_open"), hint("n", "hint_new"), hint("?", "hint_help"), hint("q", "hint_quit"),
			hint("e", "hint_edit"), hint("d", "hint_delete"), hint("s", "hint_schedule"), hint("p", "hint_post"),
			hint("space", "hint_select"), hint("f", "hint_filter"), hint("r", "hint_repurpose"), hint("i", "hint_import"),
			hint("tab", "hint_tab"), hint("f1", "hint_manual")}
		if m.filterCampaign != "" {
			h = append([][2]string{hint("esc", "hint_clear")}, h...)
		}
		return h
	case 2: // queue
		return [][2]string{hint("enter", "hint_open"), hint("?", "hint_help"), hint("q", "hint_quit"), hint("e", "hint_edit"),
			hint("d", "hint_delete"), hint("tab", "hint_tab"), hint("f1", "hint_manual")}
	case 3: // history
		return [][2]string{hint("enter", "hint_open"), hint("?", "hint_help"), hint("q", "hint_quit"),
			hint("x", "hint_export"), hint("tab", "hint_tab"), hint("f1", "hint_manual")}
	case 0: // dashboard
		return [][2]string{hint("enter", "hint_open"), hint("?", "hint_help"), hint("q", "hint_quit"), hint("n", "hint_new"),
			hint("tab", "hint_tab"), hint("i", "hint_import"), hint("f1", "hint_manual")}
	}
	return [][2]string{hint("?", "hint_help"), hint("q", "hint_quit"), hint("enter", "hint_open"),
		hint("tab", "hint_tab"), hint("f1", "hint_manual")}
}

func (m Model) footerLine() string {
	w := m.innerW()
	right := ""
	switch {
	case m.statusMessage != "":
		right = ui.Toast(ui.Info, m.statusMessage)
	case m.maxCursorItems() > 0 && (m.activeTab == 1 || m.activeTab == 2):
		right = dimStyle.Render(fmt.Sprintf("%d/%d", m.cursor+1, m.maxCursorItems()))
	}
	left := statusbar.Hints(max(w-lipgloss.Width(right)-2, 10), m.hintsFor()...)
	return statusbar.Line(w, left, right)
}

// ── view assembly ─────────────────────────────────────────────────────────────

func (m Model) frameHeight() int {
	if m.height <= 0 {
		return 0
	}
	return max(m.height-1, 6) // one spare row against alt-screen redraw artifacts, like the other tools
}

// bodyDims is the size left for the tab content.
func (m Model) bodyDims() (int, int) {
	h := 24
	if fh := m.frameHeight(); fh > 0 {
		h = fh - len(m.headerLines()) - 1
	}
	return m.innerW(), max(h, 3)
}

// indentLines left-pads every line and clamps it to the terminal width, so
// views that still use fixed-size boxes can never overflow a narrow window.
func (m Model) indentLines(s string) string {
	pad := strings.Repeat(" ", indent)
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = ansi.Truncate(pad+l, m.termW(), "")
	}
	return strings.Join(lines, "\n")
}

func (m Model) helpBody(w, h int) string {
	rows := []struct{ k, t string }{
		{"tab / shift+tab", "help_tab"}, {"↑/k  ↓/j", "help_up"}, {"enter", "help_enter"}, {"n", "help_new_post"},
		{"e", "help_edit_post"}, {"s", "help_schedule"}, {"p", "help_post"}, {"i", "help_import"}, {"d", "help_delete"},
		{"r", "help_repurpose"}, {"f", "help_filter"}, {"esc", "help_esc"}, {"f1 / R", "help_readme"},
		{"?", "help_toggle"}, {"q / ctrl+c", "help_quit"},
	}
	var b strings.Builder
	for _, r := range rows {
		b.WriteString(ui.KeyCap(r.k) + " " + dimStyle.Render(Tr(r.t)) + "\n")
	}
	return ui.Panel(w, min(len(rows)+2, h), Tr("help_title"), strings.TrimRight(b.String(), "\n"), true)
}

func (m Model) tabBody(w, h int) string {
	switch {
	case m.selectedPost != nil:
		return m.renderDetailView()
	case m.selectedHistory != nil:
		return m.renderHistoryDetailView()
	case m.showReadme:
		return m.renderReadme()
	case m.showHelp:
		return m.helpBody(w, h)
	}
	switch m.activeTab {
	case 0:
		return m.renderDashboard(w, h)
	case 1:
		return m.renderPostList(w, h)
	case 2:
		return m.renderSchedule(w, h)
	case 3:
		return m.renderHistory()
	case 4:
		return m.renderAnalytics()
	case 5:
		return m.renderSettings()
	case 6:
		return m.renderLogs()
	}
	return ""
}

// ── shared bits ───────────────────────────────────────────────────────────────

func padRight(s string, w int) string {
	return s + strings.Repeat(" ", max(w-lipgloss.Width(s), 0))
}

var platformNames = map[string]string{
	models.PlatformTwitter: "Twitter/X", models.PlatformLinkedIn: "LinkedIn", models.PlatformThreads: "Threads",
	models.PlatformMastodon: "Mastodon", models.PlatformBluesky: "Bluesky", models.PlatformFacebook: "Facebook",
}

func platformName(p string) string {
	if n, ok := platformNames[p]; ok {
		return n
	}
	if p == "" {
		return "?"
	}
	return strings.ToUpper(p[:1]) + p[1:]
}

func platformPill(p string) string { return ui.Pill(platformName(p), ui.Info) }

// statusPill is the post status as a fixed-width pill: draft muted, scheduled
// info, posted ok, failed err.
func statusPill(status string) string {
	k, label := ui.Muted, "DRAFT"
	switch status {
	case models.StatusScheduled:
		k, label = ui.Info, "SCHED"
	case models.StatusPosted:
		k, label = ui.OK, "POSTED"
	case models.StatusFailed:
		k, label = ui.Err, "FAILED"
	}
	return ui.Pill(fmt.Sprintf("%-6s", label), k)
}

// platformLimit is the character limit the editor already enforces (0 = unknown).
func platformLimit(platform string) int {
	switch platform {
	case "twitter":
		return 280
	case "bluesky":
		return 300
	case "mastodon", "threads":
		return 500
	case "linkedin":
		return 3000
	case "telegram":
		return 4096
	case "discord":
		return 2000
	case "devto", "hashnode", "medium":
		return 100000
	case "reddit":
		return 40000
	}
	return 0
}

// whenText: when a post goes/went out, relative ("in 2d 09:00").
func whenText(p models.Post, now time.Time) string {
	switch {
	case p.Status == models.StatusScheduled && p.ScheduledAt != nil:
		return ui.RelTime(*p.ScheduledAt, now) + " " + p.ScheduledAt.Format("15:04")
	case p.Status == models.StatusPosted && p.PostedAt != nil:
		return ui.RelTime(*p.PostedAt, now) + " " + p.PostedAt.Format("15:04")
	case p.Status == models.StatusFailed:
		return "failed"
	}
	return ""
}

// postLen is the character count the limit bar measures: the longest tweet of
// a thread, else the body.
func postLen(p models.Post) int {
	if p.Type == "thread" && len(p.Tweets) > 0 {
		n := 0
		for _, t := range p.Tweets {
			n = max(n, t.CharCount())
		}
		return n
	}
	return utf8.RuneCountInString(p.Body)
}

// limitBar renders "▓▓▓░░ 212/280"; empty for platforms without a known limit.
func limitBar(p models.Post, width int) string {
	lim := platformLimit(p.Platform)
	if lim == 0 || lim >= 100000 {
		return ""
	}
	n := postLen(p)
	return ui.Bar(width, float64(n)/float64(lim), true) + dimStyle.Render(fmt.Sprintf(" %d/%d", n, lim))
}

func emptyBody(w, h int, title, hintText string) string {
	return emptystate.Render(w, h, "✎", strings.TrimSpace(title), hintText)
}
