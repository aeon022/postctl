# Show HN: postctl – A local-first, markdown-native social media manager in Go

*Suggested Title:* Show HN: postctl – A local-first, markdown-native social media manager in Go
*Submission URL:* https://postctl.sh
*GitHub Repository:* https://github.com/aeon022/postctl

---

Hi HN,

I built `postctl` because I got tired of using clunky, heavy web dashboards just to schedule tweets and LinkedIn updates. As developers, we version-control our code, write in Markdown, and operate in the terminal. Why should our social media pipeline be any different?

`postctl` is a terminal-native CLI and TUI tool written in Go that treats your postings as code.

### How it works
1. **Write in Markdown:** Author posts as `.md` files. Frontmatter defines platforms (Twitter, LinkedIn, Threads, Bluesky, Facebook), campaign names, schedules, and local image paths.
2. **Import & Validate:** Run `./postctl import ./posts/`. The tool parses files, validates platform character limits (e.g. 280 for Twitter) and image existence, and saves them to a local SQLite DB.
3. **Interactive TUI:** Run `./postctl tui` to browse drafts, view calendar schedules, edit metadata on the fly (suspends to Vim/Neovim on `ctrl+v`), and trigger publication.
4. **Headless background daemon:** Run `./postctl daemon` in the background on your Mac, or cross-compile it for Linux (`GOOS=linux`) and run it on a $4/mo VPS or Raspberry Pi for 24/7 scheduler uptime.

### The "AI-as-Operator" Principle
Most modern tools focus on "AI content generation" (writing captions for you). `postctl` treats AI as the *operator* of the tool. 
Every command supports `--format json` and `--dry-run`. This enables local terminal AI agents (like Claude/GPT in command-line sidecars) to:
- Generate markdown draft files based on code changes or release notes.
- Run `postctl import` and review dry-run validation results as JSON.
- Present the dry-run output to the human for approval.
- Execute the final publish command non-interactively.

### Technical highlights
- **Go + SQLite:** Single-binary, offline-first. Uses a CGO-free Go SQLite driver (`modernc.org/sqlite`).
- **AES-256 Backups:** Syncs configuration and DB files securely by exporting them as encrypted blobs (`postctl config export -o backup.bin`) utilizing AES-256-GCM.
- **Headless Chrome Evasion Fallback:** When X/Twitter rate-limits or blocks direct GraphQL posting requests, `postctl` automatically falls back to a headless Chrome instance (via `chromedp`), injects your browser session cookies, populates the composer UI, and clicks post.

It is open-source (MIT licensed) and free for up to 2 social networks. 

I'd love to hear your feedback, thoughts on the architecture, and what features you'd like to see next on the roadmap!

GitHub: https://github.com/aeon022/postctl
Landing Page: https://postctl.sh
