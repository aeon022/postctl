---
platform: twitter
type: thread
campaign: hn-client-deepdive
schedule: 2026-07-06 09:00
---
How do you open web links from a Terminal User Interface (TUI) without breaking your command-line flow?

In `hn-client`, we integrated text-based browsers like `w3m` and `lynx` directly into our Go-based Hacker News client. Here is how it works under the hood. 🧵
---
Traditional TUI programs open links in your desktop browser, forcing you to leave your terminal.

With `hn-client`, pressing `o` on a story suspends the Bubble Tea TUI, executes `w3m <url>` in the foreground, and hands control over to the terminal browser!
---
When you exit `w3m` (by pressing `q`), `hn-client` instantly resumes its UI states and refreshes the screen seamlessly.

This is made possible using Go's `os/exec` package and Bubble Tea's `tea.ExecProcess` command framework.
---
Keep your focus in your terminal. Check out the project, download the binary, or contribute:
👉 https://github.com/aeon022/hn-client
