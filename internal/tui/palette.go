package tui

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/aeon022/missionctl-core/palette"
	"github.com/aeon022/missionctl-core/ui"
)

// The ":" command palette, like in the other tools of the suite: type a command's name,
// get a live-filtered list, Enter runs it. A command either replays its single-key shortcut
// through the normal key handler or jumps to a tab ("goto:<tab>").

var paletteCommands = []palette.Command{
	{Name: "dashboard", Desc: "Dashboard", Key: "goto:0"},
	{Name: "posts", Desc: "Posts", Key: "goto:1"},
	{Name: "queue", Desc: "Queue", Key: "goto:2"},
	{Name: "calendar", Desc: "Calendar", Key: "goto:3"},
	{Name: "history", Desc: "History", Key: "goto:4"},
	{Name: "stats", Desc: "Stats / insights", Key: "goto:5"},
	{Name: "settings", Desc: "Settings", Key: "goto:6"},
	{Name: "logs", Desc: "Logs", Key: "goto:7"},
	{Name: "new", Desc: "New post", Key: "n"},
	{Name: "edit", Desc: "Edit post", Key: "e"},
	{Name: "post", Desc: "Publish now", Key: "p"},
	{Name: "schedule", Desc: "Schedule", Key: "s"},
	{Name: "delete", Desc: "Delete", Key: "d"},
	{Name: "repurpose", Desc: "Repurpose via AI", Key: "r"},
	{Name: "import", Desc: "Import posts", Key: "i"},
	{Name: "filter", Desc: "Filter campaign", Key: "f"},
	{Name: "export", Desc: "Export history", Key: "x"},
	{Name: "profile", Desc: "Switch profile", Key: "P"},
	{Name: "manual", Desc: "Open the manual", Key: "R"},
	{Name: "help", Desc: "Keyboard help", Key: "?"},
	{Name: "quit", Desc: "Quit", Key: "q"},
}

func (m Model) paletteMatches() []palette.Command {
	return palette.Match(paletteCommands, m.paletteInput.Value())
}

func (m Model) openPalette() Model {
	in := textinput.New()
	in.Placeholder = Tr("palette_placeholder")
	in.Focus()
	m.paletteInput = in
	m.paletteCursor = 0
	m.inPalette = true
	return m
}

// paletteKey handles keys while the palette is open.
func (m Model) paletteKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	matches := m.paletteMatches()
	switch msg.String() {
	case "esc", "ctrl+c":
		m.inPalette = false
		return m, nil
	case "up", "ctrl+p":
		m.paletteCursor = max(m.paletteCursor-1, 0)
		return m, nil
	case "down", "ctrl+n":
		m.paletteCursor = min(m.paletteCursor+1, max(len(matches)-1, 0))
		return m, nil
	case "enter":
		m.inPalette = false
		if len(matches) == 0 {
			return m, nil
		}
		return m.runPaletteCommand(matches[min(m.paletteCursor, len(matches)-1)])
	}
	var cmd tea.Cmd
	m.paletteInput, cmd = m.paletteInput.Update(msg)
	m.paletteCursor = 0
	return m, cmd
}

func (m Model) runPaletteCommand(c palette.Command) (Model, tea.Cmd) {
	if tab, ok := strings.CutPrefix(c.Key, "goto:"); ok {
		return m.switchTab(int(tab[0] - '0'))
	}
	r := []rune(c.Key)[0]
	nm, cmd := m.Update(tea.KeyPressMsg{Code: r, Text: c.Key})
	return nm.(Model), cmd
}

func (m Model) paletteBody(w, h int) string {
	pw := min(w, 56)
	matches := m.paletteMatches()
	capacity := max(min(h-5, 10), 3)
	start, end := window(len(matches), m.paletteCursor, capacity)
	lines := []string{": " + m.paletteInput.View(), ""}
	for i := start; i < end; i++ {
		c := matches[i]
		lines = append(lines, ui.Row(pw-4, i == m.paletteCursor, padRight(c.Name, 12)+dimStyle.Render(c.Desc)))
	}
	if len(matches) == 0 {
		lines = append(lines, dimStyle.Render(Tr("palette_none")))
	}
	return ui.Panel(pw, min(len(lines)+2, max(h, 5)), Tr("palette_title"), strings.Join(lines, "\n"), true)
}
