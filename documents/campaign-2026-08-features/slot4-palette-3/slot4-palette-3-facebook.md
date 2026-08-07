---
platform: facebook
type: single
campaign: missionctl-polish-2026-08
joplin_id: 96362732befd4c928793f3da2ddb0186
updated_at: '2026-08-07T19:52:34.559Z'
---
⌨️ Command Palette rollout: complete — 7 of 7 tools

timectl and diaryctl were the last two to get it. Every TUI in the missionctl suite (except habctl, where it started as a prototype, and postctl, which has its own design) now has a `:` command palette — press it, type an action by name, hit enter, live-filtered as you go.

The interesting part wasn't the feature itself, it's a well-known pattern from tools like k9s and lazygit. It's how we built it: one shared matching package instead of copying the logic into all 7 tools, and every single rollout tested live in a real terminal rather than trusting code review alone — which caught two genuine bugs that would've shipped invisibly otherwise.

👉 https://github.com/aeon022/missionctl
