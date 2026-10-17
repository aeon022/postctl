# Security Policy

postctl manages credentials for social media accounts, so security reports are taken seriously.

## Reporting a vulnerability

Please report suspected vulnerabilities **privately** — not as a public issue or pull request:

- E-mail: **post@abteilung83.com** (subject: `postctl security`)
- Languages: German or English
- Helpful details: affected version (`postctl --version`), platform, steps to reproduce, impact, and a proof of concept if you have one. Please do not include real tokens or account data.

You will get an acknowledgement within **3 working days** and a first assessment within **10 working days**. We aim to fix confirmed issues quickly and will coordinate a disclosure date with you; please give us a reasonable time (typically up to 90 days) before publishing details. Reporters are credited in the release notes unless they prefer to stay anonymous.

## Scope

In scope: the `postctl` CLI/TUI, the scheduler daemon, the MCP server (`postctl mcp`), license handling, local storage of tokens/backups, and the postctl.sh website.

Out of scope: vulnerabilities in third-party platforms (Instagram, Facebook, X, LinkedIn, …) or in your own tokens/devices; attacks that need prior full access to your user account or unlocked machine; denial of service by flooding; social engineering; reports without a demonstrated security impact.

## Supported versions

Only the latest release receives security fixes. Please update before reporting.

## Safe harbor

Good-faith research that follows this policy — no access to other people's data, no service disruption, no public disclosure before a fix — will not lead to legal action from us.

## How postctl protects your data (summary)

- Local-first: posts, history and credentials stay on your machine; there is no postctl cloud.
- Account tokens in the local database and exported backups are encrypted with AES-256-GCM.
- No telemetry. Network traffic goes only to the platforms you connect, to Polar.sh for license checks, and to an AI provider if you enable AI features with your own key.
- Keep your config and database in a directory only you can read, and rotate tokens if a device is lost.
