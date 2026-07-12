package tui

import (
	"fmt"
	"strings"

	"github.com/aeon022/postctl/internal/models"
	"github.com/charmbracelet/lipgloss"
)

// renderDashboard rendert die Dashboard-Ansicht (Tab 0)
func (m Model) renderDashboard() string {
	boxHeight := m.getBoxHeight()

	// Spalte 1: Campaigns & Next Up
	var col1 strings.Builder

	col1.WriteString(StyleHeader.Render(Tr("dash_campaigns")) + "\n")
	if len(m.campaigns) == 0 {
		col1.WriteString(Tr("dash_no_campaigns"))
	} else {
		innerCampaignsHeight := boxHeight - 14
		visibleCampaigns := innerCampaignsHeight / 2
		if visibleCampaigns < 2 {
			visibleCampaigns = 2
		}

		startIdx := 0
		endIdx := len(m.campaigns)
		if len(m.campaigns) > visibleCampaigns {
			startIdx = m.cursor - visibleCampaigns/2
			if startIdx < 0 {
				startIdx = 0
			}
			if startIdx+visibleCampaigns > len(m.campaigns) {
				startIdx = len(m.campaigns) - visibleCampaigns
			}
			endIdx = startIdx + visibleCampaigns
		}

		for i := startIdx; i < endIdx; i++ {
			c := m.campaigns[i]
			cursor := "  "
			if m.activeTab == 0 && i == m.cursor {
				cursor = "> "
			}
			col1.WriteString(fmt.Sprintf("%s● %s\n"+Tr("dash_campaign_format"), 
				cursor, c.Slug, len(c.Posts), c.Posted, c.Scheduled))
		}
	}
	col1.WriteString("\n")

	col1.WriteString(StyleHeader.Render(Tr("dash_next_up")) + "\n")
	if len(m.nextUp) == 0 {
		col1.WriteString(Tr("dash_no_schedules"))
	} else {
		// Maximal 5 anstehende Posts anzeigen
		limit := 5
		if len(m.nextUp) < limit {
			limit = len(m.nextUp)
		}
		for i := 0; i < limit; i++ {
			p := m.nextUp[i]
			timeStr := ""
			if p.ScheduledAt != nil {
				timeStr = p.ScheduledAt.Format("Mon 15:04")
			}
			titlePreview := p.Title
			if len(titlePreview) > 16 {
				titlePreview = titlePreview[:13] + "..."
			}
			col1.WriteString(fmt.Sprintf("◷ %-11s %-8s %-2s  %s\n", 
				timeStr, strings.ToUpper(p.Platform), strings.ToUpper(p.Language), titlePreview))
		}
	}

	// Spalte 2: Stats & Platforms
	var col2 strings.Builder

	col2.WriteString(StyleHeader.Render(Tr("dash_stats")) + "\n")
	col2.WriteString(fmt.Sprintf("%s%d\n", Tr("stats_posted"), m.stats.posted))
	col2.WriteString(fmt.Sprintf("%s%d\n", Tr("stats_scheduled"), m.stats.scheduled))
	col2.WriteString(fmt.Sprintf("%s%d\n", Tr("stats_drafts"), m.stats.drafts))
	col2.WriteString(fmt.Sprintf("%s%d\n", Tr("stats_failed"), m.stats.failed))
	col2.WriteString("\n\n")

	col2.WriteString(StyleHeader.Render(Tr("dash_platforms")) + "\n")
	platforms := []string{models.PlatformTwitter, models.PlatformLinkedIn, models.PlatformThreads, models.PlatformMastodon, models.PlatformBluesky, models.PlatformFacebook}
	for _, p := range platforms {
		status := Tr("dash_not_auth")
		if m.platforms[p] {
			status = Tr("dash_connected")
		}
		name := p
		if p == models.PlatformTwitter {
			name = "Twitter/X"
		} else if p == models.PlatformLinkedIn {
			name = "LinkedIn"
		} else if p == models.PlatformThreads {
			name = "Threads"
		} else if p == models.PlatformMastodon {
			name = "Mastodon"
		} else if p == models.PlatformBluesky {
			name = "Bluesky"
		} else if p == models.PlatformFacebook {
			name = "Facebook"
		}
		col2.WriteString(fmt.Sprintf("%-10s %s\n", name+":", status))
	}

	// Beider Spalten in Boxen verpacken
	box1 := StyleBox.Width(50).Height(boxHeight).Render(col1.String())
	box2 := StyleBox.Width(34).Height(boxHeight).Render(col2.String())

	return lipgloss.JoinHorizontal(lipgloss.Top, box1, "   ", box2)
}

// renderPostList rendert die Liste aller Posts (Tab 1)
func (m Model) renderPostList() string {
	var builder strings.Builder

	headerText := Tr("header_posts")
	if m.filterCampaign != "" {
		headerText = fmt.Sprintf(Tr("posts_header_filtered"), m.filterCampaign)
	}
	builder.WriteString(StyleHeader.Render(headerText) + "\n")

	if len(m.posts) == 0 {
		builder.WriteString(Tr("posts_none_found"))
		return StyleBox.Width(78).Height(12).Render(builder.String())
	}

	filtered := m.getFilteredPosts()
	if len(filtered) == 0 {
		builder.WriteString(fmt.Sprintf(Tr("posts_none_found_campaign"), m.filterCampaign))
		return StyleBox.Width(78).Height(12).Render(builder.String())
	}

	boxHeight := m.getBoxHeight()
	windowSize := (boxHeight - 6) / 4
	if windowSize < 2 {
		windowSize = 2
	}
	startIdx := 0
	endIdx := len(filtered)

	if len(filtered) > windowSize {
		startIdx = m.cursor - windowSize/2
		if startIdx < 0 {
			startIdx = 0
		}
		if startIdx+windowSize > len(filtered) {
			startIdx = len(filtered) - windowSize
		}
		endIdx = startIdx + windowSize
	}

	for i := startIdx; i < endIdx; i++ {
		p := filtered[i]
		cursor := "  "
		selected := false
		if m.activeTab == 1 && i == m.cursor {
			cursor = "> "
			selected = true
		}

		// Status Badge rendern
		var statusStr string
		switch p.Status {
		case models.StatusDraft:
			statusStr = StyleStatusDraft.Render(" DRAFT ")
		case models.StatusScheduled:
			timeStr := ""
			if p.ScheduledAt != nil {
				timeStr = p.ScheduledAt.Format(" 02.01. 15:04")
			}
			statusStr = StyleStatusScheduled.Render(" SCHED" + timeStr + " ")
		case models.StatusPosted:
			timeStr := ""
			if p.PostedAt != nil {
				timeStr = p.PostedAt.Format(" 02.01. 15:04")
			}
			statusStr = StyleStatusPosted.Render(" POSTED" + timeStr + " ")
		case models.StatusFailed:
			statusStr = StyleStatusFailed.Render(" FAILED ")
		}

		// Metadata Info
		metaInfo := ""
		if p.Type == "thread" {
			metaInfo = fmt.Sprintf(Tr("meta_thread"), len(p.Tweets))
		} else {
			metaInfo = Tr("meta_single")
		}
		if len(p.Images) > 0 {
			metaInfo += " · " + fmt.Sprintf(Tr("meta_images"), len(p.Images))
		}
		if p.Campaign != "" {
			metaInfo += " · 📁 " + p.Campaign
		}

		titlePreview := stripEmojis(p.Title)
		if titlePreview == "" {
			titlePreview = "(no title)"
		}
		if len(titlePreview) > 45 {
			titlePreview = titlePreview[:42] + "..."
		}

		// Listeneintrag gestalten
		lineColor := ColorLightGray
		if selected {
			lineColor = ColorSecondary
		}
		itemStyle := lipgloss.NewStyle().Foreground(lineColor)

		builder.WriteString(fmt.Sprintf("%s%s / %s %s\n", 
			cursor, strings.ToUpper(p.Platform), strings.ToUpper(p.Language), statusStr))
		builder.WriteString(itemStyle.Render(fmt.Sprintf("    %q", titlePreview)) + "\n")
		builder.WriteString(lipgloss.NewStyle().Foreground(ColorLightGray).Render(fmt.Sprintf("    %s", metaInfo)) + "\n\n")
	}

	return StyleBox.Width(84).Height(boxHeight).Render(builder.String())
}

// getBoxHeight berechnet die dynamische Höhe für die TUI-Boxen basierend auf der Terminal-Höhe
func (m Model) getBoxHeight() int {
	overhead := 12
	if m.showHelp {
		overhead = 26
	}
	
	h := m.height - overhead
	if h < 10 {
		return 12 // Mindesthöhe
	}
	if h > 24 {
		return 24 // Maximale Standardhöhe für Listenboxen
	}
	return h
}
