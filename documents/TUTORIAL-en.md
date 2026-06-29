# Go Tutorial — postctl

This tutorial guides you through learning Go using the postctl project. Each phase builds a feature and teaches new Go concepts.

## Course Concept

This tutorial is written in parallel with the project and can be published as a paid course:

- **Format**: Video series (screen recording + voiceover) + written tutorial
- **Title**: "Build a Real CLI Tool in Go — From Zero to Production"
- **Target Audience**: Developers who want to learn Go by building a real project
- **Teaser**: Phase 1 + 2 free as blog posts on Dev.to (traffic + leads)
- **Full Version**: Phase 3-7 as a paid course ($49-79)
- **Platform**: Self-hosted site (Lemon Squeezy) or Udemy
- **Languages**: German (primary) + English

Each phase ends with a **Challenge** — a task the learner solves themselves before seeing the solution.

---

## Prerequisites

```bash
# Install Go
brew install go

# Verify version (1.22+)
go version

# Editor: VS Code + Go Extension (gopls)
code --install-extension golang.go
```

## Phase 1: Hello Go — Project Setup

### What we build
Basic structure with Cobra CLI: `postctl version`, `postctl help`

### Go Concepts
- **Go Modules**: Dependency Management (`go.mod`)
- **Packages**: Code organization in folders
- **`func main()`**: Entry Point
- **`fmt.Println`**: Output
- **Strings**: String formatting with `fmt.Sprintf`

### Steps

```bash
# 1. Create project
mkdir postctl && cd postctl
go mod init github.com/aeon022/postctl

# 2. Install Cobra
go get github.com/spf13/cobra@latest

# 3. Create folders
mkdir -p cmd
```

**`main.go`** — Entry Point:
```go
package main

import "github.com/aeon022/postctl/cmd"

func main() {
    cmd.Execute()
}
```

**`cmd/root.go`** — Root Command:
```go
package cmd

import (
    "fmt"
    "os"

    "github.com/spf13/cobra"
)

// Version — set at build time
var Version = "dev"

var rootCmd = &cobra.Command{
    Use:   "postctl",
    Short: "Social media posting from the terminal",
    Long:  "postctl manages social media posts from Markdown files.\nTwitter/X, LinkedIn, and Threads — from one CLI.",
}

// Execute — called by main()
func Execute() {
    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func init() {
    // Version subcommand
    rootCmd.AddCommand(&cobra.Command{
        Use:   "version",
        Short: "Print version",
        Run: func(cmd *cobra.Command, args []string) {
            fmt.Printf("postctl version %s\n", Version)
        },
    })
}
```

### Challenge 1
Add a command `postctl hello <name>` that prints `Hello <name>! Welcome to Go.`. (Tip: Check `cobra.ExactArgs(1)`).

---

## Phase 2: File Reader — Markdown Parser

### What we build
A parser that reads a Markdown file, extracts YAML Frontmatter (metadata like platform, schedule, images) and the content body.

### Go Concepts
- **Structs**: Custom data types for Post/Tweet
- **File I/O**: `os.ReadFile`, `os.Open`
- **Error Handling**: `error` interface and `if err != nil`
- **Strings/Slices**: `strings.Split`, `strings.HasPrefix`
- **Pointers**: Passing by reference vs value

### Code Example: Metadata Structs
```go
package models

import "time"

type Post struct {
    ID          string
    Platform    string    // twitter, linkedin, threads
    Type        string    // single, thread
    Campaign    string
    ScheduledAt *time.Time
    Body        string
    Images      []string
    Status      string    // draft, scheduled, posted, failed
    Error       string
}
```

### Challenge 2
Write a function `ParseFrontmatter(content string) (models.Post, error)` that splits the YAML block (between `---`) and parses key-value pairs manually without using external libraries.

---

## Phase 3: Local Database — SQLite Integration

### What we build
Save imported posts to a local SQLite database. Update post status (`draft`, `scheduled`, `posted`) and save publishing history.

### Go Concepts
- **Database Connection**: `database/sql` package
- **Driver**: Go SQLite driver (CGO-free via `modernc.org/sqlite`)
- **SQL Operations**: `CREATE TABLE`, `INSERT`, `UPDATE`, `SELECT`
- **Time Parsing**: `time.Parse` and time zone handling

### Challenge 3
Create a database table `posts` and write a method `SavePost(post models.Post) error` that inserts a new post or updates an existing one if the ID already exists (UPSERT).

---

## Phase 4: API Clients — Publishing to Platforms

### What we build
HTTP client implementations for Twitter/X, LinkedIn (OIDC), and Threads API.

### Go Concepts
- **Interfaces**: Define a `Platform` interface that all clients implement:
  ```go
  type Platform interface {
      Auth(ctx context.Context) error
      Post(ctx context.Context, post *models.Post) (string, error)
      IsAuthenticated(ctx context.Context) bool
  }
  ```
- **HTTP Requests**: `net/http` package, custom headers, JSON payloads
- **OAuth 2.0 Flow**: Token exchange and local web callback server

### Challenge 4
Implement the LinkedIn client using the `openid` and `w_member_social` scopes to publish a simple status message.

---

## Phase 5: Browser Automation — Headless Fallback

### What we build
A headless browser fallback for X/Twitter when API calls fail, utilizing `chromedp` to set login cookies, navigate the web interface, type tweets, and post them.

### Go Concepts
- **Context**: Timeout and cancellation propagation
- **Goroutines & Channels**: Network response interception to capture GQL tweet IDs
- **DevTools Protocol**: Driving Chrome via CDP

### Challenge 5
Set Chrome flags to bypass automated browser checks (e.g., hiding `navigator.webdriver` via `AutomationControlled`).

---

## Phase 6: Terminal User Interface (TUI) — Bubble Tea

### What we build
An interactive terminal dashboard with tabs (Dashboard, Posts, Schedule, Settings) showing campaigns, posts, and scheduler status.

### Go Concepts
- **Elm Architecture**: Model-Update-View pattern in CLI
- **Bubble Tea Framework**: `tea.Model`, `tea.Msg`, `tea.Cmd`
- **Terminal Rendering**: `lipgloss` for styling, borders, and layouts

### Challenge 6
Build a custom text input field that opens an external Vim/Neovim editor, suspends the TUI, and restores the terminal state upon editor exit.

---

## Phase 7: Background Scheduler — Daemon

### What we build
A background daemon (`postctl daemon`) that runs in the background, checks the database every 10 seconds, and publishes due posts automatically.

### Go Concepts
- **Ticker**: `time.NewTicker` for periodic tasks
- **Signals**: Catching OS signals (`SIGINT`, `SIGTERM`) for graceful shutdown
- **File Locks**: Prevent starting multiple scheduler processes concurrently

### Challenge 7
Implement a locking mechanism using a pidfile (`postctl.pid`) to ensure only one daemon runs at a time.
