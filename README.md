# postctl.sh — published site (do not edit here)

This branch (`deploy/landing`) contains only the built static site — this
is what Plesk pulls and serves directly as the domain's document root.

- Source lives on branch `deploy/landing-src` (worktree `postctl/.worktree-landing/`).
- To publish a change: edit + `npm run build` in the source worktree, then
  copy the fresh `dist/*` into this worktree (`postctl/.worktree-landing-publish/`),
  commit, and push.
