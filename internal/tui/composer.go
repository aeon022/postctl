package tui

import (
	"regexp"
	"strings"
)

// Multi-platform composer: one text, several platforms. adaptCaption tailors the text to a
// platform's habits before saving, and the platforms are staggered in time like the manual plan.

// maxTrailingTags is how many hashtags of the closing hashtag block a platform keeps (-1 = all).
var maxTrailingTags = map[string]int{
	"instagram": -1,
	"threads":   1, // one topic tag per post is what Threads uses
	"facebook":  3,
	"linkedin":  3,
	"twitter":   2,
	"bluesky":   2,
	"mastodon":  -1,
}

const staggerMinutes = 30 // gap between platforms when several are scheduled at once

var tagOnlyLine = regexp.MustCompile(`^\s*(#[\p{L}\p{N}_]+\s*)+$`)

// adaptCaption trims the last line when it is a pure hashtag block to the platform's limit.
// Text without such a block is returned unchanged.
func adaptCaption(platform, body string) string {
	keep, ok := maxTrailingTags[platform]
	if !ok || keep < 0 {
		return body
	}
	lines := strings.Split(strings.TrimRight(body, "\n "), "\n")
	last := len(lines) - 1
	if last < 0 || !tagOnlyLine.MatchString(lines[last]) {
		return body
	}
	tags := strings.Fields(lines[last])
	if len(tags) <= keep {
		return body
	}
	lines[last] = strings.Join(tags[:keep], " ")
	return strings.Join(lines, "\n")
}

// targetPlatforms are the selected platforms in selector order (or just the highlighted one).
func (m Model) targetPlatforms() []string {
	var out []string
	for _, p := range editorPlatforms {
		if m.editorTargets[p] {
			out = append(out, p)
		}
	}
	if len(out) == 0 || m.editorPostID != "" {
		return []string{m.editorPlatform}
	}
	return out
}
