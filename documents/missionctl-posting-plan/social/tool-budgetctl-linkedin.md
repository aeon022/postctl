---
platform: linkedin
campaign: missionctl-tools
---
💰 Introducing budgetctl: Local-First, Privacy-Preserving Expense & Subscription Tracking

Connecting your bank account to cloud-based personal finance apps gives companies access to your complete transaction history. For privacy advocates, this is unacceptable.

budgetctl is a command-line budget manager written in Go that processes bank transactions entirely on your local machine.

Key Features:
1️⃣ Local Imports: Import standard bank CSV exports (ING, N26, Deutsche Bank, etc.) directly into a local SQLite database.
2️⃣ Regex Classification: Create classification rules using regex patterns (`budgetctl tag "Netflix" --category streaming`) to auto-categorize transactions.
3️⃣ Subscription Detection: Analyzes transactions to detect recurring monthly/yearly charges and identify potential unused SaaS subscriptions.
4️⃣ Budget Goals: Set category spending limits and track progress via the TUI dashboard.
5️⃣ MCP Integration: Exposes 9 tools to Claude (including `budget_summary`, `detect_recurring_payments`), enabling a local AI assistant to review your finances and suggest cuts.

Keep your financials secure, local, and private.

👉 Build budgetctl from source: https://github.com/aeon022/budgetctl

#golang #terminal #bubbletea #personalfinance #budgeting #mcp #opensource #privacy
