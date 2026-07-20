---
platform: twitter
type: thread
campaign: hn-client-deepdive
schedule: 2026-07-07 09:00
---
Hacker News comments are notorious for raw HTML tags, embedded links, and code snippets. How do you display them cleanly in a monospace terminal?

In `hn-client`, we built a custom HTML parser & renderer using Lipgloss to make comments highly readable. 🧵
---
The pipeline:
1. Parse the HTML comment text using `golang.org/x/net/html`.
2. Extract and format links (`<a>`) and inline code (`<code>`).
3. Render nested comment indentations with color-coded border lines based on nesting depth.
---
This turns raw HTML strings like `<i>hello</i>` or `<pre>code</pre>` into styled, syntax-highlighted terminal outputs.

Plus, we persist read states and flag fresh replies using `[NEU]` tags!
---
Beautiful terminal typography. Check out the project and star it:
👉 https://github.com/aeon022/hn-client
