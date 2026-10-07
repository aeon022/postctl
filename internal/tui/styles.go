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

// Styles
var StyleTitle = lipgloss.NewStyle().
	Bold(true).
	Foreground(ColorBgFg).
	Background(ColorPrimary).
	Padding(0, 1)
