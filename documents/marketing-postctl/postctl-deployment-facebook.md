---
id: postctl-deployment-facebook
platform: facebook
type: single
campaign: postctl-marketing
schedule: "2026-07-01 09:00"
---
🚀 Meet postctl v1.2 — Local-first Social Media Scheduler for Developers

Worried about your scheduled posts failing when your MacBook is closed? `postctl` has you covered:

1️⃣ Auto-Catchup: When you wake your computer, the scheduler immediately detects missed slots in the SQLite database and posts them instantly.
2️⃣ 24/7 Cloud Daemon: Cross-compile the Go binary for Linux and deploy it to a cheap VPS or a Raspberry Pi in under 2 minutes:
`GOOS=linux GOARCH=amd64 go build -o postctl-linux main.go`

Deploy config, run `./postctl-linux daemon` in the background, and you're good to go.

Learn more: https://postctl.sh
