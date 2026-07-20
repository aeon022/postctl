---
platform: linkedin
campaign: postctl-features
schedule: 2026-07-05 17:00
---
💻 Social Media as Code: Versioning & Syncing your Posts with Git

Why do we version control our code, documentation, and infrastructure, but treat social media posts as throwaway inputs in a SaaS web form?

postctl applies the "as code" philosophy to social media management.

Here is the Git-integrated workflow for multi-device sync:
1️⃣ Markdown Files: All posts are written as local Markdown files with YAML frontmatter, versioned directly in your project's Git repository.
2️⃣ SQLite Local DB: The postctl import tool converts these files into a local SQLite database (`postctl.db`).
3️⃣ AES-256 Encryption: Run `postctl config export -o backup.bin` to export all tokens, history, and schedules into a single encrypted binary file.
4️⃣ Git Versioning: Push the encrypted `backup.bin` to your private dotfiles or project repository.
5️⃣ Cross-Device Sync: Pull the repo on your VPS or laptop, run `postctl config import -f backup.bin`, and keep your publishing daemon running 24/7.

It's secure, free, and puts your social queue under your control.

👉 GitHub Repository: https://github.com/aeon022/postctl
🌐 Website: https://postctl.sh

#git #devops #socialmediaascode #localfirst #golang #security #cryptography
