---
platform: facebook
campaign: hn-client-deepdive
schedule: 2026-07-07 09:00
---
Rich Monospace Styling: HTML Comment Sanitization in hn-client 🎨

Most command line tools display raw HTML comments, rendering tags like `<pre>` or `<code>` literally. hn-client solves this by integrating a custom HTML parser that sanitizes text and uses Lipgloss styles to render borders, inline links, code blocks, and indentation.

Check out the code:
👉 https://github.com/aeon022/hn-client
