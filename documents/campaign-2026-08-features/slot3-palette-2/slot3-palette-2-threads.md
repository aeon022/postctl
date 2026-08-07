---
platform: threads
type: single
campaign: missionctl-polish-2026-08
joplin_id: 1991978f72c14e558e0577690a0a9af5
updated_at: '2026-08-07T19:52:34.468Z'
---
⌨️ mailctl & budgetctl just got the `:` command palette too.

Press `:`, type an action, enter — live-filtered, prefix matches first.

Caught a real bug shipping this one: the list wasn't shrinking for the palette, so on a full terminal the input line got pushed off-screen. Fixed. Live-tested this time, not just code review.

👉 https://github.com/aeon022/missionctl
