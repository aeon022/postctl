---
platform: twitter
type: thread
campaign: hn-client-deepdive
schedule: 2026-07-06 15:30
---
How do you load hundreds of Hacker News stories concurrently in a terminal interface without freezing the UI?

In `hn-client`, we combined the Elm TUI architecture with Go's powerful concurrency primitives. Here's the structure. 🧵
---
`hn-client` follows the Model-Update-View pattern of Bubble Tea.

To keep the UI responsive, we never perform API calls directly inside the Update loop. Instead, we run them in background goroutines using Bubble Tea Commands (`tea.Cmd`).
---
When you switch tabs (e.g. Top to Ask), a goroutine fetches the story details in parallel.

Once complete, the Go channel pipes a `statusMsg` back to the main loop, updating the model and triggering a clean UI re-render.
---
Fast, concurrent terminal apps. Explore the codebase and star the repo:
👉 https://github.com/aeon022/hn-client
