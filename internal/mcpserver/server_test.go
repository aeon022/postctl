package mcpserver

import (
	"context"
	"encoding/json"
	"github.com/aeon022/missionctl-core/activity"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

// call invokes a handler directly with args, against an isolated database:
// openStore() resolves POSTCTL_DATA_DIR, pointed at a temp dir by isolate().
func call(t *testing.T, h func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error), args map[string]any) (text string, isErr bool) {
	t.Helper()
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	res, err := h(context.Background(), req)
	if err != nil || res == nil || len(res.Content) == 0 {
		t.Fatalf("handler returned %v, %v", res, err)
	}
	tc, ok := res.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("content is %T, want text", res.Content[0])
	}
	return tc.Text, res.IsError
}

func isolate(t *testing.T) { t.Helper(); t.Setenv("POSTCTL_DATA_DIR", t.TempDir()) }

func create(t *testing.T, args map[string]any) string {
	t.Helper()
	text, isErr := call(t, handleCreatePost, args)
	if isErr {
		t.Fatalf("create_post: %s", text)
	}
	var out struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(text), &out); err != nil || out.ID == "" {
		t.Fatalf("create_post result %q: %v", text, err)
	}
	return out.ID
}

func TestCreateGetAndList(t *testing.T) {
	isolate(t)
	id := create(t, map[string]any{"platform": "bluesky", "body": "hallo welt", "title": "T", "campaign": "launch"})

	text, isErr := call(t, handleGetPost, map[string]any{"id": id})
	if isErr || !strings.Contains(text, "hallo welt") || !strings.Contains(text, `"draft"`) {
		t.Errorf("get_post = %v %s", isErr, text)
	}

	create(t, map[string]any{"platform": "linkedin", "body": "anderer"})
	text, _ = call(t, handleListPosts, map[string]any{"platform": "bluesky"})
	if !strings.Contains(text, id) || strings.Contains(text, "anderer") {
		t.Errorf("platform filter leaked: %s", text)
	}
	text, _ = call(t, handleListPosts, map[string]any{"campaign": "launch"})
	if !strings.Contains(text, id) {
		t.Errorf("campaign filter lost the post: %s", text)
	}
	if text, _ = call(t, handleListPosts, map[string]any{"limit": float64(1)}); strings.Count(text, `"id"`) != 1 {
		t.Errorf("limit=1 returned: %s", text)
	}
}

func TestTwitterThreadSplit(t *testing.T) {
	isolate(t)
	id := create(t, map[string]any{"platform": "twitter", "body": "eins\n---\nzwei\n---\ndrei"})
	text, _ := call(t, handleGetPost, map[string]any{"id": id})
	if !strings.Contains(text, `"thread"`) || !strings.Contains(text, "zwei") {
		t.Errorf("thread not split: %s", text)
	}
}

func TestCreateWithScheduleAndReschedule(t *testing.T) {
	isolate(t)
	id := create(t, map[string]any{"platform": "mastodon", "body": "später", "schedule": "2026-12-01T09:00:00+01:00"})
	text, _ := call(t, handleGetPost, map[string]any{"id": id})
	if !strings.Contains(text, `"scheduled"`) {
		t.Errorf("schedule at create should set status scheduled: %s", text)
	}

	draft := create(t, map[string]any{"platform": "mastodon", "body": "entwurf"})
	text, isErr := call(t, handleSchedulePost, map[string]any{"id": draft, "schedule": "2026-12-24T18:00:00Z"})
	if isErr || !strings.Contains(text, "2026-12-24T18:00:00Z") {
		t.Errorf("schedule_post = %v %s", isErr, text)
	}
	if text, _ = call(t, handleGetPost, map[string]any{"id": draft}); !strings.Contains(text, `"scheduled"`) {
		t.Errorf("draft not scheduled: %s", text)
	}
}

func TestValidationErrors(t *testing.T) {
	isolate(t)
	cases := []struct {
		name string
		h    func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)
		args map[string]any
		want string
	}{
		{"create without body", handleCreatePost, map[string]any{"platform": "twitter"}, "required"},
		{"create bad schedule", handleCreatePost, map[string]any{"platform": "twitter", "body": "x", "schedule": "morgen"}, "invalid schedule"},
		{"get without id", handleGetPost, map[string]any{}, "required"},
		{"get unknown id", handleGetPost, map[string]any{"id": "nope"}, "not found"},
		{"schedule without time", handleSchedulePost, map[string]any{"id": "x"}, "required"},
		{"schedule bad time", handleSchedulePost, map[string]any{"id": "x", "schedule": "bald"}, "invalid schedule"},
		{"publish without id", handlePublishPost, map[string]any{}, "required"},
		{"publish unknown id", handlePublishPost, map[string]any{"id": "nope"}, "not found"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			text, isErr := call(t, c.h, c.args)
			if !isErr || !strings.Contains(text, c.want) {
				t.Errorf("got isErr=%v %q, want error containing %q", isErr, text, c.want)
			}
		})
	}
}

func TestPublishDryRun(t *testing.T) {
	isolate(t)
	id := create(t, map[string]any{"platform": "bluesky", "body": "trockenübung"})
	text, isErr := call(t, handlePublishPost, map[string]any{"id": id, "dry_run": true})
	if isErr || !strings.Contains(text, "dryrun-") {
		t.Errorf("dry-run publish = %v %s", isErr, text)
	}
}

func TestListCampaignsEmpty(t *testing.T) {
	isolate(t)
	if text, isErr := call(t, handleListCampaigns, nil); isErr {
		t.Errorf("list_campaigns on empty DB errored: %s", text)
	}
}

func TestSchedulePostLogsOneScheduledEvent(t *testing.T) {
	isolate(t)
	t.Setenv("MISSIONCTL_DATA_DIR", t.TempDir())
	id := create(t, map[string]any{"platform": "mastodon", "body": "entwurf", "title": "Mein Beitrag"})
	if _, isErr := call(t, handleSchedulePost, map[string]any{"id": id, "schedule": "2026-12-24T18:00:00Z"}); isErr {
		t.Fatal("schedule_post failed")
	}
	from, to := activity.Day(time.Now())
	evs, err := activity.Read(from, to)
	if err != nil || len(evs) != 1 || evs[0].Action != "scheduled" || !strings.HasPrefix(evs[0].Title, "Mein Beitrag → mastodon (2026-12-24 ") {
		t.Fatalf("events = %+v, %v; want exactly one scheduled event", evs, err)
	}
}

func TestCreateWithScheduleLogsAndPlainDraftDoesNot(t *testing.T) {
	isolate(t)
	t.Setenv("MISSIONCTL_DATA_DIR", t.TempDir())
	create(t, map[string]any{"platform": "bluesky", "body": "nur ein Entwurf"})
	from, to := activity.Day(time.Now())
	if evs, _ := activity.Read(from, to); len(evs) != 0 {
		t.Fatalf("a draft is not an activity: %+v", evs)
	}
	create(t, map[string]any{"platform": "bluesky", "body": "geplant", "schedule": "2026-12-01T09:00:00+01:00"})
	if evs, _ := activity.Read(from, to); len(evs) != 1 || evs[0].Action != "scheduled" {
		t.Errorf("scheduled create must log once: %+v", evs)
	}
}
