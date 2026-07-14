package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/aeon022/postctl/internal/config"
)

func TestRSSCommands(t *testing.T) {
	// Setup custom exit handler
	var capturedExitCode int
	originalExitFunc := exitFunc
	exitFunc = func(code int) {
		capturedExitCode = code
	}
	defer func() {
		exitFunc = originalExitFunc
		_ = capturedExitCode
	}()

	// Set temporary test HOME directory to isolate config/db
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Save active config
	oldActiveConfig := config.ActiveConfig
	defer func() {
		config.ActiveConfig = oldActiveConfig
	}()

	// Mock RSS feed server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8" ?>
<rss version="2.0">
<channel>
 <title>Test Blog</title>
 <link>http://example.com</link>
 <description>A test blog</description>
 <item>
  <title>First post</title>
  <link>http://example.com/first</link>
  <description>This is the first post description.</description>
 </item>
</channel>
</rss>`))
	}))
	defer server.Close()

	// 1. Test RSS list (empty)
	config.ActiveConfig.RSSFeeds = nil
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"rss", "list"})
	rootCmd.Execute()

	output := buf.String()
	if !testingContains(output, "Keine RSS-Feeds konfiguriert") {
		t.Errorf("expected empty list message, got: %q", output)
	}

	// 2. Test RSS add
	buf.Reset()
	rootCmd.SetArgs([]string{"rss", "add", server.URL})
	rootCmd.Execute()

	output = buf.String()
	if !testingContains(output, "erfolgreich hinzugefügt") {
		t.Errorf("expected success message on add, got: %q", output)
	}

	if len(config.ActiveConfig.RSSFeeds) != 1 || config.ActiveConfig.RSSFeeds[0] != server.URL {
		t.Errorf("expected feed to be added to config, got: %v", config.ActiveConfig.RSSFeeds)
	}

	// 3. Test RSS list (non-empty)
	buf.Reset()
	rootCmd.SetArgs([]string{"rss", "list"})
	rootCmd.Execute()

	output = buf.String()
	if !testingContains(output, server.URL) {
		t.Errorf("expected list to contain feed URL, got: %q", output)
	}

	// 4. Test RSS import
	buf.Reset()
	rootCmd.SetArgs([]string{"rss", "import"})
	rootCmd.Execute()

	output = buf.String()
	if !testingContains(output, "First post") || !testingContains(output, "1 neue Beiträge") {
		t.Errorf("expected import success message, got: %q", output)
	}

	// 5. Test RSS remove
	buf.Reset()
	rootCmd.SetArgs([]string{"rss", "remove", server.URL})
	rootCmd.Execute()

	output = buf.String()
	if !testingContains(output, "erfolgreich entfernt") {
		t.Errorf("expected success message on remove, got: %q", output)
	}

	if len(config.ActiveConfig.RSSFeeds) != 0 {
		t.Errorf("expected feed to be removed, got: %v", config.ActiveConfig.RSSFeeds)
	}
}
