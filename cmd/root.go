package cmd

import (
	"fmt"
	"os"

	"github.com/aeon022/postctl/internal/config"
	"github.com/spf13/cobra"
)

// Version — wird beim Build gesetzt
var Version = "dev"

var FormatFlag string
var DryRunFlag bool

var rootCmd = &cobra.Command{
	Use:   "postctl",
	Short: "Social media posting from the terminal",
	Long:  "postctl manages social media posts from Markdown files.\nTwitter/X, LinkedIn, and Threads — from one CLI.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return config.LoadConfig()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTUI()
	},
}

// Execute — wird von main() aufgerufen
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Globale Flags registrieren
	rootCmd.PersistentFlags().StringVar(&FormatFlag, "format", "human", "Output format (human|json)")
	rootCmd.PersistentFlags().BoolVar(&DryRunFlag, "dry-run", false, "Simulate execution without side effects")

	// Version subcommand
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("postctl %s\n", Version)
		},
	})
}
