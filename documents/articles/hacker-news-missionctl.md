---
joplin_id: 9f48ce77dc3a45d482741fcdc26c603b
updated_at: '2026-08-07T19:27:09.574Z'
---
# Show HN: missionctl – 9 local-first terminal tools that give AI agents hands (Go, MCP)

*Suggested Title:* Show HN: missionctl – 9 local-first terminal tools that give AI agents hands (Go, MCP)
*Submission URL:* https://missionctl.sh
*GitHub Repository:* https://github.com/aeon022/missionctl

---

Hey HN,

I got tired of the pattern where "connecting AI to your daily life" means handing a cloud SaaS your email, calendar, and financial data just so an assistant can read it. So I built `missionctl` instead: 9 local-first, terminal-native Go tools — mail, calendar, tasks, notes, budget, habits, time tracking, a dev diary, and social post scheduling — each one a single static binary with its own local SQLite cache and its own MCP server.

Point Claude Desktop (or any MCP client) at all 9 and it gets 66 tools total, and can genuinely orchestrate across them — "what's on my calendar tomorrow and do I have overdue tasks" hits two different tools automatically, no explicit selection needed.

### How each tool is built
Same shape every time: read from the real source (Apple Mail/Calendar/Reminders via AppleScript+EventKit, an Obsidian vault, a bank CSV) into a local SQLite cache, expose it through a Bubble Tea TUI and a plain CLI, and start an MCP server (`<tool> mcp`) that lets an AI agent call the exact same operations. Nothing round-trips through a cloud service — sync is local, reads are local, the MCP server just talks stdio to Claude.

### What's new this month
This was a deliberate polish pass, not new tools:
- **A `:` command palette** rolled out to all 7 tools that didn't have it yet — press `:`, type an action by name, live-filtered, Enter to run. Prototyped in one tool first, then generalized into a shared matching package (`missionctl-core/palette`) instead of copy-pasting it 7 times.
- **Real binary downloads.** Until now the only install path was "clone + `go build`." Now there's a GoReleaser pipeline publishing to GitHub Releases and a Homebrew tap: `brew install aeon022/tap/<tool>`.
- **A few real bugs**, found by actually running the TUIs in real terminal sizes rather than trusting unit tests alone — two separate palette overflow bugs and an unbounded-scroll bug in one tool's detail view.
- **Cross-device sync**, done carefully: point a tool's SQLite database at a folder you already sync yourself (iCloud Drive, Dropbox, Syncthing) and it automatically switches journal modes (WAL splits state across up to 3 files a sync client has no idea must land together — rollback-journal doesn't), takes a same-machine advisory lock, and detects iCloud's zero-byte eviction placeholders instead of silently opening an empty database.

### Pricing, honestly
All 9 tools are MIT licensed and free — full stop, no feature you already have gets taken away. Two specific things (AI transaction categorization in the budget tool, and using more than one Obsidian vault in the notes tool) are part of an optional lifetime Bundle, because they either cost real inference or add real complexity. Everything else, including cross-device sync and the full CLI/TUI/MCP surface of every tool, stays free forever.

GitHub: https://github.com/aeon022/missionctl
Site: https://missionctl.sh

Happy to go deep on the sync-safety engineering or the MCP tool design in the comments — genuinely curious what people would want wired up next.
