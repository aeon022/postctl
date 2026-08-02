package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var styleProfilePickerRow = lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary)

// profilePickerModel is a small standalone Bubble Tea program shown before
// the main app when postctl is launched interactively with no --profile
// flag/POSTCTL_PROFILE set and at least one named profile already exists
// — otherwise there'd be no way to reach anything but the default profile
// short of remembering the flag every time.
type profilePickerModel struct {
	profiles []string // "" = default, else the profile name
	cursor   int
	chosen   string
	selected bool
}

func (m profilePickerModel) Init() tea.Cmd { return nil }

func (m profilePickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch keyMsg.String() {
	case "ctrl+c", "q", "esc":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.profiles)-1 {
			m.cursor++
		}
	case "enter":
		m.chosen = m.profiles[m.cursor]
		m.selected = true
		return m, tea.Quit
	}
	return m, nil
}

func profileLabel(p string) string {
	if p == "" {
		return "default"
	}
	return p
}

func (m profilePickerModel) View() string {
	var b strings.Builder
	b.WriteString(StyleTitle.Render("postctl") + "\n\n")
	b.WriteString("Choose a profile:\n\n")
	for i, p := range m.profiles {
		label := profileLabel(p)
		if i == m.cursor {
			b.WriteString("  > " + styleProfilePickerRow.Render(label) + "\n")
		} else {
			b.WriteString("    " + label + "\n")
		}
	}
	b.WriteString("\n↑/↓ navigate  ·  enter: select  ·  esc: quit\n")
	return b.String()
}

// RunProfilePicker shows the picker over profiles (already including ""
// for the default, first) and returns the chosen one, or ok=false if the
// user quit without choosing.
func RunProfilePicker(profiles []string) (chosen string, ok bool, err error) {
	result, err := tea.NewProgram(profilePickerModel{profiles: profiles}).Run()
	if err != nil {
		return "", false, err
	}
	final := result.(profilePickerModel)
	if !final.selected {
		return "", false, nil
	}
	return final.chosen, true, nil
}
