package store

import (
	"context"
	"os"
	"testing"
)

func TestTokenKeyMigrationKeepsTokensReadable(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("POSTCTL_ENCRYPTION_KEY", "")
	s, err := NewSQLiteStore(":memory:", false)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()

	// existing install: token written under the legacy key (a key file exists only after migrate)
	enc, _ := encrypt("legacy-token")
	if _, err := s.db.Exec("INSERT INTO auth_tokens (platform, token) VALUES ('threads', ?)", enc); err != nil {
		t.Fatal(err)
	}
	if KeySource() != "legacy" {
		t.Fatalf("source = %s, want legacy", KeySource())
	}
	if n, err := s.MigrateTokenKey(ctx); err != nil || n != 1 {
		t.Fatalf("migrate: n=%d err=%v", n, err)
	}
	if KeySource() != "keyfile" {
		t.Fatalf("source after migrate = %s", KeySource())
	}
	if fi, err := os.Stat(KeyFilePath()); err != nil || fi.Mode().Perm() != 0o600 {
		t.Fatalf("key file must be 0600: %v %v", fi, err)
	}
	tok, _, _, err := s.GetToken(ctx, "threads")
	if err != nil || tok != "legacy-token" {
		t.Fatalf("token after migrate: %q %v", tok, err)
	}
	// the stored ciphertext no longer opens with the public default key alone
	var raw string
	s.db.QueryRow("SELECT token FROM auth_tokens WHERE platform='threads'").Scan(&raw)
	os.Remove(KeyFilePath())
	if _, _, _, err := s.GetToken(ctx, "threads"); err == nil {
		t.Fatal("without the key file the migrated token must not decrypt")
	}
}

func TestFreshInstallGetsKeyFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("POSTCTL_ENCRYPTION_KEY", "")
	s, _ := NewSQLiteStore(":memory:", false)
	defer s.Close()
	if err := s.SaveToken(context.Background(), "bluesky", "tok", "", nil); err != nil {
		t.Fatal(err)
	}
	if KeySource() != "keyfile" {
		t.Fatalf("a fresh install must start with a random key, got %s", KeySource())
	}
}
