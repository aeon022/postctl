package markdown

import (
	"testing"
	"time"

	"github.com/aeon022/postctl/internal/models"
)

func TestCleanID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World", "hello-world"},
		{"My_Awesome_File", "my_awesome_file"},
		{"post-2026---cool", "post-2026-cool"},
		{"!!!Special Chars!!!", "special-chars"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := cleanID(tt.input)
			if got != tt.expected {
				t.Errorf("cleanID(%q) = %q; want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestTwitterLength(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected int
	}{
		{"Short text", "Hello", 5},
		{"German umlauts", "Grüße", 5},
		{"Short URL", "https://x.com", 23},
		{"Long URL", "https://verylongdomainname.com/some/path/to/resource?param=value", 23},
		{"Mixed text", "Check out https://google.com for info!", 37}, // "Check out " (10) + URL (23) + " for info!" (10) = 43? Wait:
		// "Check out " is 10 chars.
		// "https://google.com" is replaced by 23 chars.
		// " for info!" is 10 chars.
		// Total: 10 + 23 + 10 = 43 chars. Let's verify.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tweet := models.Tweet{Content: tt.content}
			got := tweet.CharCount()
			// For "Check out https://google.com for info!":
			// "Check out " = 10
			// URL = 23
			// " for info!" = 10
			// Total = 43. Let's adjust the expected value in our test.
			expected := tt.expected
			if tt.name == "Mixed text" {
				expected = 43
			}
			if got != expected {
				t.Errorf("CharCount(%q) = %d; want %d", tt.content, got, expected)
			}
		})
	}
}

func TestParseContentSingle(t *testing.T) {
	content := `---
platform: linkedin
type: single
campaign: launch-2026
schedule: 2026-06-23T09:00
images:
  - screenshots/dashboard.png
---
This is a single LinkedIn post body.
It can have multiple paragraphs.
`
	posts, err := ParseContent(content, "test-post.md")
	if err != nil {
		t.Fatalf("ParseContent failed: %v", err)
	}

	if len(posts) != 1 {
		t.Fatalf("expected 1 post, got %d", len(posts))
	}

	post := posts[0]
	if post.ID != "test-post-linkedin" {
		t.Errorf("expected ID test-post-linkedin, got %q", post.ID)
	}
	if post.Platform != models.PlatformLinkedIn {
		t.Errorf("expected platform linkedin, got %q", post.Platform)
	}
	if post.Type != "single" {
		t.Errorf("expected type single, got %q", post.Type)
	}
	if post.Campaign != "launch-2026" {
		t.Errorf("expected campaign launch-2026, got %q", post.Campaign)
	}
	if post.ScheduledAt == nil {
		t.Fatalf("expected ScheduledAt to be set")
	}
	
	expectedTime := time.Date(2026, 6, 23, 9, 0, 0, 0, time.Local)
	if !post.ScheduledAt.Equal(expectedTime) {
		t.Errorf("expected schedule time %v, got %v", expectedTime, *post.ScheduledAt)
	}

	if post.Body != "This is a single LinkedIn post body.\nIt can have multiple paragraphs." {
		t.Errorf("unexpected body: %q", post.Body)
	}

	if len(post.Images) != 1 || post.Images[0] != "screenshots/dashboard.png" {
		t.Errorf("unexpected images: %v", post.Images)
	}
}

func TestParseContentThread(t *testing.T) {
	content := `---
platform: twitter
type: thread
campaign: thread-test
images:
  - img1.png
  - img2.png
---
## Tweet 1
First tweet content.
<!-- image: inline-img.png -->

## Tweet 2
Second tweet content.

## Tweet 3
Third tweet content.

## Reply
Reply content with link: https://github.com/aeon022/postctl
`
	posts, err := ParseContent(content, "test-thread.md")
	if err != nil {
		t.Fatalf("ParseContent failed: %v", err)
	}

	if len(posts) != 1 {
		t.Fatalf("expected 1 post, got %d", len(posts))
	}

	post := posts[0]
	if post.Platform != models.PlatformTwitter {
		t.Errorf("expected platform twitter, got %q", post.Platform)
	}
	if len(post.Tweets) != 4 {
		t.Fatalf("expected 4 tweets, got %d", len(post.Tweets))
	}

	// Check Tweet 1 (inline image overrides frontmatter)
	if post.Tweets[0].Content != "First tweet content." {
		t.Errorf("Tweet 1 content mismatch: %q", post.Tweets[0].Content)
	}
	if post.Tweets[0].Image != "inline-img.png" {
		t.Errorf("Tweet 1 image mismatch: %q", post.Tweets[0].Image)
	}

	// Check Tweet 2 (gets images[0] because it's index 2 (1-based index 2, so images[2-2] = images[0]))
	if post.Tweets[1].Content != "Second tweet content." {
		t.Errorf("Tweet 2 content mismatch: %q", post.Tweets[1].Content)
	}
	if post.Tweets[1].Image != "img1.png" {
		t.Errorf("Tweet 2 image mismatch: %q", post.Tweets[1].Image)
	}

	// Check Tweet 3 (gets images[1] = img2.png)
	if post.Tweets[2].Content != "Third tweet content." {
		t.Errorf("Tweet 3 content mismatch: %q", post.Tweets[2].Content)
	}
	if post.Tweets[2].Image != "img2.png" {
		t.Errorf("Tweet 3 image mismatch: %q", post.Tweets[2].Image)
	}

	// Check Reply (isReply = true, content has URL parsed correctly)
	reply := post.Tweets[3]
	if !reply.IsReply {
		t.Errorf("expected last tweet to be a reply")
	}
	if reply.Content != "Reply content with link: https://github.com/aeon022/postctl" {
		t.Errorf("Reply content mismatch: %q", reply.Content)
	}
}

func TestParseContentAll(t *testing.T) {
	content := `---
platform: all
type: single
---
Post everywhere!
`
	posts, err := ParseContent(content, "everywhere.md")
	if err != nil {
		t.Fatalf("ParseContent failed: %v", err)
	}

	if len(posts) != 3 {
		t.Fatalf("expected 3 posts (twitter, linkedin, threads), got %d", len(posts))
	}

	platforms := make(map[string]bool)
	for _, p := range posts {
		platforms[p.Platform] = true
	}

	if !platforms["twitter"] || !platforms["linkedin"] || !platforms["threads"] {
		t.Errorf("missing platforms, got: %v", platforms)
	}
}
