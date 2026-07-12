package tui

import (
	"fmt"
	"math"
	"strings"

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

	// 1. API-Info-Banner
	infoText := "ℹ  Für Twitter/X, LinkedIn und Threads sind eigene API-Zugangsdaten nötig, um echte Interaktionsdaten abzurufen. Ohne API werden diese als 0 angezeigt. Mastodon und Bluesky nutzen Live-Daten."
	infoBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 1).
		Width(78).
		Render(infoText)
	builder.WriteString(infoBox + "\n\n")

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

	// Beides in Boxen verpacken (37 + 3 + 38 = 78)
	box1 := StyleBox.Width(37).Height(7).Render(strings.TrimRight(sumCol.String(), "\n"))
	box2 := StyleBox.Width(38).Height(7).Render(strings.TrimRight(chartCol.String(), "\n"))
	builder.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, box1, "   ", box2) + "\n\n")

	// 3b. Trend Chart (Letzte 30 Tage)
	var trendCol strings.Builder
	trendCol.WriteString(StyleHeader.Render("ENGAGEMENT-TREND (LETZTE 30 TAGE - Likes/Shares/Comments)") + "\n\n")
	trendCol.WriteString(renderTrendChart(data.dailyEngagement) + "\n")
	boxTrend := StyleBox.Width(78).Render(strings.TrimRight(trendCol.String(), "\n"))
	builder.WriteString(boxTrend + "\n\n")

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

	builder.WriteString(StyleBox.Width(78).Render(strings.TrimRight(breakdown.String(), "\n")))

	return builder.String()
}

// renderTrendChart zeichnet ein vertikales ASCII-Balkendiagramm für die TUI
func renderTrendChart(engagement []int) string {
	maxVal := 0
	for _, v := range engagement {
		if v > maxVal {
			maxVal = v
		}
	}
	if maxVal == 0 {
		maxVal = 1
	}

	chartHeight := 5
	var lines []string

	for r := chartHeight; r > 0; r-- {
		var line strings.Builder
		valAtStep := int(float64(maxVal) * float64(r) / float64(chartHeight))
		line.WriteString(fmt.Sprintf("%3d │ ", valAtStep))
		
		for _, val := range engagement {
			fraction := float64(val) / float64(maxVal)
			threshold := float64(r) / float64(chartHeight)
			prevThreshold := float64(r-1) / float64(chartHeight)
			
			if fraction >= threshold {
				line.WriteString("█ ")
			} else if fraction >= prevThreshold + (threshold-prevThreshold)/2 {
				line.WriteString("▄ ")
			} else {
				line.WriteString("  ")
			}
		}
		lines = append(lines, line.String())
	}

	// X-Achse zeichnen (Präfix "  0 └──" hat Länge 7, danach 30 * 2 = 60 Zeichen für die Achse)
	var xAxis strings.Builder
	xAxis.WriteString("  0 └──")
	for i := 0; i < len(engagement); i++ {
		xAxis.WriteString("──")
	}
	lines = append(lines, xAxis.String())

	// Labels (Präfix "     " -> 5 Leerzeichen)
	var labels strings.Builder
	labels.WriteString("     30 Tage her")
	// Graph beginnt bei Index 7. "30 Tage her" beginnt bei Index 5, Länge 11 (endet bei 16).
	// "Heute" soll am Ende unter dem letzten Balken stehen (Balken-Start bei Index 7 + 29*2 = 65).
	// Abstand bis zum Start von "Heute": 65 - 16 = 49 Leerzeichen.
	labels.WriteString(strings.Repeat(" ", 43))
	labels.WriteString("Heute")
	lines = append(lines, labels.String())

	return strings.Join(lines, "\n")
}
