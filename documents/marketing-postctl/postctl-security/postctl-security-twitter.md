---
id: postctl-security-twitter
platform: twitter
type: single
campaign: postctl-security
schedule: "2026-07-07 10:00"
---
## Tweet 1
Why trust the cloud with your social media API keys? 🔑

postctl keeps your access tokens secure on your local machine using industry-standard AES-256-GCM encryption. Here is how the security architecture works. 👇

## Tweet 2
🔒 Local SQLite + AES-256:
All client secrets, OAuth tokens, and campaign histories are stored in a local SQLite file.

🔑 User-Derived Key:
The database is encrypted using a key derived from a passphrase you control. No plain text tokens on disk.

## Tweet 3
Zero third-party servers, zero telemetry, zero risk of SaaS database leaks. Your keys remain strictly on your own hardware.

Control your workflow:
👉 https://github.com/aeon022/postctl
🌐 https://postctl.sh
