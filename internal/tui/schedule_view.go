package tui

import (
	"fmt"
	"strings"

	"github.com/aeon022/postctl/internal/config"
	"github.com/charmbracelet/lipgloss"
)

// renderSchedule rendert den Schedule-Tab (Tab 2) mit Kampagnen-Gruppierung und Scroll-Unterstützung
func (m Model) renderSchedule() string {
	var builder strings.Builder

	builder.WriteString(StyleHeader.Render("SCHEDULED POSTS") + "\n")
	if len(m.nextUp) == 0 {
		builder.WriteString("No posts currently scheduled.\n")
		return StyleBox.Width(84).Height(12).Render(builder.String())
	}

	type lineItem struct {
		text      string
		isHeader  bool
		postIndex int // Index in m.nextUp, -1 für Header
	}
	var items []lineItem

	var lastCampaign string
	for idx, p := range m.nextUp {
		if p.Campaign != lastCampaign || idx == 0 {
			lastCampaign = p.Campaign
			campaignName := lastCampaign
			if campaignName == "" {
				if strings.ToLower(config.ActiveConfig.Defaults.Language) == "de" {
					campaignName = "EINZELPOSTS"
				} else {
					campaignName = "INDIVIDUAL POSTS"
				}
			}
			items = append(items, lineItem{
				text:      "\n" + StyleHeader.Render("📁 "+strings.ToUpper(campaignName)),
				isHeader:  true,
				postIndex: -1,
			})
		}

		timeStr := ""
		if p.ScheduledAt != nil {
			timeStr = p.ScheduledAt.Format("02.01.2006 15:04")
		}

		titlePreview := stripEmojis(p.Title)
		if len(titlePreview) > 40 {
			titlePreview = titlePreview[:37] + "..."
		}

		items = append(items, lineItem{
			text:      fmt.Sprintf("◷ %-17s %-8s %-2s  %s", timeStr, strings.ToUpper(p.Platform), strings.ToUpper(p.Language), titlePreview),
			isHeader:  false,
			postIndex: idx,
		})
	}

	// Index des aktuell ausgewählten Elements in der Zeilenliste (items) ermitteln
	selectedItemIdx := -1
	for idx, item := range items {
		if item.postIndex == m.cursor {
			selectedItemIdx = idx
			break
		}
	}

	// Viewport-Größe (Höhe des Inhalts-Bereichs)
	boxHeight := m.getBoxHeight()
	viewportHeight := boxHeight - 6
	if viewportHeight < 5 {
		viewportHeight = 5
	}
	startIdx := 0
	endIdx := len(items)

	if len(items) > viewportHeight {
		startIdx = selectedItemIdx - viewportHeight/2
		if startIdx < 0 {
			startIdx = 0
		}
		if startIdx+viewportHeight > len(items) {
			startIdx = len(items) - viewportHeight
		}
		endIdx = startIdx + viewportHeight
	}

	for idx := startIdx; idx < endIdx; idx++ {
		item := items[idx]
		if item.isHeader {
			builder.WriteString(item.text + "\n")
		} else {
			cursor := "  "
			selected := false
			if m.activeTab == 2 && item.postIndex == m.cursor {
				cursor = "> "
				selected = true
			}

			itemStyle := lipgloss.NewStyle().Foreground(ColorLightGray)
			if selected {
				itemStyle = lipgloss.NewStyle().Foreground(ColorSecondary)
			}

			p := m.nextUp[item.postIndex]
			timeStr := ""
			if p.ScheduledAt != nil {
				timeStr = p.ScheduledAt.Format("02.01.2006 15:04")
			}
			titlePreview := stripEmojis(p.Title)
			if len(titlePreview) > 40 {
				titlePreview = titlePreview[:37] + "..."
			}

			platformStr := fmt.Sprintf("%-8s", strings.ToUpper(p.Platform))

			builder.WriteString(fmt.Sprintf("%s◷ %-17s %s %-2s  %s\n", 
				cursor,
				timeStr,
				itemStyle.Render(platformStr),
				strings.ToUpper(p.Language),
				titlePreview,
			))
		}
	}

	return StyleBox.Width(84).Height(boxHeight).Render(builder.String())
}
