Show HN: missionctl – Local-first macOS TUI suite + MCP server to give AI agents 'hands'

Hi HN,

I built `missionctl` because I wanted my local AI agent (like Claude Desktop) to actually do things in my digital life, rather than just talk about them, without giving up my privacy or locking my data in third-party SaaS cloud platforms.

`missionctl` is a suite of six focused, terminal-native CLI and TUI tools written in Go. Each tool manages one domain of your life locally in SQLite caches, and exposes a standardized Model Context Protocol (MCP) server over stdio.

### The Tools:
1. **mailctl:** Apple Mail bridge (AppleScript). Syncs inbox, searches threads, drafts/sends emails from Markdown files.
2. **calctl:** Apple Calendar bridge (EventKit). Manages events, calculates free slots.
3. **taskctl:** Apple Reminders bridge (EventKit). Command-line task manager with background sync daemon.
4. **notectl:** Obsidian vault bridge. Full-text search, writes to notes, templates daily notes.
5. **budgetctl:** Bank CSV importer. Category rules, budget goals, recurring payment detection.
6. **postctl:** Multi-platform publisher (Twitter/X, LinkedIn, Threads, Mastodon, Bluesky, Facebook).

### Why local-first?
Most AI tools require syncing your mail, calendar, and financials to their cloud databases. `missionctl` keeps all credentials, SQLite databases, and caches on your local machine. Nothing is transmitted except when you explicitly publish (postctl) or send (mailctl). 

Every tool compiles to a single static Go binary with no runtime dependencies.

Once all six MCP servers are wired up to Claude Desktop, the AI gains 43 tools. You can ask Claude workflows like:
* *"Give me my morning briefing based on unread emails, calendar events, and tasks."*
* *"I just had a meeting, here are my notes. Create tasks, save the meeting note to Obsidian, and draft the follow-up email."*
* *"Analyze my transactions last month and flag recurring subscriptions."*

### Pricing / Monetization:
The binaries are open source. You can connect up to two networks/domains for free. A lifetime license for unlimited domains is $9 per tool, or $39 for the complete six-tool bundle on Polar.sh.

I'd love to hear your feedback on the architecture, local-first workflows, and what bridges/platforms you'd like to see supported next!

Project Repository: https://github.com/aeon022/missionctl
Website: https://postctl.sh
