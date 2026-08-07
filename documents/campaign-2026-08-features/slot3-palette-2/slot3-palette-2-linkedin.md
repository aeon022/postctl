---
platform: linkedin
type: single
campaign: missionctl-polish-2026-08
joplin_id: e50c378d335444df821a25dfe52116d9
updated_at: '2026-08-07T19:52:34.457Z'
---
⌨️ Command Palette rollout continues: mailctl and budgetctl

Same pattern as the previous round (calctl, taskctl, notectl): press `:`, type an action by name, enter. Live-filtered, prefix matches ranked first.

Two things worth sharing from this round specifically:

budgetctl's help screen mixes regular key/description rows with a freeform paragraph (explaining how accounts work in the tool). Rather than special-case it, we extended the shared help-builder package with a small `Text()` method for freeform lines — the abstraction earns its keep instead of being bent to fit one tool.

We also caught a real bug live, not in code review: on a full-height terminal, the transaction list wasn't shrinking to make room for the palette, so opening it could push the palette's own input line off the top of the screen. Testing this meant actually launching the TUI in a real terminal at real dimensions — the bug was invisible in the code and in unit tests alike.

That's 5 of 7 tools done. Two more this week.

👉 GitHub: https://github.com/aeon022/missionctl
🌐 Website: https://missionctl.sh

#golang #tui #cli #bubbletea #developertools #softwareengineering
