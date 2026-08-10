package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/models"
	"github.com/aeon022/postctl/internal/platforms"
	"github.com/aeon022/postctl/internal/store"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete [id]",
	Short: "Delete a post locally and remotely",
	Long:  `Delete a post from the local database. If the post has already been published, it will also be deleted from the remote platform.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		postID := args[0]
		ctx := context.Background()

		dbPath := config.GetDBPath()
		s, err := store.NewSQLiteStore(dbPath, config.Shared())
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

		// Real platform delete must succeed before the local record goes
		// away — used to only print a warning on failure and delete the
		// local row regardless, which left the post live and untracked on
		// the remote platform while postctl believed it gone. Same bug
		// class fixed in calctl's DeleteEvent on 2026-08-01.
		if post.Status == models.StatusPosted && post.PlatformID != "" {
			fmt.Printf("Deleting post %q from remote platform %s...\n", postID, post.Platform)
			plat, err := platforms.GetPlatform(post.Platform, s, false)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: failed to get platform instance: %v\n", err)
				fmt.Fprintln(os.Stderr, "Local record kept — rerun once the platform is reachable.")
				os.Exit(1)
			}
			if err := plat.Delete(ctx, post.PlatformID); err != nil {
				fmt.Fprintf(os.Stderr, "Error: failed to delete post from remote platform: %v\n", err)
				fmt.Fprintln(os.Stderr, "Local record kept — the post is still live remotely.")
				os.Exit(1)
			}
			fmt.Println("Successfully deleted post from remote platform.")
		}

		if err := s.DeletePost(ctx, postID); err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to delete post locally: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Successfully deleted post %q from local database.\n", postID)
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
