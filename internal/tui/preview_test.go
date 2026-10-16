package tui

import (
	"strings"
	"testing"

	"github.com/aeon022/missionctl-core/tuitest"
)

func TestEditorPreviewFoldsCaptionAndWarnsAboutMissingAlt(t *testing.T) {
	m := layoutModel(t, 150, 42)
	mm, _ := tuitest.Keys(m, "tab", "n")
	em := mm.(Model)
	em.editorPlatform = "instagram"
	em.editorImages.SetValue("https://x/a.png, https://x/b.png")
	em.editorAlts.SetValue("nur eins")
	em.editorBody.SetValue(strings.Repeat("Langer Text ", 30) + "#a #b #c")

	text := tuitest.Text(em)
	for _, want := range []string{"Preview", "… more", "alt text missing for 1", "no place tag", "draft"} {
		if !strings.Contains(text, want) {
			t.Errorf("preview should show %q:\n%s", want, text)
		}
	}

	narrow := tuitest.Text(func() Model { x := layoutModel(t, 100, 42); y, _ := tuitest.Keys(x, "tab", "n"); return y.(Model) }())
	if strings.Contains(narrow, "Preview") {
		t.Error("below the wide breakpoint the form keeps the full width (no preview panel)")
	}
}

func TestMissingAlts(t *testing.T) {
	for _, c := range []struct {
		imgs, alts string
		want       int
	}{{"", "", 0}, {"a.png", "", 1}, {"a.png, b.png", "x | y", 0}, {"a.png, b.png", "x", 1}, {"a.png", " | ", 1}} {
		if got := missingAlts(c.imgs, c.alts); got != c.want {
			t.Errorf("missingAlts(%q,%q) = %d, want %d", c.imgs, c.alts, got, c.want)
		}
	}
}
