package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/models"
	"github.com/aeon022/postctl/internal/store"
	"github.com/spf13/cobra"
)

var cancelCmd = &cobra.Command{
	Use:   "cancel [id]",
	Short: "Cancel a scheduled post",
	Long:  `Cancel a scheduled post, resetting its status back to draft and removing it from the queue.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		postID := args[0]
		ctx := context.Background()

		dbPath := config.GetDBPath()
		s, err := store.NewSQLiteStore(dbPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: open store: %v\n", err)
			os.Exit(1)
		}
		defer s.Close()

		post, err := s.GetPost(ctx, postID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: post with ID %q not found\n", postID)
			os.Exit(1)
		}

		if post.Status == models.StatusPosted {
			fmt.Fprintf(os.Stderr, "Error: post with ID %q is already published. Use 'postctl delete %s' to remove it.\n", postID, postID)
			os.Exit(1)
		}

		post.Status = models.StatusDraft
		post.ScheduledAt = nil
		post.Error = ""
		
		if err := s.SavePost(ctx, post); err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to cancel post: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Successfully cancelled scheduled post %q. Status reset to draft.\n", postID)
	},
}

func init() {
	rootCmd.AddCommand(cancelCmd)
}
