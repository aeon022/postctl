---
platform: twitter
type: thread
campaign: missionctl-polish-2026-08
images:
  - campaign-2026-08-features/diaryctl-heatmap.png
joplin_id: 645de32c1c814eb0b1be21a853a3878e
updated_at: '2026-08-07T19:52:34.726Z'
---
Gave diaryctl's start screen a real facelift. New heatmap, tighter layout, an actual loading state. Screenshot below. 🧵
---
The heatmap used to be a flat 30-day grid with 4 color tiers and no legend. Now it's GitHub-contribution-graph style: weeks run horizontally instead of stacking as rows, so the same panel width shows ~13 weeks instead of 30 days — 5-tier gradient, legend included.
---
Panels used to stretch to fill the whole terminal regardless of content — at 120×40 that meant ~34-row boxes for a 9-line heatmap. Now they size to their content, with a "Today" summary and Recent Entries digest filling the space usefully.
---
Also fixed a real bug: scrolling down in the entry detail view had no upper bound, so scrolling past the end slowly emptied the panel instead of just stopping. Caught live in tmux, not in review.
---
👉 https://github.com/aeon022/diaryctl

#golang #tui #cli #bubbletea #opensource #uidesign
