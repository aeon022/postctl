package generator

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// RepurposedPostData holds the title and body of the repurposed social post.
type RepurposedPostData struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// RepurposeResult holds the repurposed posts for the target platforms and a suggested slug.
type RepurposeResult struct {
	Slug  string                        `json:"slug"`
	Posts map[string]RepurposedPostData `json:"posts"`
}

const repurposeSystemPrompt = `You are an expert social media copywriter and content repurposer.
Your task is to take an existing social media post and rewrite/repurpose it for one or more target platforms.

Requirements for each target platform:
- For 'twitter' target (thread):
  - Must start with a strong hook tweet.
  - Split into a logical sequence of tweets. Each tweet must be separated by '## Tweet 1', '## Tweet 2', etc.
  - The final tweet must be a reply (separated by '## Reply') containing a call to action or link.
  - Each tweet must be strictly under 280 characters. Note that a URL counts as exactly 23 characters.
- For 'linkedin' target (single):
  - Engaging, professional, and scannable. Use short paragraphs, bullet points, and appropriate hashtags.
  - This is a single post, not a thread.
- For 'threads' target (single):
  - Similar to Twitter but as a single concise post. Friendly, engaging, conversational.

You must return ONLY a raw JSON object matching this schema:
{
  "slug": "suggested-lowercase-url-friendly-slug",
  "posts": {
    // Generate only the requested target platforms here:
    // "twitter": { "title": "...", "content": "..." },
    // "linkedin": { "title": "...", "content": "..." },
    // "threads": { "title": "...", "content": "..." }
  }
}

Do not include any chat prefix/suffix or markdown formatting like ` + "`" + "```json" + "`" + ` or similar outside the JSON object. Return ONLY the JSON object. Keep the output clean and parseable by standard JSON decoders.`

// RepurposeContent contacts the LLM API to repurpose a post to the target platforms.
func RepurposeContent(ctx context.Context, cfg GeneratorConfig, srcPlatform, srcType, srcTitle, srcContent string, targets []string, tone string) (*RepurposeResult, error) {
	targetsStr := strings.Join(targets, ", ")
	tonePrompt := ""
	if tone != "" {
		tonePrompt = fmt.Sprintf("\nTone of Voice / Schreibstil: %s\nBitte passe den Schreibstil des Ziel-Beitrags entsprechend an.", tone)
	}
	userPrompt := fmt.Sprintf(
		"Source Post Metadata:\n- Platform: %s\n- Type: %s\n- Title: %s\n\nSource Post Content:\n%s\n\nTarget Platforms to generate: %s\n%s\n\nPlease repurpose the post to the specified target platforms.",
		srcPlatform, srcType, srcTitle, srcContent, targetsStr, tonePrompt,
	)

	var rawResponse string
	var err error

	provider := strings.ToLower(cfg.Provider)
	switch provider {
	case "claude":
		rawResponse, err = callClaude(ctx, cfg, repurposeSystemPrompt, userPrompt)
	case "ollama":
		rawResponse, err = callOllama(ctx, cfg, repurposeSystemPrompt, userPrompt)
	case "openai", "":
		rawResponse, err = callOpenAI(ctx, cfg, repurposeSystemPrompt, userPrompt)
	default:
		return nil, fmt.Errorf("unknown ai provider: %s", cfg.Provider)
	}

	if err != nil {
		return nil, err
	}

	// Clean and parse JSON response
	cleaned := cleanJSONResponse(rawResponse)
	var result RepurposeResult
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return nil, fmt.Errorf("failed to parse AI response as JSON: %w (Raw response: %q)", err, rawResponse)
	}

	// Ensure slug is clean
	if result.Slug == "" {
		result.Slug = "repurposed-post"
	}
	result.Slug = CleanSlug(result.Slug)

	return &result, nil
}

// SaveRepurposedToMarkdownFiles writes the repurposed posts into separate markdown files in the specified directory.
func SaveRepurposedToMarkdownFiles(result *RepurposeResult, dir string, campaign string) ([]string, error) {
	var writtenFiles []string

	for platform, data := range result.Posts {
		postType := "single"
		if platform == "twitter" {
			postType = "thread"
		}

		fm := FileFrontmatter{
			Platform: platform,
			Type:     postType,
			Title:    data.Title,
			Campaign: campaign,
		}

		fmBytes, err := marshalYAML(fm)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal frontmatter for platform %s: %w", platform, err)
		}

		// Construct final Markdown content
		var sb strings.Builder
		sb.WriteString("---\n")
		sb.Write(fmBytes)
		sb.WriteString("---\n")
		sb.WriteString(data.Content)
		sb.WriteString("\n")

		fileName := fmt.Sprintf("%s-repurposed-to-%s.md", result.Slug, platform)
		filePath := fileName
		if dir != "" {
			filePath = fmt.Sprintf("%s/%s", dir, fileName)
		}

		if err := writeFile(filePath, []byte(sb.String())); err != nil {
			return nil, fmt.Errorf("failed to write file %s: %w", filePath, err)
		}
		writtenFiles = append(writtenFiles, filePath)
	}

	return writtenFiles, nil
}
