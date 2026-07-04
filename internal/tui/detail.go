package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderDetailView rendert die Detail- bzw. Previewansicht eines Posts
func (m Model) renderDetailView() string {
	if m.selectedPost == nil {
		return ""
	}

	p := m.selectedPost
	var builder strings.Builder

	// Header der Detailansicht
	titleStr := fmt.Sprintf(" PREVIEW: %s %s ", strings.ToUpper(p.Platform), strings.ToUpper(p.Language))
	builder.WriteString(lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorBgFg).
		Background(ColorSecondary).
		Padding(0, 1).
		Render(titleStr))
	builder.WriteString("\n\n")

	// Metadaten
	builder.WriteString(fmt.Sprintf("Campaign: %s\n", p.Campaign))
	builder.WriteString(fmt.Sprintf("Type:     %s\n", p.Type))
	statusStr := strings.ToUpper(p.Status)
	if p.ScheduledAt != nil {
		statusStr += fmt.Sprintf(" (Scheduled at: %s)", p.ScheduledAt.Format("02.01.2006 15:04"))
	}
	builder.WriteString(fmt.Sprintf("Status:   %s\n", statusStr))
	if p.Error != "" {
		builder.WriteString(lipgloss.NewStyle().Foreground(ColorFailed).Render(fmt.Sprintf("Error:    %s\n", p.Error)))
	}
	builder.WriteString(fmt.Sprintf("File:     %s\n", p.SourceFile))
	builder.WriteString("\n")

	// Post Inhalt rendern
	if p.Type == "thread" {
		// Thread Tweets nacheinander auflisten
		for i, tweet := range p.Tweets {
			charCount := tweet.CharCount()
			charLimitOk := tweet.IsValid()
			
			// Charakter-Info
			charStyle := lipgloss.NewStyle().Foreground(ColorPosted)
			charText := fmt.Sprintf("[%d / 280 chars ✓]", charCount)
			if !charLimitOk {
				charStyle = lipgloss.NewStyle().Foreground(ColorFailed).Bold(true)
				charText = fmt.Sprintf("[%d / 280 chars ✗ - TOO LONG]", charCount)
			}
			
			titleLine := fmt.Sprintf("Tweet %d/%d", i+1, len(p.Tweets))
			if tweet.IsReply {
				titleLine += " (Reply)"
			}

			// Header Zeile des einzelnen Tweets
			builder.WriteString(lipgloss.JoinHorizontal(lipgloss.Top,
				StyleHeader.Render(titleLine),
				" ",
				charStyle.Render(charText),
			) + "\n")

			// Box für Tweet-Inhalt
			tweetBoxStyle := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorDarkGray).
				Width(65).
				Padding(0, 1)
			builder.WriteString(tweetBoxStyle.Render(tweet.Content) + "\n")

			// Bilder-Info
			if tweet.Image != "" {
				builder.WriteString(lipgloss.NewStyle().Foreground(ColorSecondary).Render(fmt.Sprintf("📎 Image: %s\n", tweet.Image)))
			} else {
				builder.WriteString(lipgloss.NewStyle().Foreground(ColorLightGray).Render("📎 No image\n"))
			}
			builder.WriteString("\n")
		}
	} else {
		// Single Post Body rendern
		builder.WriteString(StyleHeader.Render("Post Body") + "\n")
		bodyBoxStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorDarkGray).
			Width(65).
			Padding(1, 2)
		builder.WriteString(bodyBoxStyle.Render(p.Body) + "\n")

		if len(p.Images) > 0 {
			builder.WriteString(StyleHeader.Render("Images:") + "\n")
			for _, img := range p.Images {
				builder.WriteString(lipgloss.NewStyle().Foreground(ColorSecondary).Render(fmt.Sprintf("📎 %s\n", img)))
			}
		}
	}

	// Legend / Action-Guide für den Footer
	builder.WriteString("\n")
	builder.WriteString(StyleHelp.Render("esc: back  ·  e: edit  ·  d: delete  ·  r: repurpose via AI"))

	// Verwende eine große Box für die Detailansicht
	return StyleBox.Width(78).Height(20).Render(builder.String())
}
