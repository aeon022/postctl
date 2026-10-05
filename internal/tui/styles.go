package tui

import (
	"image/color"
	"os"

	"charm.land/lipgloss/v2"
)

// adaptive resolves a light/dark ANSI color pair once, at startup — v2 dropped
// AdaptiveColor, and these package-level styles are built once, not per render.
var adaptive = func() func(light, dark string) color.Color {
	pick := lipgloss.LightDark(lipgloss.HasDarkBackground(os.Stdin, os.Stdout))
	return func(light, dark string) color.Color { return pick(lipgloss.Color(light), lipgloss.Color(dark)) }
}()

// Palette — aligned with missionctl design system
var (
	ColorPrimary   = adaptive("25", "33")   // blue (header, borders)
	ColorSecondary = adaptive("30", "43")   // teal (active/selected)
	ColorDarkGray  = adaptive("250", "244") // subtle (inactive borders)
	ColorLightGray = adaptive("243", "246") // muted (metadata, help)
	ColorBgFg      = adaptive("232", "255") // badge foreground (dark/light swap)

	// Status colors
	ColorDraft     = adaptive("250", "239") // subtle gray
	ColorScheduled = adaptive("214", "220") // amber
	ColorPosted    = adaptive("28", "42")   // green
	ColorFailed    = adaptive("160", "203") // red

	// Per-status badge foregrounds. ColorBgFg (black in light mode, white in
	// dark mode) only works for Draft, whose background follows the same
	// light/dark split. Scheduled's amber is bright in *both* modes (214
	// and 220 are both high-luminance yellow/orange — dark 220 in
	// particular is close to pure yellow), so white text on it in dark mode
	// was nearly invisible. Posted's and Failed's Light/Dark background
	// values are inverted relative to Draft's (the light-mode green/red are
	// the darker of their pair), so they need ColorBgFg's swap flipped, not
	// dropped.
	ColorOnScheduled = adaptive("232", "232") // always dark text — amber is light in both modes
	ColorOnPosted    = adaptive("255", "232") // inverted: light-mode green is dark, dark-mode green is bright
	ColorOnFailed    = adaptive("255", "255") // always light text — both reds are mid-dark
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
