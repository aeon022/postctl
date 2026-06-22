package cmd

import (
	"bytes"
	"testing"

	"github.com/aeon022/postctl/internal/config"
)

func TestAnalyticsCmdEmptyDB(t *testing.T) {
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
	config.ActiveConfig.DBPath = tempDir + "/test_analytics_empty.db"

	buf := new(bytes.Buffer)
	analyticsCmd.SetOut(buf)
	analyticsCmd.SetErr(buf)

	// Set flags manually since we are running the Run function directly
	analyticsDays = 5
	FormatFlag = "human"

	// Run command directly
	analyticsCmd.Run(analyticsCmd, []string{})

	if capturedExitCode != 0 {
		t.Errorf("expected exit code 0 on empty DB, got %d", capturedExitCode)
	}
}
