package tui

import (
	"fmt"
	"strings"

	"github.com/aeon022/missionctl-core/ui"
)

// kv is one dimmed "label  value" metadata row.
func kv(label, value string) string { return dimStyle.Render(padRight(label, 13)) + value }

// renderDetailView is the post preview: a scrollable focused panel.
func (m Model) renderDetailView(w, h int) string {
	if m.selectedPost == nil {
		return ""
	}
	p := m.selectedPost
	iw := panelRowW(w)

	lines := []string{platformPill(p.Platform) + " " + statusPill(p.Status), ""}
	lines = append(lines, kv(Tr("detail_campaign"), p.Campaign), kv(Tr("detail_type"), p.Type))
	status := strings.ToUpper(p.Status)
	if p.ScheduledAt != nil {
		status += " · " + Tr("detail_sched_at") + " " + p.ScheduledAt.Format("02.01.2006 15:04")
	}
	lines = append(lines, kv(Tr("detail_status"), status))
	if p.Error != "" {
		lines = append(lines, kv(Tr("detail_error"), ui.Pill(p.Error, ui.Err)))
	}
	lines = append(lines, kv(Tr("detail_file"), p.SourceFile), "")

	if p.Type == "thread" {
		for i, tweet := range p.Tweets {
			title := fmt.Sprintf("%d/%d", i+1, len(p.Tweets))
			if tweet.IsReply {
				title += " · " + Tr("detail_reply")
			}
			count := fmt.Sprintf("%d/280", tweet.CharCount())
			counter := ui.Pill(count+" ✓", ui.OK)
			if !tweet.IsValid() {
				counter = ui.Pill(count+" ✗ "+Tr("detail_too_long"), ui.Err)
			}
			lines = append(lines, ui.Divider(iw, title), counter)
			for _, l := range wrapLines(tweet.Content, iw-2) {
				lines = append(lines, "  "+l)
			}
			if tweet.Image != "" {
				lines = append(lines, dimStyle.Render("📎 "+tweet.Image))
			} else {
				lines = append(lines, dimStyle.Render("📎 "+Tr("detail_no_image")))
			}
			lines = append(lines, "")
		}
	} else {
		lines = append(lines, ui.Divider(iw, strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(Tr("editor_label_body")), ":"))))
		for _, l := range wrapLines(p.Body, iw-2) {
			lines = append(lines, "  "+l)
		}
		if len(p.Images) > 0 {
			lines = append(lines, "", ui.Divider(iw, Tr("panel_images")))
			for _, img := range p.Images {
				lines = append(lines, "📎 "+img)
			}
		}
	}
	return scrollPanel(w, h, fmt.Sprintf("%s · %s %s", Tr("panel_preview"), strings.ToUpper(p.Platform), strings.ToUpper(p.Language)),
		lines, m.detailScrollOffset, true)
}
