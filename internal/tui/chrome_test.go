package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/aeon022/missionctl-core/tuitest"
	"github.com/aeon022/postctl/internal/models"
	"github.com/aeon022/postctl/internal/platforms"
	"github.com/charmbracelet/x/ansi"
)

// chromeViews builds every secondary view (reached the way a user would, or by
// setting the state the view reads) on the layoutModel.
func chromeViews(t *testing.T, w, h int) map[string]Model {
	t.Helper()
	SetReadmeContent("# Manual\n\nIntro text with `code` and **bold**.\n\n## First\n\n* one\n* two\n\n```\ncode block\n```\n\n## Second\n\n" + strings.Repeat("A long paragraph line to scroll through. ", 40) + "\n")
	platforms.Log("a background log line")
	now := time.Now()
	hist := []models.HistoryEntry{
		{ID: "h1", PostID: "a", Action: "posted", PlatformID: "123", CreatedAt: now},
		{ID: "h2", PostID: "b", Action: "failed", Error: "401 unauthorized\nsecond line of the error " + strings.Repeat("x", 120), CreatedAt: now},
	}
	keys := func(k ...string) Model {
		m := layoutModel(t, w, h)
		m.history = hist
		mm, _ := tuitest.Keys(m, k...)
		m = mm.(Model)
		m.analyticsLoading = false
		m.analyticsData = &analyticsLoadedMsg{totalPosts: 3, totalLikes: 10, totalShares: 4, totalComments: 2, totalImpressions: 900,
			platStats:       map[string]*platMetricSummary{"bluesky": {Posts: 2, Likes: 10, Shares: 4, Comments: 2, Impressions: 900}},
			dailyEngagement: make([]int, 30)}
		for i := range m.analyticsData.dailyEngagement {
			m.analyticsData.dailyEngagement[i] = i % 7
		}
		return m
	}
	views := map[string]Model{
		"detail":         keys("tab", "enter"),
		"history":        keys("tab", "tab", "tab"),
		"history detail": keys("tab", "tab", "tab", "j", "enter"),
		"analytics":      keys("tab", "tab", "tab", "tab"),
		"settings":       keys("tab", "tab", "tab", "tab", "tab"),
		"logs":           keys("tab", "tab", "tab", "tab", "tab", "tab"),
		"help":           keys("?"),
		"help posts":     keys("tab", "?"),
		"readme toc":     keys("f1"),
		"readme content": keys("f1", "enter", "j"),
		"editor new":     keys("tab", "n"),
		"edit":           keys("tab", "e"),
		"calendar":       keys("tab", "n", "tab", "tab", "ctrl+d"),
	}
	for f := 1; f <= 6; f++ {
		views[fmt.Sprintf("editor focus %d", f)] = keys(append([]string{"tab", "n"}, strings.Split(strings.Repeat("tab,", f), ",")[:f]...)...)
	}
	th := keys("tab", "enter")
	p := *th.selectedPost
	p.Type = "thread"
	p.Tweets = []models.Tweet{{Index: 1, Content: strings.Repeat("first tweet ", 30)}, {Index: 2, Content: "second", IsReply: true, Image: "a.png"}}
	th.selectedPost = &p
	views["thread detail"] = th
	slots := keys("tab", "tab", "tab", "tab", "tab")
	slots.editingQueueSlots = true
	views["queue slots"] = slots
	return views
}

func TestEveryViewKeepsTheChrome(t *testing.T) {
	for _, sz := range layoutSizes {
		for name, m := range chromeViews(t, sz[0], sz[1]) {
			lines := strings.Split(m.View().Content, "\n")
			if len(lines) != sz[1]-1 {
				t.Errorf("%s %dx%d: %d lines, want %d (constant height)", name, sz[0], sz[1], len(lines), sz[1]-1)
			}
			for i, l := range lines {
				if lw := lipgloss.Width(l); lw > sz[0] {
					t.Errorf("%s %dx%d: line %d is %d wide: %q", name, sz[0], sz[1], i, lw, ansi.Strip(l))
					break
				}
			}
			plain := ansi.Strip(m.View().Content)
			pl := strings.Split(plain, "\n")
			if !strings.Contains(pl[0], "postctl") || !strings.Contains(pl[1], "──") || !strings.Contains(pl[tabsRow], m.tabLabels()[m.activeTab]) {
				t.Errorf("%s %dx%d: header/divider/tabs missing: %q", name, sz[0], sz[1], pl[:3])
			}
			foot := ansi.Strip(m.footerLine())
			if strings.Contains(foot, "\n") || lipgloss.Width(foot) > m.innerW() || strings.TrimSpace(foot) == "" {
				t.Errorf("%s %dx%d: footer is not one line within the width: %q", name, sz[0], sz[1], foot)
			}
			if strings.Contains(plain, "StyleBox") || strings.Contains(plain, "detail_") || strings.Contains(plain, "panel_") || strings.Contains(plain, "hint_") {
				t.Errorf("%s %dx%d: untranslated key on screen:\n%s", name, sz[0], sz[1], plain)
			}
		}
	}
}

func TestSecondaryViewFootersLeadWithEscBack(t *testing.T) {
	for name, m := range chromeViews(t, 140, 36) {
		foot := ansi.Strip(m.footerLine())
		switch name {
		case "detail", "thread detail", "history detail", "readme toc", "readme content":
			if !strings.HasPrefix(foot, "esc back") {
				t.Errorf("%s footer should start with 'esc back': %q", name, foot)
			}
		case "queue slots":
			if !strings.HasPrefix(foot, "esc cancel") || !strings.Contains(foot, "enter save") {
				t.Errorf("%s footer: %q", name, foot)
			}
		case "editor new", "edit":
			if !strings.HasPrefix(foot, "esc cancel") || !strings.Contains(foot, "tab next field") {
				t.Errorf("%s footer: %q", name, foot)
			}
		case "calendar":
			if !strings.HasPrefix(foot, "esc close") || !strings.Contains(foot, "enter pick") {
				t.Errorf("%s footer: %q", name, foot)
			}
		}
	}
}

func TestViewContentShowsTheRealThing(t *testing.T) {
	v := chromeViews(t, 140, 40)
	for name, wants := range map[string][]string{
		"detail":         {"Preview", "BLUESKY", "Campaign"},
		"thread detail":  {"1/2", "reply", "no image"},
		"history":        {"POSTED", "FAILED", "Post: a (ID: 123)"},
		"history detail": {"History entry", "Timestamp", "second line of the error"},
		"analytics":      {"Summary", "Interaction split", "Platform details", "Bluesky", "30 days ago"},
		"settings":       {"Settings", "Platform accounts", "Backup & sync", "Scheduler queue slots"},
		"logs":           {"a background log line"},
		"readme toc":     {"Table of contents", "Manual", "First"},
		"readme content": {"Manual", "back to contents"},
		"editor new":     {"Platform", "SAVE", "[TWITTER]"},
		"calendar":       {"Calendar", "Mo Tu We Th Fr Sa Su"},
		"help":           {"KEYBOARD HELP"},
	} {
		text := tuitest.Text(v[name])
		for _, want := range wants {
			if !strings.Contains(text, want) {
				t.Errorf("%s: missing %q:\n%s", name, want, text)
			}
		}
	}
}

func TestEditorKeepsTheFocusedFieldVisibleOnASmallTerminal(t *testing.T) {
	v := chromeViews(t, 80, 24)
	if text := tuitest.Text(v["editor focus 5"]); !strings.Contains(text, "[ SAVE ]") {
		t.Errorf("save button must be visible when focused:\n%s", text)
	}
	if text := tuitest.Text(v["editor focus 1"]); !strings.Contains(text, "➔") {
		t.Errorf("focused campaign field must be visible:\n%s", text)
	}
}

func TestHistoryRowSelectionFollowsTheCursor(t *testing.T) {
	m := chromeViews(t, 100, 30)["history"]
	if !strings.Contains(tuitest.Text(m), "▌") {
		t.Error("selected history row should carry the accent bar")
	}
	mm, _ := tuitest.Keys(m, "j")
	if mm.(Model).cursor != 1 {
		t.Errorf("cursor = %d", mm.(Model).cursor)
	}
}

func TestProfilePickerUsesTheChrome(t *testing.T) {
	for _, sz := range layoutSizes {
		m := profilePickerModel{profiles: []string{"", "work", "side"}, cursor: 1, width: sz[0], height: sz[1]}
		lines := strings.Split(m.viewContent(), "\n")
		if len(lines) != sz[1]-1 {
			t.Errorf("%dx%d: %d lines, want %d", sz[0], sz[1], len(lines), sz[1]-1)
		}
		for _, l := range lines {
			if lw := lipgloss.Width(l); lw > sz[0] {
				t.Errorf("%dx%d: line is %d wide: %q", sz[0], sz[1], lw, ansi.Strip(l))
				break
			}
		}
		plain := ansi.Strip(m.viewContent())
		for _, want := range []string{"postctl", "work", "▌"} {
			if !strings.Contains(plain, want) {
				t.Errorf("%dx%d: picker missing %q", sz[0], sz[1], want)
			}
		}
		if foot := strings.TrimSpace(ansi.Strip(lines[len(lines)-1])); !strings.Contains(foot, "esc quit") {
			t.Errorf("%dx%d footer: %q", sz[0], sz[1], foot)
		}
	}
}
