package cmd

import (
	"fmt"
	"os"

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
	// If the user didn't explicitly pick a profile (flag or env), and at
	// least one named profile already exists, offer a quick picker rather
	// than silently defaulting — otherwise reaching anything but the
	// default profile from the interactive TUI would require remembering
	// --profile every time. Non-interactive commands (list, post, ...)
	// deliberately skip this: it would break scripting.
	if ProfileFlag == "" && os.Getenv("POSTCTL_PROFILE") == "" {
		named, err := config.ListProfiles()
		if err == nil && len(named) > 0 {
			profiles := append([]string{""}, named...)
			chosen, ok, err := tui.RunProfilePicker(profiles)
			if err != nil {
				return fmt.Errorf("profile picker: %w", err)
			}
			if !ok {
				return nil
			}
			if err := config.LoadConfigProfile(chosen); err != nil {
				return fmt.Errorf("load profile %q: %w", chosen, err)
			}
		}
	}

	dbPath := config.GetDBPath()
	s, err := store.NewSQLiteStore(dbPath, config.Shared())
	if err != nil {
		return fmt.Errorf("open sqlite store for TUI: %w", err)
	}
	defer s.Close()

	model := tui.NewModel(s)
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("run bubbletea program: %w", err)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}
