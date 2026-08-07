---
platform: facebook
type: single
campaign: missionctl-polish-2026-08
joplin_id: 574380eedc5c40d188a9692837798293
updated_at: '2026-08-07T19:27:09.838Z'
---
🦙 Local AI, No API Key Required: Ollama Support Lands in budgetctl & diaryctl

missionctl has always meant "local-first" for your data. Now it means that for AI too.

budgetctl's transaction categorization and diaryctl's AI-narrated diary entries can both run against a local Ollama model instead of Anthropic's Claude — no API key, no per-token billing, no request ever leaving your machine.

Point either tool at a local model with one environment variable (BUDGETCTL_OLLAMA_MODEL / DIARYCTL_OLLAMA_MODEL), and it uses that instead of the cloud. Anthropic stays the default and the more capable option for anyone who wants it — this is an option, not a replacement.

Under the hood, a shared missionctl-core/ai package handles both providers behind one interface, so adding Ollama support to a tool that already had Claude support was a small, focused change rather than a rewrite.

For anyone who's chosen this whole suite specifically because their data doesn't leave their laptop: now the AI features can make that same promise.

👉 https://missionctl.sh
