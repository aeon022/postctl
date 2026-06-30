---
id: postctl-deployment-threads
platform: threads
type: single
campaign: postctl-marketing
schedule: "2026-07-01 09:00"
---
💻 "What happens to my scheduled posts if my laptop is closed?"

`postctl` handles this in two ways:

1️⃣ Auto-Catchup: When you boot your machine and open the TUI or start the daemon, it immediately publishes missed scheduled posts.

2️⃣ Cloud Daemon: Cross-compile to Linux and run on a cheap VPS or a Pi:
`GOOS=linux GOARCH=amd64 go build -o postctl-linux main.go`

No heavy infrastructure or databases. Just files.

Guide: https://postctl.sh/docs
