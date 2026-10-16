package store

import (
	"os"
	"testing"
)

// TestMain runs every test of this package with a throw-away HOME, so a test can never read or
// overwrite the real ~/.config/postctl (config, tokens, license) or ~/.local/share/postctl.
func TestMain(m *testing.M) {
	home, err := os.MkdirTemp("", "postctl-test-home-")
	if err != nil {
		panic(err)
	}
	os.Setenv("HOME", home)
	code := m.Run()
	os.RemoveAll(home)
	os.Exit(code)
}
