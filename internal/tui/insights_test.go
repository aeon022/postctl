package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
)

func TestTopPostsSortedByInteractions(t *testing.T) {
	posts := []postMetric{
		{Title: "low", Platform: "instagram", Likes: 1},
		{Title: "best", Platform: "threads", Likes: 5, Comments: 4, Shares: 2},
		{Title: "mid", Platform: "facebook", Likes: 4},
	}
	lines := topPostLines(posts, 2, 80)
	if len(lines) != 2 || !strings.Contains(ansi.Strip(lines[0]), "best") || !strings.Contains(ansi.Strip(lines[1]), "mid") {
		t.Errorf("top posts = %q", lines)
	}
	if got := topPostLines(nil, 5, 80); len(got) != 1 {
		t.Errorf("empty input shows one hint line, got %q", got)
	}
}

func TestHeatmapMarksTheSlotPostsWentOut(t *testing.T) {
	at := time.Date(2026, 10, 19, 18, 0, 0, 0, time.Local) // Monday 18:00
	lines := heatmapLines([]postMetric{{At: at, Likes: 9}, {At: at.AddDate(0, 0, 1), Likes: 0}})
	if len(lines) != 8 {
		t.Fatalf("header + 7 weekdays expected, got %d lines", len(lines))
	}
	mon, tue, wed := ansi.Strip(lines[1]), ansi.Strip(lines[2]), ansi.Strip(lines[3])
	if !strings.Contains(mon, "■") && !strings.Contains(mon, "◼") {
		t.Errorf("monday row should have a strong cell: %q", mon)
	}
	if strings.Count(wed, "▪") != 17 {
		t.Errorf("wednesday has no posts (all empty cells): %q", wed)
	}
	_ = tue
	if got := heatmapLines(nil); len(got) != 1 {
		t.Errorf("no data → one hint line, got %q", got)
	}
}
