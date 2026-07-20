---
platform: threads
campaign: hn-client-deepdive
schedule: 2026-07-06 09:00
---
How to browse Hacker News inside your terminal! 🖥️

hn-client uses Go's `os/exec` and Bubble Tea's `tea.ExecProcess` to suspend the TUI and hand control to text browsers like `w3m` or `lynx`. When you close the browser, the TUI resumes seamlessly.

No GUI context-switching required!

Repo: https://github.com/aeon022/hn-client
