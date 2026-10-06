package scheduler

import (
	"strings"

	"github.com/aeon022/missionctl-core/activity"
	"github.com/aeon022/postctl/internal/models"
)

// LogPost writes a "published" or "scheduled" event to the suite activity
// log. Title: the post's title (else the first 60 characters of its text)
// plus " → <platform>"; scheduled events add the date. Best-effort.
func LogPost(action string, p *models.Post) {
	t := strings.TrimSpace(p.Title)
	if t == "" {
		t = strings.TrimSpace(p.Body)
		if t == "" && len(p.Tweets) > 0 {
			t = strings.TrimSpace(p.Tweets[0].Content)
		}
		if r := []rune(strings.Join(strings.Fields(t), " ")); len(r) > 60 {
			t = string(r[:60]) + "…"
		} else {
			t = string(r)
		}
	}
	t += " → " + p.Platform
	if action == "scheduled" && p.ScheduledAt != nil {
		t += " (" + p.ScheduledAt.Format("2006-01-02 15:04") + ")"
	}
	activity.Log("postctl", action, t)
}
