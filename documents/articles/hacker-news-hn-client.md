# Show HN: hn-client – Keyboard-driven Hacker News reader with color-coded comments

*Suggested Title:* Show HN: hn-client – Keyboard-driven Hacker News reader with color-coded comments
*Submission URL:* 
*GitHub Repository:* https://github.com/aeon022/hn-client

---

Hi HN,

I spend a lot of time reading Hacker News, but I hate when my browser accumulates dozens of tabs just from comments and story links. As someone who spends most of my day in the terminal, I wanted a native, fast, keyboard-driven way to browse HN without context-switching to a heavy web browser.

So I built `hn-client` — a terminal-native Hacker News client written in Go using the **Bubble Tea** (TUI framework) and **Lipgloss** (styling) libraries.

Here is why it's different from standard terminal web scrapers:

### 1. Color-Coded Comment Threads
HN comment trees can get deep and hard to follow on small screens. `hn-client` renders nested comments with vertical, color-coded borders depending on their indentation level. This makes tracing back a conversation thread extremely easy on the eyes.

### 2. Clean HTML-to-Terminal Parsing
Raw HN comments contain raw HTML elements, unformatted URLs, and messy preformatted code blocks. The client parses and sanitizes comments:
- Translates HTML links (`<a>`) into readable text with underlined terminal links.
- Cleans and standardizes code blocks (`<code>` and `<pre>`) so code snippets format properly on monospace grids.
- Decodes HTML entities (like `&quot;` and `&#x27;`) on the fly.

### 3. Native Vim-Style Nav & Mouse Wheel Hijack
You can navigate using standard `j`/`k` keys, the arrow keys, or your mouse scroll wheel. The list views and comment viewports support wheel scrolling directly. We also fixed scroll drift and viewport height calculations, so the view remains stable as you traverse long comment threads.

### 4. Global Read History
No one wants to click on the same story twice. The tool tracks your read stories in `~/.hn-history.json` globally. Read stories are grayed out in the story feeds, and the client automatically handles file rotation and cleaning of old history entries.

### 5. Interactive Real-Time Search
Press `/` in the story list to filter the loaded feed in real-time. You can filter by story titles, authors, or categories (Top, New, Best, Ask, Show).

### Under the hood
- **Go / Golang:** Compiles to a single static binary. Zero dependencies at runtime.
- **Official HN API:** Integrates with the official Hacker News Firebase API.
- **Robust timeouts:** A 10-second timeout on all HTTP requests prevents the TUI from freezing during network drops.

I’d love to hear your feedback, bug reports, and suggestions for additional features!

GitHub: https://github.com/aeon022/hn-client
