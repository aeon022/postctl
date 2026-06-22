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
		s, err := store.NewSQLiteStore(dbPath)
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
