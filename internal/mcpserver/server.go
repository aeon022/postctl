package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/models"
	"github.com/aeon022/postctl/internal/scheduler"
	"github.com/aeon022/postctl/internal/store"
	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func Serve() error {
	s := mcpserver.NewMCPServer("postctl", "1.0.0",
		mcpserver.WithToolCapabilities(true),
	)
	s.AddTool(toolListPosts(), handleListPosts)
	s.AddTool(toolGetPost(), handleGetPost)
	s.AddTool(toolCreatePost(), handleCreatePost)
	s.AddTool(toolPublishPost(), handlePublishPost)
	s.AddTool(toolSchedulePost(), handleSchedulePost)
	s.AddTool(toolListCampaigns(), handleListCampaigns)
	return mcpserver.ServeStdio(s)
}

// ── Tool definitions ──────────────────────────────────────────────────────────

func toolListPosts() mcp.Tool {
	return mcp.NewTool("list_posts",
		mcp.WithDescription("List social media posts from the local postctl database. Filter by platform, status, or campaign. Returns title, platform, status, scheduled_at, body preview."),
		mcp.WithString("platform", mcp.Description("Filter by platform: twitter, linkedin, threads, mastodon, bluesky")),
		mcp.WithString("status", mcp.Description("Filter by status: draft, scheduled, posted, failed")),
		mcp.WithString("campaign", mcp.Description("Filter by campaign name")),
		mcp.WithNumber("limit", mcp.Description("Max results (default 50)")),
	)
}

func toolGetPost() mcp.Tool {
	return mcp.NewTool("get_post",
		mcp.WithDescription("Get a single post by ID, including full content and metadata."),
		mcp.WithString("id", mcp.Required(), mcp.Description("Post ID")),
	)
}

func toolCreatePost() mcp.Tool {
	return mcp.NewTool("create_post",
		mcp.WithDescription("Create a new social media post draft in the postctl database. Use publish_post to send it immediately or set schedule to publish later."),
		mcp.WithString("platform", mcp.Required(), mcp.Description("Target platform: twitter, linkedin, threads, mastodon, bluesky")),
		mcp.WithString("body", mcp.Required(), mcp.Description("Post content. For Twitter threads, separate tweets with '---' on its own line.")),
		mcp.WithString("title", mcp.Description("Optional title for reference")),
		mcp.WithString("campaign", mcp.Description("Campaign name to group related posts")),
		mcp.WithString("schedule", mcp.Description("Schedule time in RFC3339 format (e.g. 2026-07-10T09:00:00+02:00). Omit to save as draft.")),
	)
}

func toolPublishPost() mcp.Tool {
	return mcp.NewTool("publish_post",
		mcp.WithDescription("Publish an existing post immediately to its configured platform. The post must exist in the database (use create_post first or list_posts to find existing drafts)."),
		mcp.WithString("id", mcp.Required(), mcp.Description("Post ID to publish")),
		mcp.WithBoolean("dry_run", mcp.Description("If true, validate without actually posting (default: false)")),
	)
}

func toolSchedulePost() mcp.Tool {
	return mcp.NewTool("schedule_post",
		mcp.WithDescription("Update the scheduled publish time of an existing post."),
		mcp.WithString("id", mcp.Required(), mcp.Description("Post ID")),
		mcp.WithString("schedule", mcp.Required(), mcp.Description("New schedule time in RFC3339 format")),
	)
}

func toolListCampaigns() mcp.Tool {
	return mcp.NewTool("list_campaigns",
		mcp.WithDescription("List all unique campaign names and their post counts/status breakdown."),
	)
}

// ── Handlers ──────────────────────────────────────────────────────────────────

func handleListPosts(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	platform := req.GetString("platform", "")
	status := req.GetString("status", "")
	campaign := req.GetString("campaign", "")
	limit := int(req.GetFloat("limit", 50))

	s, err := openStore()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	defer s.Close()

	posts, err := s.ListPosts(context.Background(), platform, status, campaign)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if limit > 0 && len(posts) > limit {
		posts = posts[:limit]
	}

	type row struct {
		ID          string  `json:"id"`
		Platform    string  `json:"platform"`
		Status      string  `json:"status"`
		Campaign    string  `json:"campaign,omitempty"`
		Title       string  `json:"title,omitempty"`
		Preview     string  `json:"preview"`
		ScheduledAt *string `json:"scheduled_at,omitempty"`
		PostedAt    *string `json:"posted_at,omitempty"`
	}
	var rows []row
	for _, p := range posts {
		r := row{
			ID:       p.ID,
			Platform: p.Platform,
			Status:   p.Status,
			Campaign: p.Campaign,
			Title:    p.Title,
			Preview:  previewBody(&p),
		}
		if p.ScheduledAt != nil {
			t := p.ScheduledAt.Format(time.RFC3339)
			r.ScheduledAt = &t
		}
		if p.PostedAt != nil {
			t := p.PostedAt.Format(time.RFC3339)
			r.PostedAt = &t
		}
		rows = append(rows, r)
	}
	return jsonResult(rows)
}

func handleGetPost(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := req.GetString("id", "")
	if id == "" {
		return mcp.NewToolResultError("id is required"), nil
	}
	s, err := openStore()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	defer s.Close()

	post, err := s.GetPost(context.Background(), id)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("post not found: %v", err)), nil
	}
	return jsonResult(post)
}

func handleCreatePost(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	platform := req.GetString("platform", "")
	body := req.GetString("body", "")
	title := req.GetString("title", "")
	campaign := req.GetString("campaign", "")
	scheduleStr := req.GetString("schedule", "")

	if platform == "" || body == "" {
		return mcp.NewToolResultError("platform and body are required"), nil
	}

	post := &models.Post{
		ID:        uuid.New().String(),
		Platform:  platform,
		Title:     title,
		Campaign:  campaign,
		Status:    models.StatusDraft,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Split Twitter threads by "---"
	if platform == models.PlatformTwitter && strings.Contains(body, "\n---\n") {
		parts := strings.Split(body, "\n---\n")
		post.Type = "thread"
		for i, p := range parts {
			post.Tweets = append(post.Tweets, models.Tweet{Index: i + 1, Content: strings.TrimSpace(p)})
		}
	} else {
		post.Type = "single"
		post.Body = body
	}

	if scheduleStr != "" {
		t, err := time.Parse(time.RFC3339, scheduleStr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid schedule time: %v", err)), nil
		}
		post.ScheduledAt = &t
		post.Status = models.StatusScheduled
	}

	s, err := openStore()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	defer s.Close()

	if err := s.SavePost(context.Background(), post); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return jsonResult(map[string]any{"ok": true, "id": post.ID, "status": post.Status})
}

func handlePublishPost(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := req.GetString("id", "")
	dryRun := req.GetBool("dry_run", false)

	if id == "" {
		return mcp.NewToolResultError("id is required"), nil
	}
	s, err := openStore()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	defer s.Close()

	post, err := s.GetPost(context.Background(), id)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("post not found: %v", err)), nil
	}

	platformID, err := scheduler.PublishPost(context.Background(), s, post, dryRun)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return jsonResult(map[string]any{"ok": true, "platform_id": platformID, "platform": post.Platform})
}

func handleSchedulePost(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := req.GetString("id", "")
	scheduleStr := req.GetString("schedule", "")

	if id == "" || scheduleStr == "" {
		return mcp.NewToolResultError("id and schedule are required"), nil
	}
	t, err := time.Parse(time.RFC3339, scheduleStr)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid schedule time: %v", err)), nil
	}

	s, err := openStore()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	defer s.Close()

	post, err := s.GetPost(context.Background(), id)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("post not found: %v", err)), nil
	}
	post.ScheduledAt = &t
	post.Status = models.StatusScheduled
	post.UpdatedAt = time.Now()
	if err := s.SavePost(context.Background(), post); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return jsonResult(map[string]any{"ok": true, "id": id, "scheduled_at": t.Format(time.RFC3339)})
}

func handleListCampaigns(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s, err := openStore()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	defer s.Close()

	posts, err := s.ListPosts(context.Background(), "", "", "")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	type campStat struct {
		Name     string         `json:"name"`
		Total    int            `json:"total"`
		ByStatus map[string]int `json:"by_status"`
	}
	camps := map[string]*campStat{}
	for _, p := range posts {
		name := p.Campaign
		if name == "" {
			name = "(no campaign)"
		}
		if _, ok := camps[name]; !ok {
			camps[name] = &campStat{Name: name, ByStatus: map[string]int{}}
		}
		camps[name].Total++
		camps[name].ByStatus[p.Status]++
	}
	var result []*campStat
	for _, v := range camps {
		result = append(result, v)
	}
	return jsonResult(result)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func openStore() (store.Store, error) {
	dbPath := config.GetDBPath()
	return store.NewSQLiteStore(dbPath)
}

func jsonResult(v any) (*mcp.CallToolResult, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(string(b)), nil
}

func previewBody(p *models.Post) string {
	if p.Body != "" {
		if len(p.Body) > 100 {
			return p.Body[:97] + "…"
		}
		return p.Body
	}
	if len(p.Tweets) > 0 {
		content := p.Tweets[0].Content
		if len(content) > 100 {
			return content[:97] + "…"
		}
		return content
	}
	return ""
}
