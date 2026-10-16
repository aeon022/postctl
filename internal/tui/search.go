package tui

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/aeon022/missionctl-core/ui"
	"github.com/aeon022/postctl/internal/models"
)

// "/" in the Posts tab: live search over title, text, platform, campaign and id.
// Enter keeps the filter, Esc clears it.

func (m Model) matchesSearch(p models.Post) bool {
	q := strings.ToLower(strings.TrimSpace(m.searchQuery))
	if q == "" {
		return true
	}
	hay := strings.ToLower(strings.Join([]string{p.Title, p.Body, p.Platform, p.Campaign, p.ID, p.Status}, "\n"))
	for _, word := range strings.Fields(q) { // every word must appear somewhere
		if !strings.Contains(hay, word) {
			return false
		}
	}
	return true
}

func (m Model) openSearch() Model {
	in := textinput.New()
	in.Placeholder = Tr("search_placeholder")
	in.SetValue(m.searchQuery)
	in.Focus()
	m.searchInput = in
	m.searching = true
	return m
}

func (m Model) searchKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.searching, m.searchQuery, m.cursor = false, "", 0
		return m, nil
	case "enter":
		m.searching = false
		return m, nil
	}
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	m.searchQuery = m.searchInput.Value()
	m.cursor = 0
	return m, cmd
}

func (m Model) searchBody(w int) string {
	pw := min(w, 56)
	n := len(m.getFilteredPosts())
	return ui.Panel(pw, 3, Tr("search_title"), "/ "+m.searchInput.View()+"  "+dimStyle.Render(strings.Repeat(" ", 1)+itoa(n)+" "+Tr("search_hits")), true)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for ; n > 0; n /= 10 {
		b = append([]byte{byte('0' + n%10)}, b...)
	}
	return string(b)
}
