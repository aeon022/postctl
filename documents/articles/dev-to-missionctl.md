---
title: Giving Claude Hands, Locally — Building a 9-Tool MCP Suite in Go
published: false
description: 9 local-first CLI/TUI tools, 66 MCP tools, one shared core package. How the architecture, the command palette rollout, and the cross-device sync safety actually work.
tags: go, mcp, opensource, showdev
joplin_id: a42593c3293a4938a38b1b7e948e2f97
updated_at: '2026-08-07T19:27:09.531Z'
---
# Giving Claude Hands, Locally — Building a 9-Tool MCP Suite in Go

Most "AI can act on your life" tools work the same way: your email, your calendar, your bank transactions go up to a cloud service, and an AI reads them from there. That's a reasonable trade for a lot of products. It's also not one I wanted to make for my own inbox, calendar, or budget.

So `missionctl` takes the opposite shape: 9 small Go tools — `mailctl`, `calctl`, `taskctl`, `notectl`, `budgetctl`, `habctl`, `timectl`, `diaryctl`, `postctl` — each a single static binary, each with its own local SQLite cache, each exposing an MCP server over stdio. Claude Desktop (or any MCP client) gets 66 tools total. Nothing round-trips through anything I don't run myself.

## The shape every tool follows

Every tool is the same four pieces:

1. **A local source of truth.** Apple Mail/Calendar/Reminders via AppleScript and EventKit, an Obsidian vault read straight off disk, a bank CSV you import by hand. No tool invents its own cloud account.
2. **A local SQLite cache.** `<tool> sync` pulls from the real source into `~/.local/share/<tool>/`. Every other command reads from the cache — fast, and it works offline.
3. **A Bubble Tea TUI and a plain CLI.** Same operations, two interfaces. `calctl` opens a week view; `calctl list --from 2026-08-01` prints JSON-friendly output for scripts.
4. **An MCP server.** `<tool> mcp` starts a stdio server exposing the same operations as typed tools with descriptions — `find_free_slots`, `create_task`, `detect_recurring_payments`. Claude reads the descriptions and picks the right tool across the right app; a prompt like *"what's on my calendar tomorrow, and do I have overdue tasks"* fans out to `calctl` and `taskctl` automatically, with no explicit tool selection in the prompt.

## Not duplicating the same fix 7 times

The suite started as 7 independent tools that grew independently, which is exactly how you end up implementing the same feature 7 slightly-different ways. Two examples from a recent polish pass fixed that pattern directly.

**The command palette.** One tool (`habctl`) had a `:`-triggered command palette — press `:`, type an action by name, live-filtered, Enter replays the matching keypress. It's the k9s/lazygit pattern, and it's a genuinely nice way to discover functionality without memorizing single-key shortcuts. Instead of copy-pasting the matching logic into the other 7 tools, it became a real shared package:

```go
// missionctl-core/palette
type Command struct {
    Name string
    Desc string
    Key  string
}
func Match(cmds []Command, query string) []Command // prefix-then-contains
```

Every tool's palette is now: a list of `Command` structs, a call to `Match`, and a render loop. One implementation, one test suite, seven call sites.

**Cross-device sync.** Pointing a tool's SQLite file at an iCloud Drive or Dropbox folder sounds simple and isn't, for three specific reasons — and getting any one of them wrong means silent data loss, not a crash:

- **WAL journal mode splits state across up to three files** (`.db`, `.db-wal`, `.db-shm`). A sync client uploads whichever one changed, whenever — with zero cross-file atomicity guarantee. `missionctl-core/syncdir` switches a tool to classic rollback-journal mode the moment it's pointed at a user-configured directory: one main file, plus a `-journal` sidecar that exists only mid-transaction and is gone the instant a commit finishes. The private default directory stays on WAL, untouched — this only kicks in once you opt in.
- **Two processes on one machine** (two terminal tabs, a crashed session that never let go of its handle) must not write concurrently. An advisory `flock` enforces that, and — because it's a kernel-held lock, not an app-managed one — releases itself automatically the moment the holding process exits, however it exits. No stale-lock cleanup logic needed.
- **macOS can evict an iCloud Drive file** to free local disk space, replacing it with a zero-byte placeholder. Every tool checks for that placeholder and forces a re-download before touching the file, instead of opening what looks like a valid-but-empty database.

```go
// missionctl-core/config
func ResolveDir(tool, override string) (dir string, shared bool) {
    if override == "" {
        return DataDir(tool), false // private default, WAL is fine
    }
    // user-configured — safe to treat as possibly synced
    return expandedOverride, true
}
```

Every tool calls the same `ResolveDir` + `syncdir.JournalMode(shared)` pair. `data_dir` in the tool's config (or a `<TOOL>_DATA_DIR` env var) opts a single tool into it; the private default is unaffected.

## Real bugs, found by actually running the thing

Unit tests caught plenty, but two overflow bugs and an unbounded-scroll bug only showed up by literally launching each TUI in a real terminal at a real size (`tmux` scripted sessions, in this case) and looking at the rendered output. One was a windowing bug that counted *rows* when it needed to count *physical lines* — a header costs 2 lines, an event costs 1, and a view sized for N rows overflowed the moment headers were involved. No amount of testing the row-count logic in isolation would have caught that; it only shows up once you render it.

## Pricing, since it comes up

All 9 tools are MIT licensed and free — that's not a trial. Two specific features (AI transaction categorization in the budget tool, using more than one Obsidian vault in the notes tool) sit behind an optional lifetime Bundle, because they either cost real inference or add real complexity; everything else, sync included, stays free indefinitely. The license check is the same real Polar.sh license-key flow across every gated tool — no telemetry, just a key you can activate or leave alone.

## Try it

```bash
brew tap aeon022/tap https://github.com/aeon022/homebrew-tap
brew install aeon022/tap/calctl   # or any of: mailctl taskctl notectl budgetctl diaryctl timectl habctl
```

- **Site:** https://missionctl.sh
- **Docs:** https://missionctl.sh/docs
- **GitHub:** https://github.com/aeon022/missionctl

Curious what other local-first + MCP combinations people are building — drop a comment if you're doing something similar.
