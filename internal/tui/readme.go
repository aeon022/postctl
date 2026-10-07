package tui

import (
	_ "embed"
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/aeon022/missionctl-core/ui"
	"github.com/charmbracelet/x/ansi"
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

// renderReadmeTOC is the table of contents: one focused panel with
// selectable rows indented by heading level.
func (m Model) renderReadmeTOC(w, h int) string {
	rw := panelRowW(w)
	var rows []string
	for i, item := range m.readmeTOC {
		pad := strings.Repeat("  ", max(0, item.level-1))
		title := ansi.Truncate(item.title, max(rw-lipgloss.Width(pad)-2, 4), "…")
		style := lipgloss.NewStyle()
		if item.level == 1 {
			style = style.Bold(true).Foreground(ColorPrimary)
		}
		rows = append(rows, ui.Row(rw, i == m.tocCursor, pad+style.Render(title)))
	}
	start, end := window(len(rows), m.tocCursor, max(h-2, 1))
	return ui.Panel(w, h, Tr("panel_readme")+" · "+Tr("panel_toc"), strings.Join(rows[start:end], "\n"), true)
}

func (m Model) renderReadmeContent(w, h int) string {
	rw := panelRowW(w)
	viewport := max(h-2, 1)
	back := dimStyle.Render("  " + Tr("readme_back_to_top"))

	var out []string
	inCodeBlock := false
	// code block state at m.readmeScroll
	for i := 0; i < m.readmeScroll && i < len(m.readmeLines); i++ {
		if strings.HasPrefix(strings.TrimSpace(m.readmeLines[i]), "```") {
			inCodeBlock = !inCodeBlock
		}
	}
	for i := m.readmeScroll; i < min(m.readmeScroll+viewport, len(m.readmeLines)); i++ {
		line := m.readmeLines[i]
		trimmed := strings.TrimSpace(line)

		switch {
		case strings.HasPrefix(trimmed, "```"): // marker hidden, block styled by color
			inCodeBlock = !inCodeBlock
		case inCodeBlock:
			out = append(out, "  "+lipgloss.NewStyle().Foreground(ColorPosted).Render(line))
		case trimmed == "---":
			out = append(out, ui.Divider(rw, ""))
		case strings.HasPrefix(trimmed, "# "):
			if i > 0 {
				out = append(out, back)
			}
			out = append(out, lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render("█ "+strings.ToUpper(stripEmojis(strings.TrimPrefix(trimmed, "# ")))))
		case strings.HasPrefix(trimmed, "## "):
			if i > 0 {
				out = append(out, back)
			}
			out = append(out, lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary).Render("❯ "+stripEmojis(strings.TrimPrefix(trimmed, "## "))))
		case strings.HasPrefix(trimmed, "### "):
			out = append(out, lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render("  "+stripEmojis(strings.TrimPrefix(trimmed, "### "))))
		case strings.HasPrefix(trimmed, "#### "):
			out = append(out, lipgloss.NewStyle().Bold(true).Underline(true).Render("  "+stripEmojis(strings.TrimPrefix(trimmed, "#### "))))
		case strings.HasPrefix(trimmed, "* ") || strings.HasPrefix(trimmed, "- "):
			text := ""
			if len(trimmed) > 2 {
				text = trimmed[2:]
			}
			out = append(out, lipgloss.NewStyle().Foreground(ColorSecondary).Render("•")+" "+formatInlineMarkdown(text))
		default:
			out = append(out, formatInlineMarkdown(line))
		}
	}
	title := Tr("panel_readme")
	if n := len(m.readmeLines); n > 0 {
		title += fmt.Sprintf("  %d/%d", min(m.readmeScroll+1, n), n)
	}
	return ui.Panel(w, h, title, strings.Join(out, "\n"), true)
}

func (m Model) renderReadme(w, h int) string {
	if m.readmeFocus == 0 {
		return m.renderReadmeTOC(w, h)
	}
	return m.renderReadmeContent(w, h)
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
