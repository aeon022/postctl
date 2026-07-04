package tui

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var readmeContent string

// SetReadmeContent registers the embedded README text from main.go
func SetReadmeContent(content string) {
	readmeContent = content
}

type tocItem struct {
	title string
	line  int
	level int
}

// stripEmojis removes emojis and Variation Selectors from a string to ensure precise terminal cell-width calculation
func stripEmojis(s string) string {
	var sb strings.Builder
	for _, r := range s {
		// Emojis are generally in these Unicode ranges:
		// U+1F300 to U+1F9FF, U+2600 to U+26FF, U+2700 to U+27BF, and Variation Selector U+FE0F
		if (r >= 0x1F300 && r <= 0x1F9FF) || (r >= 0x2600 && r <= 0x26FF) || (r >= 0x2700 && r <= 0x27BF) || r == 0xFE0F {
			continue
		}
		sb.WriteRune(r)
	}
	res := sb.String()
	res = strings.ReplaceAll(res, "  ", " ") // remove double spaces
	return strings.TrimSpace(res)
}

// wrapLine wraps a single markdown line to a maximum character limit while preserving prefixes (bullets, spaces)
func wrapLine(line string, limit int) []string {
	if len(line) <= limit {
		return []string{line}
	}

	// Find the prefix (spaces, bullet points, numbers)
	prefix := ""
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return []string{line}
	}

	indentCount := strings.Index(line, trimmed)
	if indentCount > 0 {
		prefix = strings.Repeat(" ", indentCount)
	}

	// Bullet or list marker check
	rest := trimmed
	
	// Support double-digit numbered lists like "10. "
	dotIdx := strings.Index(trimmed, ". ")
	if strings.HasPrefix(trimmed, "* ") {
		prefix += "  "
		if len(trimmed) > 2 {
			rest = trimmed[2:]
		} else {
			rest = ""
		}
	} else if strings.HasPrefix(trimmed, "- ") {
		prefix += "  "
		if len(trimmed) > 2 {
			rest = trimmed[2:]
		} else {
			rest = ""
		}
	} else if dotIdx > 0 && dotIdx < 4 { // Matches "1. ", "10. ", etc.
		// Make sure all characters before the dot are digits
		isNum := true
		for i := 0; i < dotIdx; i++ {
			if trimmed[i] < '0' || trimmed[i] > '9' {
				isNum = false
				break
			}
		}
		if isNum {
			prefix += strings.Repeat(" ", dotIdx+2)
			rest = trimmed[dotIdx+2:]
		}
	}

	words := strings.Fields(rest)
	if len(words) == 0 {
		return []string{line}
	}

	var result []string
	// The first line gets the original prefix (e.g. "1. " or "* ")
	currentLine := line[:len(line)-len(rest)] + words[0]

	for _, word := range words[1:] {
		if len(currentLine)+1+len(word) > limit {
			result = append(result, currentLine)
			currentLine = prefix + word
		} else {
			currentLine += " " + word
		}
	}
	result = append(result, currentLine)
	return result
}

func getReadmeData() ([]string, []tocItem) {
	rawLines := strings.Split(readmeContent, "\n")
	var wrappedLines []string
	var toc []tocItem

	for _, line := range rawLines {
		trimmed := strings.TrimSpace(line)
		
		// If it's a header, record it in the TOC pointing to the exact current index in wrappedLines
		if strings.HasPrefix(trimmed, "#") {
			parts := strings.SplitN(trimmed, " ", 2)
			if len(parts) == 2 && strings.HasPrefix(parts[0], "#") {
				level := len(parts[0])
				title := strings.TrimSpace(parts[1])
				
				// Clean formatting & strip emojis
				title = strings.ReplaceAll(title, "`", "")
				title = strings.ReplaceAll(title, "**", "")
				title = strings.ReplaceAll(title, "*", "")
				title = stripEmojis(title)
				
				toc = append(toc, tocItem{
					title: title,
					line:  len(wrappedLines), // line index in wrappedLines
					level: level,
				})
			}
		}

		// Wrap and append to the final slice
		if trimmed == "" || strings.HasPrefix(trimmed, "```") {
			wrappedLines = append(wrappedLines, line)
		} else {
			// Limit to 64 chars to comfortably fit inside any scaled box width >= 72
			wrappedLines = append(wrappedLines, wrapLine(line, 64)...)
		}
	}

	return wrappedLines, toc
}

func (m Model) renderReadmeTOC() string {
	var builder strings.Builder

	// Dynamic Sizing
	outerWidth := 78
	outerHeight := 22
	if m.width > 10 {
		outerWidth = max(78, min(100, m.width - 4))
	}
	if m.height > 10 {
		outerHeight = max(22, m.height - 4)
	}

	innerWidth := outerWidth - 6
	innerHeight := outerHeight - 4

	// Header
	headerStr := " SYSTEM DOKUMENTATION & README — INHALTSVERZEICHNIS "
	builder.WriteString(lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorBgFg).
		Background(ColorSecondary).
		Padding(0, 1).
		Render(headerStr))
	builder.WriteString("\n\n")

	// TOC List
	var tocBuilder strings.Builder
	for i, item := range m.readmeTOC {
		cursor := "  "
		selected := false
		if i == m.tocCursor {
			cursor = "> "
			selected = true
		}

		// Indentation based on heading level
		indent := strings.Repeat("  ", max(0, item.level-1))
		title := item.title
		
		// Limit length to fit in column
		runes := []rune(title)
		maxLen := (innerWidth - 6) - len(indent)
		if len(runes) > maxLen {
			if maxLen > 2 {
				title = string(runes[:maxLen-2]) + ".."
			} else {
				title = ".."
			}
		}

		lineStyle := lipgloss.NewStyle().Foreground(ColorLightGray)
		if selected {
			lineStyle = lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true)
		} else if item.level == 1 {
			lineStyle = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
		} else if item.level > 2 {
			lineStyle = lipgloss.NewStyle().Foreground(ColorLightGray)
		}

		tocBuilder.WriteString(fmt.Sprintf("%s%s%s\n", cursor, indent, lineStyle.Render(title)))
	}

	// Pad remaining height in TOC list
	linesRendered := len(m.readmeTOC)
	if linesRendered < innerHeight {
		for k := 0; k < innerHeight-linesRendered; k++ {
			tocBuilder.WriteString("\n")
		}
	}

	tocBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorSecondary).
		Width(innerWidth).
		Height(innerHeight)

	tocBox := tocBoxStyle.Render(tocBuilder.String())
	builder.WriteString(tocBox)
	builder.WriteString("\n\n")

	// Help bar
	helpStr := "↑/↓/j/k: Navigation  ·  enter: Auswählen/Springen  ·  esc/q: Schließen"
	builder.WriteString(StyleHelp.Render(helpStr))

	return StyleBox.Width(outerWidth).Height(outerHeight).Render(builder.String())
}

func (m Model) renderReadmeContent() string {
	var builder strings.Builder

	// Dynamic Sizing
	outerWidth := 78
	outerHeight := 22
	if m.width > 10 {
		outerWidth = max(78, min(100, m.width - 4))
	}
	if m.height > 10 {
		outerHeight = max(22, m.height - 4)
	}

	innerWidth := outerWidth - 6
	innerHeight := outerHeight - 4
	viewportHeight := innerHeight - 2

	// Header
	headerStr := " SYSTEM DOKUMENTATION & README "
	builder.WriteString(lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorBgFg).
		Background(ColorSecondary).
		Padding(0, 1).
		Render(headerStr))
	builder.WriteString("\n\n")

	// Content Viewport
	var contentBuilder strings.Builder
	
	inCodeBlock := false
	
	// Determine code block state at m.readmeScroll
	for i := 0; i < m.readmeScroll && i < len(m.readmeLines); i++ {
		if strings.HasPrefix(strings.TrimSpace(m.readmeLines[i]), "```") {
			inCodeBlock = !inCodeBlock
		}
	}

	endLine := m.readmeScroll + viewportHeight
	if endLine > len(m.readmeLines) {
		endLine = len(m.readmeLines)
	}

	for i := m.readmeScroll; i < endLine; i++ {
		line := m.readmeLines[i]
		trimmed := strings.TrimSpace(line)

		// Codeblock marker toggle (cleanly hidden, styled by indentation)
		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			continue
		}

		if inCodeBlock {
			contentBuilder.WriteString("  " + lipgloss.NewStyle().Foreground(ColorPosted).Render(line) + "\n")
			continue
		}

		// Horizontal rule divider
		if trimmed == "---" {
			contentBuilder.WriteString(lipgloss.NewStyle().Foreground(ColorDarkGray).Render(strings.Repeat("─", innerWidth-4)) + "\n")
			continue
		}

		// Formatting headers without markdown hashes and strip emojis for border safety
		if strings.HasPrefix(trimmed, "# ") {
			title := strings.ToUpper(strings.TrimPrefix(trimmed, "# "))
			title = stripEmojis(title)
			if i > 0 {
				contentBuilder.WriteString(lipgloss.NewStyle().Foreground(ColorDarkGray).Render("  ▲ t: Zurück zum Inhaltsverzeichnis / Back to Top") + "\n")
			}
			contentBuilder.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render("█ "+title) + "\n")
		} else if strings.HasPrefix(trimmed, "## ") {
			title := strings.TrimPrefix(trimmed, "## ")
			title = stripEmojis(title)
			if i > 0 {
				contentBuilder.WriteString(lipgloss.NewStyle().Foreground(ColorDarkGray).Render("  ▲ t: Zurück zum Inhaltsverzeichnis / Back to Top") + "\n")
			}
			contentBuilder.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render("❯ "+title) + "\n")
		} else if strings.HasPrefix(trimmed, "### ") {
			title := strings.TrimPrefix(trimmed, "### ")
			title = stripEmojis(title)
			contentBuilder.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render("  "+title) + "\n")
		} else if strings.HasPrefix(trimmed, "#### ") {
			title := strings.TrimPrefix(trimmed, "#### ")
			title = stripEmojis(title)
			contentBuilder.WriteString(lipgloss.NewStyle().Bold(true).Underline(true).Foreground(ColorLightGray).Render("  "+title) + "\n")
		} else {
			// Format bullet points
			if strings.HasPrefix(trimmed, "* ") || strings.HasPrefix(trimmed, "- ") {
				bullet := "•"
				var text string
				if len(trimmed) > 2 {
					text = trimmed[2:]
				}
				text = formatInlineMarkdown(text)
				contentBuilder.WriteString(lipgloss.NewStyle().Foreground(ColorSecondary).Render(bullet) + " " + text + "\n")
			} else {
				contentBuilder.WriteString(formatInlineMarkdown(line) + "\n")
			}
		}
	}

	// Pad remaining height if text is shorter than viewport height
	linesRendered := endLine - m.readmeScroll
	if linesRendered < viewportHeight {
		for k := 0; k < viewportHeight-linesRendered; k++ {
			contentBuilder.WriteString("\n")
		}
	}

	contentBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorSecondary).
		Width(innerWidth).
		Height(innerHeight)

	contentBox := contentBoxStyle.Render(contentBuilder.String())
	builder.WriteString(contentBox)
	builder.WriteString("\n\n")

	// Help bar
	helpStr := "↑/↓/j/k: Scrollen  ·  t/backspace: Zum Inhaltsverzeichnis  ·  esc/q: Schließen"
	builder.WriteString(StyleHelp.Render(helpStr))

	return StyleBox.Width(outerWidth).Height(outerHeight).Render(builder.String())
}

func (m Model) renderReadme() string {
	if m.readmeFocus == 0 {
		return m.renderReadmeTOC()
	}
	return m.renderReadmeContent()
}

// Simple inline markdown formatting (e.g. code -> cyan, bold -> bold)
func formatInlineMarkdown(text string) string {
	// 1. Format markdown links [label](url) -> underlined label
	text = formatLinks(text)

	// 2. Format backticks
	parts := strings.Split(text, "`")
	for idx := 1; idx < len(parts); idx += 2 {
		parts[idx] = lipgloss.NewStyle().Foreground(ColorSecondary).Render(parts[idx])
	}
	text = strings.Join(parts, "")

	// 3. Format bold markers
	boldParts := strings.Split(text, "**")
	for idx := 1; idx < len(boldParts); idx += 2 {
		boldParts[idx] = lipgloss.NewStyle().Bold(true).Foreground(ColorLightGray).Render(boldParts[idx])
	}
	return strings.Join(boldParts, "")
}

// formatLinks parses markdown link syntax [label](url) and keeps only the label formatted as underlined cyan
func formatLinks(text string) string {
	var result strings.Builder
	current := text
	for {
		start := strings.Index(current, "[")
		if start == -1 {
			result.WriteString(current)
			break
		}
		result.WriteString(current[:start])
		current = current[start:]

		endLabel := strings.Index(current, "]")
		if endLabel == -1 {
			result.WriteString(current)
			break
		}

		if endLabel+1 >= len(current) || current[endLabel+1] != '(' {
			result.WriteString(current[:endLabel+1])
			current = current[endLabel+1:]
			continue
		}

		endUrl := strings.Index(current[endLabel+1:], ")")
		if endUrl == -1 {
			result.WriteString(current)
			break
		}
		endUrlIdx := endLabel + 1 + endUrl

		label := current[1:endLabel]
		styledLabel := lipgloss.NewStyle().Underline(true).Foreground(ColorSecondary).Render(label)
		result.WriteString(styledLabel)
		current = current[endUrlIdx+1:]
	}
	return result.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
