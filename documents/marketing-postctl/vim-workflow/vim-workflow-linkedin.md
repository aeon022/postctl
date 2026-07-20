---
platform: linkedin
campaign: vim-workflow
schedule: 2026-07-03 08:30:00
---
⌨️ Keyboard-Only: Writing Social Media Posts inside Neovim

Context switching is a productivity killer. As developers, we spend hours optimizing our terminal workflows, only to break our flow when we have to copy/paste text into a web-based social media dashboard.

With postctl, you can draft, edit, and queue posts without ever leaving your editor. 

Here is how the Neovim integration works:
1️⃣ While using the postctl TUI (`./postctl tui`), press `ctrl+v` in any form field.
2️⃣ The TUI suspends and launches your system editor (Vim/Neovim).
3️⃣ Write your post using standard Markdown. The file includes a dynamic helper header showing character rulers for the active platform (Twitter: 280, Bluesky: 300, Mastodon: 500).
4️⃣ Edit metadata (campaign, schedule, images) directly in the YAML frontmatter.
5️⃣ Save and exit (`:wq`). The TUI resumes instantly and syncs your text and metadata back into the database.

It’s social media management built for developers who live in the terminal.

👉 GitHub Repository: https://github.com/aeon022/postctl
🌐 Learn more: https://postctl.sh

#neovim #vim #terminal #developerproductivity #bubbletea #golang #opensource #commandline
