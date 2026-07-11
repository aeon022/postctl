package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/markdown"
	"github.com/aeon022/postctl/internal/models"
	"github.com/aeon022/postctl/internal/store"
	"github.com/spf13/cobra"
)

var listFlag bool

// scheduleCmd repräsentiert den schedule-Befehl
var scheduleCmd = &cobra.Command{
	Use:   "schedule [id] [datetime]",
	Short: "Schedule a post for a specific time",
	Long:  `Schedule the post with the given ID for a specific date and time, or list all currently scheduled posts.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		dbPath := config.GetDBPath()
		s, err := store.NewSQLiteStore(dbPath)
		if err != nil {
			reportScheduleError(fmt.Errorf("open store: %w", err), 2)
			return
		}
		defer s.Close()

		// Wenn --list gesetzt ist, alle geplanten Posts anzeigen
		if listFlag {
			reportScheduledList(ctx, s)
			return
		}

		if len(args) < 2 {
			cmd.Help()
			return
		}

		postID := args[0]
		timeStr := args[1]

		// 1. Zeit parsen
		parsedTime, err := markdown.ParseScheduleTime(timeStr)
		if err != nil {
			reportScheduleError(fmt.Errorf("invalid datetime format %q: %w", timeStr, err), 1) // 1 = Validierungsfehler
			return
		}

		// 2. Post laden
		post, err := s.GetPost(ctx, postID)
		if err != nil {
			reportScheduleError(fmt.Errorf("post with ID %q not found: %w", postID, err), 1)
			return
		}

		// 3. Post als geplant markieren
		post.Status = models.StatusScheduled
		post.ScheduledAt = &parsedTime
		post.Error = ""
		
		if err := s.SavePost(ctx, post); err != nil {
			reportScheduleError(fmt.Errorf("save scheduled post: %w", err), 2)
			return
		}

		reportScheduleSuccess(post)
	},
}

type scheduledListJSON struct {
	OK    bool          `json:"ok"`
	Posts []models.Post `json:"posts"`
}

type scheduleSuccessJSON struct {
	OK          bool      `json:"ok"`
	PostID      string    `json:"post_id"`
	Status      string    `json:"status"`
	ScheduledAt time.Time `json:"scheduled_at"`
}

type scheduleErrorJSON struct {
	OK     bool     `json:"ok"`
	Code   int      `json:"code"`
	Errors []string `json:"errors"`
}

func reportScheduledList(ctx context.Context, s *store.SQLiteStore) {
	posts, err := s.ListPosts(ctx, "all", models.StatusScheduled, "")
	if err != nil {
		reportScheduleError(fmt.Errorf("list scheduled posts: %w", err), 2)
		return
	}

	if FormatFlag == "json" {
		out := scheduledListJSON{
			OK:    true,
			Posts: posts,
		}
		jsonBytes, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(jsonBytes))
	} else {
		fmt.Println("Scheduled Posts:")
		if len(posts) == 0 {
			fmt.Println(" - No posts currently scheduled.")
			return
		}
		for _, p := range posts {
			timeStr := ""
			if p.ScheduledAt != nil {
				timeStr = p.ScheduledAt.Format("02.01.2006 15:04")
			}
			fmt.Printf(" - [%s] %-10s %-2s (ID: %s)\n", timeStr, strings.ToUpper(p.Platform), strings.ToUpper(p.Language), p.ID)
		}
	}
}

func reportScheduleSuccess(post *models.Post) {
	scheduledTime := time.Time{}
	if post.ScheduledAt != nil {
		scheduledTime = *post.ScheduledAt
	}

	if FormatFlag == "json" {
		out := scheduleSuccessJSON{
			OK:          true,
			PostID:      post.ID,
			Status:      "scheduled",
			ScheduledAt: scheduledTime,
		}
		jsonBytes, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(jsonBytes))
	} else {
		fmt.Printf("Post %s successfully scheduled for %s.\n", post.ID, scheduledTime.Format("02.01.2006 15:04"))
	}
}

func reportScheduleError(err error, exitCode int) {
	if FormatFlag == "json" {
		out := scheduleErrorJSON{
			OK:     false,
			Code:   exitCode,
			Errors: []string{err.Error()},
		}
		jsonBytes, _ := json.MarshalIndent(out, "", "  ")
		fmt.Fprintln(os.Stderr, string(jsonBytes))
	} else {
		fmt.Fprintf(os.Stderr, "Schedule Error: %v\n", err)
	}
	os.Exit(exitCode)
}

func init() {
	scheduleCmd.Flags().BoolVar(&listFlag, "list", false, "List all scheduled posts")
	rootCmd.AddCommand(scheduleCmd)
}
