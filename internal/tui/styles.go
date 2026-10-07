package tui

import (
	"charm.land/lipgloss/v2"
	"github.com/aeon022/missionctl-core/theme"
)

// Palette — theme-based, so the default `terminal` preset (and any other
// preset) applies; see missionctl-core/theme.
var (
	ColorPrimary   = theme.BlueV2   // header, borders
	ColorSecondary = theme.GreenV2  // active/selected
	ColorDarkGray  = theme.SubtleV2 // inactive borders
	ColorLightGray = theme.MutedV2  // metadata, help
	ColorBgFg      = theme.OnAccentV2

	// Status colors
	ColorDraft     = theme.SubtleV2
	ColorScheduled = theme.AmberV2
	ColorPosted    = theme.GreenV2
	ColorFailed    = theme.RedV2

	// Text on a status background (theme's "on accent" works for all of them).
	ColorOnScheduled = theme.OnAccentV2
	ColorOnPosted    = theme.OnAccentV2
	ColorOnFailed    = theme.OnAccentV2
)

// Styles
var (
	StyleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorBgFg).
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
				Foreground(ColorBgFg).
				Background(ColorDraft).
				Padding(0, 1).
				Bold(true)

	StyleStatusScheduled = lipgloss.NewStyle().
				Foreground(ColorOnScheduled).
				Background(ColorScheduled).
				Padding(0, 1).
				Bold(true)

	StyleStatusPosted = lipgloss.NewStyle().
				Foreground(ColorOnPosted).
				Background(ColorPosted).
				Padding(0, 1).
				Bold(true)

	StyleStatusFailed = lipgloss.NewStyle().
				Foreground(ColorOnFailed).
				Background(ColorFailed).
				Padding(0, 1).
				Bold(true)

	StyleHelp = lipgloss.NewStyle().
			Foreground(ColorLightGray).
			Italic(true)
)
