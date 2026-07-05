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

### Milestone 4: AI-as-Operator & MCP (Completed)
- [x] Local LLM integration for AI-powered repurposing.
- [x] Machine-readable output formats (`--format json`) and safe `--dry-run` operations.
- [x] Native **Model Context Protocol (MCP)** server over stdio for agent integration.

---

## 🔮 Future Roadmap (New Feature Ideas)

### 📊 1. Monospace TUI Analytics Dashboard
- [ ] Add a **TUI Analytics Tab** displaying real-time post engagement (likes, shares, comments, impressions).
- [ ] Fetch stats asynchronously using background workers to avoid UI freezes.
- [ ] Generate comparative charts (ASCII bar charts) comparing campaign performances.

### 🧵 2. Multi-Platform Threads
- [ ] Extend thread-splitting support to **Mastodon**, **Bluesky**, and **Threads** (currently only Twitter/X is fully thread-native).
- [ ] Implement sequential posting loops with status checks for all thread-capable platforms.

### 📅 3. Smart Queues & Auto-Scheduling
- [ ] Support defining standard **Publishing Windows** (e.g., weekdays at 09:00, 15:00, 18:00) per campaign or platform.
- [ ] Add an auto-queue function: importing a draft automatically assigns it to the next available publishing window slot.

### 📂 4. Media Library & Video Support
- [ ] Implement video uploads for platforms that support it.
- [ ] Add a TUI-based **Media Library browser** to preview and reuse uploaded images or assets.

### 👥 5. Multi-Profile Management
- [ ] Support multiple accounts or profiles per platform (e.g. `@personal` vs. `@company`).
- [ ] Allow toggling active profiles directly inside the TUI settings dashboard.
