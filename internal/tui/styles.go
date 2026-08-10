package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Palette — aligned with missionctl design system
var (
	ColorPrimary   = lipgloss.AdaptiveColor{Light: "25", Dark: "33"}   // blue (header, borders)
	ColorSecondary = lipgloss.AdaptiveColor{Light: "30", Dark: "43"}   // teal (active/selected)
	ColorDarkGray  = lipgloss.AdaptiveColor{Light: "250", Dark: "244"} // subtle (inactive borders)
	ColorLightGray = lipgloss.AdaptiveColor{Light: "243", Dark: "246"} // muted (metadata, help)
	ColorBgFg      = lipgloss.AdaptiveColor{Light: "232", Dark: "255"} // badge foreground (dark/light swap)

	// Status colors
	ColorDraft     = lipgloss.AdaptiveColor{Light: "250", Dark: "239"} // subtle gray
	ColorScheduled = lipgloss.AdaptiveColor{Light: "214", Dark: "220"} // amber
	ColorPosted    = lipgloss.AdaptiveColor{Light: "28", Dark: "42"}   // green
	ColorFailed    = lipgloss.AdaptiveColor{Light: "160", Dark: "203"} // red

	// Per-status badge foregrounds. ColorBgFg (black in light mode, white in
	// dark mode) only works for Draft, whose background follows the same
	// light/dark split. Scheduled's amber is bright in *both* modes (214
	// and 220 are both high-luminance yellow/orange — dark 220 in
	// particular is close to pure yellow), so white text on it in dark mode
	// was nearly invisible. Posted's and Failed's Light/Dark background
	// values are inverted relative to Draft's (the light-mode green/red are
	// the darker of their pair), so they need ColorBgFg's swap flipped, not
	// dropped.
	ColorOnScheduled = lipgloss.AdaptiveColor{Light: "232", Dark: "232"} // always dark text — amber is light in both modes
	ColorOnPosted    = lipgloss.AdaptiveColor{Light: "255", Dark: "232"} // inverted: light-mode green is dark, dark-mode green is bright
	ColorOnFailed    = lipgloss.AdaptiveColor{Light: "255", Dark: "255"} // always light text — both reds are mid-dark
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
