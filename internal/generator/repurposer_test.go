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

func TestRepurposeContent(t *testing.T) {
	expectedResult := RepurposeResult{
		Slug: "repurposed-mock",
		Posts: map[string]RepurposedPostData{
			"linkedin": {
				Title:   "LinkedIn Version",
				Content: "Hello LinkedIn repurposed!",
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		responseBytes, _ := json.Marshal(expectedResult)

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
		APIKey:   "secret",
		BaseURL:  server.URL,
	}

	result, err := RepurposeContent(context.Background(), cfg, "twitter", "thread", "Original Title", "Original Content", []string{"linkedin"})
	if err != nil {
		t.Fatalf("RepurposeContent failed: %v", err)
	}

	if result.Slug != "repurposed-mock" {
		t.Errorf("expected slug 'repurposed-mock', got %q", result.Slug)
	}

	linkedInPost, ok := result.Posts["linkedin"]
	if !ok {
		t.Fatalf("expected LinkedIn post in results")
	}

	if linkedInPost.Title != "LinkedIn Version" {
		t.Errorf("expected Title 'LinkedIn Version', got %q", linkedInPost.Title)
	}
}

func TestSaveRepurposedToMarkdownFiles(t *testing.T) {
	tempDir := t.TempDir()
	result := &RepurposeResult{
		Slug: "test-slug",
		Posts: map[string]RepurposedPostData{
			"linkedin": {
				Title:   "LI Title",
				Content: "LI Body content",
			},
		},
	}

	files, err := SaveRepurposedToMarkdownFiles(result, tempDir, "campaign-1")
	if err != nil {
		t.Fatalf("SaveRepurposedToMarkdownFiles failed: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 written file, got %d", len(files))
	}

	expectedPath := filepath.Clean(filepath.Join(tempDir, "test-slug-repurposed-to-linkedin.md"))
	if filepath.Clean(files[0]) != expectedPath {
		t.Errorf("expected path %q, got %q", expectedPath, files[0])
	}

	data, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	content := string(data)
	expectedContents := []string{
		"platform: linkedin",
		"type: single",
		"title: LI Title",
		"campaign: campaign-1",
		"LI Body content",
	}

	for _, expected := range expectedContents {
		if !testingContains(content, expected) {
			t.Errorf("expected file to contain %q, but got:\n%q", expected, content)
		}
	}
}
