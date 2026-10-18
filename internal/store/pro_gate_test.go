package store

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/aeon022/postctl/internal/config"
)

func TestPlatformAllowedFollowsTheSubscription(t *testing.T) {
	ctx := context.Background()
	s, err := NewSQLiteStore(filepath.Join(t.TempDir(), "p.db"), false)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	saved := config.ActiveConfig
	t.Cleanup(func() { config.ActiveConfig = saved })

	// four platforms are connected (a former Pro user)
	config.ActiveConfig.LicenseStatus = "granted"
	for _, p := range []string{"instagram", "facebook", "threads", "twitter"} {
		if err := s.SaveToken(ctx, p, "tok-"+p, "", nil); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range []string{"instagram", "twitter"} {
		if !s.PlatformAllowed(ctx, p) {
			t.Errorf("Pro: %s must be allowed", p)
		}
	}

	// the subscription ends: the first two connected platforms keep publishing, the others pause
	config.ActiveConfig.LicenseStatus = "revoked"
	want := map[string]bool{"instagram": true, "facebook": true, "threads": false, "twitter": false}
	for p, ok := range want {
		if got := s.PlatformAllowed(ctx, p); got != ok {
			t.Errorf("lapsed: PlatformAllowed(%s) = %v, want %v", p, got, ok)
		}
	}
}
