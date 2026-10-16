package tui

import (
	"charm.land/lipgloss/v2"
	"github.com/aeon022/missionctl-core/theme"
)

// Palette — theme-based, so the default `terminal` preset (and any other
// preset) applies; see missionctl-core/theme.
var (
	ColorPrimary   = theme.BlueV2  // header, borders
	ColorSecondary = theme.GreenV2 // active/selected
	ColorLightGray = theme.MutedV2 // metadata, help
	ColorBgFg      = theme.OnAccentV2

	// Status colors
	ColorPosted = theme.GreenV2
	ColorFailed = theme.RedV2
)

func init() {
	// live theme switching: rebuild everything that cached a color
	theme.OnChange(refreshStyles)
}

func refreshStyles() {
	ColorPrimary, ColorSecondary, ColorLightGray, ColorBgFg = theme.BlueV2, theme.GreenV2, theme.MutedV2, theme.OnAccentV2
	ColorPosted, ColorFailed = theme.GreenV2, theme.RedV2
	StyleTitle = lipgloss.NewStyle().Bold(true).Foreground(ColorBgFg).Background(ColorPrimary).Padding(0, 1)
	dimStyle = lipgloss.NewStyle().Foreground(theme.MutedV2)
}

// Styles
var StyleTitle = lipgloss.NewStyle().
	Bold(true).
	Foreground(ColorBgFg).
	Background(ColorPrimary).
	Padding(0, 1)
