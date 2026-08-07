---
platform: linkedin
type: single
campaign: missionctl-polish-2026-08
joplin_id: e951117fc9984ab5bd541dcaaff0cc9b
updated_at: '2026-08-07T19:52:34.570Z'
---
⌨️ Command Palette rollout: complete — 7 of 7 tools

timectl and diaryctl were the last two. Every TUI in the missionctl suite (except habctl, where the pattern started as a prototype, and postctl, which has its own design language) now has a `:` command palette: press it, type an action by name, enter — live-filtered, prefix matches ranked first.

What made this rollout worth writing about wasn't the feature itself — it's a well-known pattern from k9s and lazygit. It's the process:

→ One shared matching implementation (`missionctl-core/palette`), not seven copies
→ Every single-key shortcut already goes through one handler per tool; the palette just replays the same keypress through it, so there's no separate action-dispatch logic that could drift out of sync
→ Each rollout got tested live in a real terminal at real dimensions, not just reviewed as code — and that caught two genuine overflow bugs (one in taskctl-style tools, one specific to how calctl windows its event list) that would have shipped invisibly otherwise

Small feature, but a good reminder that "does it compile and pass unit tests" and "does it actually look right on screen" are two different questions.

👉 GitHub: https://github.com/aeon022/missionctl
🌐 Website: https://missionctl.sh

#golang #tui #cli #bubbletea #developertools #softwareengineering #testing
