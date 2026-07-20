# postctl Project Roadmap

This document outlines the milestones, current achievements, and planned features for `postctl`.

---

## 🚀 Milestones & Current State

### Milestone 1: The Core CLI & Storage (Completed)
- [x] SQLite local storage integration with CGO-free driver.
- [x] Standard CLI commands (`import`, `list`, `post`, `schedule`).
- [x] Deterministic ID generation for campaign synchronization.
- [x] Secure AES-256-GCM database export/import for multi-device Git sync.

### Milestone 2: Markdown & YAML Parser (Completed)
- [x] YAML Frontmatter metadata parser.
- [x] Unified thread segmentation support (`---` separator and `## Tweet X` headers).
- [x] Character limit validators per platform.
- [x] Image path validation and resolution.

### Milestone 3: Interactive TUI Dashboard (Completed)
- [x] Bubble Tea/Lip Gloss keyboard-driven terminal dashboard.
- [x] Responsive layout with dynamic sizing and scroll viewports.
- [x] Suspended Vim/Neovim integration (`ctrl+v`) with live character rulers.
- [x] Terminal calendar datepicker (`ctrl+d`).
- [x] Multi-platform status checker and setup wizard.

### Milestone 4.5: Power-User Utilities & Platform Integration (Completed)
- [x] Connection Diagnostic Tool (`postctl config test`) to safely test all credential states.
- [x] RSS Auto-Importer (`postctl rss`) to fetch and queue posts from blog feeds.
- [x] Terminal Image Preview in TUI Editor using high-fidelity ANSI 24-bit half-blocks.
- [x] TUI Bulk Actions (multi-select with Space, bulk delete with `d`, bulk schedule to queues with `s`).
- [x] Platform Expansion Category A & B (Telegram, Discord, Reddit, Dev.to, Hashnode, Medium).

---

## 🔮 Future Roadmap (Completed)

### 📊 1. Monospace TUI Analytics Dashboard
- [x] Add a **TUI Analytics Tab** displaying real-time post engagement (likes, reposts, comments, impressions) using ASCII/Sparkline charts.
- [x] Fetch stats asynchronously using background workers to avoid TUI thread locks.
- [x] Implement command line analytics output (`postctl analytics`) with ASCII formatting.

### 🧵 2. Multi-Platform Threads & Live Editor Preview
- [x] Extend thread-splitting support (`---` separator) to **Mastodon**, **Bluesky**, and **Threads**.
- [x] Implement live character-limit warning indicator and thread-view in the TUI Editor.
- [x] Implement sequential posting loops with custom safety delays between replies to prevent rate limits.

### 📅 3. Smart Queues & Auto-Scheduling
- [x] Support defining standard **Publishing Windows** (e.g. weekdays at 09:00, 15:00, 18:00) per campaign/platform in config.
- [x] Add support for `schedule: queue` in frontmatter or `postctl schedule <ID> --queue`.
- [x] Automatically assign posts to the next available publishing window slot.

### 🔄 4. Git Hooks & Static Site Generator (SSG) Pipeline
- [x] Create `postctl git-hook install` command to automatically import posts from a specific directory on commit/push.
- [x] Write integration guidelines/scripts for Hugo, Jekyll, and Astro to extract frontmatter social teasers.

### 🤖 5. Offline LLM & Repurposing (Ollama Integration)
- [x] Integrate local Ollama support (`llama3`, `mistral`, `phi3`) alongside OpenAI and Anthropic.
- [x] Add tone selection flag for repurposing: `postctl repurpose <ID> --platform <TARGET> --tone <TONE>` (e.g. professional, shitpost, educational).
- [x] Implement AI-based automatic ALT-text generator for uploaded images.

---

## 🚀 Milestone 5: Multi-Platform Expansion (Planned & In Progress)

Integrate additional platforms to support a wider range of developer community, tech blogging, and visual/video networks.

### 💬 Category A: Developer & Community Channels (Easiest)
- [x] **Telegram:** Broadcast to channels/groups using a bot token and chat ID.
- [x] **Discord:** Send updates to server channels via Webhook URLs.
- [x] **Reddit:** Submit text or link posts to subreddits using OAuth API.

### ✍️ Category B: Developer Blogging (Markdown-Native)
- [x] **Dev.to:** Publish tech articles directly via personal API token (native Markdown/Frontmatter).
- [x] **Hashnode:** Publish articles to personal blogs using GraphQL API and personal access token.
- [x] **Medium:** Publish articles to Medium publication drafts or posts via Integration Token.

### 📸 Category C: Visual & Video Networks (Media-Focused)
- [ ] **Instagram:** Publish single images, carousels, or Reels using Facebook Graph API.
- [ ] **Pinterest:** Create image-rich Pins with links using Pinterest API.
- [ ] **YouTube / YouTube Shorts:** Upload teaser videos via YouTube API.

