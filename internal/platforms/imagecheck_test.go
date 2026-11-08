package platforms

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
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

func TestOpenImageURL(t *testing.T) {
	png := "\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/feed/a.png" {
			_, _ = w.Write([]byte(png))
			return
		}
		if r.URL.Path == "/key.png" {
			_, _ = w.Write([]byte("-----BEGIN OPENSSH PRIVATE KEY-----"))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	ctx := context.Background()

	rc, name, err := openImage(ctx, srv.URL+"/feed/a.png?v=c697e69")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(rc)
	if name != "a.png" || string(b) != png {
		t.Errorf("got name %q, %d bytes", name, len(b))
	}
	if _, _, err := openImage(ctx, srv.URL+"/missing.png"); err == nil {
		t.Error("404 must fail")
	}
	if _, _, err := openImage(ctx, srv.URL+"/key.png"); err == nil {
		t.Error("non-image must be refused")
	}
}
