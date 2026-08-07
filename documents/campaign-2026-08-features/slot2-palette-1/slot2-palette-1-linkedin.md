---
platform: linkedin
type: single
campaign: missionctl-polish-2026-08
images:
  - campaign-2026-08-features/taskctl-palette.png
  - campaign-2026-08-features/calctl-palette.png
  - campaign-2026-08-features/notectl-palette.png
joplin_id: f10c14389a484210ae5d7ff65ff57aeb
updated_at: '2026-08-07T19:52:34.352Z'
---
⌨️ Command Palettes are coming to the missionctl suite

Terminal power users know the pattern from k9s and lazygit: press a single key, type what you want to do, hit enter. No more memorizing which of twenty single-letter shortcuts does what.

We just rolled this out to calctl, taskctl, and notectl (four more tools follow this week):

🔹 Press `:` to open the palette
🔹 Type a few letters — matches filter live, prefix matches ranked first
🔹 ↑/↓ to select, Enter to run

The implementation detail I'm proud of: the palette doesn't have its own action-dispatch logic. It replays the exact keypress the shortcut already maps to, through the same handler every normal keystroke goes through. That means the palette can never drift out of sync with what the shortcuts actually do — there's only one source of truth.

The matching logic (prefix-then-contains ranking, same idea fzf and most fuzzy-finders use) now lives in a small shared package, `missionctl-core/palette`, instead of being copy-pasted into every tool.

We also caught two real overflow bugs while rolling this out and testing live in real terminals rather than trusting the code alone — worth doing if you're shipping anything with fixed-height terminal layouts.

📸 Screenshots: taskctl, calctl, and notectl's palettes in action.

👉 GitHub: https://github.com/aeon022/missionctl
🌐 Website: https://missionctl.sh

#golang #tui #cli #bubbletea #developertools #opensource #softwareengineering
