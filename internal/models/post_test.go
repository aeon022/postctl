package models

import (
	"testing"
)

func TestDeriveTitle(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected string
	}{
		{
			name:     "Simple first line",
			body:     "This is the first line.\nThis is the second line.",
			expected: "This is the first line.",
		},
		{
			name:     "Markdown styling removed",
			body:     "**Bold** and _italic_ text here.\nNext line.",
			expected: "Bold and italic text here.",
		},
		{
			name:     "Skipping empty lines and headers",
			body:     "\n\n## Header\n---\nActual content start here.\nMore content.",
			expected: "Actual content start here.",
		},
		{
			name:     "Truncating long first line",
			body:     "This is an exceptionally long first line of the post that should definitely be truncated because it exceeds forty characters.",
			expected: "This is an exceptionally long first l...",
		},
		{
			name:     "Thread post with dividers and tweet headers",
			body:     "## Tweet 1\nHello world!\n---\n## Tweet 2\nAnother tweet.",
			expected: "Hello world!",
		},
		{
			name:     "Empty body fallback",
			body:     "",
			expected: "Unbenannter Beitrag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeriveTitle(tt.body)
			if got != tt.expected {
				t.Errorf("DeriveTitle() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestPrepareTweets(t *testing.T) {
	t.Run("Thread with global images and empty tweet images", func(t *testing.T) {
		post := &Post{
			Type:   "thread",
			Images: []string{"img1.png", "img2.png"},
			Tweets: []Tweet{
				{Index: 1, Content: "First tweet"},
				{Index: 2, Content: "Second tweet"},
				{Index: 3, Content: "Third tweet"},
			},
		}
		post.PrepareTweets()

		if post.Tweets[0].Image != "img1.png" {
			t.Errorf("expected Tweets[0].Image to be 'img1.png', got %q", post.Tweets[0].Image)
		}
		if post.Tweets[1].Image != "img2.png" {
			t.Errorf("expected Tweets[1].Image to be 'img2.png', got %q", post.Tweets[1].Image)
		}
		if post.Tweets[2].Image != "" {
			t.Errorf("expected Tweets[2].Image to be '', got %q", post.Tweets[2].Image)
		}
	})

	t.Run("Thread with some preset tweet images", func(t *testing.T) {
		post := &Post{
			Type:   "thread",
			Images: []string{"img1.png", "img2.png"},
			Tweets: []Tweet{
				{Index: 1, Content: "First tweet"},
				{Index: 2, Content: "Second tweet", Image: "preset.png"},
			},
		}
		post.PrepareTweets()

		// Presets should prevent fallback distribution
		if post.Tweets[0].Image != "" {
			t.Errorf("expected Tweets[0].Image to remain empty, got %q", post.Tweets[0].Image)
		}
		if post.Tweets[1].Image != "preset.png" {
			t.Errorf("expected Tweets[1].Image to remain 'preset.png', got %q", post.Tweets[1].Image)
		}
	})

	t.Run("Single post with global images", func(t *testing.T) {
		post := &Post{
			Type:   "single",
			Images: []string{"img1.png"},
			Tweets: []Tweet{
				{Index: 1, Content: "Single tweet content"},
			},
		}
		post.PrepareTweets()

		if post.Tweets[0].Image != "" {
			t.Errorf("expected Tweets[0].Image to remain empty, got %q", post.Tweets[0].Image)
		}
	})
}

