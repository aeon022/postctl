package cmd

import (
	"bytes"
	"testing"

	"github.com/aeon022/postctl/internal/config"
)

func TestRepurposeCmdMissingToFlag(t *testing.T) {
	// Temporarily capture exit code
	var capturedExitCode int
	originalExitFunc := exitFunc
	exitFunc = func(code int) {
		capturedExitCode = code
	}
	defer func() {
		exitFunc = originalExitFunc
	}()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	// Invoke repurpose without the --to flag
	rootCmd.SetArgs([]string{"repurpose", "some-post-id"})
	rootCmd.Execute()

	if capturedExitCode != 1 {
		t.Errorf("expected exit code 1 due to missing --to flag, got %d", capturedExitCode)
	}
}

func TestRepurposeCmdPostNotFound(t *testing.T) {
	// Temporarily capture exit code
	var capturedExitCode int
	originalExitFunc := exitFunc
	exitFunc = func(code int) {
		capturedExitCode = code
	}
	defer func() {
		exitFunc = originalExitFunc
	}()

	// Save active config values and temporarily set DB path to a non-existent temp file
	oldActiveConfig := config.ActiveConfig
	defer func() {
		config.ActiveConfig = oldActiveConfig
	}()

	tempDir := t.TempDir()
	config.ActiveConfig.DBPath = tempDir + "/test_repurpose_not_found.db"

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	// Invoke repurpose with a valid --to flag but a post ID that won't exist in a fresh DB
	rootCmd.SetArgs([]string{"repurpose", "non-existent-id", "--to", "linkedin"})
	rootCmd.Execute()

	if capturedExitCode != 1 {
		t.Errorf("expected exit code 1 for missing post, got %d", capturedExitCode)
	}
}
