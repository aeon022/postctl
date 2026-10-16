package tui

import (
	"context"
	"fmt"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/aeon022/missionctl-core/theme"
	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/models"
	"github.com/aeon022/postctl/internal/service"
	"github.com/aeon022/postctl/internal/store"
)

// healthInfo is what the dashboard needs to warn early: when tokens expire, whether the
// scheduler runs, and whether scheduled posts are overdue with nothing to publish them.
type healthInfo struct {
	expires     map[string]time.Time // platform → token expiry (known ones only)
	schedulerOn bool
	overdue     int
	autoPublish bool
}

const instagramTokenDays = 60

func collectHealth(ctx context.Context, s *store.SQLiteStore, posts []models.Post) healthInfo {
	h := healthInfo{expires: map[string]time.Time{}, autoPublish: config.AutoPublishEnabled()}
	for _, p := range dashPlatforms {
		if _, _, exp, err := s.GetToken(ctx, p); err == nil && exp != nil && !exp.IsZero() {
			h.expires[p] = *exp
		}
	}
	// Instagram's token lives in the config; it lasts 60 days from issue/refresh.
	if iss, err := time.ParseInLocation("2006-01-02", config.ActiveConfig.Instagram.TokenIssued, time.Local); err == nil && config.ActiveConfig.Instagram.AccessToken != "" {
		h.expires[models.PlatformInstagram] = iss.AddDate(0, 0, instagramTokenDays)
	}
	h.schedulerOn = service.Running(config.ActiveProfile)
	now := time.Now()
	for _, p := range posts {
		if p.Status == models.StatusScheduled && p.ScheduledAt != nil && p.ScheduledAt.Before(now.Add(-2*time.Minute)) {
			h.overdue++
		}
	}
	return h
}

var amberStyle = func() lipgloss.Style { return lipgloss.NewStyle().Foreground(theme.AmberV2) }

// expiryText is "· 52 d" (amber below 14 days, red when expired), "" when unknown.
func (h healthInfo) expiryText(platform string, now time.Time) string {
	exp, ok := h.expires[platform]
	if !ok {
		return ""
	}
	days := int(exp.Sub(now).Hours() / 24)
	switch {
	case exp.Before(now):
		return lipgloss.NewStyle().Foreground(theme.RedV2).Render("· " + Tr("health_expired"))
	case days < 14:
		return amberStyle().Render(fmt.Sprintf("· %d %s ⚠", days, Tr("health_days")))
	}
	return dimStyle.Render(fmt.Sprintf("· %d %s", days, Tr("health_days")))
}

// healthLines are the scheduler status and warnings shown under the platform list.
func (m Model) healthLines() []string {
	h := m.health
	var out []string
	if h.schedulerOn {
		out = append(out, lipgloss.NewStyle().Foreground(theme.GreenV2).Render("● ")+dimStyle.Render(Tr("health_sched_on")))
	} else {
		out = append(out, dimStyle.Render("○ "+Tr("health_sched_off")))
	}
	if h.overdue > 0 && !h.schedulerOn && !h.autoPublish {
		out = append(out, lipgloss.NewStyle().Foreground(theme.RedV2).Render(fmt.Sprintf("⚠ "+Tr("health_overdue"), h.overdue)))
	} else if h.overdue > 0 {
		out = append(out, amberStyle().Render(fmt.Sprintf("⚠ "+Tr("health_overdue_wait"), h.overdue)))
	}
	if m.stats.failed > 0 {
		out = append(out, lipgloss.NewStyle().Foreground(theme.RedV2).Render(fmt.Sprintf("✗ "+Tr("health_failed"), m.stats.failed)))
	}
	return out
}
