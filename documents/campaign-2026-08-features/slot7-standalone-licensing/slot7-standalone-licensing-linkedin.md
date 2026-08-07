---
platform: linkedin
type: single
campaign: missionctl-polish-2026-08
joplin_id: ce972736cdb844d1a9fb1f8d62c9ac50
updated_at: '2026-08-07T19:27:09.794Z'
---
🔑 Buy Just the One Tool You Need — Standalone Licensing Is Here

Until now, unlocking AI features anywhere in the missionctl suite meant buying the full $39 Bundle, even if you only cared about one tool's AI feature.

Five tools now sell standalone via Polar.sh: mailctl, calctl, budgetctl, and habctl each get a one-time $9 unlock for their AI features (draft generation, meeting summaries, categorization, coaching reviews). notectl's $9 unlocks a second vault instead of AI.

Core functionality in every tool stays free forever — the $9 unlock is purely additive, not a trial that expires.

The technical bit worth sharing: this runs on missionctl-core/licensing, a shared package wrapping Polar.sh's license-key API. Each tool validates against its own benefit ID OR the shared Bundle's benefit ID — either grants access — instead of five separate ad-hoc licensing implementations copy-pasted across the suite. We also removed an old public "family/dev" bypass that had crept into one tool's Pro-check, so the licensing gate now matches what's actually for sale.

Prefer everything at once? The Bundle is still $39 one-time for all 9 tools, and is the better deal once you want three or more of the paid unlocks.

👉 https://missionctl.sh

#golang #buildinpublic #opensource #indiehacker #saas #localfirst
