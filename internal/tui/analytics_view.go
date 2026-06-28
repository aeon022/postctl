package tui

import (
	"fmt"
	"math"
	"strings"

	"github.com/aeon022/postctl/internal/config"
	"github.com/charmbracelet/lipgloss"
)

// renderAnalytics rendert die Analytics-Ansicht (Tab 4)
func (m Model) renderAnalytics() string {
	if m.analyticsLoading {
		return "\n  📈 Lade Social Analytics & Engagement-Daten...\n"
	}

	if m.analyticsData == nil {
		return "\n  Keine Daten geladen.\n"
	}

	data := m.analyticsData
	if data.err != nil {
		return fmt.Sprintf("\n  [FEHLER BEIM LADEN]: %v\n", data.err)
	}

	var builder strings.Builder

	// 1. Lizenz-Banner
	var banner string
	if !config.IsPro() {
		bannerText := "⚡ " + StyleStatusScheduled.Render(" CORE TIERS ") + " Analyse auf die letzten 3 Tage beschränkt. Hol dir postctl Pro für unbegrenzte Historie!"
		banner = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorScheduled).
			Padding(0, 1).
			Width(78).
			Render(bannerText)
	} else {
		bannerText := "🌟 " + StyleStatusPosted.Render(" PRO ACTIVE ") + " Vollständige Social Analytics & Engagement-Statistiken freigeschaltet."
		banner = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorSecondary).
			Padding(0, 1).
			Width(78).
			Render(bannerText)
	}
	builder.WriteString(banner + "\n\n")

	// 2. Zusammenfassung
	var sumCol strings.Builder
	sumCol.WriteString(StyleHeader.Render("ZUSAMMENFASSUNG") + "\n")
	sumCol.WriteString(fmt.Sprintf("Veröffentlichte Beiträge: %d\n", data.totalPosts))
	sumCol.WriteString(fmt.Sprintf("Likes (Gefällt mir):      %d\n", data.totalLikes))
	sumCol.WriteString(fmt.Sprintf("Shares (Teilungen):       %d\n", data.totalShares))
	sumCol.WriteString(fmt.Sprintf("Comments (Kommentare):    %d\n", data.totalComments))
	sumCol.WriteString(fmt.Sprintf("Impressions (Ansichten):  %d\n", data.totalImpressions))

	// 3. Engagement-Verteilung
	var chartCol strings.Builder
	chartCol.WriteString(StyleHeader.Render("INTERAKTIONS-VERTEILUNG") + "\n")

	totalInteractions := data.totalLikes + data.totalShares + data.totalComments
	platforms := []string{"twitter", "linkedin", "threads", "mastodon", "bluesky", "facebook"}
	platNames := map[string]string{
		"twitter":  "Twitter/X",
		"linkedin": "LinkedIn",
		"threads":  "Threads",
		"mastodon": "Mastodon",
		"bluesky":  "Bluesky",
		"facebook": "Facebook",
	}

	for _, p := range platforms {
		sum, exists := data.platStats[p]
		if !exists || sum.Posts == 0 {
			continue
		}
		interactions := sum.Likes + sum.Shares + sum.Comments
		percentage := 0.0
		if totalInteractions > 0 {
			percentage = float64(interactions) / float64(totalInteractions) * 100.0
		}

		// Balken zeichnen (Breite: 15 Blöcke)
		barWidth := 15
		filledCount := int(math.Round(float64(barWidth) * (percentage / 100.0)))
		if filledCount > barWidth {
			filledCount = barWidth
		}
		
		barStr := strings.Repeat("█", filledCount) + strings.Repeat("░", barWidth-filledCount)
		// Einfärben des Balkens
		styledBar := lipgloss.NewStyle().Foreground(ColorSecondary).Render(barStr)
		chartCol.WriteString(fmt.Sprintf("%-10s %s %3.0f%%\n", platNames[p], styledBar, percentage))
	}

	// Beides in Boxen verpacken
	box1 := StyleBox.Width(37).Height(7).Render(sumCol.String())
	box2 := StyleBox.Width(38).Height(7).Render(chartCol.String())
	builder.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, box1, "   ", box2) + "\n\n")

	// 4. Tabellen-Breakdown pro Plattform
	var breakdown strings.Builder
	breakdown.WriteString(StyleHeader.Render("PLATTFORMEN-DETAILS") + "\n")
	breakdown.WriteString(fmt.Sprintf("  %-12s | %-5s | %-5s | %-5s | %-5s | %-6s\n", "Plattform", "Posts", "Likes", "Share", "Comm.", "Impr."))
	breakdown.WriteString("  " + strings.Repeat("-", 53) + "\n")

	hasPlats := false
	for _, p := range platforms {
		sum, exists := data.platStats[p]
		if exists && sum.Posts > 0 {
			hasPlats = true
			breakdown.WriteString(fmt.Sprintf("  %-12s | %-5d | %-5d | %-5d | %-5d | %-6d\n",
				platNames[p], sum.Posts, sum.Likes, sum.Shares, sum.Comments, sum.Impressions))
		}
	}
	if !hasPlats {
		breakdown.WriteString("  Keine Beitragsdetails vorhanden.\n")
	}

	builder.WriteString(StyleBox.Width(78).Render(breakdown.String()))

	return builder.String()
}
