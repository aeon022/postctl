package tui

import "github.com/charmbracelet/lipgloss"

// Farben
var (
	ColorPrimary   = lipgloss.Color("#8b7cf8") // Lavender / Purple
	ColorSecondary = lipgloss.Color("#00f5d4") // Cyan
	ColorDarkGray  = lipgloss.Color("#2d3748") // Dark Slate
	ColorLightGray = lipgloss.Color("#718096") // Gray
	ColorBg        = lipgloss.Color("#1a202c") // Very Dark Slate
	ColorText      = lipgloss.Color("#f7fafc") // White/Off-white

	ColorDraft     = lipgloss.Color("#a0aec0") // Gray
	ColorScheduled = lipgloss.Color("#ecc94b") // Yellow
	ColorPosted    = lipgloss.Color("#48bb78") // Green
	ColorFailed    = lipgloss.Color("#f56565") // Red
)

// Lipgloss Stile
var (
	StyleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorBg).
			Background(ColorPrimary).
			Padding(0, 1)

	StyleTabActive = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSecondary).
			Border(lipgloss.Border{
			Top:         " ",
			Bottom:      "─",
			Left:        "│",
			Right:       "│",
			TopLeft:     "┌",
			TopRight:    "┐",
			BottomLeft:  "┴",
			BottomRight: "┴",
		}, true, false, true, true).
		BorderForeground(ColorPrimary).
		Padding(0, 1)

	StyleTabInactive = lipgloss.NewStyle().
				Foreground(ColorLightGray).
				Border(lipgloss.Border{
			Top:         " ",
			Bottom:      "─",
			Left:        "│",
			Right:       "│",
			TopLeft:     "┌",
			TopRight:    "┐",
			BottomLeft:  "┼",
			BottomRight: "┼",
		}, true, false, true, true).
		BorderForeground(ColorDarkGray).
		Padding(0, 1)

	StyleBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorDarkGray).
			Padding(1, 2)

	StyleHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary)

	StyleStatusDraft = lipgloss.NewStyle().
				Foreground(ColorBg).
				Background(ColorDraft).
				Padding(0, 1).
				Bold(true)

	StyleStatusScheduled = lipgloss.NewStyle().
				Foreground(ColorBg).
				Background(ColorScheduled).
				Padding(0, 1).
				Bold(true)

	StyleStatusPosted = lipgloss.NewStyle().
				Foreground(ColorBg).
				Background(ColorPosted).
				Padding(0, 1).
				Bold(true)

	StyleStatusFailed = lipgloss.NewStyle().
				Foreground(ColorText).
				Background(ColorFailed).
				Padding(0, 1).
				Bold(true)

	StyleHelp = lipgloss.NewStyle().
			Foreground(ColorLightGray).
			Italic(true)
)
