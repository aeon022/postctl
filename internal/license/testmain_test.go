package license

import (
	"os"
	"testing"
)

// TestMain runs the package with a throw-away HOME so no test can touch the real ~/.config/postctl.
func TestMain(m *testing.M) {
	home, err := os.MkdirTemp("", "postctl-license-test-")
	if err != nil {
		panic(err)
	}
	os.Setenv("HOME", home)
	code := m.Run()
	os.RemoveAll(home)
	os.Exit(code)
}
