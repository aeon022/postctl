---
title: "missionctl: Giving Local AI Agents 'Hands' via Monospace CLI/TUI Tools"
published: false
description: Stop building complex cloud integrations for AI. Give Claude/GPT access to your local email, calendar, tasks, notes, and budget via lightweight, local-first Go binaries + MCP.
tags: go, open-source, mcp, ai
---

# missionctl: Giving Local AI Agents 'Hands' via Monospace CLI/TUI Tools

AI assistants are incredibly smart. They can summarize books, draft code, and plan marketing strategies. But when it comes to *actually doing* things, they are completely helpless. 

They can't read your local email. They can't schedule an event in your Apple Calendar. They can't add a task to your Apple Reminders list or check if you're over budget for groceries this month.

To bridge this gap, most modern tools try to build complex cloud sync databases, locking your private data into their SaaS platforms.

I wanted a different path: a local-first, privacy-respecting, terminal-native suite that turns your local applications into a scriptable, AI-accessible interface.

Meet **missionctl** — a suite of six focused CLI/TUI tools for macOS that manages your digital life, stores everything locally in SQLite, and exposes a clean Model Context Protocol (MCP) server so local AI agents can interact with your real data.

---

## 🛠️ The Architecture: Monospace & Local-First

Each tool in `missionctl` is written in Go and compiles to a single, static binary with no runtime dependencies. 

Every tool has four layers:
1. **Bubble Tea TUI**: A beautiful keyboard-driven terminal dashboard for when you want to use it manually.
2. **Cobra CLI**: A scriptable command-line interface for shell scripting.
3. **JSON Output**: Every read command supports `--json` so tools can easily pipe data to other programs.
4. **MCP Server**: Implements the Model Context Protocol (over stdio) so AI clients like Claude Desktop can read and write data directly.

Here is how the data flows:
```
Apple Mail / Calendar / Reminders
        │ AppleScript / EventKit
        ▼
   Local SQLite cache  ◄──── Bank CSV (budgetctl)
        │                    Obsidian vault (notectl)
   ┌────┴──────────┐         Markdown files (postctl)
   │               │
  TUI           MCP server (stdio)
(Bubble Tea)        │
                    ▼
             Claude Desktop /
             any MCP client
```

Each TUI and MCP server share the same SQLite database via WAL mode, letting both run simultaneously without lockups.

---

## 📦 The Tool Suite

Here are the six standalone tools that make up the suite:

### 1. `mailctl` — Email from the Terminal
*   **Bridge:** Apple Mail via AppleScript.
*   **What it does:** Syncs your inbox locally, lets you search messages, and compose/reply from Markdown files.
*   **MCP Tools:** `inbox`, `search_email`, `email_thread`, `send_email`, `draft_email`, `sync_inbox`.

### 2. `calctl` — Calendar from the Terminal
*   **Bridge:** Apple Calendar via Swift EventKit.
*   **What it does:** Lists events, creates appointments, and calculates free slots within working hours.
*   **MCP Tools:** `list_events`, `today`, `this_week`, `sync`, `find_free_slots`, `create_event`, `delete_event`.

### 3. `taskctl` — Tasks from the Terminal
*   **Bridge:** Apple Reminders via Swift EventKit.
*   **What it does:** Manages tasks and syncs with Apple Reminders. Includes a background sync daemon.
*   **MCP Tools:** `today_tasks`, `week_tasks`, `list_tasks`, `sync`, `create_task`, `complete_task`, `delete_task`.

### 4. `notectl` — Notes from the Terminal
*   **Bridge:** Local Markdown files (Obsidian/Logseq vaults).
*   **What it does:** Indexes your Obsidian vault, creates daily notes, writes to sections, and searches notes.
*   **MCP Tools:** `list_notes`, `read_note`, `write_note`, `search_notes`, `sync_notes`, `get_daily_note`, `append_daily_note`.

### 5. `budgetctl` — Budget from the Terminal
*   **Bridge:** Bank CSV exports.
*   **What it does:** Imports bank exports, categorizes transactions, sets monthly category goals, and detects recurring subscriptions.
*   **MCP Tools:** `list_transactions`, `budget_summary`, `import_transactions`, `tag_transactions`, `apply_category_rules`, `list_budget_goals`, `set_budget_goal`, `delete_budget_goal`, `detect_recurring_payments`.

### 6. `postctl` — Social Media from the Terminal
*   **Bridge:** Local SQLite + social network APIs.
*   **What it does:** Markdown-first scheduler and publisher for Twitter/X, LinkedIn, Threads, Bluesky, Mastodon, and Facebook.
*   **MCP Tools:** `list_posts`, `get_post`, `create_post`, `publish_post`, `schedule_post`, `list_campaigns`, `get_campaign`.

---

## 🤖 AI Workflows in Action

Once you configure all six servers in Claude Desktop, the AI gains 43 specialized tools. Here are a few workflows that become possible:

### Daily Briefing
> "What's on my plate today?"
1. Claude calls `calctl today`
2. Claude calls `taskctl today_tasks`
3. Claude calls `mailctl inbox --unread`
4. Claude compiles and renders a natural-language briefing.

### Meeting Capture
> "I just had a product meeting. Here are my notes: [paste]. Create tasks for all action items, write a meeting note to my vault, and draft the follow-up email."
1. Claude creates tasks using `taskctl create_task`
2. Claude writes a structured meeting note to Obsidian using `notectl write_note`
3. Claude drafts the follow-up email in Apple Mail using `mailctl draft_email`.

### Financial Review
> "Analyze my spending last month, compare it to my goals, and highlight any subscriptions I should cancel."
1. Claude fetches data via `budgetctl budget_summary`
2. Claude reviews limits via `budgetctl list_budget_goals`
3. Claude identifies recurring payments via `budgetctl detect_recurring_payments`
4. Claude provides an analysis highlighting anomalies.

---

## 🔮 The Roadmap

`missionctl` is rolling out throughout 2026/2027:
*   **Q3 2026:** `postctl` v1.0 + MCP, `calctl` v0.1.
*   **Q4 2026:** `calctl` v0.5, `mailctl` v0.1 + v0.5, `budgetctl` v0.1 (Bundle Alpha on Polar.sh).
*   **Q1 2027:** `mailctl` v1.0 + TUI, `budgetctl` v0.5, `notectl` v0.1, `taskctl` v0.1 (Full Bundle v1.0).
*   **Q2 2027:** All tools v1.0 + complete MCP Suite.

---

## 📦 Lifetime Access & Pricing
We believe in lifetime access. No monthly subscription traps. You can connect up to two networks or domains for free, forever. Unlimited license keys can be purchased on Polar.sh:
*   Individual tools: $9 (one-time)
*   **missionctl Bundle (All six tools): $39** (one-time)
*   Interactive Go & TUI Tutorial: $19 (one-time)
*   Full Bundle + Tutorial: $49 (one-time)

If you love the command line, want absolute ownership of your data, and want your local AI agent to be highly capable, check it out!

👉 Website: https://postctl.sh / https://github.com/aeon022/missionctl
