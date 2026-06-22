package generator

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestScrapeURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `
			<!DOCTYPE html>
			<html>
			<head>
				<title>My Awesome Article</title>
				<style>body { color: red; }</style>
				<script>console.log("hello");</script>
			</head>
			<body>
				<h1>Welcome to the Future!</h1>
				<p>This is standard content to extract. <!-- comment --></p>
			</body>
			</html>
		`)
	}))
	defer server.Close()

	scraped, err := ScrapeURL(server.URL)
	if err != nil {
		t.Fatalf("ScrapeURL failed: %v", err)
	}

	if scraped.Title != "My Awesome Article" {
		t.Errorf("expected title 'My Awesome Article', got %q", scraped.Title)
	}

	// Verify style, script, comments are removed, content is cleaned
	if !testingContains(scraped.Content, "Welcome to the Future!") || !testingContains(scraped.Content, "This is standard content to extract.") {
		t.Errorf("expected content to contain key sentences, got: %q", scraped.Content)
	}

	if testingContains(scraped.Content, "color: red") || testingContains(scraped.Content, "console.log") || testingContains(scraped.Content, "comment") {
		t.Errorf("scraped content contained script/style/comment content: %q", scraped.Content)
	}
}

func testingContains(s, substr string) bool {
	// A simple helper using standard Go strings
	return len(s) >= len(substr) && (s == substr || (len(substr) > 0 && (len(s) > len(substr) && (s[0:len(substr)] == substr || testingContains(s[1:], substr)))))
}
