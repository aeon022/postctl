package cmd

import (
	"fmt"

	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/store"
	"github.com/aeon022/postctl/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

// tuiCmd repräsentiert den TUI-Befehl
var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Start the Terminal User Interface",
	Long:  `Launch the interactive terminal dashboard to manage posts, schedules, and view posting history.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTUI()
	},
}

// runTUI initialisiert den Store, lädt das Bubbletea-Programm und startet die TUI
func runTUI() error {
	dbPath := config.GetDBPath()
	s, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		return fmt.Errorf("open sqlite store for TUI: %w", err)
	}
	defer s.Close()

	model := tui.NewModel(s)
	p := tea.NewProgram(model, tea.WithAltScreen())
	
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("run bubbletea program: %w", err)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}
