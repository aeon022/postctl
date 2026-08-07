---
platform: facebook
type: single
campaign: missionctl-polish-2026-08
images:
  - campaign-2026-08-features/diaryctl-heatmap.png
joplin_id: ce33e19c10664f5fb5dabf3ba0463f17
updated_at: '2026-08-07T19:52:34.690Z'
---
🎨 diaryctl's start screen got a real facelift

diaryctl generates AI-narrated developer diary entries from your git history. Its start screen (a commit heatmap next to your recent entries) hadn't changed since launch.

What's new:
📊 GitHub-style heatmap — weeks run horizontally now instead of stacking as rows, so the same panel width covers ~13 weeks instead of 30 days, with a proper 5-tier color gradient and legend.
📐 Content-sized panels — they used to stretch to fill the whole terminal no matter how little was in them. Now they size to their content, and the extra space goes to a prominent "Today" summary and a new Recent Entries digest.
⏳ A real loading spinner — diaryctl was missing one entirely.
🐛 A real bug fix — scrolling down in the entry view had no upper bound and would slowly empty the panel out past the end of the content. Caught by actually scrolling through it live, not just reading the code.

Screenshot: the new start screen in action.

👉 https://github.com/aeon022/diaryctl
