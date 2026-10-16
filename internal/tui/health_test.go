package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/aeon022/missionctl-core/tuitest"
)

func TestHealthExpiryTextWarnsEarly(t *testing.T) {
	now := time.Now()
	h := healthInfo{expires: map[string]time.Time{
		"instagram": now.AddDate(0, 0, 40),
		"threads":   now.AddDate(0, 0, 9),
		"twitter":   now.AddDate(0, 0, -1),
	}}
	if got := h.expiryText("instagram", now); !strings.Contains(got, "· 39 d") && !strings.Contains(got, "· 40 d") {
		t.Errorf("instagram: %q", got)
	}
	if got := h.expiryText("threads", now); !strings.Contains(got, "⚠") {
		t.Errorf("under 14 days must carry a warning, got %q", got)
	}
	if got := h.expiryText("twitter", now); !strings.Contains(got, "expired") {
		t.Errorf("past expiry must say expired, got %q", got)
	}
	if h.expiryText("facebook", now) != "" {
		t.Error("unknown expiry shows nothing")
	}
}

func TestDashboardShowsSchedulerAndOverdueWarning(t *testing.T) {
	m := layoutModel(t, 120, 40)
	m.platforms = map[string]bool{"instagram": true}
	m.health = healthInfo{schedulerOn: false, overdue: 3, expires: map[string]time.Time{"instagram": time.Now().AddDate(0, 0, 5)}}
	text := tuitest.Text(m)
	for _, want := range []string{"Scheduler off", "3 post(s) overdue", "⚠"} {
		if !strings.Contains(text, want) {
			t.Errorf("dashboard should show %q:\n%s", want, text)
		}
	}
	m.health.schedulerOn = true
	if text := tuitest.Text(m); !strings.Contains(text, "Scheduler running") || strings.Contains(text, "nothing will publish") {
		t.Errorf("with the scheduler on there is no 'nothing will publish' warning:\n%s", text)
	}
}
