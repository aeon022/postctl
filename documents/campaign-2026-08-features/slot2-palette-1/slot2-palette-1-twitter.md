---
platform: twitter
type: thread
campaign: missionctl-polish-2026-08
images:
  - campaign-2026-08-features/taskctl-palette.png
joplin_id: 97f8f6b5458e496dbd6942d0338eb8e0
updated_at: '2026-08-07T19:52:34.377Z'
---
New in calctl, taskctl & notectl: a `:` command palette, like k9s or lazygit.

Instead of memorizing single-key shortcuts across a dozen views, press `:`, type what you want ("new", "sync", "filter"), hit enter. Live-filtered as you type, prefix matches first. 🧵
---
It reuses the exact same key handling every shortcut already goes through — so behavior is guaranteed identical to pressing the key directly. No separate action-dispatch logic to drift out of sync.

Screenshot: taskctl's palette mid-filter. 📸
---
Same matching logic (prefix-then-contains, fzf-style) now lives in a shared `missionctl-core/palette` package — one implementation, not copy-pasted seven times across the suite.
---
Free & open source:
👉 https://github.com/aeon022/calctl
👉 https://github.com/aeon022/taskctl
👉 https://github.com/aeon022/notectl

#golang #tui #cli #bubbletea #opensource
