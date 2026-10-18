package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/models"
	"github.com/aeon022/postctl/internal/store"
	"github.com/spf13/cobra"
)

var markPostedAt, markPostedPlatformID string

var markPostedCmd = &cobra.Command{
	Use:   "mark-posted [id]",
	Short: "Mark a failed or scheduled post as already published (after checking the platform)",
	Long: `Sets a post to "posted" without publishing anything. Use it when a post is live on the platform but postctl
recorded a failure, e.g. the platform answered too late (timeout) although it published. Check the platform first:
marking a post that is not live means it will never be published by postctl.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		s, err := store.NewSQLiteStore(config.GetDBPath(), config.Shared())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: open store: %v\n", err)
			os.Exit(1)
		}
		defer s.Close()

		post, err := s.GetPost(ctx, args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: post with ID %q not found\n", args[0])
			os.Exit(1)
		}
		if post.Status == models.StatusPosted {
			fmt.Printf("Post %q is already marked as posted.\n", args[0])
			return
		}
		at := time.Now()
		if markPostedAt != "" {
			t, err := time.ParseInLocation("2006-01-02 15:04", markPostedAt, time.Local)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: --at must look like 2026-10-18 18:30: %v\n", err)
				os.Exit(1)
			}
			at = t
		}
		if DryRunFlag { // --dry-run must never change anything
			fmt.Printf("[DRY RUN] Would mark post %q (status %s) as posted at %s\n", args[0], post.Status, at.Format(time.RFC3339))
			return
		}
		post.Status = models.StatusPosted
		post.PostedAt = &at
		post.Error = ""
		if markPostedPlatformID != "" {
			post.PlatformID = markPostedPlatformID
		}
		if err := s.SavePost(ctx, post); err != nil {
			fmt.Fprintf(os.Stderr, "Error: could not save: %v\n", err)
			os.Exit(1)
		}
		_ = s.AddHistoryEntry(ctx, &models.HistoryEntry{PostID: post.ID, Action: "posted"})
		fmt.Printf("Post %q marked as posted (%s).\n", args[0], at.Format("02.01.2006 15:04"))
	},
}

func init() {
	markPostedCmd.Flags().StringVar(&markPostedAt, "at", "", "time it went live, e.g. \"2026-10-18 18:36\" (default: now)")
	markPostedCmd.Flags().StringVar(&markPostedPlatformID, "platform-id", "", "the platform's id of the live post, if known")
	rootCmd.AddCommand(markPostedCmd)
}
