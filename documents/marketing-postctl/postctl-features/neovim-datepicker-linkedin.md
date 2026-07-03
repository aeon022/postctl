---
platform: linkedin
campaign: postctl-features
schedule: 2026-07-05 10:00
---
⌨️ Keyboard-Only Social Media: Writing inside Neovim & TUI Datepicker

Context switching is a productivity killer. As developers, we optimize our terminal workflows, only to break our flow when we have to copy/paste text into a web-based social media dashboard.

With postctl, you can draft, edit, and queue posts without ever leaving your editor. 

Here is how the Neovim integration works:
1️⃣ Suspend and Launch: While using the postctl TUI (`./postctl tui`), press `ctrl+v` in any form field to spawn your system editor (Vim/Neovim).
2️⃣ Live Character Rulers: Write your post in Markdown. The file includes a dynamic helper header showing character rulers for the active platform.
3️⃣ YAML Frontmatter Sync: Edit metadata directly in the YAML header. Save and exit (`:wq`) to sync your text and metadata back into the TUI.
4️⃣ Terminal Calendar: Focus the schedule field and press `ctrl+d` to open a terminal datepicker. Navigate with hjkl or arrows, and hit enter to set the time.

It’s social media management built for developers who live in the terminal.

👉 GitHub Repository: https://github.com/aeon022/postctl
🌐 Learn more: https://postctl.sh

#neovim #vim #terminal #developerproductivity #bubbletea #golang #opensource
