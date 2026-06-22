package tui

import (
	"fmt"
	"strings"

	"github.com/aeon022/postctl/internal/models"
	"github.com/charmbracelet/lipgloss"
)

// renderDashboard rendert die Dashboard-Ansicht (Tab 0)
func (m Model) renderDashboard() string {
	// Spalte 1: Campaigns & Next Up
	var col1 strings.Builder

	col1.WriteString(StyleHeader.Render("CAMPAIGNS") + "\n")
	if len(m.campaigns) == 0 {
		col1.WriteString("No campaigns found.\n")
	} else {
		for i, c := range m.campaigns {
			cursor := "  "
			if m.activeTab == 0 && i == m.cursor {
				cursor = "> "
			}
			col1.WriteString(fmt.Sprintf("%s● %s\n   %d posts (%d posted, %d scheduled)\n", 
				cursor, c.Slug, len(c.Posts), c.Posted, c.Scheduled))
		}
	}
	col1.WriteString("\n")

	col1.WriteString(StyleHeader.Render("NEXT UP") + "\n")
	if len(m.nextUp) == 0 {
		col1.WriteString("No posts scheduled.\n")
	} else {
		// Maximal 3 anstehende Posts anzeigen
		limit := 3
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
			if len(titlePreview) > 30 {
				titlePreview = titlePreview[:27] + "..."
			}
			col1.WriteString(fmt.Sprintf("◷ %-11s %-8s %-2s  %s\n", 
				timeStr, strings.ToUpper(p.Platform), strings.ToUpper(p.Language), titlePreview))
		}
	}

	// Spalte 2: Stats & Platforms
	var col2 strings.Builder

	col2.WriteString(StyleHeader.Render("STATS") + "\n")
	col2.WriteString(fmt.Sprintf("Posted:    %d\n", m.stats.posted))
	col2.WriteString(fmt.Sprintf("Scheduled: %d\n", m.stats.scheduled))
	col2.WriteString(fmt.Sprintf("Drafts:    %d\n", m.stats.drafts))
	col2.WriteString(fmt.Sprintf("Failed:    %d\n", m.stats.failed))
	col2.WriteString("\n\n")

	col2.WriteString(StyleHeader.Render("PLATFORMS") + "\n")
	platforms := []string{models.PlatformTwitter, models.PlatformLinkedIn, models.PlatformThreads}
	for _, p := range platforms {
		status := "○ not auth'd"
		if m.platforms[p] {
			status = "✓ connected"
		}
		name := p
		if p == models.PlatformTwitter {
			name = "Twitter/X"
		} else if p == models.PlatformLinkedIn {
			name = "LinkedIn"
		} else if p == models.PlatformThreads {
			name = "Threads"
		}
		col2.WriteString(fmt.Sprintf("%-10s %s\n", name+":", status))
	}

	// Beider Spalten in Boxen verpacken
	box1 := StyleBox.Width(40).Height(14).Render(col1.String())
	box2 := StyleBox.Width(35).Height(14).Render(col2.String())

	return lipgloss.JoinHorizontal(lipgloss.Top, box1, "   ", box2)
}

// renderPostList rendert die Liste aller Posts (Tab 1)
func (m Model) renderPostList() string {
	var builder strings.Builder

	headerText := "POSTS"
	if m.filterCampaign != "" {
		headerText = fmt.Sprintf("POSTS (Filter: Campaign = %s) [ESC to clear]", m.filterCampaign)
	}
	builder.WriteString(StyleHeader.Render(headerText) + "\n")

	if len(m.posts) == 0 {
		builder.WriteString("No posts found. Use 'postctl import <path>' to import markdown posts.\n")
		return StyleBox.Width(78).Height(12).Render(builder.String())
	}

	filtered := m.getFilteredPosts()
	if len(filtered) == 0 {
		builder.WriteString(fmt.Sprintf("No posts found for campaign %q.\n", m.filterCampaign))
		return StyleBox.Width(78).Height(12).Render(builder.String())
	}

	for i, p := range filtered {
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
			metaInfo = fmt.Sprintf("thread · %d tweets", len(p.Tweets))
		} else {
			metaInfo = "single"
		}
		if len(p.Images) > 0 {
			metaInfo += fmt.Sprintf(" · 📎 %d images", len(p.Images))
		}
		if p.Campaign != "" {
			metaInfo += fmt.Sprintf(" · 📁 %s", p.Campaign)
		}

		titlePreview := p.Title
		if titlePreview == "" {
			titlePreview = "(no title)"
		}
		if len(titlePreview) > 45 {
			titlePreview = titlePreview[:42] + "..."
		}

		// Listeneintrag gestalten
		lineColor := lipgloss.Color("#cbd5e0")
		if selected {
			lineColor = ColorSecondary
		}
		itemStyle := lipgloss.NewStyle().Foreground(lineColor)

		builder.WriteString(fmt.Sprintf("%s%s / %s %s\n", 
			cursor, strings.ToUpper(p.Platform), strings.ToUpper(p.Language), statusStr))
		builder.WriteString(itemStyle.Render(fmt.Sprintf("    %q\n", titlePreview)))
		builder.WriteString(lipgloss.NewStyle().Foreground(ColorLightGray).Render(fmt.Sprintf("    %s\n\n", metaInfo)))
	}

	return StyleBox.Width(78).Height(16).Render(builder.String())
}
