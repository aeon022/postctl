---
platform: threads
campaign: hn-client-deepdive
schedule: 2026-07-06 09:00
joplin_id: 4eada529b5b9465e8987b0c401e82fa9
updated_at: '2026-08-07T19:27:10.156Z'
---
How to browse Hacker News inside your terminal! 🖥️

hn-client uses Go's `os/exec` and Bubble Tea's `tea.ExecProcess` to suspend the TUI and hand control to text browsers like `w3m` or `lynx`. When you close the browser, the TUI resumes seamlessly.

No GUI context-switching required!

Repo: https://github.com/aeon022/hn-client
