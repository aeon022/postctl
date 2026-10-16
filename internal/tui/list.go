package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/aeon022/missionctl-core/theme"
	"github.com/aeon022/missionctl-core/ui"
	"github.com/aeon022/postctl/internal/models"
	"github.com/charmbracelet/x/ansi"
)

// ── dashboard (tab 0) ─────────────────────────────────────────────────────────

var dashPlatforms = []string{models.PlatformTwitter, models.PlatformLinkedIn, models.PlatformThreads,
	models.PlatformMastodon, models.PlatformBluesky, models.PlatformFacebook, models.PlatformInstagram}

// window returns the [start,end) slice of n items that fits cap rows and keeps
// cursor roughly centered.
func window(n, cursor, capacity int) (int, int) {
	capacity = max(capacity, 1)
	if n <= capacity {
		return 0, n
	}
	start := min(max(cursor-capacity/2, 0), n-capacity)
	return start, start + capacity
}

// panelRowW is the usable text width inside a ui.Panel of the given width.
func panelRowW(w int) int { return max(w-3, 4) }

func (m Model) campaignsTitle() string {
	if len(m.campaigns) == 0 && m.needsOnboarding() {
		return Tr("onb_title")
	}
	return Tr("dash_campaigns")
}

func (m Model) campaignsOrOnboarding(w, capacity int) []string {
	if len(m.campaigns) == 0 && m.needsOnboarding() {
		return m.onboardingLines()
	}
	return m.campaignLines(w, capacity)
}

func (m Model) campaignLines(w, capacity int) []string {
	if len(m.campaigns) == 0 {
		return []string{dimStyle.Render(Tr("dash_no_campaigns"))}
	}
	start, end := window(len(m.campaigns), m.cursor, capacity)
	now := time.Now()
	var out []string
	for i := start; i < end; i++ {
		c := m.campaigns[i]
		out = append(out, ui.Row(w, m.activeTab == tabDashboard && i == m.cursor, campaignRow(c, w-2, now)))
	}
	return out
}

// campaignRow is "● slug  ████░░░░ 7/20 ✗1 · in 2d": progress bar, counts, failures, next post.
func campaignRow(c models.Campaign, w int, now time.Time) string {
	total := len(c.Posts)
	ratio := 0.0
	if total > 0 {
		ratio = float64(c.Posted) / float64(total)
	}
	failed := 0
	var next *time.Time
	for _, p := range c.Posts {
		switch p.Status {
		case models.StatusFailed:
			failed++
		case models.StatusScheduled:
			if p.ScheduledAt != nil && p.ScheduledAt.After(now) && (next == nil || p.ScheduledAt.Before(*next)) {
				next = p.ScheduledAt
			}
		}
	}
	right := ui.Bar(8, ratio, false) + dimStyle.Render(fmt.Sprintf(" %d/%d", c.Posted, total))
	if failed > 0 {
		right += " " + lipgloss.NewStyle().Foreground(theme.RedV2).Render(fmt.Sprintf("✗%d", failed))
	}
	if next != nil && w >= 60 {
		right += dimStyle.Render(" · " + ui.RelTime(*next, now))
	}
	slug := ansi.Truncate(c.Slug, max(w-lipgloss.Width(right)-4, 6), "…")
	gap := strings.Repeat(" ", max(w-lipgloss.Width(slug)-lipgloss.Width(right)-3, 1))
	return "● " + slug + gap + right
}

func (m Model) nextUpLines(w, capacity int, now time.Time) []string {
	if len(m.nextUp) == 0 {
		return []string{dimStyle.Render(Tr("dash_no_schedules"))}
	}
	var out []string
	for i := 0; i < len(m.nextUp) && i < capacity; i++ {
		p := m.nextUp[i]
		when := ""
		if p.ScheduledAt != nil {
			when = ui.RelTime(*p.ScheduledAt, now) + " " + p.ScheduledAt.Format("15:04")
		}
		title := ansi.Truncate(stripEmojis(p.Title), max(w-lipgloss.Width(when)-14, 6), "…")
		out = append(out, "  "+dimStyle.Render(padRight(when, 15))+platformPill(p.Platform)+" "+title)
	}
	return out
}

func (m Model) statsLines() []string {
	label := func(k string) string { return strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(Tr(k)), ":")) }
	pill := func(k, trKey string, n int, kind ui.Kind) string {
		if n == 0 && kind == ui.Err {
			kind = ui.Muted
		}
		return ui.Pill(fmt.Sprintf("%s %d", label(trKey), n), kind)
	}
	return []string{
		pill("", "stats_posted", m.stats.posted, ui.OK) + " " + pill("", "stats_scheduled", m.stats.scheduled, ui.Info),
		pill("", "stats_drafts", m.stats.drafts, ui.Muted) + " " + pill("", "stats_failed", m.stats.failed, ui.Err),
	}
}

func (m Model) platformLines() []string {
	var out []string
	for _, p := range dashPlatforms {
		kind, status := ui.Muted, Tr("dash_not_auth")
		if m.platforms[p] {
			kind, status = ui.OK, Tr("dash_connected")
		}
		line := ui.Dot(kind) + " " + padRight(platformName(p), 12) + dimStyle.Render(status)
		if m.platforms[p] {
			if e := m.health.expiryText(p, time.Now()); e != "" {
				line += " " + e
			}
		}
		out = append(out, line)
	}
	if hl := m.healthLines(); len(hl) > 0 {
		out = append(out, "")
		out = append(out, hl...)
	}
	out = append(out, "", dimStyle.Render(Tr("dash_connect_hint")))
	return out
}

// renderDashboard lays the four panels out responsively: two columns from
// dashTwoCol, one below; sizes follow the terminal instead of fixed boxes.
func (m Model) renderDashboard(w, h int) string {
	now := time.Now()
	focus := m.activeTab == tabDashboard
	statsH := 4

	if w >= dashTwoCol {
		lw := (w - 1) * 11 / 20
		rw := w - 1 - lw
		hc := max(h/2, 5)
		hn := max(h-hc, 5)
		left := ui.Panel(lw, hc, m.campaignsTitle(), strings.Join(m.campaignsOrOnboarding(panelRowW(lw), hc-2), "\n"), focus) + "\n" +
			ui.Panel(lw, hn, Tr("dash_next_up"), strings.Join(m.nextUpLines(panelRowW(lw), hn-2, now), "\n"), false)
		right := ui.Panel(rw, statsH, Tr("dash_stats"), strings.Join(m.statsLines(), "\n"), false) + "\n" +
			ui.Panel(rw, max(h-statsH, 6), Tr("dash_platforms"), strings.Join(m.platformLines(), "\n"), false)
		return joinColumns(left, right, lw, 1)
	}

	rest := max(h-statsH, 6)
	hc, hn := max(rest/3, 4), max(rest/3, 4)
	hp := max(rest-hc-hn, 4)
	return strings.Join([]string{
		ui.Panel(w, statsH, Tr("dash_stats"), strings.Join(m.statsLines(), "\n"), false),
		ui.Panel(w, hc, m.campaignsTitle(), strings.Join(m.campaignsOrOnboarding(panelRowW(w), hc-2), "\n"), focus),
		ui.Panel(w, hn, Tr("dash_next_up"), strings.Join(m.nextUpLines(panelRowW(w), hn-2, now), "\n"), false),
		ui.Panel(w, hp, Tr("dash_platforms"), strings.Join(m.platformLines(), "\n"), false),
	}, "\n")
}

// joinColumns places two multi-line blocks side by side, padding the left one
// to leftW so the right block lines up.
func joinColumns(left, right string, leftW, gap int) string {
	l, r := strings.Split(left, "\n"), strings.Split(right, "\n")
	n := max(len(l), len(r))
	var out []string
	for i := 0; i < n; i++ {
		a, b := "", ""
		if i < len(l) {
			a = l[i]
		}
		if i < len(r) {
			b = r[i]
		}
		out = append(out, padRight(a, leftW)+strings.Repeat(" ", gap)+b)
	}
	return strings.Join(out, "\n")
}

// ── posts (tab 1) ─────────────────────────────────────────────────────────────

// postRow is one post as a single selectable line: checkbox, platform pill,
// status pill, title, and campaign/when on the right.
func (m Model) postRow(p models.Post, w int, selected bool, now time.Time) string {
	chk := "[ ] "
	if m.selectedPosts[p.ID] {
		chk = lipgloss.NewStyle().Foreground(theme.BlueV2).Bold(true).Render("[x] ")
	}
	left := chk + padRight(platformPill(p.Platform), 11) + statusPill(p.Status) + " "
	right := whenText(p, now)
	if p.Campaign != "" && w >= 90 {
		right = strings.TrimSpace("📁 " + p.Campaign + "  " + right)
	}
	if right != "" {
		right = " " + dimStyle.Render(right)
	}
	title := stripEmojis(p.Title)
	if title == "" {
		title = "(no title)"
	}
	titleW := max(w-2-lipgloss.Width(left)-lipgloss.Width(right), 4)
	title = ansi.Truncate(title, titleW, "…")
	pad := strings.Repeat(" ", max(titleW-lipgloss.Width(title), 0))
	return ui.Row(w, selected, left+title+pad+right)
}

func (m Model) postsTitle() string {
	if m.filterCampaign != "" {
		return fmt.Sprintf(Tr("posts_header_filtered"), m.filterCampaign)
	}
	return Tr("header_posts")
}

// previewPanel shows the selected post: pills, schedule, campaign, the body
// and — where the platform has a known limit — a character-limit bar.
func (m Model) previewPanel(p *models.Post, w, h int) string {
	var b strings.Builder
	if p == nil {
		return ui.Panel(w, h, "Preview", dimStyle.Render("—"), false)
	}
	cw := panelRowW(w)
	b.WriteString(platformPill(p.Platform) + " " + statusPill(p.Status) + "\n")
	if t := stripEmojis(p.Title); t != "" {
		b.WriteString("\n" + ansi.Truncate(t, cw, "…") + "\n")
	}
	now := time.Now()
	meta := []string{}
	if wt := whenText(*p, now); wt != "" {
		meta = append(meta, wt)
	}
	if p.Campaign != "" {
		meta = append(meta, "📁 "+p.Campaign)
	}
	if p.Type == "thread" {
		meta = append(meta, fmt.Sprintf(Tr("meta_thread"), len(p.Tweets)))
	}
	if len(p.Images) > 0 {
		meta = append(meta, fmt.Sprintf(Tr("meta_images"), len(p.Images)))
	}
	if len(meta) > 0 {
		b.WriteString(dimStyle.Render(strings.Join(meta, " · ")) + "\n")
	}
	if bar := limitBar(*p, min(cw-9, 24)); bar != "" {
		b.WriteString(bar + "\n")
	}
	b.WriteString("\n")
	body := p.Body
	if p.Type == "thread" {
		var parts []string
		for i, t := range p.Tweets {
			parts = append(parts, fmt.Sprintf("%d/%d  %s", i+1, len(p.Tweets), t.Content))
		}
		body = strings.Join(parts, "\n\n")
	}
	b.WriteString(ansi.Wrap(body, cw, " -"))
	if p.Error != "" {
		b.WriteString("\n\n" + ui.Toast(ui.Err, ansi.Truncate(p.Error, cw-2, "…")))
	}
	return ui.Panel(w, h, "Preview", strings.TrimRight(b.String(), "\n"), false)
}

// renderPostList renders the posts tab: single-line rows, and from wideBreak
// columns a Preview panel for the selected post.
func (m Model) renderPostList(w, h int) string {
	if len(m.posts) == 0 {
		return emptyBody(w, h, Tr("posts_none_found"), Tr("posts_empty_hint"))
	}
	filtered := m.getFilteredPosts()
	if len(filtered) == 0 {
		return emptyBody(w, h, fmt.Sprintf(Tr("posts_none_found_campaign"), m.filterCampaign), Tr("posts_empty_hint"))
	}
	now := time.Now()
	var sel *models.Post
	if m.cursor >= 0 && m.cursor < len(filtered) {
		sel = &filtered[m.cursor]
	}

	if w >= wideBreak {
		lw := (w - 1) * 58 / 100
		rw := w - 1 - lw
		start, end := window(len(filtered), m.cursor, h-2)
		var rows []string
		for i := start; i < end; i++ {
			rows = append(rows, m.postRow(filtered[i], panelRowW(lw), m.activeTab == tabPosts && i == m.cursor, now))
		}
		left := ui.Panel(lw, h, m.postsTitle(), strings.Join(rows, "\n"), true)
		return joinColumns(left, m.previewPanel(sel, rw, h), lw, 1)
	}

	start, end := window(len(filtered), m.cursor, h-1)
	rows := []string{ui.Divider(w, m.postsTitle())}
	for i := start; i < end; i++ {
		rows = append(rows, m.postRow(filtered[i], w, m.activeTab == tabPosts && i == m.cursor, now))
	}
	return strings.Join(rows, "\n")
}
