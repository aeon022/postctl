---
id: postctl-deployment-mastodon
platform: mastodon
type: single
campaign: postctl-marketing
schedule: "2026-07-01 09:00"
---
💻 "What happens to my scheduled posts if my laptop is closed?"

With local-first postctl, you have two options:

1️⃣ Auto-Catchup: When you boot and open the TUI or start the daemon, postctl publishes missed posts.
2️⃣ 24/7 Cloud: postctl compiles to a single Go binary. Run it on a VPS or Raspberry Pi:
`GOOS=linux GOARCH=amd64 go build -o postctl-linux`
`nohup ./postctl-linux daemon > daemon.log 2>&1 &`

👉 https://postctl.sh/docs

#selfhosted #golang #tui
