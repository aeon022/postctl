# AI Agent Instructions — postctl Landing Page Deployment

This guide explains how to edit, build, and publish/deploy changes to the `postctl.sh` website.

## Git Repository & Branch Structure

- The website source code lives on the branch **`deploy/landing`** of the `postctl` repository (NOT `missionctl`).
- The `main` branch of the `postctl` repository contains only the CLI & TUI tool source code.

## How to Work on the Landing Page & Docs

1. **Access the Website Source Code:**
   If a git worktree for the website does not exist, create it:
   ```bash
   git worktree add ../postctl-landing deploy/landing
   ```
   All editing and building must be done in the directory `../postctl-landing` (relative to the `postctl` root) or `~/Developing/Projects/missionctl/postctl-landing`.

2. **Synchronize Docs (if code changes were made on main):**
   The website docs are rendered directly from the repository Markdown files. If you modified `README.md` or files in `docs/` or `documents/` on the `main` branch, synchronize them to the landing page directory:
   ```bash
   cp /Users/gweiher/Developing/Projects/missionctl/postctl/README.md /Users/gweiher/Developing/Projects/missionctl/postctl-landing/README.md
   cp -r /Users/gweiher/Developing/Projects/missionctl/postctl/docs/ /Users/gweiher/Developing/Projects/missionctl/postctl-landing/docs/
   cp -r /Users/gweiher/Developing/Projects/missionctl/postctl/documents/ /Users/gweiher/Developing/Projects/missionctl/postctl-landing/documents/
   ```

3. **Install Dependencies & Start Dev Server:**
   ```bash
   cd /Users/gweiher/Developing/Projects/missionctl/postctl-landing
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
