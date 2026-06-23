package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap definiert alle verfügbaren Tastaturbelegungen
type KeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	Tab      key.Binding
	ShiftTab key.Binding
	Enter    key.Binding
	Esc      key.Binding
	Post     key.Binding
	Schedule key.Binding
	Edit      key.Binding
	NewPost   key.Binding
	Delete    key.Binding
	Repurpose key.Binding
	Import    key.Binding
	Readme    key.Binding
	Quit      key.Binding
	Help      key.Binding
}

// Keys ist die globale KeyMap Instanz
var Keys = KeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Left: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←/h", "left"),
	),
	Right: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→/l", "right"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next tab"),
	),
	ShiftTab: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("shift+tab", "prev tab"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select / preview"),
	),
	Esc: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	Post: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "post now"),
	),
	Schedule: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "schedule"),
	),
	Edit: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "edit"),
	),
	NewPost: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "new post"),
	),
	Delete: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "delete"),
	),
	Repurpose: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "repurpose via AI"),
	),
	Import: key.NewBinding(
		key.WithKeys("i"),
		key.WithHelp("i", "import posts"),
	),
	Readme: key.NewBinding(
		key.WithKeys("f1", "R"),
		key.WithHelp("f1/R", "open readme"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle quick help"),
	),
}
