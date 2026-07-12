# postctl

Terminal social media manager. Write posts in Markdown, schedule them, and publish to Twitter/X, LinkedIn, Threads, Mastodon, and Bluesky from the command line or a full TUI.

**Supported platforms:** Twitter/X · LinkedIn · Threads · Mastodon · Bluesky

---

## Quick Start

1. **Install**

   ```bash
   git clone https://github.com/aeon022/postctl && cd postctl
   ./setup.sh
   ```

2. **Authenticate with a platform**

   ```bash
   postctl auth --platform twitter
   ```

3. **Write a post** — create a Markdown file (see [Post Format](#post-markdown-format)):

   ```markdown
   ---
   platform: twitter
   title: My first post
   ---

   Hello from postctl.
   ```

4. **Import the file**

   ```bash
   postctl import my-post.md
   ```

5. **Publish immediately or schedule**

   ```bash
   postctl post <ID>
   postctl schedule <ID> --time 2026-07-10T09:00:00+02:00
   ```

6. **Open the TUI to manage everything**

   ```bash
   postctl tui
   ```

---

## Cheatsheet

```
postctl                                  Open TUI (default)
postctl tui                              Open TUI explicitly

postctl auth --platform PLATFORM         Authenticate with a platform
postctl config [--show] [--set K V]      View or set config values

postctl import FILE_OR_DIR               Import Markdown post(s)
postctl list [--platform P] [--status S] [--campaign C] [--format human|json]
postctl template --platform PLATFORM     Generate a post template

postctl post ID [--dry-run]              Publish a post immediately
postctl schedule ID [--time DATETIME] [--queue] Schedule a post (RFC3339) or to the queue
postctl campaign list                    List all campaigns
postctl campaign post NAME [--dry-run]   Publish all posts in a campaign

postctl generate URL                     AI-generate a post from a URL
postctl repurpose ID --platform TARGET [--tone TONE] Repurpose a post with custom tone

postctl git-hook install [--dir DIR]     Install a post-commit git hook
postctl git-hook uninstall               Uninstall the git hook

postctl analytics [--platform P] [--format human|json]
postctl daemon [--dry-run]               Run the background scheduler
postctl mcp                              Start the MCP server (stdio)
postctl version                          Print version
```

---

## Post Markdown Format

### Frontmatter Fields

| Field      | Required | Values / Format                                                       | Description                       |
|------------|----------|-----------------------------------------------------------------------|-----------------------------------|
| `platform` | Yes      | `twitter`, `linkedin`, `threads`, `mastodon`, `bluesky`               | Target platform                   |
| `title`    | No       | String                                                                | Internal label (not published)    |
| `campaign` | No       | String slug                                                           | Groups posts into a campaign      |
| `schedule` | No       | RFC3339 or `"queue"`                                                  | Scheduled publish time or Smart Queue |

### Body Format

Write the post body in plain Markdown below the closing `---` of the frontmatter block.

- **LinkedIn, Threads, Mastodon, Bluesky:** Single body. No separators.
- **Twitter/X threads:** Separate individual tweets with a line containing only `---`. Each segment becomes one tweet in the thread.

### Twitter Thread Example

```markdown
---
platform: twitter
title: Launch announcement
campaign: product-launch
schedule: 2026-07-10T09:00:00+02:00
---

This is the first tweet. Max 280 characters for Twitter/X.

---

Second tweet in the thread.

---

Third tweet. Threads are Twitter-only.
```

---

## CLI Reference

### Authentication and Configuration

| Command | Description |
|---------|-------------|
| `postctl auth --platform PLATFORM` | Authenticate with the given platform (OAuth flow) |
| `postctl config --show` | Print the current configuration |
| `postctl config --set KEY VALUE` | Set a configuration value |

### Content Management

| Command | Description |
|---------|-------------|
| `postctl import FILE_OR_DIR` | Import one Markdown file or a directory of files |
| `postctl list` | List posts; filter with `--platform`, `--status`, `--campaign`; format with `--format human\|json` |
| `postctl template --platform PLATFORM` | Print a Markdown template for the given platform |
| `postctl generate URL` | AI-generate a draft post from the article at URL |
| `postctl repurpose ID --platform TARGET [--tone TONE]` | Repurpose an existing post with an optional custom tone |

### Publishing

| Command | Description |
|---------|-------------|
| `postctl post ID` | Publish post immediately |
| `postctl post ID --dry-run` | Simulate publishing without sending |
| `postctl schedule ID --time DATETIME` | Set or update the scheduled publish time |
| `postctl schedule ID --queue` | Schedule a post to the next available queue slot |
| `postctl campaign list` | List all campaigns with post counts |
| `postctl campaign post NAME` | Publish all posts in a campaign |
| `postctl campaign post NAME --dry-run` | Dry-run campaign publish |
| `postctl git-hook install [--dir DIR]` | Install local git post-commit hook for auto-import |
| `postctl git-hook uninstall` | Remove local git post-commit hook |
| `postctl daemon` | Start the background scheduler daemon |
| `postctl daemon --dry-run` | Run daemon in dry-run mode |

### Analytics

| Command | Description |
|---------|-------------|
| `postctl analytics` | Show analytics across all platforms |
| `postctl analytics --platform PLATFORM` | Filter to one platform |
| `postctl analytics --format json` | Output as JSON |

### MCP Server

| Command | Description |
|---------|-------------|
| `postctl mcp` | Start the MCP server on stdio for use by AI agents |

---

## TUI Guide

Launch with `postctl` or `postctl tui`.

### Views

| View | Description |
|------|-------------|
| Posts list | Main view; shows all posts with status badges (draft / scheduled / posted / failed) |
| Detail | Full post content and metadata |
| Editor | Write or edit post body and frontmatter fields |
| Schedule | Set or adjust the scheduled time for a post |
| Analytics | Platform-level metrics overview |
| History | Log of past publish events |
| Settings | App configuration |
| Readme | In-app documentation |

Switch between views using tabs or the keybindings below.

### Keybindings

**Posts List**

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate up and down |
| `Enter` | Open detail view |
| `n` | New post |
| `e` | Edit selected post |
| `d` | Delete selected post |
| `p` | Publish selected post |
| `s` | Schedule selected post |
| `Tab` | Switch tabs |
| `q` | Quit |

**Detail View**

| Key | Action |
|-----|--------|
| `Esc` | Back to list |
| `e` | Edit post |
| `p` | Publish post |
| `r` | Repurpose post |

**Editor**

| Key | Action |
|-----|--------|
| `Ctrl+S` | Save |
| `Esc` | Cancel |
| `Tab` | Move between fields |

---

## MCP — AI Integration

postctl ships a built-in MCP server that exposes all core operations to AI agents. This lets tools like Claude Desktop create, schedule, and publish posts on your behalf.

### Claude Desktop Configuration

Add the following to `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "postctl": {
      "command": "postctl",
      "args": ["mcp"]
    }
  }
}
```

Restart Claude Desktop after saving. The `postctl` binary must be on your `PATH`.

### MCP Tools

| Tool | Parameters | Description |
|------|------------|-------------|
| `list_posts` | `platform`, `status`, `campaign` (all optional) | List posts with optional filters |
| `get_post` | `id` | Retrieve full post content and metadata by ID |
| `create_post` | `platform`, `body`, `title`, `campaign`, `schedule` | Create a draft or scheduled post |
| `publish_post` | `id`, `dry_run` | Publish a post immediately |
| `schedule_post` | `id`, `schedule` (RFC3339) | Set or update the scheduled publish time |
| `list_campaigns` | — | List all campaigns with total count and per-status breakdown |
| `get_campaign` | `name`, `status` (optional filter) | Get all posts in a campaign with full content |

For Twitter threads, separate tweets with `\n---\n` in the `body` field when calling `create_post`.

Schedule values must be RFC3339, e.g. `2026-07-10T09:00:00+02:00`.

### AI Workflow Examples

**Plan a campaign from an article**

> "Read the article at https://example.com/blog/launch, then create a five-post campaign called `launch-week` with one post per day starting Monday. Use twitter for three posts and linkedin for two."

Claude calls `create_post` for each post with appropriate bodies, the campaign name, and staggered `schedule` values derived from the article content.

**Review and publish scheduled posts**

> "Show me everything scheduled for this week and publish any posts that look ready."

Claude calls `list_posts` with `status: scheduled`, presents the results for your review, then calls `publish_post` for each approved post — or all of them at once if you confirm.

**Repurpose a blog post across platforms**

> "Take post abc123 and create adapted versions for LinkedIn and Threads."

Claude calls `get_post` to retrieve the original, then calls `create_post` twice — once for `linkedin` and once for `threads` — adapting tone and length for each platform automatically.

---

## Platform Notes

| Platform | Character Limit | Threads | Images |
|----------|-----------------|---------|--------|
| Twitter/X | 280 per tweet | Yes, separate with `---` | Supported |
| LinkedIn | ~3,000 recommended | No | Supported |
| Threads | 500 | No | At least one recommended (Meta requirement) |
| Mastodon | 500 (instance default) | No | Supported |
| Bluesky | 300 | No | Supported |

Twitter threads have no hard post count limit, but keep threads focused. LinkedIn, Threads, Mastodon, and Bluesky do not support thread-style multi-part posts — use a single body for those platforms.

> [!WARNING]
> **API Rate Limits & Bulk Publishing:** Publishing multiple posts simultaneously or in quick succession can lead to API rate limits or permanent account bans (especially on federated networks like Mastodon). Always space out posts over time (e.g., at least 15-30 minutes delay between consecutive publishing events).

---

## Architecture

```
Markdown files
      |
   postctl import
      |
      v
SQLite  (~/.local/share/postctl/postctl.db)
      |
      +---> TUI (Bubbletea)    ---> Platform APIs  (Twitter, LinkedIn, Threads, Mastodon, Bluesky)
      |
      +---> MCP server (stdio) ---> AI agents  (Claude Desktop, etc.)
      |
      +---> postctl daemon     ---> scheduled publish via platform APIs
```

**Requirements:** macOS or Linux · Go 1.21+ · API credentials for each platform you use

---

## License

See [LICENSE](LICENSE).
