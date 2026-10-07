package tui

import (
	"fmt"
	"math"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/aeon022/missionctl-core/theme"
	"github.com/aeon022/missionctl-core/ui"
)

var analyticsPlatforms = []string{"twitter", "linkedin", "threads", "mastodon", "bluesky", "facebook"}

// renderAnalytics is the analytics tab: info line, summary + split panels
// (side by side from 82 columns), trend chart and the per-platform table, in
// the shared panel style. Panels that do not fit the height are cut off.
func (m Model) renderAnalytics(w, h int) string {
	switch {
	case m.analyticsLoading:
		return emptyBody(w, h, Tr("an_loading"), "")
	case m.analyticsData == nil:
		return emptyBody(w, h, Tr("an_no_data"), "")
	case m.analyticsData.err != nil:
		return emptyBody(w, h, fmt.Sprintf(Tr("an_load_error"), m.analyticsData.err), "")
	}
	data := m.analyticsData
	rw := panelRowW(w)

	infoLines := wrapLines(Tr("an_info"), rw)[:]
	if len(infoLines) > 2 {
		infoLines = infoLines[:2]
	}
	info := ui.Panel(w, 4, "ℹ", dimStyle.Render(strings.Join(infoLines, "\n")), false)

	sum := []string{
		kvw(Tr("an_posts"), fmt.Sprint(data.totalPosts)), kvw(Tr("an_likes"), fmt.Sprint(data.totalLikes)),
		kvw(Tr("an_shares"), fmt.Sprint(data.totalShares)), kvw(Tr("an_comments"), fmt.Sprint(data.totalComments)),
		kvw(Tr("an_impressions"), fmt.Sprint(data.totalImpressions)),
	}

	total := data.totalLikes + data.totalShares + data.totalComments
	var dist, table []string
	table = append(table, dimStyle.Render(fmt.Sprintf("%-12s %6s %6s %6s %6s %8s", Tr("an_col_platform"), "Posts", "Likes", "Share", "Comm.", "Impr.")))
	for _, p := range analyticsPlatforms {
		s, ok := data.platStats[p]
		if !ok || s.Posts == 0 {
			continue
		}
		ratio := 0.0
		if total > 0 {
			ratio = float64(s.Likes+s.Shares+s.Comments) / float64(total)
		}
		dist = append(dist, fmt.Sprintf("%-10s %s %3.0f%%", platformName(p), ui.Bar(12, ratio, false), math.Round(ratio*100)))
		table = append(table, fmt.Sprintf("%-12s %6d %6d %6d %6d %8d", platformName(p), s.Posts, s.Likes, s.Shares, s.Comments, s.Impressions))
	}
	if len(table) == 1 {
		table = append(table, dimStyle.Render(Tr("an_no_details")))
	}

	trend := strings.Split(renderTrendChart(data.dailyEngagement, rw), "\n")
	sty := lipgloss.NewStyle().Foreground(theme.BlueV2)
	for i := range trend {
		trend[i] = sty.Render(trend[i])
	}

	var parts []string
	parts = append(parts, info)
	if w >= 82 {
		lw := (w - 1) / 2
		left := ui.Panel(lw, 7, Tr("panel_summary"), strings.Join(sum, "\n"), false)
		right := ui.Panel(w-1-lw, 7, Tr("panel_dist"), strings.Join(dist, "\n"), false)
		parts = append(parts, joinColumns(left, right, lw, 1))
	} else {
		parts = append(parts, ui.Panel(w, 7, Tr("panel_summary"), strings.Join(sum, "\n"), false),
			ui.Panel(w, 7, Tr("panel_dist"), strings.Join(dist, "\n"), false))
	}
	parts = append(parts, ui.Panel(w, 9, Tr("panel_trend"), strings.Join(trend, "\n"), false),
		ui.Panel(w, len(table)+2, Tr("panel_details"), strings.Join(table, "\n"), false))
	return strings.Join(parts, "\n")
}

// kvw is a dimmed label (padded to 26 cells) followed by its value.
func kvw(label, value string) string { return dimStyle.Render(padRight(label, 26)) + value }

// renderTrendChart draws a 5-row bar chart of the newest days that fit width
// (two cells per day), with an x axis and start/end labels underneath.
func renderTrendChart(engagement []int, width int) string {
	if n := max((width-6)/2, 1); len(engagement) > n {
		engagement = engagement[len(engagement)-n:]
	}
	maxVal := 1
	for _, v := range engagement {
		maxVal = max(maxVal, v)
	}

	const chartHeight = 5
	var lines []string
	for r := chartHeight; r > 0; r-- {
		var line strings.Builder
		fmt.Fprintf(&line, "%3d │ ", maxVal*r/chartHeight)
		threshold := float64(r) / chartHeight
		prev := float64(r-1) / chartHeight
		for _, val := range engagement {
			fraction := float64(val) / float64(maxVal)
			switch {
			case fraction >= threshold:
				line.WriteString("█ ")
			case fraction >= prev+(threshold-prev)/2:
				line.WriteString("▄ ")
			default:
				line.WriteString("  ")
			}
		}
		lines = append(lines, line.String())
	}
	lines = append(lines, "  0 └─"+strings.Repeat("──", len(engagement)))

	start, end := Tr("an_axis_start"), Tr("an_axis_end")
	gap := max(len(engagement)*2-lipgloss.Width(start)-lipgloss.Width(end), 1)
	return strings.Join(append(lines, "     "+start+strings.Repeat(" ", gap)+end), "\n")
}
