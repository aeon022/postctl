package cmd

import (
	"os"

	"github.com/aeon022/missionctl-core/doctor"
	"github.com/aeon022/postctl/internal/config"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check config and database health",
	Run: func(cmd *cobra.Command, args []string) {
		profile := config.ActiveProfile
		if profile == "" {
			profile = "default"
		}
		checks := []doctor.Check{
			{Label: "Profile", OK: true, Detail: profile},
			doctor.CheckSQLite("Database", config.GetDBPath(), "posts"),
			doctor.CheckDataDir("Data directory", config.GetDBPath(), config.Shared()),
		}
		if !doctor.PrintReport(checks) {
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
