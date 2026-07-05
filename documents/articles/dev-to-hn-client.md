---
title: Why I Built a Terminal-Native Hacker News Client in Go (Using Bubble Tea)
published: false
description: Read Hacker News like a developer. Introducing hn-client, a fast, keyboard-driven terminal client written in Go with nested color-coded comments.
tags: go, open-source, terminal, showdev
---

# Why I Built a Terminal-Native Hacker News Client in Go (Using Bubble Tea)

As developers, we love our command-line tools. We run git, build code, tail logs, and edit configuration files without ever touching our mouse. 

Yet, when we want to check what's trending on **Hacker News**, we leave our development environment, open a browser tab, and start clicking. Soon, we are buried in 25 open tabs of comment sections and blog links.

I wanted a better way. I wanted to browse top stories, search Ask/Show HN, and read comment threads directly in my terminal—without leaving the keyboard.

So I built **hn-client** — a fast, minimalist Hacker News client written in Go using the **Bubble Tea** (TUI framework) and **Lipgloss** (terminal styling) libraries.

---

## 🚀 Key Features

*   **Keyboard & Mouse Navigation:** Vim-style `j`/`k` list navigation alongside seamless mouse scroll wheel support.
*   **Color-Coded Thread Indentation:** Tracing deep conversations is simplified with vertically styled, color-coded borders indicating nesting depth.
*   **HTML-to-Terminal Sanitizer:** Parses raw HTML comments, converts links to clickable terminal anchors, and formats `<code>`/`<pre>` blocks to respect monospace grids.
*   **Real-time Search:** Filter the stories feed dynamically by title, author, or category using `/`.
*   **Global History:** Remembers read stories across directories in `~/.hn-history.json` and grays them out in the feed.

---

## 🛠️ The Architecture: Go + Bubble Tea + Lipgloss

Building interactive terminal interfaces can be challenging because terminals handle layouts line-by-line. To build `hn-client`, I used the Elm-inspired **Bubble Tea** framework.

### The Model-View-Update (MVU) Loop
In Bubble Tea, everything revolves around three core components:
1.  **Model:** The state of your application (active tab, loaded stories, scroll offset, cursor position, active comment details).
2.  **Update:** A function that receives terminal events (key presses, mouse scrolls, API data loaded) and modifies the Model.
3.  **View:** A pure function that renders the Model as a single string of text.

Here is a simplified example of our Go event handler for vim-navigation:

```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "q", "ctrl+c":
            return m, tea.Quit
        case "j", "down":
            if m.cursor < len(m.stories)-1 {
                m.cursor++
            }
        case "k", "up":
            if m.cursor > 0 {
                m.cursor--
            }
        case "enter":
            return m, m.loadCommentsCmd()
        }
    }
    return m, nil
}
```

---

## 🎨 Parsing HTML for Monospaced Terminals

Hacker News comments are returned by the Firebase API as raw HTML snippets containing markup like `<p>`, `<code>`, `<pre>`, and `<a>` links. 

Simply stripping the tags creates unreadable blocks of text. The client implements a custom regex-based parser that:
1.  Converts paragraph tags (`<p>`) into standard terminal double newlines (`\n\n`).
2.  Identifies links (`<a href="...">...</a>`), extracts the destination URL, and underlines it using ANSI styles.
3.  Wraps preformatted blocks (`<pre><code>`) in distinct gray borders so code remains monospaced and readable.

### The Nested Comment Border Hack
To render the nested comment threads, the client utilizes Lipgloss borders. We define an array of color gradients:

```go
var CommentBorders = []lipgloss.Style{
    lipgloss.NewStyle().Border(lipgloss.Border{Left: "│"}).BorderForeground(lipgloss.Color("33")),  // Blue
    lipgloss.NewStyle().Border(lipgloss.Border{Left: "│"}).BorderForeground(lipgloss.Color("43")),  // Teal
    lipgloss.NewStyle().Border(lipgloss.Border{Left: "│"}).BorderForeground(lipgloss.Color("243")), // Gray
}
```
For each nested comment, we wrap the text dynamically in the styled border representing its depth:
```go
depthStyle := CommentBorders[comment.Depth % len(CommentBorders)]
renderedComment := depthStyle.PaddingLeft(2).Render(comment.Text)
```

This creates a beautiful, readable vertical outline for nested replies.

---

## Try it out!

`hn-client` is completely open-source (MIT licensed). You can install it on macOS or Linux using the quick installation script:

```bash
git clone https://github.com/aeon022/hn-client.git
cd hn-client
chmod +x setup.sh
./setup.sh
```

Check out the repository, give it a star, and let me know your thoughts on the Bubble Tea architecture!

*   **GitHub Repository:** [https://github.com/aeon022/hn-client](https://github.com/aeon022/hn-client)
