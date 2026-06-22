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

// campaignCmd repräsentiert das Hauptkommando campaign
var campaignCmd = &cobra.Command{
	Use:   "campaign",
	Short: "Manage and publish post campaigns",
	Long:  `Group, list, and publish entire campaigns of posts at once.`,
}

// campaignListCmd repräsentiert das Unterkommando campaign list
var campaignListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all campaigns and their status",
	Long:  `Scan the database and aggregate statistics for each campaign.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		dbPath := config.GetDBPath()
		s, err := store.NewSQLiteStore(dbPath)
		if err != nil {
			reportCampaignError(fmt.Errorf("open database: %w", err), 2)
			return
		}
		defer s.Close()

		// Alle Posts holen, um Kampagnen zu gruppieren
		posts, err := s.ListPosts(ctx, "all", "all", "")
		if err != nil {
			reportCampaignError(fmt.Errorf("fetch posts: %w", err), 2)
			return
		}

		campaignMap := make(map[string]*models.Campaign)
		for _, p := range posts {
			if p.Campaign == "" {
				continue
			}
			c, ok := campaignMap[p.Campaign]
			if !ok {
				c = &models.Campaign{Slug: p.Campaign}
				campaignMap[p.Campaign] = c
			}
			c.Posts = append(c.Posts, p)
			switch p.Status {
			case models.StatusPosted:
				c.Posted++
			case models.StatusScheduled:
				c.Scheduled++
			case models.StatusDraft:
				c.Drafts++
			}
		}

		var campaigns []models.Campaign
		for _, c := range campaignMap {
			campaigns = append(campaigns, *c)
		}

		reportCampaignsList(campaigns)
	},
}

// campaignPostCmd repräsentiert das Unterkommando campaign post
var campaignPostCmd = &cobra.Command{
	Use:   "post <campaign_slug>",
	Short: "Publish all unpublished posts in a campaign",
	Long:  `Publish all posts belonging to the specified campaign that are not yet successfully posted.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		campaignSlug := args[0]
		ctx := context.Background()

		dbPath := config.GetDBPath()
		s, err := store.NewSQLiteStore(dbPath)
		if err != nil {
			reportCampaignError(fmt.Errorf("open store: %w", err), 2)
			return
		}
		defer s.Close()

		// Posts dieser Kampagne holen
		posts, err := s.ListPosts(ctx, "all", "all", campaignSlug)
		if err != nil {
			reportCampaignError(fmt.Errorf("fetch campaign posts: %w", err), 2)
			return
		}

		if len(posts) == 0 {
			reportCampaignError(fmt.Errorf("no posts found for campaign %q", campaignSlug), 1) // 1 = Validierung / Not Found
			return
		}



		var triggeredCount int
		var successCount int
		var failedCount int
		var failedInfos []failedPostInfo

		for _, p := range posts {
			// Nur posten, wenn nicht bereits gepostet
			if p.Status != models.StatusPosted {
				triggeredCount++
				
				_, err := scheduler.PublishPost(ctx, s, &p, DryRunFlag)
				if err != nil {
					failedCount++
					failedInfos = append(failedInfos, failedPostInfo{
						PostID: p.ID,
						Error:  err.Error(),
					})
				} else {
					successCount++
				}
			}
		}

		reportCampaignPublishResult(campaignSlug, triggeredCount, successCount, failedCount, failedInfos)
	},
}

// JSON-Output Typen

type campaignListJSON struct {
	Campaigns []campaignStatsJSON `json:"campaigns"`
	Total     int                 `json:"total"`
}

type campaignStatsJSON struct {
	Campaign  string `json:"campaign"`
	Total     int    `json:"total"`
	Posted    int    `json:"posted"`
	Scheduled int    `json:"scheduled"`
	Drafts    int    `json:"drafts"`
}

type campaignPublishResultJSON struct {
	OK              bool             `json:"ok"`
	Campaign        string           `json:"campaign"`
	DryRun          bool             `json:"dry_run"`
	PostsTriggered  int              `json:"posts_triggered"`
	PostsSuccessful int              `json:"posts_successful"`
	PostsFailed     int              `json:"posts_failed"`
	Errors          []failedPostInfo `json:"errors,omitempty"`
}

type failedPostInfo struct {
	PostID string `json:"post_id"`
	Error  string `json:"error"`
}

type campaignErrorJSON struct {
	OK     bool     `json:"ok"`
	Code   int      `json:"code"`
	Errors []string `json:"errors"`
}

func reportCampaignsList(campaigns []models.Campaign) {
	if FormatFlag == "json" {
		statsList := []campaignStatsJSON{}
		for _, c := range campaigns {
			statsList = append(statsList, campaignStatsJSON{
				Campaign:  c.Slug,
				Total:     len(c.Posts),
				Posted:    c.Posted,
				Scheduled: c.Scheduled,
				Drafts:    c.Drafts,
			})
		}
		out := campaignListJSON{
			Campaigns: statsList,
			Total:     len(campaigns),
		}
		jsonBytes, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(jsonBytes))
	} else {
		fmt.Println("Campaigns Status Summary:")
		if len(campaigns) == 0 {
			fmt.Println(" - No campaigns found in database.")
			return
		}
		for _, c := range campaigns {
			fmt.Printf(" - Campaign: %-20s (Total: %d | Posted: %d | Scheduled: %d | Drafts: %d)\n",
				c.Slug, len(c.Posts), c.Posted, c.Scheduled, c.Drafts)
		}
	}
}

func reportCampaignPublishResult(campaign string, triggered, success, failed int, failedInfos []failedPostInfo) {
	// Falls Fehler aufgetreten sind, ist OK = false
	isSuccess := failed == 0

	if FormatFlag == "json" {
		out := campaignPublishResultJSON{
			OK:              isSuccess,
			Campaign:        campaign,
			DryRun:          DryRunFlag,
			PostsTriggered:  triggered,
			PostsSuccessful: success,
			PostsFailed:     failed,
			Errors:          failedInfos,
		}
		jsonBytes, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(jsonBytes))
	} else {
		prefix := ""
		if DryRunFlag {
			prefix = "[DRY RUN] "
		}
		action := "published"
		if DryRunFlag {
			action = "validated"
		}
		fmt.Printf("%sCampaign %q: Successfully %s %d/%d posts. (%d failed)\n", 
			prefix, campaign, action, success, triggered, failed)
		
		if failed > 0 {
			fmt.Fprintln(os.Stderr, "Errors:")
			for _, f := range failedInfos {
				fmt.Fprintf(os.Stderr, " - Post %s: %s\n", f.PostID, f.Error)
			}
		}
	}

	// Falls Fehler auftraten, beenden wir den Prozess mit Exit-Code 2 (API-Fehler/Veröffentlichungsfehler)
	if failed > 0 {
		os.Exit(2)
	}
}

func reportCampaignError(err error, exitCode int) {
	if FormatFlag == "json" {
		out := campaignErrorJSON{
			OK:     false,
			Code:   exitCode,
			Errors: []string{err.Error()},
		}
		jsonBytes, _ := json.MarshalIndent(out, "", "  ")
		fmt.Fprintln(os.Stderr, string(jsonBytes))
	} else {
		fmt.Fprintf(os.Stderr, "Campaign Error: %v\n", err)
	}
	os.Exit(exitCode)
}

func init() {
	campaignCmd.AddCommand(campaignListCmd)
	campaignCmd.AddCommand(campaignPostCmd)
	rootCmd.AddCommand(campaignCmd)
}
