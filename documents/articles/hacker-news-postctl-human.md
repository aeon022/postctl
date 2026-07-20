# Show HN: postctl – Local-first, markdown-native social media scheduler in Go

*Suggested Title:* Show HN: postctl – Local-first, markdown-native social media scheduler in Go
*Submission URL:* https://postctl.sh
*GitHub Repository:* https://github.com/aeon022/postctl

---

Hey HN,

I built `postctl` because I got tired of using heavy, clunky web dashboards just to schedule a few tweets or LinkedIn updates. Since we write code in Markdown and version-control everything, I thought: why not treat our social media pipeline the same way?

`postctl` is a terminal-native CLI and TUI tool in Go that treats your postings as code.

The workflow is simple:
1. **Write in Markdown:** You write your posts as local Markdown files. The frontmatter defines the platform, campaign, and local image paths.
2. **Import & Validate:** Running `postctl import` parses the files, validates character limits (e.g. 280 for Twitter), and saves them to a local SQLite database.
3. **Interactive TUI:** You can run `postctl tui` to browse drafts, view the calendar schedule, and edit metadata on the fly (pressing `ctrl+v` suspends to Vim/Neovim).
4. **Headless background daemon:** Running `postctl daemon` in the background on your machine or VPS dispatches the posts at their scheduled times.

One feature I spent some time on is Chrome headless evasion. When Twitter blocks direct API posting requests, `postctl` automatically falls back to a headless Chrome instance (via `chromedp`), injects your browser session cookies, populates the composer, and clicks post. It's a bit of a hack, but it works surprisingly well.

Every command also supports `--format json` and `--dry-run`, making it easy to integrate with local terminal AI assistants or scripts to generate and import drafts.

It's open source (MIT licensed) and free for up to 2 social networks. I'd love to hear your thoughts on the workflow!

GitHub: https://github.com/aeon022/postctl
Site: https://postctl.sh
