package tui

import (
	"strings"
	"testing"

	"github.com/aeon022/missionctl-core/tuitest"
	"github.com/aeon022/postctl/internal/models"
)

func TestPostsSearchFiltersLiveAndEscClears(t *testing.T) {
	m := layoutModel(t, 120, 40)
	m.posts = []models.Post{
		{ID: "1", Platform: "instagram", Title: "Cradle Collective", Campaign: "okt", Status: models.StatusScheduled},
		{ID: "2", Platform: "threads", Title: "Iris hört zu", Campaign: "okt", Status: models.StatusScheduled},
		{ID: "3", Platform: "instagram", Title: "Martin Moro", Campaign: "nov", Status: models.StatusDraft},
	}
	mm, _ := m.switchTab(tabPosts)

	typed, _ := tuitest.Keys(mm, "/", "i", "n", "s", "t", "a")
	tm := typed.(Model)
	if !tm.searching || len(tm.getFilteredPosts()) != 2 {
		t.Fatalf("'insta' matches the two instagram posts, got %d (searching=%v)", len(tm.getFilteredPosts()), tm.searching)
	}
	both, _ := tuitest.Keys(tm, " ", "m", "o")
	if got := both.(Model).getFilteredPosts(); len(got) != 1 || got[0].ID != "3" {
		t.Errorf("every word must match: 'insta mo' → Martin Moro only, got %v", got)
	}

	kept, _ := tuitest.Keys(tm, "enter")
	km := kept.(Model)
	if km.searching || km.searchQuery != "insta" {
		t.Errorf("enter keeps the filter and closes the input (searching=%v query=%q)", km.searching, km.searchQuery)
	}
	if text := tuitest.Text(km); !strings.Contains(text, "search: insta") {
		t.Errorf("the header shows the active search:\n%s", text)
	}
	cleared, _ := tuitest.Keys(km, "esc")
	if cm := cleared.(Model); cm.searchQuery != "" || len(cm.getFilteredPosts()) != 3 {
		t.Error("esc clears the search")
	}
}
