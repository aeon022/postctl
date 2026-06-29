---
title: Why I Built a Terminal-Native, Markdown-First Social Media Manager in Go
published: false
description: Stop using bloated web dashboards. Manage your Twitter/X, LinkedIn, and Threads posts as Markdown files versioned in Git.
tags: go, open-source, showdev, developer
---

# Why I Built a Terminal-Native, Markdown-First Social Media Manager in Go

As developers, we love version control. We write code in Markdown, review configurations in Pull Requests, and track history in Git. 

Yet, when it comes to social media distribution, we are forced to leave our terminals. We log into heavy web dashboards, click through complex scheduling forms, manually upload media files, and paste captions. 

Even worse: most modern "AI social tools" focus on writing captions for you, rather than letting your terminal AI agent actually *operate* the tool.

That is why I built **postctl** — a local-first, terminal-native CLI and TUI tool written in Go that manages, previews, and schedules postings across Twitter/X, LinkedIn, Threads, Bluesky, and Facebook directly from your codebase.

---

## The Philosophy: Postings as Code

With `postctl`, your social media pipeline lives alongside your code. A post is just a simple Markdown file with a YAML frontmatter:

```markdown
---
platform: twitter, linkedin
type: thread
campaign: release-v1.2
schedule: 2026-06-30 09:00
images: ["screenshots/dashboard.png"]
---

## Tweet 1
🚀 Announcing postctl v1.2 — The terminal-native social media manager is now fully bilingual with robust headless Chrome fallbacks!

## Tweet 2
🤖 "AI-as-Operator" in action:
With `--format json` and dry-runs, your terminal AI agent can import, schedule, and publish posts autonomously while you sleep.
```

To schedule it, you just run:
```bash
./postctl import ./documents/marketing-plan/
```
The CLI parses the frontmatter, validates character counts (e.g., 280 for Twitter, 300 for Bluesky) and image paths, and stores it in a local, encrypted SQLite database (`postctl.db`).

---

## 🤖 The "AI-as-Operator" Principle

We designed `postctl` for AI terminal agents (like Claude Engineer, GPT, or Aider). Instead of simple copywriting prompts, the AI operates the entire pipeline:

1. **Non-interactive flags:** Every command runs cleanly scriptable.
2. **`--format json`**: Every query returns machine-readable JSON that LLMs parse easily.
3. **`--dry-run`**: AI agents can simulate a posting pipeline, verify integrity, present it to the human in chat for approval, and then execute the live publish command.

Here is the exact terminal pipeline an AI agent runs:
```bash
# Import the Markdown files
./postctl import ./marketing/

# Dry-run validation (JSON output)
./postctl campaign post launch-campaign --dry-run --format json

# Human approves in chat -> Publish
./postctl campaign post launch-campaign --format json
```

---

## 💻 Under the Hood: The Tech Stack

* **Language:** Go (Golang) — compiles to a single, static binary. No docker, no node_modules, no configurations.
* **TUI (Terminal User Interface):** Built using the Elm-inspired **Bubble Tea** framework and **Lipgloss** by Charm. Allows managing drafts, reviewing schedules, and configuring credentials in a beautiful command-line dashboard.
* **Storage:** Local SQLite database, utilizing a CGO-free Go SQLite driver (`modernc.org/sqlite`).
* **Encryption:** AES-256-GCM with PBKDF2 key-stretching to securely export and sync your credentials across devices (e.g., Mac Studio and MacBook).
* **Headless Evasion Fallback:** When X/Twitter rate-limits or blocks direct HTTP/GraphQL requests, `postctl` spins up a headless Google Chrome instance (via `chromedp`), loads your session cookies, types the text in the composer, and posts it.

---

## ☁️ Going Offline: 24/7 Cloud Daemon

Since it's local-first, what happens if your laptop is closed at the scheduled post time? 
1. **Auto-Catchup:** When you wake your machine, `postctl` immediately publishes any missed schedules.
2. **Cloud Daemon:** You can easily cross-compile the binary to Linux (`GOOS=linux GOARCH=amd64 go build`) and run the background scheduler daemon on a $4/mo VPS or a Raspberry Pi:
   ```bash
   nohup ./postctl-linux daemon > daemon.log 2>&1 &
   ```

---

## Open Source & Lifetime Access

`postctl` is open-source. You can connect up to 2 social networks for free, forever. If you need unlimited networks, you can buy a lifetime license on Polar.sh.

Check it out:
* **Website:** [https://postctl.sh](https://postctl.sh)
* **GitHub:** [https://github.com/aeon022/postctl](https://github.com/aeon022/postctl)

Give it a star and let me know what features you want next on the roadmap!
