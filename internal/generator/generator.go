package generator

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// GeneratedPostData holds the title and body of the generated social post.
type GeneratedPostData struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// GeneratedPosts holds the posts for all three target platforms and a suggested slug.
type GeneratedPosts struct {
	Slug     string            `json:"slug"`
	Twitter  GeneratedPostData `json:"twitter"`
	LinkedIn GeneratedPostData `json:"linkedin"`
	Threads  GeneratedPostData `json:"threads"`
}

// GeneratorConfig holds the AI provider configuration.
type GeneratorConfig struct {
	Provider string
	APIKey   string
	Model    string
	BaseURL  string
}

// OpenAIRequest represents the payload for OpenAI/Ollama Chat Completions API.
type OpenAIRequest struct {
	Model       string          `json:"model"`
	Messages    []OpenAIMessage `json:"messages"`
	Temperature float64         `json:"temperature"`
}

type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAIResponse represents the response from OpenAI/Ollama Chat Completions API.
type OpenAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// ClaudeRequest represents the payload for Anthropic Claude Messages API.
type ClaudeRequest struct {
	Model       string          `json:"model"`
	System      string          `json:"system,omitempty"`
	Messages    []ClaudeMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens"`
	Temperature float64         `json:"temperature"`
}

type ClaudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ClaudeResponse represents the response from Anthropic Claude Messages API.
type ClaudeResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

const systemPrompt = `You are an expert social media manager and content repurposer.
Your job is to read the provided text content of a web page/article and repurpose it into three high-performing social media posts:
1. A Twitter/X thread (platform: twitter, type: thread)
2. A LinkedIn post (platform: linkedin, type: single)
3. A Threads post (platform: threads, type: single)

Requirements:
- Twitter/X Thread:
  - Must start with a strong hook tweet.
  - Split into a logical sequence of tweets. Each tweet must be separated by '## Tweet 1', '## Tweet 2', etc.
  - The final tweet must be a reply (separated by '## Reply') containing a call to action or reference to the source.
  - Each tweet must be strictly under 280 characters. Note that a URL counts as exactly 23 characters, so if a tweet contains a URL, the other characters must be under 257.
- LinkedIn Post:
  - Engaging, professional, and scannable. Use short paragraphs, bullet points, and appropriate hashtags.
  - This is a single post, not a thread.
- Threads Post:
  - Similar to Twitter but as a single concise post. Friendly, engaging, conversational.

You must return ONLY a raw JSON object matching this schema:
{
  "slug": "suggested-lowercase-url-friendly-slug",
  "twitter": {
    "title": "A short descriptive title for the twitter post",
    "content": "## Tweet 1\n[Content of tweet 1]\n\n## Tweet 2\n[Content of tweet 2]\n\n## Reply\n[Content of the reply tweet with CTA/Link]"
  },
  "linkedin": {
    "title": "A short descriptive title for the linkedin post",
    "content": "[Engaging body of LinkedIn post]"
  },
  "threads": {
    "title": "A short descriptive title for the threads post",
    "content": "[Engaging body of Threads post]"
  }
}

Do not include any chat prefix/suffix or markdown formatting like ` + "`" + "```json" + "`" + ` or similar outside the JSON object. Return ONLY the JSON object. Keep the output clean and parseable by standard JSON decoders.`

// GenerateContent contacts the configured LLM API and generates the content.
func GenerateContent(ctx context.Context, cfg GeneratorConfig, sourceURL, title, text string) (*GeneratedPosts, error) {
	userPrompt := fmt.Sprintf("Source URL: %s\nTitle: %s\nContent:\n%s", sourceURL, title, text)

	var rawResponse string
	var err error

	provider := strings.ToLower(cfg.Provider)
	switch provider {
	case "claude":
		rawResponse, err = callClaude(ctx, cfg, systemPrompt, userPrompt)
	case "ollama":
		rawResponse, err = callOllama(ctx, cfg, systemPrompt, userPrompt)
	case "openai", "":
		rawResponse, err = callOpenAI(ctx, cfg, systemPrompt, userPrompt)
	default:
		return nil, fmt.Errorf("unknown ai provider: %s", cfg.Provider)
	}

	if err != nil {
		return nil, err
	}

	// Clean and parse JSON response
	cleaned := cleanJSONResponse(rawResponse)
	var posts GeneratedPosts
	if err := json.Unmarshal([]byte(cleaned), &posts); err != nil {
		return nil, fmt.Errorf("failed to parse AI response as JSON: %w (Raw response: %q)", err, rawResponse)
	}

	// Ensure fields are not empty
	if posts.Slug == "" {
		posts.Slug = "generated-post"
	}
	// Slugify slug to be extra safe
	posts.Slug = CleanSlug(posts.Slug)

	return &posts, nil
}

func callOpenAI(ctx context.Context, cfg GeneratorConfig, sysPrompt, userPrompt string) (string, error) {
	url := cfg.BaseURL
	if url == "" {
		url = "https://api.openai.com/v1/chat/completions"
	}

	model := cfg.Model
	if model == "" {
		model = "gpt-4o-mini"
	}

	payload := OpenAIRequest{
		Model: model,
		Messages: []OpenAIMessage{
			{Role: "system", Content: sysPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.7,
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal openai payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return "", fmt.Errorf("create openai request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("execute openai request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read openai response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var openAIResp OpenAIResponse
		_ = json.Unmarshal(bodyBytes, &openAIResp)
		if openAIResp.Error != nil && openAIResp.Error.Message != "" {
			return "", fmt.Errorf("openai API error (%s): %s", resp.Status, openAIResp.Error.Message)
		}
		return "", fmt.Errorf("openai API returned non-OK status: %s (response: %s)", resp.Status, string(bodyBytes))
	}

	var openAIResp OpenAIResponse
	if err := json.Unmarshal(bodyBytes, &openAIResp); err != nil {
		return "", fmt.Errorf("unmarshal openai response: %w", err)
	}

	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf("openai API returned 0 choices")
	}

	return openAIResp.Choices[0].Message.Content, nil
}

func callClaude(ctx context.Context, cfg GeneratorConfig, sysPrompt, userPrompt string) (string, error) {
	url := cfg.BaseURL
	if url == "" {
		url = "https://api.anthropic.com/v1/messages"
	}

	model := cfg.Model
	if model == "" {
		model = "claude-3-5-sonnet-latest"
	}

	payload := ClaudeRequest{
		Model: model,
		System: sysPrompt,
		Messages: []ClaudeMessage{
			{Role: "user", Content: userPrompt},
		},
		MaxTokens:   4000,
		Temperature: 0.7,
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal claude payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return "", fmt.Errorf("create claude request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", cfg.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("execute claude request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read claude response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var claudeResp ClaudeResponse
		_ = json.Unmarshal(bodyBytes, &claudeResp)
		if claudeResp.Error != nil && claudeResp.Error.Message != "" {
			return "", fmt.Errorf("claude API error (%s): %s", resp.Status, claudeResp.Error.Message)
		}
		return "", fmt.Errorf("claude API returned non-OK status: %s (response: %s)", resp.Status, string(bodyBytes))
	}

	var claudeResp ClaudeResponse
	if err := json.Unmarshal(bodyBytes, &claudeResp); err != nil {
		return "", fmt.Errorf("unmarshal claude response: %w", err)
	}

	if len(claudeResp.Content) == 0 || claudeResp.Content[0].Type != "text" {
		return "", fmt.Errorf("claude API did not return text content")
	}

	return claudeResp.Content[0].Text, nil
}

func callOllama(ctx context.Context, cfg GeneratorConfig, sysPrompt, userPrompt string) (string, error) {
	// Ollama implements OpenAI compatible chat completions
	// If base_url is not set, we default to the standard localhost Ollama endpoint
	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://localhost:11434/v1/chat/completions"
	}
	return callOpenAI(ctx, cfg, sysPrompt, userPrompt)
}

func cleanJSONResponse(raw string) string {
	raw = strings.TrimSpace(raw)

	// Strip markdown code block wrapper if present
	if strings.HasPrefix(raw, "```json") {
		raw = strings.TrimPrefix(raw, "```json")
		raw = strings.TrimSuffix(raw, "```")
	} else if strings.HasPrefix(raw, "```") {
		raw = strings.TrimPrefix(raw, "```")
		raw = strings.TrimSuffix(raw, "```")
	}

	return strings.TrimSpace(raw)
}

// CleanSlug converts a string to a safe filename/id slug.
func CleanSlug(s string) string {
	s = strings.ToLower(s)
	// Replace non-alphanumeric/hyphen/underscore with hyphen
	var result strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			result.WriteRune(r)
		} else {
			result.WriteRune('-')
		}
	}
	res := result.String()
	// Replace multiple hyphens with single hyphen
	for strings.Contains(res, "--") {
		res = strings.ReplaceAll(res, "--", "-")
	}
	return strings.Trim(res, "-")
}

type FileFrontmatter struct {
	Platform string   `yaml:"platform"`
	Type     string   `yaml:"type"`
	Title    string   `yaml:"title,omitempty"`
	Campaign string   `yaml:"campaign,omitempty"`
}

// SaveToMarkdownFiles writes the generated posts into three separate markdown files in the specified directory.
func SaveToMarkdownFiles(posts *GeneratedPosts, dir string, campaign string) ([]string, error) {
	// Import "path/filepath" and "os" and "gopkg.in/yaml.v3"
	// To make sure they are available, we should import yaml.v3 in generator.go.
	// Let's do that in a replace chunk or ensure it's imported.
	platforms := []struct {
		name     string
		plat     string
		postType string
		data     GeneratedPostData
	}{
		{"twitter", "twitter", "thread", posts.Twitter},
		{"linkedin", "linkedin", "single", posts.LinkedIn},
		{"threads", "threads", "single", posts.Threads},
	}

	var writtenFiles []string

	for _, p := range platforms {
		fm := FileFrontmatter{
			Platform: p.plat,
			Type:     p.postType,
			Title:    p.data.Title,
			Campaign: campaign,
		}

		fmBytes, err := marshalYAML(fm)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal frontmatter for %s: %w", p.name, err)
		}

		// Construct final Markdown content
		var sb strings.Builder
		sb.WriteString("---\n")
		sb.Write(fmBytes)
		sb.WriteString("---\n")
		sb.WriteString(p.data.Content)
		sb.WriteString("\n")

		fileName := fmt.Sprintf("%s-%s.md", posts.Slug, p.name)
		var filePath string
		if dir != "" {
			// clean path helper
			filePath = filepath.Clean(filepath.Join(dir, fileName))
		} else {
			filePath = fileName
		}

		if err := writeFile(filePath, []byte(sb.String())); err != nil {
			return nil, fmt.Errorf("failed to write file %s: %w", filePath, err)
		}
		writtenFiles = append(writtenFiles, filePath)
	}

	return writtenFiles, nil
}

func marshalYAML(v interface{}) ([]byte, error) {
	return yaml.Marshal(v)
}

func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}

// GenerateAltText liest ein Bild ein und generiert eine Barrierefreiheits-Beschreibung via Vision LLM
func GenerateAltText(ctx context.Context, cfg GeneratorConfig, imagePath string) (string, error) {
	if cfg.APIKey == "" && strings.ToLower(cfg.Provider) != "ollama" {
		if strings.ToLower(cfg.Provider) == "claude" {
			cfg.APIKey = os.Getenv("ANTHROPIC_API_KEY")
		} else {
			cfg.APIKey = os.Getenv("OPENAI_API_KEY")
		}
		if cfg.APIKey == "" {
			return "", nil
		}
	}

	fileBytes, err := os.ReadFile(imagePath)
	if err != nil {
		return "", fmt.Errorf("read image file: %w", err)
	}

	mimeType := "image/jpeg"
	if strings.HasSuffix(strings.ToLower(imagePath), ".png") {
		mimeType = "image/png"
	} else if strings.HasSuffix(strings.ToLower(imagePath), ".gif") {
		mimeType = "image/gif"
	} else if strings.HasSuffix(strings.ToLower(imagePath), ".webp") {
		mimeType = "image/webp"
	}

	base64Data := base64.StdEncoding.EncodeToString(fileBytes)

	provider := strings.ToLower(cfg.Provider)
	if provider == "claude" {
		return callClaudeVision(ctx, cfg, base64Data, mimeType)
	} else if provider == "openai" || provider == "" {
		return callOpenAIVision(ctx, cfg, base64Data, mimeType)
	}

	return "", nil
}

func callOpenAIVision(ctx context.Context, cfg GeneratorConfig, base64Data, mimeType string) (string, error) {
	url := cfg.BaseURL
	if url == "" {
		url = "https://api.openai.com/v1/chat/completions"
	}

	model := cfg.Model
	if model == "" || model == "gpt-4o-mini" {
		model = "gpt-4o-mini"
	}

	payload := map[string]interface{}{
		"model": model,
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": "Describe this image in detail, but concisely, to be used as social media alt text (accessible description). Return ONLY the description, no intro or quotes.",
					},
					{
						"type": "image_url",
						"image_url": map[string]interface{}{
							"url": fmt.Sprintf("data:%s;base64,%s", mimeType, base64Data),
						},
					},
				},
			},
		},
		"max_tokens": 150,
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("openai vision status %d: %s", resp.StatusCode, string(body))
	}

	var openAIResp OpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
		return "", err
	}

	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf("no choice returned")
	}

	return strings.TrimSpace(openAIResp.Choices[0].Message.Content), nil
}

func callClaudeVision(ctx context.Context, cfg GeneratorConfig, base64Data, mimeType string) (string, error) {
	url := cfg.BaseURL
	if url == "" {
		url = "https://api.anthropic.com/v1/messages"
	}

	model := cfg.Model
	if model == "" {
		model = "claude-3-5-sonnet-latest"
	}

	payload := map[string]interface{}{
		"model":      model,
		"max_tokens": 150,
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type": "image",
						"source": map[string]interface{}{
							"type":       "base64",
							"media_type": mimeType,
							"data":       base64Data,
						},
					},
					{
						"type": "text",
						"text": "Describe this image in detail, but concisely, to be used as social media alt text (accessible description). Return ONLY the description, no intro or quotes.",
					},
				},
			},
		},
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return "", err
	}

	req.Header.Set("x-api-key", cfg.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("claude vision status %d: %s", resp.StatusCode, string(body))
	}

	var claudeResp ClaudeResponse
	if err := json.NewDecoder(resp.Body).Decode(&claudeResp); err != nil {
		return "", err
	}

	if len(claudeResp.Content) == 0 {
		return "", fmt.Errorf("no content returned")
	}

	return strings.TrimSpace(claudeResp.Content[0].Text), nil
}

