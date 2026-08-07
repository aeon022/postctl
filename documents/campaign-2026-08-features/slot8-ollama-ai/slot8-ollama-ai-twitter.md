---
platform: twitter
type: thread
campaign: missionctl-polish-2026-08
joplin_id: 3830c73281414341b20f8a59747d2906
updated_at: '2026-08-07T19:27:09.868Z'
---
missionctl's whole pitch is local-first data. The AI features were the one exception — until now. 🧵
---
budgetctl (transaction categorization) and diaryctl (AI diary generation) now support Ollama as a provider alongside Anthropic. Point either tool at a local model and the request never leaves your machine.
---
One env var switches it: BUDGETCTL_OLLAMA_MODEL / DIARYCTL_OLLAMA_MODEL. Built on a shared missionctl-core/ai package so adding a second provider meant extending one abstraction, not forking per-tool code.
---
Claude's still the default (still the more capable option for most people) — Ollama's there for anyone who wants zero cloud dependency, full stop.

👉 https://missionctl.sh

#golang #localfirst #ollama
