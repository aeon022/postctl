# AI Agent Instructions — postctl Landing Page Deployment

This guide explains how to edit, build, and publish/deploy changes to the `postctl.sh` website.

## Git Repository & Branch Structure

Two branches, two purposes — kept separate so Plesk's `git pull` only ever fetches the site, never the Astro source, docs, or stray build artifacts:

- **`deploy/landing-src`** — the Astro source (components, pages, docs, README, etc.). All editing happens here.
- **`deploy/landing`** — the *built* static site only (what used to be `dist/`), flattened to the branch root. This is the branch Plesk's Git integration is connected to; its document root can point straight at the repo root, no subfolder config needed.
- The `main` branch of the `postctl` repository contains only the CLI & TUI tool source code.
- **Never** push landing-page changes to `main`, and never leave them on a throwaway `worktree-agent-*` branch — those are session-local and must be deleted (locally and on the remote, if pushed) once the work is merged or abandoned.

## Folder Structure Rule (applies to every project with its own landing page, not just postctl)

Two hidden worktree subfolders **inside the project it belongs to**, never sibling folders one level up:

```
missionctl/postctl/                          ← postctl repo, branch main (CLI/TUI source)
missionctl/postctl/.worktree-landing/         ← same repo, branch deploy/landing-src (Astro source — edit here)
missionctl/postctl/.worktree-landing-publish/ ← same repo, branch deploy/landing (built site only — Plesk pulls this)
```

Both are listed in `postctl/.gitignore` so neither shows up as untracked content on `main`.

## How to Work on the Landing Page & Docs

1. **Access the worktrees:**
   If they don't exist yet, create them from inside the `postctl` repo root:
   ```bash
   cd /Users/gweiher/Developing/Projects/missionctl/postctl
   git worktree add ./.worktree-landing deploy/landing-src
   git worktree add ./.worktree-landing-publish deploy/landing
   ```
   All editing happens in `.worktree-landing`. `.worktree-landing-publish` is a build target — don't hand-edit files there.

2. **Synchronize Docs (if code changes were made on main):**
   The website docs are rendered directly from the repository Markdown files. If you modified `README.md` or files in `docs/` or `documents/` on the `main` branch, synchronize them to the source worktree:
   ```bash
   cp /Users/gweiher/Developing/Projects/missionctl/postctl/README.md /Users/gweiher/Developing/Projects/missionctl/postctl/.worktree-landing/README.md
   cp -r /Users/gweiher/Developing/Projects/missionctl/postctl/docs/ /Users/gweiher/Developing/Projects/missionctl/postctl/.worktree-landing/docs/
   cp -r /Users/gweiher/Developing/Projects/missionctl/postctl/documents/ /Users/gweiher/Developing/Projects/missionctl/postctl/.worktree-landing/documents/
   ```

3. **Install Dependencies & Start Dev Server:**
   ```bash
   cd /Users/gweiher/Developing/Projects/missionctl/postctl/.worktree-landing
   npm install --legacy-peer-deps
   npm run dev
   ```
   *(The dev server will run on http://localhost:4322 since http://localhost:4321 is typically taken by the main missionctl landing page).*

4. **Build & Sync to the Publish Worktree:**
   Run the helper script — it builds the source worktree and rsyncs the fresh `dist/` output into `.worktree-landing-publish`, then stages the changes there for review:
   ```bash
   /Users/gweiher/Developing/Projects/missionctl/postctl/.worktree-landing/scripts/publish.sh
   ```
   Review the `git status`/`git diff` it prints before committing. Never push straight from the source worktree — the site branch only ever gets built output, never `src/`.

5. **Commit and Push (Unsandboxed Git):**
   The environment's default sandboxed terminal blocks outbound TCP requests to GitHub. You MUST request `unsandboxed(git)` permission to run the push command.
   ```bash
   cd /Users/gweiher/Developing/Projects/missionctl/postctl/.worktree-landing-publish
   git commit -m "build: regenerate static production pages"
   git push origin deploy/landing
   ```
   If source changes should also be preserved/shared, commit and push `deploy/landing-src` from `.worktree-landing` too (not required for the site to go live, but keeps source history from being local-only).

## Cleaning Up After a Session

If you created any extra worktree (e.g. an isolated agent worktree for a one-off task) that is not `.worktree-landing/` or `.worktree-landing-publish/`, remove it and its branch once done — don't leave it in the repo:
```bash
git worktree remove <path>
git branch -D <branch>
git push origin --delete <branch>   # only if it was pushed
```
