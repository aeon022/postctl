package tui

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"charm.land/lipgloss/v2"
	"github.com/aeon022/missionctl-core/theme"
	"github.com/aeon022/missionctl-core/ui"
	"github.com/aeon022/postctl/internal/config"
	"github.com/charmbracelet/x/ansi"
)

// The editor's live preview: how the post will look on the chosen platform — image,
// caption cut where the platform folds it, character/hashtag counters, alt text, place
// tag and the planned time — so mistakes show up before anything is scheduled.

// foldAt is where a feed folds a long caption behind "… more" (0 = no fold shown).
var foldAt = map[string]int{"instagram": 125, "facebook": 477, "linkedin": 210, "twitter": 280, "bluesky": 300, "threads": 500, "mastodon": 500}

var hashtagRe = regexp.MustCompile(`#[\p{L}\p{N}_]+`)

// editorImagePaths splits the images field and resolves local files.
func (m Model) editorImagePaths() []string {
	var out []string
	for _, part := range strings.Split(m.editorImages.Value(), ",") {
		if t := strings.TrimSpace(part); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func localImage(src string) (string, bool) {
	if strings.HasPrefix(src, "https://") || strings.HasPrefix(src, "http://") {
		return "", false
	}
	for _, p := range []string{src, filepath.Join(config.ActiveConfig.Defaults.ImageDir, src)} {
		if _, err := os.Stat(p); err == nil {
			return p, true
		}
	}
	return "", false
}

func (m Model) previewContent(w, h int) string {
	pf := m.editorPlatform
	body := m.editorBody.Value()
	var b strings.Builder

	head := platformPill(pf)
	if chip := profileChip(config.ActiveProfile); chip != "" {
		head += " " + chip
	}
	b.WriteString(head + "\n\n")

	// image
	imgs := m.editorImagePaths()
	if len(imgs) > 0 {
		if p, ok := localImage(imgs[0]); ok {
			if pic := renderImageANSI(p, min(w-2, 40), max(min(h/3, 12), 4)); pic != "" {
				b.WriteString(pic + "\n")
			}
		} else {
			name := imgs[0]
			if u, err := url.Parse(imgs[0]); err == nil && u.Path != "" {
				name = path.Base(u.Path)
			}
			b.WriteString(dimStyle.Render("▨ "+ansi.Truncate(name, w-4, "…")) + "\n\n")
		}
		if len(imgs) > 1 {
			b.WriteString(dimStyle.Render(fmt.Sprintf("  ▤ %d "+Tr("prev_images"), len(imgs))) + "\n\n")
		}
	} else if pf == "instagram" && m.editorType != "story" {
		b.WriteString(amberStyle().Render("⚠ "+Tr("prev_need_image")) + "\n\n")
	}

	// caption, cut where the feed folds it
	caption := body
	more := ""
	if fold := foldAt[pf]; fold > 0 && utf8.RuneCountInString(caption) > fold {
		r := []rune(caption)
		caption = strings.TrimRight(string(r[:fold]), " \n") + "…"
		more = " " + dimStyle.Render(Tr("prev_more"))
	}
	if strings.TrimSpace(caption) == "" {
		caption = dimStyle.Render(Tr("prev_empty"))
	}
	for _, l := range wrapLines(caption, w-2) {
		b.WriteString(l + "\n")
	}
	if more != "" {
		b.WriteString(strings.TrimLeft(more, " ") + "\n")
	}
	b.WriteString("\n")

	// counters
	n := utf8.RuneCountInString(body)
	if lim := platformLimit(pf); lim > 0 && lim < 100000 {
		kind := ui.OK
		if n > lim {
			kind = ui.Err
		}
		b.WriteString(ui.Dot(kind) + fmt.Sprintf(" %d/%d %s\n", n, lim, Tr("prev_chars")))
	} else {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  %d %s", n, Tr("prev_chars"))) + "\n")
	}
	if tags := len(hashtagRe.FindAllString(body, -1)); tags > 0 {
		line := fmt.Sprintf("  #%d %s", tags, Tr("prev_hashtags"))
		if tags > 10 {
			line = amberStyle().Render(line + " – " + Tr("prev_too_many"))
		} else {
			line = dimStyle.Render(line)
		}
		b.WriteString(line + "\n")
	}
	if len(imgs) > 0 {
		if miss := missingAlts(m.editorImages.Value(), m.editorAlts.Value()); miss > 0 {
			b.WriteString(amberStyle().Render(fmt.Sprintf("  "+Tr("editor_alt_missing"), miss)) + "\n")
		} else {
			b.WriteString(lipgloss.NewStyle().Foreground(theme.GreenV2).Render("  ✓ "+Tr("prev_alt_ok")) + "\n")
		}
	}
	if pf == "instagram" && m.editorType != "story" {
		if config.ActiveConfig.Instagram.LocationID != "" {
			b.WriteString(dimStyle.Render("  📍 "+Tr("prev_place_on")) + "\n")
		} else {
			b.WriteString(dimStyle.Render("  📍 "+Tr("prev_place_off")) + "\n")
		}
	}

	// when
	when := Tr("prev_draft")
	if s := strings.TrimSpace(strings.ToLower(m.editorScheduledAt.Value())); s == "now" || s == "jetzt" {
		when = Tr("prev_now")
	} else if s != "" {
		if t, err := time.ParseInLocation("02.01.2006 15:04", m.editorScheduledAt.Value(), time.Local); err == nil {
			when = t.Format("Mon 02.01. 15:04")
		} else {
			when = amberStyle().Render("⚠ " + Tr("prev_bad_date"))
		}
	}
	b.WriteString("\n" + dimStyle.Render("⏱ ") + when)
	return b.String()
}

// renderEditor shows the form, with the live preview beside it on wide terminals.
func (m Model) renderEditor(w, h int) string {
	if w < wideBreak {
		return m.renderEditorForm(w, h)
	}
	lw := w * 5 / 9
	rw := w - lw - 1
	form := m.renderEditorForm(lw, h)
	prev := ui.Panel(rw, h, Tr("preview_title"), m.previewContent(rw-4, h-2), false)
	return joinColumns(form, prev, lw, 1)
}
