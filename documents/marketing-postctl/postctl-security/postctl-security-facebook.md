---
id: postctl-security-facebook
platform: facebook
type: single
campaign: postctl-security
schedule: "2026-07-07 10:00"
---
🔒 Why Local-First Security is the Only Way to Manage Social API Keys

Many developer tools require you to paste your LinkedIn, Twitter, or Threads API client secrets into their cloud databases. But exposing these high-privilege credentials to a third-party server is a massive security risk.

When building postctl, I decided to make it local-first and offline-first to eliminate this risk entirely.

Here is how the local security architecture works:
1️⃣ Local Encryption: All access tokens and client secrets are stored in a local SQLite database on your machine.
2️⃣ AES-256-GCM: The database is encrypted using AES-256-GCM. 
3️⃣ Passphrase Key Derivation: The encryption key is derived directly from a master passphrase you set during setup. Without this passphrase, the SQLite database is unreadable.
4️⃣ No Telemetry: postctl communicates directly with social media APIs (using OIDC and OAuth 2.0 flow). No middleman server intermediates the requests, and no tracking data is collected.

👉 Star on GitHub: https://github.com/aeon022/postctl
👉 Read more: https://postctl.sh
