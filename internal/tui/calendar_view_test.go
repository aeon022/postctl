package tui

import (
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
