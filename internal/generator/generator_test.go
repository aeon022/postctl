package generator

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestCleanSlug(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World!", "hello-world"},
		{"post_2026---amazing", "post_2026-amazing"},
		{"Some Page URL here", "some-page-url-here"},
		{"!!!special#chars!!!", "special-chars"},
	}

	for _, tt := range tests {
		got := CleanSlug(tt.input)
		if got != tt.expected {
			t.Errorf("CleanSlug(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}

func TestCleanJSONResponse(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"   {\"key\": \"val\"}   ", "{\"key\": \"val\"}"},
		{"```json\n{\"key\": \"val\"}\n```", "{\"key\": \"val\"}"},
		{"```\n{\"key\": \"val\"}\n```", "{\"key\": \"val\"}"},
	}

	for _, tt := range tests {
		got := cleanJSONResponse(tt.input)
		if got != tt.expected {
			t.Errorf("cleanJSONResponse(%q) = %q; want %q", tt.input, got, tt.expected)
		}
	}
}

func TestGenerateContentOpenAI(t *testing.T) {
	expectedPosts := GeneratedPosts{
		Slug: "mocked-post",
		Twitter: GeneratedPostData{
			Title:   "Twitter Title",
			Content: "## Tweet 1\nHello Twitter",
		},
		LinkedIn: GeneratedPostData{
			Title:   "LinkedIn Title",
			Content: "Hello LinkedIn",
		},
		Threads: GeneratedPostData{
			Title:   "Threads Title",
			Content: "Hello Threads",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected JSON content-type")
		}
		if r.Header.Get("Authorization") != "Bearer secret-key" {
			t.Errorf("expected bearer token")
		}

		responseBytes, _ := json.Marshal(expectedPosts)

		openAIResp := OpenAIResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{
				{
					Message: struct {
						Content string `json:"content"`
					}{
						Content: string(responseBytes),
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(openAIResp)
	}))
	defer server.Close()

	cfg := GeneratorConfig{
		Provider: "openai",
		APIKey:   "secret-key",
		Model:    "gpt-4o-mini",
		BaseURL:  server.URL,
	}

	posts, err := GenerateContent(context.Background(), cfg, "http://example.com", "Test Title", "Sample body text")
	if err != nil {
		t.Fatalf("GenerateContent failed: %v", err)
	}

	if posts.Slug != "mocked-post" {
		t.Errorf("expected slug 'mocked-post', got %q", posts.Slug)
	}
	if posts.Twitter.Title != "Twitter Title" {
		t.Errorf("expected twitter title 'Twitter Title', got %q", posts.Twitter.Title)
	}
	if posts.LinkedIn.Content != "Hello LinkedIn" {
		t.Errorf("expected linkedin content 'Hello LinkedIn', got %q", posts.LinkedIn.Content)
	}
}

func TestGenerateContentClaude(t *testing.T) {
	expectedPosts := GeneratedPosts{
		Slug: "mocked-post-claude",
		Twitter: GeneratedPostData{
			Title:   "Twitter Title",
			Content: "## Tweet 1\nHello Twitter",
		},
		LinkedIn: GeneratedPostData{
			Title:   "LinkedIn Title",
			Content: "Hello LinkedIn",
		},
		Threads: GeneratedPostData{
			Title:   "Threads Title",
			Content: "Hello Threads",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected JSON content-type")
		}
		if r.Header.Get("x-api-key") != "claude-secret" {
			t.Errorf("expected x-api-key header")
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Errorf("expected anthropic-version header")
		}

		responseBytes, _ := json.Marshal(expectedPosts)

		claudeResp := ClaudeResponse{
			Content: []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{
				{
					Type: "text",
					Text: string(responseBytes),
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(claudeResp)
	}))
	defer server.Close()

	cfg := GeneratorConfig{
		Provider: "claude",
		APIKey:   "claude-secret",
		Model:    "claude-3-5-sonnet",
		BaseURL:  server.URL,
	}

	posts, err := GenerateContent(context.Background(), cfg, "http://example.com", "Test Title", "Sample body text")
	if err != nil {
		t.Fatalf("GenerateContent failed: %v", err)
	}

	if posts.Slug != "mocked-post-claude" {
		t.Errorf("expected slug 'mocked-post-claude', got %q", posts.Slug)
	}
	if posts.Threads.Title != "Threads Title" {
		t.Errorf("expected threads title 'Threads Title', got %q", posts.Threads.Title)
	}
}

func TestSaveToMarkdownFiles(t *testing.T) {
	tempDir := t.TempDir()
	posts := &GeneratedPosts{
		Slug: "test-repurpose",
		Twitter: GeneratedPostData{
			Title:   "Twitter Title",
			Content: "## Tweet 1\nHello X!",
		},
		LinkedIn: GeneratedPostData{
			Title:   "LinkedIn Title",
			Content: "Hello LinkedIn!",
		},
		Threads: GeneratedPostData{
			Title:   "Threads Title",
			Content: "Hello Threads!",
		},
	}

	files, err := SaveToMarkdownFiles(posts, tempDir, "test-campaign")
	if err != nil {
		t.Fatalf("SaveToMarkdownFiles failed: %v", err)
	}

	if len(files) != 3 {
		t.Errorf("expected 3 files, got %d", len(files))
	}

	expectedFiles := map[string]string{
		filepath.Join(tempDir, "test-repurpose-twitter.md"):   "platform: twitter\ntype: thread\ntitle: Twitter Title\ncampaign: test-campaign\n---\n## Tweet 1\nHello X!\n",
		filepath.Join(tempDir, "test-repurpose-linkedin.md"):  "platform: linkedin\ntype: single\ntitle: LinkedIn Title\ncampaign: test-campaign\n---\nHello LinkedIn!\n",
		filepath.Join(tempDir, "test-repurpose-threads.md"):   "platform: threads\ntype: single\ntitle: Threads Title\ncampaign: test-campaign\n---\nHello Threads!\n",
	}

	for path, expectedContent := range expectedFiles {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("failed to read file %s: %v", path, err)
			continue
		}

		content := string(data)
		if !testingContains(content, expectedContent) {
			t.Errorf("file %s mismatch.\nExpected to contain:\n%q\nGot:\n%q", path, expectedContent, content)
		}
	}
}

