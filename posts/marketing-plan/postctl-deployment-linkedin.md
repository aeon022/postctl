---
id: postctl-deployment-linkedin
platform: linkedin
type: single
campaign: postctl-marketing
schedule: "2026-07-01 09:00"
---
💻 "What happens to my scheduled posts if my laptop is closed?"

This is a classic question for local-first developer tools. When using postctl, you have two elegant solutions:

1️⃣ Auto-Catchup: When you boot your machine and open the TUI or start the daemon, postctl immediately detects any missed scheduled posts in the database and publishes them instantly.

2️⃣ 24/7 Cloud Deployment: Because postctl is written in Go, it compiles to a single static binary and uses a self-contained SQLite file. You can easily deploy it on a $4 VPS or a Raspberry Pi in under 2 minutes:

• Cross-compile for Linux:
`GOOS=linux GOARCH=amd64 go build -o postctl-linux main.go`

• Copy your `~/.config/postctl/` folder to the server.
• Run the background scheduler:
`nohup ./postctl-linux daemon > daemon.log 2>&1 &`

No heavy infrastructure. No database hosting. Just code and files.

Read the guide: https://postctl.sh/docs
