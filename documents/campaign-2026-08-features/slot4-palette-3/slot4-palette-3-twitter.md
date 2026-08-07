---
platform: twitter
type: thread
campaign: missionctl-polish-2026-08
joplin_id: 1b8ab4877f6f4e598d0b003f7df8630e
updated_at: '2026-08-07T19:52:34.594Z'
---
Last two: timectl and diaryctl now have the `:` command palette. That's 7 of 7 tools done. 🧵
---
Started as a prototype in habctl, rolled out one tool at a time across the whole suite, matching logic centralized in a shared `missionctl-core/palette` package instead of copy-pasting it seven times.
---
Every rollout got tested live in a real terminal (tmux), not just code review — caught real overflow bugs along the way that unit tests alone wouldn't have surfaced.
---
Free & open source:
👉 https://github.com/aeon022/timectl
👉 https://github.com/aeon022/diaryctl
👉 https://github.com/aeon022/missionctl

#golang #tui #cli #bubbletea #opensource
