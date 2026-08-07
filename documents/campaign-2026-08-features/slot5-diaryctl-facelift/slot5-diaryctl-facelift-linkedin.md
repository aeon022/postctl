---
platform: linkedin
type: single
campaign: missionctl-polish-2026-08
images:
  - campaign-2026-08-features/diaryctl-heatmap.png
joplin_id: fe47e762f80d460d8970388a262b2e07
updated_at: '2026-08-07T19:52:34.701Z'
---
🎨 diaryctl's start screen got a real facelift

diaryctl generates AI-narrated developer diary entries from your git history. Its start screen — a commit heatmap next to a recent-entries list — hadn't been touched since launch, and it showed.

What changed:

📊 Heatmap redesign — the old version was a flat 30-day grid with 4 color tiers and no legend. It's now GitHub-contribution-graph style: weeks run horizontally instead of stacking as rows, so the same fixed panel width now covers ~13 weeks instead of 30 days, with a proper 5-tier gradient and legend.

📐 Content-sized panels — at a 120×40 terminal, the old panels stretched to ~34 rows regardless of how little content existed in them. They now size to their actual content. The reclaimed space goes to a prominent "Today" summary (pulled out of where it used to be buried) and a new Recent Entries digest.

⏳ Loading state — diaryctl was the one tool in the suite without a spinner on initial load; every other tool got one in an earlier polish round. Fixed.

🐛 A real bug, not just polish — scrolling down in the entry detail view had no upper bound. Scroll far enough past the end and the panel would visibly empty out instead of just stopping at the last page. Caught by actually scrolling 100 times in a live terminal session, not by reading the code.

Screenshot: the new start screen, full heatmap and all.

👉 GitHub: https://github.com/aeon022/diaryctl
🌐 Suite: https://missionctl.sh

#golang #tui #cli #bubbletea #uidesign #softwareengineering #opensource
