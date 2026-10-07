package tui

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/aeon022/missionctl-core/statusbar"
	"github.com/aeon022/missionctl-core/ui"
)

// profilePickerModel is a small standalone Bubble Tea program shown before
// the main app when postctl is launched interactively with no --profile
// flag/POSTCTL_PROFILE set and at least one named profile already exists
// — otherwise there'd be no way to reach anything but the default profile
// short of remembering the flag every time.
type profilePickerModel struct {
	profiles      []string // "" = default, else the profile name
	cursor        int
	chosen        string
	selected      bool
	width, height int
}

func (m profilePickerModel) Init() tea.Cmd { return nil }

func (m profilePickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.width, m.height = ws.Width, ws.Height
		return m, nil
	}
	keyMsg, ok := msg.(tea.KeyPressMsg)
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
		return Tr("picker_default")
	}
	return p
}

func (m profilePickerModel) View() tea.View {
	v := tea.NewView(m.viewContent())
	v.AltScreen = true
	return v
}

// viewContent uses the same chrome as the main app: header, divider, a panel
// with selectable rows and a one-line footer, exactly the terminal height.
func (m profilePickerModel) viewContent() string {
	w := 100
	if m.width > 0 {
		w = max(m.width-2*indent, 20)
	}
	pad := func(s string) string {
		return strings.Repeat(" ", indent) + strings.ReplaceAll(s, "\n", "\n"+strings.Repeat(" ", indent))
	}

	pw := min(w, 50)
	rows := make([]string, len(m.profiles))
	for i, p := range m.profiles {
		rows[i] = ui.Row(panelRowW(pw), i == m.cursor, profileLabel(p))
	}
	ph := min(len(rows)+2, max(m.height-5, 3))
	start, end := window(len(rows), m.cursor, ph-2)

	header := pad(ui.Header(w, "postctl · Social Media", "", time.Now().Format("Mon 02 Jan")) + "\n" + ui.Divider(w, "") + "\n")
	body := pad(ui.Panel(pw, ph, Tr("picker_title"), strings.Join(rows[start:end], "\n"), true))
	footer := pad(statusbar.Line(w, statusbar.Hints(w, hint("enter", "hint_open"), hint("esc", "hint_quit"), hint("↑↓", "hint_move")), ""))
	return ui.Frame(max(m.height-1, 0), header, body, footer)
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
