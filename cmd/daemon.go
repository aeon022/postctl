package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/scheduler"
	"github.com/aeon022/postctl/internal/service"
	"github.com/aeon022/postctl/internal/store"
	"github.com/spf13/cobra"
)

var checkIntervalSec int

// daemonCmd repräsentiert den daemon-Befehl
var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Start the background scheduler daemon",
	Long:  `Run postctl as a background daemon that periodically checks the database and publishes due scheduled posts.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Signal-Handling für Graceful Shutdown (Ctrl+C / SIGTERM)
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-sigChan
			fmt.Fprintln(os.Stderr, "\nSignal empfangen. Shutdown eingeleitet...")
			cancel()
		}()

		dbPath := config.GetDBPath()
		s, err := store.NewSQLiteStore(dbPath, config.Shared())
		if err != nil {
			return fmt.Errorf("open sqlite store for daemon: %w", err)
		}
		defer s.Close()

		interval := time.Duration(checkIntervalSec) * time.Second

		// Daemon starten
		return scheduler.RunDaemon(ctx, s, interval, DryRunFlag)
	},
}

func init() {
	daemonCmd.Flags().IntVar(&checkIntervalSec, "interval", 30, "Database check interval in seconds")
	rootCmd.AddCommand(daemonCmd)
}

var daemonInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Run the scheduler at login for the active profile (macOS launchd)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := service.Install(config.ActiveProfile); err != nil {
			return err
		}
		fmt.Printf("✓ Scheduler installed (%s). Log: %s\n", service.Label(config.ActiveProfile), service.LogPath(config.ActiveProfile))
		return nil
	},
}

var daemonUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Stop and remove the scheduler launch agent for the active profile",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := service.Uninstall(config.ActiveProfile); err != nil {
			return err
		}
		fmt.Println("✓ Scheduler removed.")
		return nil
	},
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show whether the scheduler launch agent runs for the active profile",
	Run: func(cmd *cobra.Command, args []string) {
		if service.Running(config.ActiveProfile) {
			fmt.Printf("● running (%s)\n", service.Label(config.ActiveProfile))
		} else {
			fmt.Println("○ not running")
		}
	},
}

func init() {
	daemonCmd.AddCommand(daemonInstallCmd, daemonUninstallCmd, daemonStatusCmd)
}
