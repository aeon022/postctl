package markdown

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFileReadsAltTexts(t *testing.T) {
	f := filepath.Join(t.TempDir(), "p.md")
	src := "---\nplatform: instagram\ntype: single\nimages:\n  - https://x/a.png\n  - https://x/b.png\nalt:\n  - Plakat A\n  - Plakat B\n---\nText\n"
	if err := os.WriteFile(f, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	posts, err := ParseFile(f)
	if err != nil || len(posts) != 1 {
		t.Fatalf("posts=%d err=%v", len(posts), err)
	}
	if got := posts[0].AltAt(1); got != "Plakat B" {
		t.Errorf("AltAt(1) = %q", got)
	}
}
