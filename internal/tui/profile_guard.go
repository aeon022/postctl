package tui

import (
	"fmt"
	"hash/fnv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/aeon022/missionctl-core/theme"
	"github.com/aeon022/missionctl-core/ui"
	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/models"
)

// Profile guard: every named profile gets its own color (chip in the header, colored
// divider) and manual publishing asks first, so a post never goes to the wrong account.

var profilePalette = []string{"#E5A00D", "#B86FD9", "#22B8A5", "#E8604C", "#4C9AFF", "#8BC34A"}

// profileColor returns the profile's color, "" for the default profile.
func profileColor(name string) string {
	switch name {
	case "", "default":
		return ""
	case "bruecke":
		return "#E8142B" // the Brücke's red
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(name))
	return profilePalette[int(h.Sum32())%len(profilePalette)]
}

// profileChip is the colored "● name" badge ("" for the default profile).
func profileChip(name string) string {
	c := profileColor(name)
	if c == "" {
		return ""
	}
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color(c)).Padding(0, 1).Render("● " + name)
}

// profileDivider is the line under the header, in the profile's color when it has one.
func profileDivider(w int) string {
	c := profileColor(config.ActiveProfile)
	if c == "" {
		return ui.Divider(w, "")
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(c)).Render(strings.Repeat("─", w))
}

// publishAsk is the pending "really publish now?" question.
type publishAsk struct {
	ids   []string
	posts []models.Post
}

// askPublish opens the confirmation for a manual publish; nothing is sent yet.
func (m Model) askPublish(ids []string) (Model, tea.Cmd) {
	byID := map[string]models.Post{}
	for _, p := range m.posts {
		byID[p.ID] = p
	}
	a := &publishAsk{ids: ids}
	for _, id := range ids {
		if p, ok := byID[id]; ok {
			a.posts = append(a.posts, p)
		}
	}
	if m.selectedPost != nil && len(a.posts) == 0 {
		a.posts = []models.Post{*m.selectedPost}
	}
	m.ask = a
	return m, nil
}

// askKey handles the confirmation keys; y/enter publishes, anything else cancels.
func (m Model) askKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	a := m.ask
	m.ask = nil
	switch msg.String() {
	case "y", "Y", "enter":
		m.loading = true
		if len(a.ids) == 1 {
			m.statusMessage = Tr("st_publishing")
		} else {
			m.statusMessage = Tr("st_publishing_n")
		}
		return m, m.bulkPublishPostsCmd(a.ids)
	}
	return m, nil
}

func (m Model) askBody(w, h int) string {
	a := m.ask
	pw := min(w, 64)
	var lines []string
	who := config.ActiveProfile
	if who == "" || who == "default" {
		who = Tr("picker_default") // the default profile is shown as "Privat" / "Private"
	}
	chip := profileChip(config.ActiveProfile)
	if chip == "" {
		chip = dimStyle.Render(who)
	}
	lines = append(lines, dimStyle.Render(Tr("ask_profile"))+" "+chip, "")
	shown := a.posts
	if len(shown) > 6 {
		shown = shown[:6]
	}
	for _, p := range shown {
		t := stripEmojis(p.Title)
		if t == "" {
			t = p.ID
		}
		lines = append(lines, "  "+platformPill(p.Platform)+" "+truncateTo(t, pw-24))
	}
	if more := len(a.posts) - len(shown); more > 0 {
		lines = append(lines, dimStyle.Render(fmt.Sprintf("  … +%d", more)))
	}
	if len(a.posts) == 0 {
		lines = append(lines, "  "+fmt.Sprintf(Tr("ask_n_posts"), len(a.ids)))
	}
	lines = append(lines, "", ui.KeyCap("y")+" "+dimStyle.Render(Tr("ask_yes"))+"   "+ui.KeyCap("n")+" "+dimStyle.Render(Tr("ask_no")))
	return ui.Panel(pw, min(len(lines)+2, max(h, 5)), Tr("ask_title"), strings.Join(lines, "\n"), true)
}

func truncateTo(s string, w int) string {
	if w < 4 || lipgloss.Width(s) <= w {
		return s
	}
	r := []rune(s)
	for len(r) > 1 && lipgloss.Width(string(r))+1 > w {
		r = r[:len(r)-1]
	}
	return string(r) + "…"
}

var _ = theme.MutedV2
