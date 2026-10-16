package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/aeon022/missionctl-core/tuitest"
)

func TestAdaptCaptionTrimsOnlyTheClosingHashtagBlock(t *testing.T) {
	body := "Text #inline bleibt\n\n#a #b #c #d #e"
	if got := adaptCaption("instagram", body); got != body {
		t.Errorf("instagram keeps everything, got %q", got)
	}
	if got := adaptCaption("threads", body); !strings.HasSuffix(got, "\n\n#a") || !strings.Contains(got, "#inline") {
		t.Errorf("threads keeps one closing tag and inline tags, got %q", got)
	}
	if got := adaptCaption("facebook", body); !strings.HasSuffix(got, "#a #b #c") {
		t.Errorf("facebook keeps three, got %q", got)
	}
	plain := "kein Hashtag-Block am Ende"
	if adaptCaption("threads", plain) != plain {
		t.Error("text without a closing hashtag block is untouched")
	}
}

func TestComposerSavesOnePostPerPlatformStaggeredAndAdapted(t *testing.T) {
	m := layoutModel(t, 120, 40)
	mm, _ := tuitest.Keys(m, "tab", "n")
	em := mm.(Model)
	em.editorTargets = map[string]bool{"instagram": true, "threads": true, "facebook": true}
	em.editorPlatform = "instagram"
	em.editorScheduledAt.SetValue("20.10.2026 18:00")
	em.editorCampaign.SetValue("cmp")
	em.editorBody.SetValue("Hallo\n\n#a #b #c #d")

	if got := em.targetPlatforms(); len(got) != 3 || got[0] != "threads" && got[0] != "facebook" && got[0] != "instagram" {
		t.Fatalf("targets = %v", got)
	}
	if err := em.saveEditedPost(); err != nil {
		t.Fatal(err)
	}
	posts, err := em.store.ListPosts(context.Background(), "all", "all", "cmp")
	if err != nil || len(posts) != 3 {
		t.Fatalf("want 3 posts, got %d err=%v", len(posts), err)
	}
	times := map[string]time.Time{}
	bodies := map[string]string{}
	for _, p := range posts {
		times[p.Platform], bodies[p.Platform] = *p.ScheduledAt, p.Body
	}
	// selector order: facebook, instagram?, ... — the gap between consecutive targets is 30 minutes
	base := time.Date(2026, 10, 20, 18, 0, 0, 0, time.Local)
	seen := map[time.Duration]bool{}
	for _, tm := range times {
		seen[tm.Sub(base)] = true
	}
	for _, want := range []time.Duration{0, 30 * time.Minute, 60 * time.Minute} {
		if !seen[want] {
			t.Errorf("missing a post at +%v: %v", want, times)
		}
	}
	if !strings.HasSuffix(bodies["threads"], "#a") || strings.Contains(bodies["threads"], "#b") {
		t.Errorf("threads text adapted: %q", bodies["threads"])
	}
	if !strings.HasSuffix(bodies["instagram"], "#a #b #c #d") {
		t.Errorf("instagram text unchanged: %q", bodies["instagram"])
	}
}
