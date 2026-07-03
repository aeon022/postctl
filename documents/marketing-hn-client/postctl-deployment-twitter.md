---
id: postctl-deployment-twitter
platform: twitter
type: thread
campaign: postctl-marketing
schedule: "2026-07-01 09:00"
---

## Tweet 1

💻 "But what if my MacBook is closed when a post is scheduled?"

We get this question a lot. `postctl` is local-first, but handles this in two elegant ways:

1. **Auto-Catchup**: When you wake your Mac, it instantly catches up and posts any missed schedules.
2. **Cloud VPS**:

## Tweet 2

☁️ Deploying postctl on a $4/mo VPS or Raspberry Pi is incredibly simple:

1. Cross-compile for Linux:
`GOOS=linux GOARCH=amd64 go build -o postctl-linux main.go`

2. Copy config & SQLite DB to the server

3. Run headless background daemon:
`nohup ./postctl-linux daemon &`

## Tweet 3

No complicated server setups, docker containers, or databases to host. Just one static Go binary and a single SQLite file running in the background.

Try it out: https://postctl.sh
