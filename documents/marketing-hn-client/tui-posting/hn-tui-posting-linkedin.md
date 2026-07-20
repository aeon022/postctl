---
id: hn-tui-posting-linkedin
platform: linkedin
type: single
campaign: hn-client-tui-posting
schedule: "2026-07-08 10:00"
---
⌨️ Hacker News Browse & Post — 100% in your Terminal!

I just shipped a major update to hn-client: you can now write posts and reply to comments natively in the TUI without ever opening a browser.

How it works:
1️⃣ Secure Local Session: Log in once securely from the client. Your session cookie is stored locally in ~/.hn-config.json with strict 0600 permissions.
2️⃣ Step-by-Step TUI Wizard: Press 'w' in the feed to trigger a wizard directly in the terminal footer (Title ➔ URL ➔ Text).
3️⃣ Native Comment Replies: Press 'r' in the comment view, type your reply in the footer, and submit. The comments reload automatically.

Keep your focus in the terminal. No GUI context-switching, no browser distractions.

👉 Star the repo: https://github.com/aeon022/hn-client

#golang #terminal #hackernews #developerproductivity #tui #opensource
