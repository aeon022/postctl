---
id: postctl-deployment-bluesky
platform: bluesky
type: single
campaign: postctl-marketing
schedule: "2026-07-01 09:00"
---
💻 Closed laptop? No problem. `postctl` schedules posts in 2 ways:
1. Auto-Catchup: Booting checks SQLite and publishes missed schedules instantly.
2. VPS Daemon: Cross-compile for Linux and run on a $4 VPS:
`GOOS=linux GOARCH=amd64 go build -o postctl-linux`

Guide: https://postctl.sh/docs
