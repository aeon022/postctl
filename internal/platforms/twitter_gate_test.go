package platforms

import (
	"strings"
	"testing"

	"github.com/aeon022/postctl/internal/config"
)

func TestCookieModeGate(t *testing.T) {
	prev := config.ActiveConfig
	t.Cleanup(func() { config.ActiveConfig = prev })

	config.ActiveConfig.Twitter.AuthMode = "api"
	if err := cookieModeGate(); err != nil {
		t.Fatalf("api mode must pass: %v", err)
	}
	config.ActiveConfig.Twitter.AuthMode = "cookie"
	config.ActiveConfig.Twitter.UnofficialCookieMode = false
	if err := cookieModeGate(); err == nil || !strings.Contains(err.Error(), "unofficial_cookie_mode") {
		t.Fatalf("cookie mode without opt-in must be refused with a hint, got %v", err)
	}
	config.ActiveConfig.Twitter.UnofficialCookieMode = true
	if err := cookieModeGate(); err != nil {
		t.Fatalf("cookie mode with opt-in must pass: %v", err)
	}
}
