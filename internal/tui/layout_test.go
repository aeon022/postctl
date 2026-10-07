package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/aeon022/missionctl-core/tuitest"
	"github.com/aeon022/postctl/internal/models"
	"github.com/charmbracelet/x/ansi"
)

// layoutModel is a model with four posts in every status plus one campaign.
func layoutModel(t *testing.T, w, h int) Model {
	t.Helper()
	m := smokeModel(t, false)
	now := time.Now()
	for _, p := range []*models.Post{
		{ID: "a", Platform: "bluesky", Status: models.StatusScheduled, Title: "Launch day thread", Body: "We are live! Here is what we built.", Campaign: "launch", ScheduledAt: ptr(time.Date(now.Year(), now.Month(), now.Day()+1, 12, 0, 0, 0, now.Location())), CreatedAt: now, UpdatedAt: now},
		{ID: "b", Platform: "twitter", Status: models.StatusDraft, Title: "A second post with a rather long title to test truncation", Body: "Draft body text", Campaign: "launch", CreatedAt: now, UpdatedAt: now},
		{ID: "c", Platform: "linkedin", Status: models.StatusPosted, Title: "Posted article", Body: "Long body", PostedAt: ptr(now.Add(-48 * time.Hour)), CreatedAt: now, UpdatedAt: now},
		{ID: "d", Platform: "mastodon", Status: models.StatusFailed, Title: "Failed one", Body: "x", Error: "401 unauthorized", CreatedAt: now, UpdatedAt: now},
	} {
		if err := m.store.SavePost(context.Background(), p); err != nil {
			t.Fatal(err)
		}
	}
	res, _ := m.Update(m.loadDataCmd())
	mm, _ := tuitest.Send(res, tuitest.Resize(w, h))
	return mm.(Model)
}

func atTab(t *testing.T, m Model, tab int) Model {
	t.Helper()
	mm, _ := m.switchTab(tab)
	return mm
}

var layoutSizes = [][2]int{{40, 16}, {60, 18}, {80, 24}, {100, 30}, {140, 36}, {170, 40}}

func TestEveryTabFitsTheTerminalAndFillsExactlyItsHeight(t *testing.T) {
	for _, sz := range layoutSizes {
		for tab := 0; tab < 7; tab++ {
			m := atTab(t, layoutModel(t, sz[0], sz[1]), tab)
			out := m.View().Content
			lines := strings.Split(out, "\n")
			if len(lines) != sz[1]-1 {
				t.Errorf("%dx%d tab %d: %d lines, want %d (terminal height minus the spare row)", sz[0], sz[1], tab, len(lines), sz[1]-1)
			}
			for i, l := range lines {
				if w := lipgloss.Width(l); w > sz[0] {
					t.Errorf("%dx%d tab %d line %d is %d wide: %q", sz[0], sz[1], tab, i, w, ansi.Strip(l))
					break
				}
			}
		}
	}
}

func TestHeaderTabsAndFooterChrome(t *testing.T) {
	m := layoutModel(t, 110, 30)
	lines := strings.Split(tuitest.Text(m), "\n")
	if !strings.Contains(lines[0], "postctl · Social Media") || !strings.Contains(lines[0], time.Now().Format("Mon 02 Jan")) {
		t.Errorf("header: %q", lines[0])
	}
	if !strings.Contains(lines[1], "───") {
		t.Errorf("divider: %q", lines[1])
	}
	for _, want := range []string{"DASHBOARD", "POSTS 4", "QUEUE 1", "HISTORY", "STATS", "SETTINGS", "LOGS"} {
		if !strings.Contains(lines[2], want) {
			t.Errorf("tab row missing %q: %q", want, lines[2])
		}
	}
	foot := lines[len(lines)-1]
	if !strings.Contains(foot, "q quit") || !strings.Contains(foot, "? help") {
		t.Errorf("footer: %q", foot)
	}
}

func TestFooterIsOneLineAndKeepsHelpAndQuitWhenNarrow(t *testing.T) {
	for _, w := range []int{60, 80, 100, 140} {
		for tab := 0; tab < 7; tab++ {
			m := atTab(t, layoutModel(t, w, 30), tab)
			foot := ansi.Strip(m.footerLine())
			if strings.Contains(foot, "\n") || lipgloss.Width(foot) > m.innerW() {
				t.Errorf("w=%d tab=%d footer not one line within width: %q", w, tab, foot)
			}
			if !strings.Contains(foot, "q quit") || !strings.Contains(foot, "? help") {
				t.Errorf("w=%d tab=%d: ? and q are dropped last, got %q", w, tab, foot)
			}
		}
	}
	m := atTab(t, layoutModel(t, 100, 30), 1)
	m.filterCampaign = "launch"
	if !strings.Contains(ansi.Strip(m.footerLine()), "esc clear filter") {
		t.Error("an active campaign filter offers esc to clear it")
	}
}

func TestTabClickSelectsTheDrawnTab(t *testing.T) {
	m := layoutModel(t, 110, 30)
	row := strings.Split(tuitest.Text(m), "\n")[tabsRow]
	for i, label := range m.tabLabels() {
		col := lipgloss.Width(row[:strings.Index(row, label)]) + 1
		mm, _ := tuitest.Send(m, tuitest.Click(col, tabsRow))
		if got := mm.(Model).activeTab; got != i {
			t.Errorf("click on %q (col %d): activeTab = %d, want %d", label, col, got, i)
		}
	}
	// clicks off the tab row, or while a modal view is open, change nothing
	mm, _ := tuitest.Send(m, tuitest.Click(10, tabsRow+3))
	if mm.(Model).activeTab != 0 {
		t.Error("a click below the tab row must not switch tabs")
	}
	e := m
	e.isEditing = true
	mm, _ = tuitest.Send(e, tuitest.Click(30, tabsRow))
	if mm.(Model).activeTab != 0 {
		t.Error("tab clicks are ignored while the editor is open")
	}
	// clicking switches the same way the tab key does: analytics loads on entry
	col := lipgloss.Width(row[:strings.Index(row, "STATS")]) + 1
	st, cmd := tuitest.Send(m, tuitest.Click(col, tabsRow))
	if !st.(Model).analyticsLoading || len(cmd) == 0 {
		t.Error("clicking STATS must start the analytics load like the tab key does")
	}
}

func TestDashboardColumnRule(t *testing.T) {
	lineOf := func(text, sub string) int {
		for i, l := range strings.Split(text, "\n") {
			if strings.Contains(l, sub) {
				return i
			}
		}
		return -1
	}
	narrow := tuitest.Text(layoutModel(t, dashTwoCol-1+4, 40)) // inner width 99 → one column
	if lineOf(narrow, "STATS") >= lineOf(narrow, "CAMPAIGNS") || strings.Contains(strings.Split(narrow, "\n")[lineOf(narrow, "CAMPAIGNS")], "STATS") {
		t.Errorf("below %d columns the panels stack in one column:\n%s", dashTwoCol, narrow)
	}
	wide := tuitest.Text(layoutModel(t, dashTwoCol+4, 40)) // inner width 100 → two columns
	if l := strings.Split(wide, "\n")[lineOf(wide, "CAMPAIGNS")]; !strings.Contains(l, "STATS") {
		t.Errorf("from %d columns Campaigns and Stats share a row: %q", dashTwoCol, l)
	}
}

func TestDashboardPlatformDotsAndStats(t *testing.T) {
	m := layoutModel(t, 110, 30)
	m.platforms = map[string]bool{"bluesky": true}
	text := tuitest.Text(m)
	if strings.Count(text, "Connected ✓") != 1 || strings.Count(text, "Not connected") != 5 {
		t.Errorf("one connected, five not:\n%s", text)
	}
	if strings.Contains(text, "Press Enter") {
		t.Error("the per-platform '(Press Enter)' hint is gone")
	}
	if !strings.Contains(text, "Connect platforms in the SETTINGS tab") {
		t.Error("one hint tells where to connect")
	}
	for _, want := range []string{"Posted 1", "Scheduled 1", "Drafts 1", "Failed 1"} {
		if !strings.Contains(text, want) {
			t.Errorf("stats pill %q missing", want)
		}
	}
	// connected and not-connected dots differ
	if ui := m.platformLines(); len(ui) < 2 || ui[0] == "" {
		t.Fatal("platform lines")
	}
	if a, b := ansi.Strip(m.platformLines()[4]), ansi.Strip(m.platformLines()[0]); a == b {
		t.Error("platform rows must differ by status text")
	}
}

func TestStatusPillsAndPlatformPill(t *testing.T) {
	seen := map[string]bool{}
	for status, want := range map[string]string{
		models.StatusDraft: " DRAFT  ", models.StatusScheduled: " SCHED  ", models.StatusPosted: " POSTED ", models.StatusFailed: " FAILED ",
	} {
		p := statusPill(status)
		if ansi.Strip(p) != want {
			t.Errorf("statusPill(%s) = %q, want %q", status, ansi.Strip(p), want)
		}
		seen[p] = true
	}
	if len(seen) != 4 {
		t.Error("each status has its own color")
	}
	if ansi.Strip(platformPill("twitter")) != " Twitter/X " || ansi.Strip(platformPill("devto")) != " Devto " {
		t.Errorf("platform pills: %q %q", ansi.Strip(platformPill("twitter")), ansi.Strip(platformPill("devto")))
	}
}

func TestPostsRowsPreviewOnlyFromWideBreakAndFollowsCursor(t *testing.T) {
	below := tuitest.Text(atTab(t, layoutModel(t, wideBreak-1+4, 36), 1)) // inner 119
	if strings.Contains(below, "Preview") || strings.Contains(below, "╭") {
		t.Errorf("below %d columns: plain rows, no panel frames:\n%s", wideBreak, below)
	}
	m := atTab(t, layoutModel(t, wideBreak+4, 36), 1)
	text := tuitest.Text(m)
	if !strings.Contains(text, "Preview") || !strings.Contains(text, "We are live!") {
		t.Errorf("from %d columns the first post previews on the right:\n%s", wideBreak, text)
	}
	for _, want := range []string{"Bluesky", "SCHED", "Twitter/X", "DRAFT", "LinkedIn", "POSTED", "Mastodon", "FAILED", "tomorrow", "2d ago", "failed"} {
		if !strings.Contains(text, want) {
			t.Errorf("row/preview text %q missing", want)
		}
	}
	mm, _ := tuitest.Keys(m, "j")
	if !strings.Contains(tuitest.Text(mm), "Draft body text") {
		t.Error("the preview follows the cursor")
	}
}

func TestPostsRowSelectionAndCheckbox(t *testing.T) {
	m := atTab(t, layoutModel(t, 100, 30), 1)
	if !strings.Contains(tuitest.Text(m), "▌ [ ]") {
		t.Errorf("the cursor row carries the accent bar:\n%s", tuitest.Text(m))
	}
	mm, _ := tuitest.Keys(m, "space")
	if !strings.Contains(tuitest.Text(mm), "[x]") {
		t.Error("space ticks the checkbox")
	}
}

func TestLimitBarMath(t *testing.T) {
	tw := models.Post{Platform: "twitter", Body: strings.Repeat("x", 140)}
	if got := ansi.Strip(limitBar(tw, 20)); !strings.HasSuffix(got, " 140/280") || !strings.HasPrefix(got, "██████████░░░░░░░░░░") {
		t.Errorf("half of Twitter's 280 = half bar: %q", got)
	}
	over := models.Post{Platform: "bluesky", Body: strings.Repeat("x", 400)}
	if got := ansi.Strip(limitBar(over, 10)); !strings.HasPrefix(got, "██████████") || !strings.HasSuffix(got, " 400/300") {
		t.Errorf("over the limit: full bar, real numbers: %q", got)
	}
	if limitBar(models.Post{Platform: "medium", Body: "x"}, 10) != "" || limitBar(models.Post{Platform: "unknown"}, 10) != "" {
		t.Error("no bar for platforms without a meaningful limit")
	}
	thread := models.Post{Platform: "twitter", Type: "thread", Tweets: []models.Tweet{{Content: "short"}, {Content: strings.Repeat("y", 200)}}}
	if postLen(thread) != 200 {
		t.Errorf("a thread measures its longest tweet: %d", postLen(thread))
	}
}

func TestPlatformLimitsMatchTheEditor(t *testing.T) {
	for p, want := range map[string]int{"twitter": 280, "bluesky": 300, "mastodon": 500, "threads": 500, "linkedin": 3000, "telegram": 4096, "discord": 2000, "nope": 0} {
		if got := platformLimit(p); got != want {
			t.Errorf("platformLimit(%s) = %d, want %d", p, got, want)
		}
	}
}

func TestQueueGroupsByCampaignWithRelativeTimes(t *testing.T) {
	m := atTab(t, layoutModel(t, 100, 30), 2)
	text := tuitest.Text(m)
	for _, want := range []string{"SCHEDULED POSTS", "LAUNCH", "tomorrow", "Bluesky", "Launch day thread"} {
		if !strings.Contains(text, want) {
			t.Errorf("queue missing %q:\n%s", want, text)
		}
	}
}

func TestEmptyStatesUseTheSharedBlock(t *testing.T) {
	m := smokeModel(t, false)
	mm, _ := tuitest.Send(m, tuitest.Resize(100, 30))
	for tab, want := range map[int]string{1: "No posts found", 2: "No posts currently scheduled"} {
		got := tuitest.Text(atTab(t, mm.(Model), tab))
		if !strings.Contains(got, want) {
			t.Errorf("tab %d empty state missing %q:\n%s", tab, want, got)
		}
	}
}

func TestHelpIsAPanelWithCloseHint(t *testing.T) {
	m := layoutModel(t, 100, 30)
	mm, _ := tuitest.Keys(m, "?")
	text := tuitest.Text(mm)
	if !strings.Contains(text, "KEYBOARD HELP") || !strings.Contains(text, "Repurpose") {
		t.Errorf("help panel:\n%s", text)
	}
	if foot := ansi.Strip(mm.(Model).footerLine()); !strings.Contains(foot, "? close") {
		t.Errorf("footer while help is open: %q", foot)
	}
}
