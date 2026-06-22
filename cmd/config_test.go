package cmd

import (
	"bytes"
	"testing"

	"github.com/aeon022/postctl/internal/config"
)

func TestConfigShowCmd(t *testing.T) {
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

	rootCmd.SetArgs([]string{"config", "show"})
	rootCmd.Execute()

	if capturedExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", capturedExitCode)
	}

	output := buf.String()
	if !testingContains(output, "db_path:") {
		t.Errorf("expected output to contain db_path, got:\n%s", output)
	}
}

func TestConfigSetCmd(t *testing.T) {
	// Temporarily capture exit code
	var capturedExitCode int
	originalExitFunc := exitFunc
	exitFunc = func(code int) {
		capturedExitCode = code
	}
	defer func() {
		exitFunc = originalExitFunc
	}()

	// Save active config values and temporarily enable dry run to prevent file writes
	oldActiveConfig := config.ActiveConfig
	defer func() {
		config.ActiveConfig = oldActiveConfig
	}()

	DryRunFlag = true
	defer func() {
		DryRunFlag = false
	}()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	// Set ai.provider to a test value
	rootCmd.SetArgs([]string{"config", "set", "ai.provider", "test-claude"})
	rootCmd.Execute()

	if capturedExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", capturedExitCode)
	}

	if config.ActiveConfig.AI.Provider != "test-claude" {
		t.Errorf("expected AI provider to be updated to 'test-claude', got %q", config.ActiveConfig.AI.Provider)
	}
}
