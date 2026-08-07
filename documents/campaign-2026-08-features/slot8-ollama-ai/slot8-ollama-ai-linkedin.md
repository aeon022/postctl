---
platform: linkedin
type: single
campaign: missionctl-polish-2026-08
joplin_id: 1575d3d39b384c2ca1903b9057c1bcbd
updated_at: '2026-08-07T19:27:09.848Z'
---
🦙 Local AI, No API Key Required: Ollama Support Lands in budgetctl & diaryctl

The whole premise of missionctl is that your data — email, calendar, tasks, budget, habits — stays on your machine. Until now, the AI features were the one place that wasn't fully true: categorizing transactions or narrating a diary entry meant a request to Anthropic's API.

budgetctl and diaryctl now support Ollama as an alternative provider. Point either tool at a local model (BUDGETCTL_OLLAMA_MODEL / DIARYCTL_OLLAMA_MODEL) and categorization or diary generation runs entirely against your own hardware — no API key, no per-token cost, no network request.

This runs on missionctl-core/ai, a shared multi-provider package both tools import — adding a second backend meant extending one shared abstraction, not forking the AI-calling code per tool. Anthropic remains the default (still the more capable option for most people); Ollama is there for anyone who wants zero cloud dependency, full stop.

👉 https://missionctl.sh

#golang #localfirst #privacy #llm #ollama #opensource #buildinpublic
