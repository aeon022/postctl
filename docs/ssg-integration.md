# Static Site Generator (SSG) Integration Guide for postctl

This guide explains how to connect your blog or static website (Hugo, Jekyll, Astro) with `postctl` to automatically generate and queue social media posts when you write blog posts.

---

## 🛠️ The Architecture

The pipeline extracts social media frontmatter or automated teasers from your blog posts during the site build step (or via a local file watcher/git hook) and imports them into `postctl`.

```
[Blog Post Markdown] ---> [SSG Build / Script] ---> [postctl Markdown] ---> postctl import
```

---

## 1. Astro Integration

For Astro, you can create a custom script that runs before or after `astro build`. It reads your content collection, extracts frontmatter fields, and generates files in the `posts/` directory watched by `postctl`.

Create `scripts/export-socials.js` in your Astro project:

```javascript
import fs from 'fs';
import path from 'path';
import matter from 'gray-matter'; // npm i gray-matter

const BLOG_DIR = './src/content/blog';
const OUTPUT_DIR = './posts';

if (!fs.existsSync(OUTPUT_DIR)) {
  fs.mkdirSync(OUTPUT_DIR);
}

const files = fs.readdirSync(BLOG_DIR).filter(f => f.endsWith('.md') || f.endsWith('.mdx'));

files.forEach(file => {
  const content = fs.readFileSync(path.join(BLOG_DIR, file), 'utf-8');
  const { data, content: body } = matter(content);

  // If social teasers are configured in frontmatter
  if (data.social_twitter || data.social_linkedin) {
    const slug = file.replace(/\.(md|mdx)$/, '');

    if (data.social_twitter) {
      const outputContent = `---
platform: twitter
campaign: blog-promo
title: "Promo for ${data.title}"
---
${data.social_twitter}
`;
      fs.writeFileSync(path.join(OUTPUT_DIR, `${slug}-twitter.md`), outputContent);
    }

    if (data.social_linkedin) {
      const outputContent = `---
platform: linkedin
campaign: blog-promo
title: "Promo for ${data.title}"
---
${data.social_linkedin}
`;
      fs.writeFileSync(path.join(OUTPUT_DIR, `${slug}-linkedin.md`), outputContent);
    }
  }
});

console.log('✓ Social posts extracted for postctl.');
```

Add it to your `package.json`:
```json
"scripts": {
  "build": "node scripts/export-socials.js && astro build"
}
```

---

## 2. Hugo Integration

In Hugo, you can use Hugo's **Custom Output Formats** to generate social posts automatically during build.

Add the following to your `hugo.toml`:

```toml
[outputFormats]
  [outputFormats.SocialPosts]
    mediaType = "text/markdown"
    baseName = "social-posts"
    isPlainText = true

[outputs]
  page = ["HTML", "SocialPosts"]
```

Then, create a template file at `layouts/_default/single.socialposts.md`:

```markdown
{{- if .Params.social_teaser -}}
---
platform: all
campaign: hugo-posts
title: "Teaser: {{ .Title }}"
schedule: queue
---
{{ .Params.social_teaser }}
{{- end -}}
```

When you run `hugo`, it will generate `social-posts.md` files in your public output folders, which you can easily import:
```bash
postctl import public/blog/*/social-posts.md
```

---

## 3. Jekyll Integration

For Jekyll, add a simple Python script `_scripts/extract_socials.py` to compile socials:

```python
import os
import yaml

POSTS_DIR = '_posts'
OUTPUT_DIR = 'posts'

os.makedirs(OUTPUT_DIR, exist_ok=True)

for file in os.listdir(POSTS_DIR):
    if not file.endswith('.md'):
        continue
    
    path = os.path.join(POSTS_DIR, file)
    with open(path, 'r', encoding='utf-8') as f:
        content = f.read()
        
    if content.startswith('---'):
        parts = content.split('---', 2)
        if len(parts) >= 3:
            metadata = yaml.safe_load(parts[1])
            if 'social_teaser' in metadata:
                slug = file.replace('.md', '')
                post_content = f"""---
platform: all
campaign: jekyll-promo
title: "Jekyll: {metadata.get('title')}"
schedule: queue
---
{metadata['social_teaser']}
"""
                with open(os.path.join(OUTPUT_DIR, f"{slug}.md"), 'w') as out:
                    out.write(post_content)

print("✓ Jekyll socials generated.")
```
