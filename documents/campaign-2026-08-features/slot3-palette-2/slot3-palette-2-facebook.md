---
platform: facebook
type: single
campaign: missionctl-polish-2026-08
joplin_id: 8810811cfedb4dc7a3406d5ed3b2f84c
updated_at: '2026-08-07T19:52:34.446Z'
---
⌨️ Command Palette rollout continues: mailctl and budgetctl

Same as the last round — press `:`, type an action by name, hit enter. Live-filtered as you type.

budgetctl's help screen needed a small extension to our shared help-builder package to handle a freeform explanatory paragraph alongside the normal key list. We also caught and fixed a real bug: on a full-height terminal, the transaction list wasn't shrinking to make room for the palette, so the palette's own input line could get pushed off the top of the screen. Found by actually running the app in a real terminal, not just reading the code.

5 of 7 tools done, two more this week.

👉 https://github.com/aeon022/missionctl
