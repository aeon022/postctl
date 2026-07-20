---
platform: twitter
type: thread
campaign: missionctl-launch
---
Tired of complex SaaS cloud tools locking up your private data just to connect AI assistants to your daily life? 

Meet `missionctl` — a local-first, terminal-native suite of 6 Go tools that gives AI agents "hands" to manage your daily tasks, emails, and budget. 🧵
---
Each tool in `missionctl` manages one domain locally in SQLite and exposes a Model Context Protocol (MCP) server:

1. mailctl 📧 (Apple Mail)
2. calctl 📅 (Apple Calendar)
3. taskctl 📝 (Apple Reminders)
4. notectl 📓 (Obsidian)
5. budgetctl 💰 (Bank CSVs)
6. postctl 🐦 (Socials)
---
By running `mcp` over stdio, the tools plug directly into Claude Desktop or any MCP client, giving the AI 43 focused tools. 

No cloud database, no monthly subscription traps. Your credentials and history stay encrypted on your own local disk.
---
Read the master plan, download the binaries, or contribute:
👉 https://github.com/aeon022/missionctl
🌐 https://postctl.sh

#opensource #golang #tui #bubbletea #modelcontextprotocol #macdev
