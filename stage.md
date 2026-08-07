# postctl Landing Page — Stage Log

Session handoff notes for this worktree (`deploy/landing-src`). Read this first before touching
`src/pages/docs.astro` or any Markdown content that gets rendered on it.

---

## ⚠️ The one rule that matters here: never mix languages mid-document

**Write a whole document in ONE language first, top to bottom. Only then translate it into the
other, as a second complete pass.** Never draft in German and let English creep in halfway
through, or vice versa — that's exactly how this site ended up broken twice:

- The **first** language-toggle bug: several homepage sections ("Try the TUI", feature cards,
  "How It Works", Pricing/Donate) were written in English only and never got a German pass at
  all — so the toggle button existed and worked, but flipping it did nothing for those sections.
- The **second**, worse bug (2026-08-07): 6 of 11 `/docs` sections — including the Introduction
  tab, the *first* thing every visitor sees — had **zero** `lang-en`/`lang-de` wrapping. A real
  Bundle customer hit this live and reported it, twice, before it was actually found and fixed
  (see Session 1 below). `documents/SPEC.md` was its own special case: one document that was
  ~80% English prose followed by a ~20% German tail, added at different times by different
  passes that each only wrote in whichever language felt natural in the moment — never toggled,
  never fully bilingual either.

**The fix pattern, every time a section is added or edited:**

```astro
<div class="lang-en">
  <!-- complete English content -->
</div>
<div class="lang-de">
  <!-- complete German content -->
</div>
```

For content that lives in its own Markdown file (the API setup guides, the tutorial, the spec),
the convention is two sibling files, imported and wrapped the same way:

```
docs/api-twitter.md        ← German (base filename — matches Joplin sync, has joplin_id frontmatter)
docs/api-twitter-en.md     ← English (plain file, no frontmatter)
```

```astro
import * as TwitterDocDe from '../../docs/api-twitter.md';
import * as TwitterDocEn from '../../docs/api-twitter-en.md';
```
```astro
<div class="lang-de"><TwitterDocDe.Content /></div>
<div class="lang-en"><TwitterDocEn.Content /></div>
```

**`README.md` is the one deliberate exception** to "base filename = German" — it stays English
because it's the actual GitHub README, and GitHub convention expects that. Its German
counterpart is `README-de.md`, imported and wrapped the same way, just with the two languages
swapped from the usual pattern.

**Before considering any doc page "done," grep for the gap, don't just eyeball it:**

```bash
grep -c "lang-en\|lang-de" src/pages/docs.astro   # should track roughly with content volume
grep -o 'id="doc-[a-z-]*"[^>]*>.\{0,150\}' dist/docs/index.html   # first tag after each section
#   should show <div class="lang-en"> or <div class="lang-de"> right after every doc-body,
#   never a bare <h1>/<p> with no wrapping div at all
```

A bare `<h1>` or `<p>` with no `lang-*` div around it — anywhere — means that content shows in
one language forever, no matter what the visitor picks. That's the entire bug, every time.

---

## Session 1 — 2026-08-07

Fixed both remaining language-mix bugs, live-verified via `curl` against the deployed
`postctl.sh` (no browser tool available this session — user declined installing the Chrome
extension, so structural verification via grep/curl stood in for a visual check).

**What was actually broken**, confirmed by diffing the deployed HTML's `id="doc-*"` sections
against their immediate next tag:

- `doc-introduction` (renders `README.md`) — bare `<h1>`, no wrapping. Worst one: default/first
  tab shown on page load.
- `doc-vim-flow`, `doc-image-resolution` — inline-authored English content in `docs.astro`
  itself (not imported from a `.md` file at all), no German pass ever written.
- `doc-api-bluesky-mastodon`, `doc-api-facebook` — same: inline English only.
- `doc-system-spec` (renders `documents/SPEC.md`) — the internal English/German mix described
  above.

**Fix**: added `README-de.md`; wrote full German translations for the four inline sections and
wrapped them in matching `lang-en`/`lang-de` divs; split `SPEC.md` into a complete German version
and a new `SPEC-en.md`, wired in the same way. Also synced `README.md`/`docs/`/`documents/` from
`main` per the normal `AGENTS.md` procedure — the Profiles feature, cross-device sync, and the
`POSTCTL2026` launch-special sections existed on `main` but had never been copied into this
worktree, so they were completely missing from the live site regardless of language.

**Judgment call flagged to the user, not made silently**: `SPEC.md`'s "Future Features" section
described a subscription pricing model ($9/mo, Team Mode, Analytics Dashboard) that has nothing
to do with postctl's actual pricing (the one-time missionctl Bundle via Polar.sh). Rather than
just translating the stale numbers into English too and making them more visible, added a note
in both languages pointing at the real `/#pricing` page. Told the user this was done; didn't wait
for approval before pushing since it was a clarifying addition, not a deletion.

Built and pushed both branches (`deploy/landing-src` then `deploy/landing` via
`scripts/publish.sh`) each of the two times fixes landed. **Plesk does not auto-deploy on push in
this setup** — confirmed by checking `Last-Modified` on the live response after pushing; it
stayed on the pre-fix timestamp until a manual pull was triggered on the server. Don't assume a
push alone means it's live — say so explicitly and remind whoever's driving to trigger the pull.
