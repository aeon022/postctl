package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/models"
	"github.com/aeon022/postctl/internal/platforms"
	"github.com/aeon022/postctl/internal/store"
	"github.com/spf13/cobra"
)

var statusFlag bool

// authCmd repräsentiert den auth-Befehl
var authCmd = &cobra.Command{
	Use:   "auth [platform]",
	Short: "Authenticate with social media platforms",
	Long:  `Authenticate with Twitter/X, LinkedIn, Threads, or Mastodon using OAuth 2.0. If no platform is specified, use --status to check authentication status.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		dbPath := config.GetDBPath()
		s, err := store.NewSQLiteStore(dbPath)
		if err != nil {
			reportAuthError(fmt.Errorf("open database: %w", err), 3)
			return
		}
		defer s.Close()

		// Wenn --status gesetzt ist, Verbindungsstatus prüfen
		if statusFlag {
			reportAuthStatus(ctx, s)
			return
		}

		if len(args) == 0 {
			cmd.Help()
			return
		}

		platformName := args[0]
		
		// Platform-Instanz holen (dry-run ist bei Auth nicht aktiv)
		plat, err := platforms.GetPlatform(platformName, s, false)
		if err != nil {
			reportAuthError(err, 3)
			return
		}

		// Auth-Prozess starten
		if err := plat.Auth(ctx); err != nil {
			reportAuthError(fmt.Errorf("authentication failed: %w", err), 3)
			return
		}

		reportAuthSuccess(platformName)
	},
}

type authStatusJSON struct {
	OK        bool            `json:"ok"`
	Platforms map[string]bool `json:"platforms"`
}

type authSuccessJSON struct {
	OK       bool   `json:"ok"`
	Platform string `json:"platform"`
	Status   string `json:"status"`
}

type authErrorJSON struct {
	OK     bool     `json:"ok"`
	Code   int      `json:"code"`
	Errors []string `json:"errors"`
}

func reportAuthStatus(ctx context.Context, s store.Store) {
	plats := []string{models.PlatformTwitter, models.PlatformLinkedIn, models.PlatformThreads, models.PlatformMastodon, models.PlatformBluesky, models.PlatformFacebook}
	statusMap := make(map[string]bool)

	for _, p := range plats {
		plat, err := platforms.GetPlatform(p, s, false)
		if err == nil {
			statusMap[p] = plat.IsAuthenticated(ctx)
		} else {
			statusMap[p] = false
		}
	}

	if FormatFlag == "json" {
		out := authStatusJSON{
			OK:        true,
			Platforms: statusMap,
		}
		jsonBytes, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(jsonBytes))
	} else {
		fmt.Println("Platform Authentication Status:")
		for p, connected := range statusMap {
			statusText := "○ not connected"
			if connected {
				statusText = "✓ connected"
			}
			name := p
			if p == models.PlatformTwitter {
				name = "Twitter/X"
			} else if p == models.PlatformLinkedIn {
				name = "LinkedIn"
			} else if p == models.PlatformThreads {
				name = "Threads"
			} else if p == models.PlatformMastodon {
				name = "Mastodon"
			} else if p == models.PlatformBluesky {
				name = "Bluesky"
			} else if p == models.PlatformFacebook {
				name = "Facebook"
			}
			fmt.Printf(" - %-12s %s\n", name+":", statusText)
		}
	}
}

func reportAuthSuccess(platform string) {
	if FormatFlag == "json" {
		out := authSuccessJSON{
			OK:       true,
			Platform: platform,
			Status:   "authenticated",
		}
		jsonBytes, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(jsonBytes))
	} else {
		fmt.Printf("Successfully authenticated with %s!\n", platform)
	}
}

func reportAuthError(err error, exitCode int) {
	if FormatFlag == "json" {
		out := authErrorJSON{
			OK:     false,
			Code:   exitCode,
			Errors: []string{err.Error()},
		}
		jsonBytes, _ := json.MarshalIndent(out, "", "  ")
		fmt.Fprintln(os.Stderr, string(jsonBytes))
	} else {
		fmt.Fprintf(os.Stderr, "Auth Error: %v\n", err)
	}
	os.Exit(exitCode)
}

func init() {
	authCmd.Flags().BoolVar(&statusFlag, "status", false, "Check authentication status of all platforms")
	rootCmd.AddCommand(authCmd)
}
