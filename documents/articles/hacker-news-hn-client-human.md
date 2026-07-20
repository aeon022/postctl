# Show HN: hn-client – Keyboard-driven Hacker News reader in Go

*Suggested Title:* Show HN: hn-client – Keyboard-driven Hacker News reader in Go
*Submission URL:* 
*GitHub Repository:* https://github.com/aeon022/hn-client

---

Hey HN,

I spend a lot of time reading Hacker News, but I hate when my browser gets cluttered with dozens of tabs just from comments. As someone who lives in the terminal, I wanted a fast, keyboard-driven way to browse HN without leaving the shell.

So I built `hn-client` — a terminal-native Hacker News client written in Go using the Bubble Tea (TUI framework) and Lipgloss (styling) libraries.

I wanted to focus on a few things that annoyed me about other terminal readers:

- **Color-Coded Comment Trees:** Nested comment threads are really hard to follow on a small terminal window. I added vertical, color-coded borders for different indentation levels. It makes tracing back a conversation much easier on the eyes.
- **Clean HTML-to-Terminal Parsing:** Raw HN comments contain raw HTML links, unformatted text, and messy code blocks. I wrote a sanitizer that cleans up the HTML and formats code snippets correctly on monospace grids.
- **Global Read History:** No one wants to click on the same story twice. The tool tracks your read stories in a local JSON file globally, grays them out in the feeds, and handles old history cleanup.
- **Interactive Real-Time Search:** Press `/` in the story list to filter the loaded feed instantly by title, author, or category.

It compiles to a single static binary and connects directly to the official Firebase API (with cookie-based HTML fallbacks for your own submissions so you can still manage them if they are flagged/dead).

I'd love to hear your feedback and suggestions on how to make TUI comment trees even more readable!

GitHub: https://github.com/aeon022/hn-client
