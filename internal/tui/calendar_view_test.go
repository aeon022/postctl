package tui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"strings"
	"testing"
	"time"

	"github.com/aeon022/missionctl-core/tuitest"
	"github.com/aeon022/postctl/internal/models"
)

func calModel(t *testing.T) Model {
	t.Helper()
	m := layoutModel(t, 120, 40)
	at := func(d, hh int) *time.Time { x := time.Date(2026, 10, d, hh, 0, 0, 0, time.Local); return &x }
	m.posts = []models.Post{
		{ID: "a", Platform: "instagram", Status: models.StatusScheduled, Title: "Late", ScheduledAt: at(20, 18)},
		{ID: "b", Platform: "threads", Status: models.StatusScheduled, Title: "Early", ScheduledAt: at(20, 9)},
		{ID: "c", Platform: "facebook", Status: models.StatusPosted, Title: "Done", PostedAt: at(16, 17), ScheduledAt: at(15, 8)},
		{ID: "d", Platform: "twitter", Status: models.StatusDraft, Title: "No date"},
	}
	m.calDay = time.Date(2026, 10, 20, 0, 0, 0, 0, time.Local)
	mm, _ := m.switchTab(tabCalendar)
	return mm
}

func TestCalendarGroupsPostsByDayAndSortsByTime(t *testing.T) {
	m := calModel(t)
	by := m.postsByDay()
	if got := by["2026-10-20"]; len(got) != 2 || got[0].ID != "b" || got[1].ID != "a" {
		t.Errorf("20.10. should list Early before Late, got %+v", got)
	}
	if got := by["2026-10-16"]; len(got) != 1 || got[0].ID != "c" {
		t.Errorf("a posted post belongs to the day it went out (16.10.), got %+v", got)
	}
	if len(by["2026-10-15"]) != 0 {
		t.Error("a posted post must not also appear on its old planned day")
	}
	text := tuitest.Text(m)
	if !strings.Contains(text, "October 2026") || !strings.Contains(text, "Early") || !strings.Contains(text, "Late") {
		t.Errorf("calendar view:\n%s", text)
	}
}

func TestCalendarKeysMoveDayWeekMonthAndOpenPost(t *testing.T) {
	m := calModel(t)
	for key, want := range map[string]string{"right": "2026-10-21", "left": "2026-10-19", "down": "2026-10-27", "up": "2026-10-13", "]": "2026-11-20", "[": "2026-09-20"} {
		mm, _, ok := m.calendarKey(tuitest.Key(key))
		if !ok || dayKey(mm.calDay) != want {
			t.Errorf("%q: day = %s, want %s", key, dayKey(mm.calDay), want)
		}
	}
	mm, _, _ := m.calendarKey(tuitest.Key("enter"))
	if mm.selectedPost == nil || mm.selectedPost.ID != "b" {
		t.Errorf("enter should open the first post of the day (Early), got %+v", mm.selectedPost)
	}
}

// locate returns the screen column and row where text first appears in the rendered view.
func locate(t *testing.T, m Model, text string) (x, y int) {
	t.Helper()
	for row, line := range strings.Split(tuitest.Text(m), "\n") {
		if i := strings.Index(line, text); i >= 0 {
			return lipgloss.Width(line[:i]), row
		}
	}
	t.Fatalf("%q not found on screen:\n%s", text, tuitest.Text(m))
	return 0, 0
}

func TestCalendarMouseClickSelectsDayThenOpensIt(t *testing.T) {
	m := calModel(t) // selected: 20.10., two posts that day; 16.10. has one posted post
	x, y := locate(t, m, "16 1")
	clicked, _ := m.Update(tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft})
	cm := clicked.(Model)
	if dayKey(cm.calDay) != "2026-10-16" || cm.selectedPost != nil {
		t.Fatalf("clicking the 16th selects it (day=%s)", dayKey(cm.calDay))
	}
	again, _ := cm.Update(tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft})
	if am := again.(Model); am.selectedPost == nil || am.selectedPost.ID != "c" {
		t.Errorf("clicking the selected day opens its post, got %+v", am.selectedPost)
	}
}

func TestCalendarMouseClickOnPostRowAndWheel(t *testing.T) {
	m := calModel(t)
	x, y := locate(t, m, "18:00") // the second post of the 20th ("Late") is listed in the day panel
	clicked, _ := m.Update(tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft})
	cm := clicked.(Model)
	if cm.calCursor != 1 || cm.selectedPost != nil {
		t.Fatalf("first click on a post row selects it (cursor=%d)", cm.calCursor)
	}
	again, _ := cm.Update(tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft})
	if am := again.(Model); am.selectedPost == nil || am.selectedPost.ID != "a" {
		t.Errorf("second click opens the post, got %+v", am.selectedPost)
	}

	down, _ := m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	if dayKey(down.(Model).calDay) != "2026-10-27" {
		t.Errorf("wheel down moves one week, got %s", dayKey(down.(Model).calDay))
	}
}

func TestCalendarMouseWorksInTheCompactWeekView(t *testing.T) {
	m := layoutModel(t, 100, 20) // too small for the month grid: only the selected week is shown
	m.calDay = time.Date(2026, 10, 20, 0, 0, 0, 0, time.Local)
	mm, _ := m.switchTab(tabCalendar)
	if strings.Contains(tuitest.Text(mm), "28 ") && strings.Contains(tuitest.Text(mm), " 5 ") {
		t.Skip("month grid fits at this size")
	}
	x, y := locate(t, mm, "22")
	clicked, _ := mm.Update(tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft})
	if got := dayKey(clicked.(Model).calDay); got != "2026-10-22" {
		t.Errorf("click on the 22nd in the week view selected %s", got)
	}
}
