---
platform: facebook
campaign: hn-client-deepdive
schedule: 2026-07-07 09:00
joplin_id: e37a5404bbe440fe90da6d0cdd98f0da
updated_at: '2026-08-07T19:27:09.992Z'
---
Rich Monospace Styling: HTML Comment Sanitization in hn-client 🎨

Most command line tools display raw HTML comments, rendering tags like `<pre>` or `<code>` literally. hn-client solves this by integrating a custom HTML parser that sanitizes text and uses Lipgloss styles to render borders, inline links, code blocks, and indentation.

Check out the code:
👉 https://github.com/aeon022/hn-client
