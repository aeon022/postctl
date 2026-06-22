package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestTemplateLaunchCmd(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "my-launch.md")

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	// Set args to run template launch subcommand
	rootCmd.SetArgs([]string{"template", "launch", "-o", outputPath})
	rootCmd.Execute()

	// Check that the file was created and contains key placeholder strings
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read created template file: %v", err)
	}

	content := string(data)
	expectedSubstrings := []string{
		"platform: all",
		"type: single",
		"title: \"Product Launch: [Product Name]\"",
		"launch-[year]",
	}

	for _, sub := range expectedSubstrings {
		if !testingContains(content, sub) {
			t.Errorf("expected template to contain %q, but got:\n%s", sub, content)
		}
	}
}

func TestTemplateFeatureCmdDryRun(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "dry-run-feature.md")

	// Set global DryRunFlag to true
	DryRunFlag = true
	defer func() {
		DryRunFlag = false
	}()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"template", "feature", "-o", outputPath})
	rootCmd.Execute()

	// Under dry-run, the file MUST NOT be created
	if _, err := os.Stat(outputPath); !os.IsNotExist(err) {
		t.Errorf("expected file %s to not exist under dry-run", outputPath)
	}
}

// Simple local helper to avoid test duplication
func testingContains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(substr) > 0 && (len(s) > len(substr) && (s[0:len(substr)] == substr || testingContains(s[1:], substr)))))
}
