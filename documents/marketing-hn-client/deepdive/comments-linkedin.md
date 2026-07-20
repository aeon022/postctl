---
platform: linkedin
campaign: hn-client-deepdive
schedule: 2026-07-07 09:00
---
🎨 MONOSPACE TYPOGRAPHY: Styling raw HTML comments for the Terminal

Displaying structured text inside a terminal is easy—until you encounter arbitrary web inputs. Hacker News comments contain arbitrary HTML tags (like `<i>`, `<code>`, `<pre>`, and `<a>`), which will render as raw strings if not handled correctly.

To solve this in **hn-client**, we built a custom HTML sanitization and styling renderer.

Here is our layout pipeline:
1️⃣ HTML Tokenization: We parse the text tree using the Go package `golang.org/x/net/html` to separate text nodes from tags.
2️⃣ Style Mapping: Tags like `<code>` and `<pre>` are mapped to custom Lipgloss styles, providing distinct background boxes and padding. Links (`<a>`) are stripped of their verbose HTML tags and rendered as underlined URLs.
3️⃣ Color-Coded Nesting: Nested comment threads are indented and bounded by vertical border rules. The border colors cycle dynamically (e.g. Cyan -> Purple -> Gray) depending on the comment depth, making long sub-threads easy to scan.
4️⃣ Fresh Reply Indicators: Unread comments are automatically marked with a green `[NEU]` badge, using a local history file (`~/.hn-history.json`).

Terminal aesthetics matter just as much as GUI web styling.

👉 Read comments in terminal-comfort: https://github.com/aeon022/hn-client

#golang #tui #typography #lipgloss #webscraping #design #opensource
