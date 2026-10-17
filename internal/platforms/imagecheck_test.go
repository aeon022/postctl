package platforms

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckLocalImage(t *testing.T) {
	dir := t.TempDir()
	png := filepath.Join(dir, "a.png")
	os.WriteFile(png, []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR"), 0o600)
	secret := filepath.Join(dir, "id_ed25519")
	os.WriteFile(secret, []byte("-----BEGIN OPENSSH PRIVATE KEY-----\nabc\n"), 0o600)
	renamed := filepath.Join(dir, "key.png") // wrong extension must not help
	os.WriteFile(renamed, []byte("-----BEGIN OPENSSH PRIVATE KEY-----\nabc\n"), 0o600)
	link := filepath.Join(dir, "link.png")
	os.Symlink(secret, link)

	if err := CheckLocalImage(png); err != nil {
		t.Fatalf("real png must pass: %v", err)
	}
	if err := CheckLocalImage("https://example.com/x.png"); err != nil {
		t.Fatalf("urls pass: %v", err)
	}
	for _, bad := range []string{secret, renamed, link, dir, filepath.Join(dir, "missing.png")} {
		if err := CheckLocalImage(bad); err == nil {
			t.Errorf("%s must be refused", bad)
		}
	}
}
