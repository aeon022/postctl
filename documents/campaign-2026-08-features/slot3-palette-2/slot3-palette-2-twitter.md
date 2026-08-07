---
platform: twitter
type: thread
campaign: missionctl-polish-2026-08
joplin_id: 899b0ced074d46b1a5526fe033007b15
updated_at: '2026-08-07T19:52:34.479Z'
---
mailctl and budgetctl just joined the `:` command palette rollout across missionctl.

Same pattern as k9s/lazygit: press `:`, type an action, enter. Live-filtered, prefix matches first. 🧵
---
budgetctl's version handles something the others don't: a freeform help paragraph (how accounts work) mixed with regular key/description rows. Needed a small addition to the shared help-builder package rather than special-casing it.
---
Bug caught live in tmux, not just in code review: the transaction list wasn't shrinking while the palette was open, so on a full terminal the palette's own input line got pushed off the top of the screen. Fixed by reserving the right amount of space up front.
---
Free & open source:
👉 https://github.com/aeon022/mailctl
👉 https://github.com/aeon022/budgetctl

#golang #tui #cli #bubbletea #opensource
