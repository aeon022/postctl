package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/models"
	"github.com/aeon022/postctl/internal/scheduler"
	"github.com/aeon022/postctl/internal/store"
	"github.com/spf13/cobra"
)

// postCmd repräsentiert den post-Befehl
var postCmd = &cobra.Command{
	Use:   "post <id>",
	Short: "Publish a post immediately",
	Long:  `Publish the post with the given ID immediately to its configured platform.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		postID := args[0]
		ctx := context.Background()

		dbPath := config.GetDBPath()
		s, err := store.NewSQLiteStore(dbPath)
		if err != nil {
			reportPostError(fmt.Errorf("open store: %w", err), 2)
			return
		}
		defer s.Close()

		// 1. Post laden
		post, err := s.GetPost(ctx, postID)
		if err != nil {
			reportPostError(fmt.Errorf("post with ID %q not found: %w", postID, err), 1) // 1 = Validierung / Not Found
			return
		}

		// 2. Veröffentlichen via zentraler Scheduler-Publish-Logik
		platformID, err := scheduler.PublishPost(ctx, s, post, DryRunFlag)
		if err != nil {
			reportPostError(err, 2)
			return
		}

		// 3. Erfolg melden
		reportPostSuccess(post, platformID)
	},
}

type postSuccessJSON struct {
	OK           bool     `json:"ok"`
	Platform     string   `json:"platform"`
	TweetsPosted int      `json:"tweets_posted,omitempty"`
	ThreadID     string   `json:"thread_id,omitempty"`
	PostID       string   `json:"post_id,omitempty"`
	URLs         []string `json:"urls"`
}

type postErrorJSON struct {
	OK     bool     `json:"ok"`
	Code   int      `json:"code"`
	Errors []string `json:"errors"`
}

func reportPostSuccess(post *models.Post, platformID string) {
	urls := []string{}
	switch post.Platform {
	case models.PlatformTwitter:
		urls = append(urls, fmt.Sprintf("https://x.com/i/status/%s", platformID))
	case models.PlatformLinkedIn:
		urls = append(urls, fmt.Sprintf("https://www.linkedin.com/feed/update/%s", platformID))
	case models.PlatformThreads:
		urls = append(urls, fmt.Sprintf("https://www.threads.net/post/%s", platformID))
	}

	if FormatFlag == "json" {
		out := postSuccessJSON{
			OK:       true,
			Platform: post.Platform,
			URLs:     urls,
		}
		if post.Platform == models.PlatformTwitter {
			out.TweetsPosted = len(post.Tweets)
			out.ThreadID = platformID
		} else {
			out.PostID = platformID
		}

		jsonBytes, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(jsonBytes))
	} else {
		prefix := ""
		if DryRunFlag {
			prefix = "[DRY RUN] "
		}
		fmt.Printf("%sSuccessfully posted to %s!\n", prefix, post.Platform)
		if len(urls) > 0 {
			fmt.Printf("URL: %s\n", urls[0])
		}
	}
}

func reportPostError(err error, exitCode int) {
	if FormatFlag == "json" {
		out := postErrorJSON{
			OK:     false,
			Code:   exitCode,
			Errors: []string{err.Error()},
		}
		jsonBytes, _ := json.MarshalIndent(out, "", "  ")
		fmt.Fprintln(os.Stderr, string(jsonBytes))
	} else {
		fmt.Fprintf(os.Stderr, "Post Error: %v\n", err)
	}
	os.Exit(exitCode)
}

func init() {
	rootCmd.AddCommand(postCmd)
}
