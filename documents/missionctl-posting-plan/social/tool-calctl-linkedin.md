---
platform: linkedin
campaign: missionctl-tools
---
📅 Introducing calctl: Calendar Management & Asynchronous Scheduling for Terminal Users

Switching tabs to complex web calendars just to check your schedule or block time is a constant productivity leak. 

calctl brings calendar scheduling directly into your terminal workspace. Built in Go with Bubble Tea, it bridges Apple Calendar and exposes a secure Model Context Protocol (MCP) server so your local AI assistant can manage your time.

Key Features:
1️⃣ Native macOS Integration: Caches calendar databases locally, bridging to Apple Calendar via EventKit.
2️⃣ Slot Optimization: Run `calctl free --next 7d` to immediately calculate free time blocks inside defined working hours (e.g. 09:00 - 17:00).
3️⃣ YAML Import: Create calendar events using Markdown files with simple headers (Title, Date, Time, Duration, Attendees).
4️⃣ MCP Server: Exposes 7 tools to Claude, allowing the AI to scan, add, or delete events based on natural language commands.

Keep your calendar fast, local, and keyboard-driven.

👉 Build calctl from source: https://github.com/aeon022/calctl

#golang #terminal #bubbletea #calendar #mcp #productivity #opensource #macdev
