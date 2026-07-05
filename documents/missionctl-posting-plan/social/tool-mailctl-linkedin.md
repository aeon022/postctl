---
platform: linkedin
campaign: missionctl-tools
---
📧 Introducing mailctl: Email Management for the Monospace TUI & AI Era

Checking your inbox shouldn't mean logging into heavy webmails or bloated desktop clients. And giving AI tools access to your correspondence shouldn't require syncing your entire IMAP history to a cloud-based SaaS database.

mailctl brings your email inbox directly into your local terminal workflow. 

How it works:
1️⃣ macOS Bridge: Connects directly to Apple Mail using AppleScript to sync your inbox headers and body texts into a local, encrypted SQLite cache.
2️⃣ Markdown Drafting: Write emails in your favorite editor (like Vim/Neovim). mailctl parses the YAML frontmatter (To, Subject, CC) and body, then saves it to your Drafts folder or sends it immediately.
3️⃣ Model Context Protocol: Emits an MCP server over stdio exposing 6 tools: `inbox`, `search_email`, `email_thread`, `send_email`, `draft_email`, and `sync_inbox`.

Now, your local AI agent can read recent emails, draft replies, and organize your inbox, keeping 100% of your credentials offline.

👉 Check out the repository & build from source: https://github.com/aeon022/mailctl

#golang #terminal #email #bubbletea #modelcontextprotocol #opensource #privacy #macdev
