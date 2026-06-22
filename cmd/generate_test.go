package cmd

import (
	"bytes"
	"testing"

	"github.com/aeon022/postctl/internal/config"
)

func TestGenerateCmdInvalidURL(t *testing.T) {
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

	// Invoke generate with an invalid URL
	rootCmd.SetArgs([]string{"generate", "not-a-valid-url"})
	
	// Execute via rootCmd so that SetArgs overrides os.Args correctly
	rootCmd.Execute()

	if capturedExitCode != 1 {
		t.Errorf("expected exit code 1, got %d", capturedExitCode)
	}
}

func TestGenerateCmdMissingAPIKey(t *testing.T) {
	// Temporarily capture exit code
	var capturedExitCode int
	originalExitFunc := exitFunc
	exitFunc = func(code int) {
		capturedExitCode = code
	}
	defer func() {
		exitFunc = originalExitFunc
	}()

	// Save active config values and temporarily clear key/provider
	oldActiveConfig := config.ActiveConfig
	defer func() {
		config.ActiveConfig = oldActiveConfig
	}()

	config.ActiveConfig.AI.Provider = "openai"
	config.ActiveConfig.AI.APIKey = ""
	config.ActiveConfig.AI.BaseURL = ""

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	// Invoke generate with a valid URL format but missing key
	rootCmd.SetArgs([]string{"generate", "https://example.com/article"})
	rootCmd.Execute()

	if capturedExitCode != 1 {
		t.Errorf("expected exit code 1 for missing API key, got %d", capturedExitCode)
	}
}
