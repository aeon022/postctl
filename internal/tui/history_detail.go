package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderHistoryDetailView rendert die Detailansicht eines History-Eintrags
func (m Model) renderHistoryDetailView() string {
	if m.selectedHistory == nil {
		return ""
	}

	h := m.selectedHistory
	var builder strings.Builder

	// Header der Detailansicht
	titleStr := fmt.Sprintf(" HISTORY DETAIL: %s ", strings.ToUpper(h.Action))
	headerBg := ColorSecondary
	if h.Action == "posted" {
		headerBg = ColorPosted
	} else if h.Action == "failed" {
		headerBg = ColorFailed
	}

	builder.WriteString(lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorBgFg).
		Background(headerBg).
		Padding(0, 1).
		Render(titleStr))
	builder.WriteString("\n\n")

	// Metadaten
	builder.WriteString(fmt.Sprintf("Timestamp:   %s\n", h.CreatedAt.Format("02.01.2006 15:04:05")))
	builder.WriteString(fmt.Sprintf("Post ID:     %s\n", h.PostID))
	if h.PlatformID != "" {
		builder.WriteString(fmt.Sprintf("Platform ID: %s\n", h.PlatformID))
	}
	builder.WriteString("\n")

	// Details / Fehlermeldung
	builder.WriteString(StyleHeader.Render("Full Output / Error Message:") + "\n")
	contentBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorDarkGray).
		Width(72).
		Padding(1, 2)

	errText := h.Error
	if errText == "" {
		errText = "(No error recorded - Post published successfully)"
	}
	builder.WriteString(contentBoxStyle.Render(errText) + "\n")

	// Legend / Action-Guide für den Footer
	builder.WriteString("\n")
	builder.WriteString(StyleHelp.Render("esc: back  ·  x: export this entry to JSON  ·  j/k: scroll"))

	// Dynamic height bounding and scrolling
	boxHeight := m.getBoxHeight()
	lines := strings.Split(builder.String(), "\n")
	totalLines := len(lines)
	
	visibleLines := boxHeight - 4
	if visibleLines < 5 {
		visibleLines = 5
	}
	
	maxOffset := totalLines - visibleLines
	if maxOffset < 0 {
		maxOffset = 0
	}
	
	offset := m.detailScrollOffset
	if offset > maxOffset {
		offset = maxOffset
	}
	if offset < 0 {
		offset = 0
	}

	endIdx := offset + visibleLines
	if endIdx > totalLines {
		endIdx = totalLines
	}

	var visibleContent strings.Builder
	for i := offset; i < endIdx; i++ {
		visibleContent.WriteString(lines[i] + "\n")
	}

	return StyleBox.Width(78).Height(boxHeight).Render(visibleContent.String())
}
