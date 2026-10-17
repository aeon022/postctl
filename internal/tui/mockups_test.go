package tui

import (
	tea "charm.land/bubbletea/v2"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aeon022/missionctl-core/tuitest"
	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/models"
	"github.com/aeon022/postctl/internal/store"
)

// TestRenderLandingMockups writes ANSI screenshots of the real TUI (fictional demo data) for the
// landing page: POSTCTL_MOCKUP_DIR=/some/dir go test ./internal/tui -run TestRenderLandingMockups
func TestRenderLandingMockups(t *testing.T) {
	dir := os.Getenv("POSTCTL_MOCKUP_DIR")
	if dir == "" {
		t.Skip("set POSTCTL_MOCKUP_DIR to render landing mockups")
	}
	t.Setenv("HOME", t.TempDir())
	prevCfg, prevProfile := config.ActiveConfig, config.ActiveProfile
	t.Cleanup(func() { config.ActiveConfig, config.ActiveProfile = prevCfg, prevProfile })
	config.ActiveConfig.LicenseStatus = "granted" // in-process only, temp HOME
	config.ActiveConfig.Instagram.AccessToken = "demo"
	config.ActiveConfig.Instagram.TokenIssued = time.Now().AddDate(0, 0, -9).Format("2006-01-02")

	s, err := store.NewSQLiteStore(":memory:", false)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	now := time.Now()
	day := func(d, h, m int) *time.Time {
		x := time.Date(now.Year(), now.Month(), now.Day(), h, m, 0, 0, now.Location()).AddDate(0, 0, d)
		return &x
	}
	type dp struct {
		id, plat, camp, title, body, status string
		at                                  *time.Time
		typ                                 string
		imgs                                []string
	}
	demo := []dp{
		{"d1", "instagram", "summer-launch", "Summer Menu — teaser", "Our summer menu drops Friday 🍋\nThree new dishes, one very cold drink.\n\n#nordlicht #summermenu #local", "scheduled", day(1, 17, 30), "single", []string{"https://cdn.example.com/menu-teaser.png"}},
		{"d2", "threads", "summer-launch", "Summer Menu — teaser", "Summer menu drops Friday. Three new dishes, one very cold drink. 🍋", "scheduled", day(1, 18, 0), "single", nil},
		{"d3", "twitter", "summer-launch", "Summer Menu — teaser", "Summer menu drops Friday 🍋 Three new dishes, one very cold drink.\n\nnordlicht.example/menu", "scheduled", day(1, 18, 30), "single", nil},
		{"d4", "facebook", "summer-launch", "Summer Menu — teaser", "Our summer menu drops this Friday! Three new dishes and one very cold drink.", "scheduled", day(1, 19, 0), "single", nil},
		{"d5", "instagram", "summer-launch", "Menu day — carousel", "The new menu, dish by dish. Swipe →", "scheduled", day(3, 12, 0), "single", []string{"a.png", "b.png", "c.png"}},
		{"d6", "linkedin", "hiring", "We are hiring a line cook", "We're growing the kitchen team. If you love seasonal cooking, say hello.", "scheduled", day(4, 9, 0), "single", nil},
		{"d7", "bluesky", "hiring", "We are hiring a line cook", "We're growing the kitchen team — seasonal cooking, good people. Say hello!", "scheduled", day(4, 9, 30), "single", nil},
		{"d8", "instagram", "behind-scenes", "Prep day story", "", "scheduled", day(6, 11, 0), "story", []string{"story.png"}},
		{"d9", "twitter", "behind-scenes", "Thread: how we cure lemons", "1/ Preserved lemons take 30 days and a lot of patience.", "draft", nil, "single", nil},
		{"d10", "facebook", "behind-scenes", "Photo night recap", "Thanks to everyone who came to photo night!", "draft", nil, "single", nil},
		{"p1", "instagram", "spring-recap", "Spring recap", "Spring in review.", "posted", day(-3, 17, 30), "single", nil},
		{"p2", "threads", "spring-recap", "Spring recap", "Spring in review.", "posted", day(-3, 18, 0), "single", nil},
		{"p3", "twitter", "spring-recap", "Spring recap", "Spring in review.", "posted", day(-3, 18, 30), "single", nil},
		{"p4", "facebook", "spring-recap", "Opening hours update", "New opening hours from May.", "posted", day(-9, 10, 0), "single", nil},
		{"p5", "instagram", "spring-recap", "Wine tasting night", "Wine tasting night — recap.", "posted", day(-12, 17, 30), "single", nil},
		{"p6", "linkedin", "hiring", "Welcome, Mara!", "Welcome to the team, Mara.", "posted", day(-15, 9, 0), "single", nil},
	}
	for _, d := range demo {
		p := &models.Post{ID: d.id, Platform: d.plat, Type: d.typ, Language: "en", Campaign: d.camp, Title: d.title, Body: d.body, Images: d.imgs,
			Alts: altFor(d.imgs), Status: d.status, CreatedAt: now.AddDate(0, 0, -20), UpdatedAt: now}
		if d.status == "posted" {
			p.PostedAt, p.PlatformID = d.at, "17"+d.id
		} else {
			p.ScheduledAt = d.at
		}
		if err := s.SavePost(ctx, p); err != nil {
			t.Fatal(err)
		}
		if d.status == "posted" {
			e := &models.HistoryEntry{ID: "h" + d.id, PostID: d.id, Action: "posted", PlatformID: p.PlatformID, CreatedAt: *d.at}
			if err := s.AddHistoryEntry(ctx, e); err != nil {
				t.Fatal(err)
			}
		}
	}

	exp := now.AddDate(0, 0, 50)
	for _, pl := range []string{"twitter", "linkedin", "threads", "mastodon", "bluesky", "facebook", "instagram"} {
		if err := s.SaveToken(ctx, pl, "demo", "", &exp); err != nil {
			t.Fatal(err)
		}
	}
	m := NewModel(s)
	m.loading = false
	m.width, m.height = 118, 34
	res, _ := m.Update(m.loadDataCmd())
	os.MkdirAll(dir, 0o755)
	save := func(name string, keys ...string) {
		mm, cmds := tuitest.Keys(res, keys...)
		for round := 0; round < 3; round++ { // let async loaders (stats, logs…) finish
			var next []tea.Cmd
			for _, c := range cmds {
				ch := make(chan tea.Msg, 1)
				go func(c tea.Cmd) { ch <- c() }(c)
				select {
				case msg := <-ch:
					var nc []tea.Cmd
					mm, nc = tuitest.Send(mm, msg)
					next = append(next, nc...)
				case <-time.After(1500 * time.Millisecond):
				}
			}
			cmds = next
		}
		out := mm.View().Content
		if err := os.WriteFile(filepath.Join(dir, name+".ansi"), []byte(out), 0o644); err != nil {
			t.Fatal(err)
		}
		fmt.Println("wrote", name)
	}
	save("dashboard")
	save("posts", "tab")
	save("schedule", "tab", "tab")
	save("calendar", "tab", "tab", "tab", "right")
	save("history", "tab", "tab", "tab", "tab")
	save("settings", "tab", "tab", "tab", "tab", "tab", "tab")
	save("editor", "tab", "e")
}

func altFor(imgs []string) []string {
	var a []string
	for range imgs {
		a = append(a, "Three summer dishes on a wooden table")
	}
	return a
}
