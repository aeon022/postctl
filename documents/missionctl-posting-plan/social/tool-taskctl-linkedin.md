---
platform: linkedin
campaign: missionctl-tools
---
📝 Introducing taskctl: Fast Task Management Synced with Apple Reminders

Most command-line task managers store data in proprietary text files that don't sync with your mobile device, or force you into a cloud subscription. 

taskctl solves this by combining a fast, terminal-native CLI and TUI with native Apple Reminders integration.

Key Features:
1️⃣ macOS Integration: Utilizes a Swift-compiled bridge (EventKit) to read and write directly to your Apple Reminders app. Your tasks sync automatically to your iPhone and iPad.
2️⃣ Background Daemon: Includes a lightweight, local daemon (`taskctl daemon --install`) that runs in the background to keep your caches fresh.
3️⃣ Keyboard-First UI: Filter, sort, and complete tasks using a monospace TUI dashboard built with Go and Bubble Tea.
4️⃣ MCP Integration: Exposes 7 tools to Claude (including `today_tasks`, `week_tasks`, `create_task`, `complete_task`), allowing your local AI assistant to manage your action items.

Social media and coding tasks, managed in one command line.

👉 Build taskctl from source: https://github.com/aeon022/taskctl

#golang #terminal #bubbletea #taskmanager #mcp #productivity #opensource #macdev
