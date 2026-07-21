# AI Agent Instructions — postctl Landing Page Deployment

This guide explains how to edit, build, and publish/deploy changes to the `postctl.sh` website.

## Git Repository & Branch Structure

- The website source code lives on the branch **`deploy/landing`** of the `postctl` repository (NOT `missionctl`).
- The `main` branch of the `postctl` repository contains only the CLI & TUI tool source code.
- **Push landing-page changes only to `deploy/landing`.** Never push them to `main`, and never leave them on a throwaway `worktree-agent-*` branch — those are session-local and must be deleted (locally and on the remote, if pushed) once the work is merged or abandoned.

## Folder Structure Rule (applies to every project with its own landing page, not just postctl)

The `deploy/landing` worktree checkout is always a hidden subfolder **inside the project it belongs to**, named **`.worktree-landing/`**. It is never a sibling folder one level up (e.g. `../postctl-landing`), and a project never gets a second or third copy of it. For postctl that means:

```
missionctl/postctl/                  ← postctl repo, branch main (CLI/TUI source)
missionctl/postctl/.worktree-landing/ ← same repo, branch deploy/landing (website source)
```

`.worktree-landing/` is listed in `postctl/.gitignore` so it never shows up as untracked content on `main`.

## How to Work on the Landing Page & Docs

1. **Access the Website Source Code:**
   If the worktree does not exist yet, create it from inside the `postctl` repo root:
   ```bash
   cd /Users/gweiher/Developing/Projects/missionctl/postctl
   git worktree add ./.worktree-landing deploy/landing
   ```
   All editing and building happens in `/Users/gweiher/Developing/Projects/missionctl/postctl/.worktree-landing`.

2. **Synchronize Docs (if code changes were made on main):**
   The website docs are rendered directly from the repository Markdown files. If you modified `README.md` or files in `docs/` or `documents/` on the `main` branch, synchronize them to the landing page directory:
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

4. **Build the Site (CRITICAL):**
   Before pushing, you MUST rebuild the static production pages to update the `dist/` directory. If you do not rebuild the site, the deployed website will not show any of your changes!
   ```bash
   npm run build
   ```

5. **Commit and Push (Unsandboxed Git):**
   The environment's default sandboxed terminal blocks outbound TCP requests to GitHub. You MUST request `unsandboxed(git)` permission to run the push command.
   ```bash
   git add .
   git commit -m "build: regenerate static production pages inside dist/"
   git push origin deploy/landing
   ```

## Cleaning Up After a Session

If you created any extra worktree (e.g. an isolated agent worktree for a one-off task) that is not `.worktree-landing/`, remove it and its branch once done — don't leave it in the repo:
```bash
git worktree remove <path>
git branch -D <branch>
git push origin --delete <branch>   # only if it was pushed
```
