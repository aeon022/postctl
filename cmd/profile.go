package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/aeon022/postctl/internal/config"
	"github.com/spf13/cobra"
)

// profileCmd shows the currently active profile when run with no
// subcommand — the "current profile" question a user new to --profile
// asks most often.
var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Show or list postctl profiles (separate config, credentials, and data per profile)",
	Run: func(cmd *cobra.Command, args []string) {
		active := config.ActiveProfile
		if active == "" {
			active = "default"
		}
		if FormatFlag == "json" {
			b, _ := json.MarshalIndent(map[string]string{"active": active}, "", "  ")
			fmt.Println(string(b))
			return
		}
		fmt.Printf("Active profile: %s\n", active)
	},
}

var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all profiles that have been used at least once",
	Run: func(cmd *cobra.Command, args []string) {
		profiles, err := config.ListProfiles()
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		if FormatFlag == "json" {
			out := struct {
				Active   string   `json:"active"`
				Default  bool     `json:"default_in_use"`
				Profiles []string `json:"profiles"`
			}{Active: config.ActiveProfile, Profiles: profiles}
			b, _ := json.MarshalIndent(out, "", "  ")
			fmt.Println(string(b))
			return
		}

		mark := func(name string) string {
			if name == config.ActiveProfile {
				return " (active)"
			}
			return ""
		}
		fmt.Printf("default%s\n", mark(""))
		for _, p := range profiles {
			fmt.Printf("%s%s\n", p, mark(p))
		}
		if len(profiles) == 0 {
			fmt.Println("\nNo named profiles yet — create one by running any command with --profile <name>.")
		}
	},
}

func init() {
	profileCmd.AddCommand(profileListCmd)
	rootCmd.AddCommand(profileCmd)
}
