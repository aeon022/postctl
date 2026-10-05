package platforms

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/aeon022/postctl/internal/store"
)

// newTestStore opens an isolated on-disk SQLite store (never the real DB).
func newTestStore(t *testing.T) *store.SQLiteStore {
	t.Helper()
	s, err := store.NewSQLiteStore(filepath.Join(t.TempDir(), "postctl.db"), false)
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// testJWT builds an unsigned JWT whose exp claim is now+d.
func testJWT(d time.Duration) string {
	enc := base64.RawURLEncoding.EncodeToString
	claims, _ := json.Marshal(map[string]int64{"exp": time.Now().Add(d).Unix()})
	return enc([]byte(`{"alg":"none"}`)) + "." + enc(claims) + "." + enc([]byte("sig"))
}

func saveToken(t *testing.T, s *store.SQLiteStore, platform, token, refresh string) {
	t.Helper()
	if err := s.SaveToken(context.Background(), platform, token, refresh, nil); err != nil {
		t.Fatalf("SaveToken(%s): %v", platform, err)
	}
}
