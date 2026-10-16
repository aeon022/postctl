package cmd

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/store"
	"github.com/aeon022/postctl/internal/tui"
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
	pick := ProfileFlag == "" && os.Getenv("POSTCTL_PROFILE") == ""
	for {
		if pick {
			named, err := config.ListProfiles()
			if err == nil && len(named) > 0 {
				chosen, ok, err := tui.RunProfilePicker(append([]string{""}, named...))
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
		again, err := runTUIOnce()
		if err != nil || !again {
			return err
		}
		// P pressed: show the picker again, even if --profile was given.
		pick = true
	}
}

// runTUIOnce runs one TUI session on the active profile's store and reports
// whether the user asked to switch profile.
func runTUIOnce() (switchProfile bool, err error) {
	s, err := store.NewSQLiteStore(config.GetDBPath(), config.Shared())
	if err != nil {
		return false, fmt.Errorf("open sqlite store for TUI: %w", err)
	}
	defer s.Close()

	final, err := tea.NewProgram(tui.NewModel(s)).Run()
	if err != nil {
		return false, fmt.Errorf("run bubbletea program: %w", err)
	}
	m, ok := final.(tui.Model)
	return ok && m.WantsProfileSwitch(), nil
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}
