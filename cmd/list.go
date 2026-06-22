package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/models"
	"github.com/aeon022/postctl/internal/store"
	"github.com/spf13/cobra"
)

var (
	listPlatform string
	listStatus   string
	listCampaign string
)

// listCmd repräsentiert den list-Befehl
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all posts in the database",
	Long:  `List all posts in the database, optionally filtered by platform, status, and campaign.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		dbPath := config.GetDBPath()
		s, err := store.NewSQLiteStore(dbPath)
		if err != nil {
			reportListError(fmt.Errorf("open store: %w", err), 2)
			return
		}
		defer s.Close()

		// Posts aus DB abfragen mit Filtern
		posts, err := s.ListPosts(ctx, listPlatform, listStatus, listCampaign)
		if err != nil {
			reportListError(fmt.Errorf("list posts: %w", err), 2)
			return
		}

		reportPostsList(posts)
	},
}

type postsListJSON struct {
	Posts []models.Post `json:"posts"`
	Total int           `json:"total"`
}

type listErrorJSON struct {
	OK     bool     `json:"ok"`
	Code   int      `json:"code"`
	Errors []string `json:"errors"`
}

func reportPostsList(posts []models.Post) {
	if FormatFlag == "json" {
		out := postsListJSON{
			Posts: posts,
			Total: len(posts),
		}
		jsonBytes, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(jsonBytes))
	} else {
		fmt.Println("Posts list:")
		if len(posts) == 0 {
			fmt.Println(" - No posts found.")
			return
		}
		for _, p := range posts {
			statusText := p.Status
			if p.Status == models.StatusScheduled && p.ScheduledAt != nil {
				statusText = fmt.Sprintf("scheduled for %s", p.ScheduledAt.Format("02.01. 15:04"))
			} else if p.Status == models.StatusPosted && p.PostedAt != nil {
				statusText = fmt.Sprintf("posted at %s", p.PostedAt.Format("02.01. 15:04"))
			}

			titlePreview := p.Title
			if len(titlePreview) > 40 {
				titlePreview = titlePreview[:37] + "..."
			}

			fmt.Printf(" - (ID: %-25s) [%s] %-8s %-2s : %q\n", 
				p.ID, 
				statusText, 
				strings.ToUpper(p.Platform), 
				strings.ToUpper(p.Language), 
				titlePreview,
			)
		}
	}
}

func reportListError(err error, exitCode int) {
	if FormatFlag == "json" {
		out := listErrorJSON{
			OK:     false,
			Code:   exitCode,
			Errors: []string{err.Error()},
		}
		jsonBytes, _ := json.MarshalIndent(out, "", "  ")
		fmt.Fprintln(os.Stderr, string(jsonBytes))
	} else {
		fmt.Fprintf(os.Stderr, "List Error: %v\n", err)
	}
	os.Exit(exitCode)
}

func init() {
	listCmd.Flags().StringVar(&listPlatform, "platform", "all", "Filter by platform (all|twitter|linkedin|threads)")
	listCmd.Flags().StringVar(&listStatus, "status", "all", "Filter by status (all|draft|scheduled|posted|failed)")
	listCmd.Flags().StringVar(&listCampaign, "campaign", "", "Filter by campaign slug")
	rootCmd.AddCommand(listCmd)
}
