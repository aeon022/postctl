package generator

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var (
	titleRegex   = regexp.MustCompile(`(?i)<title[^>]*>([\s\S]*?)<\/title>`)
	scriptRegex  = regexp.MustCompile(`(?i)<script[^>]*>([\s\S]*?)<\/script>`)
	styleRegex   = regexp.MustCompile(`(?i)<style[^>]*>([\s\S]*?)<\/style>`)
	tagRegex     = regexp.MustCompile(`<[^>]+>`)
	commentRegex = regexp.MustCompile(`<!--[\s\S]*?-->`)
	spaceRegex   = regexp.MustCompile(`\s+`)
)

// ScrapedContent represents the extracted title and content of a web page
type ScrapedContent struct {
	Title   string
	Content string
}

// ScrapeURL fetches a URL and extracts clean text content and the title.
func ScrapeURL(url string) (*ScrapedContent, error) {
	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Set standard headers to prevent being blocked by anti-bot measures
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-OK status: %s", resp.Status)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	html := string(bodyBytes)

	// Extract Title
	var title string
	if titleMatch := titleRegex.FindStringSubmatch(html); len(titleMatch) > 1 {
		title = strings.TrimSpace(titleMatch[1])
	}
	if title == "" {
		title = "Generated Post"
	}

	// Clean HTML to get clean text
	cleaned := html
	cleaned = scriptRegex.ReplaceAllString(cleaned, " ")
	cleaned = styleRegex.ReplaceAllString(cleaned, " ")
	cleaned = commentRegex.ReplaceAllString(cleaned, " ")
	cleaned = tagRegex.ReplaceAllString(cleaned, " ")
	cleaned = spaceRegex.ReplaceAllString(cleaned, " ")
	cleaned = strings.TrimSpace(cleaned)

	// Limit clean content to a reasonable length for the LLM context (e.g. 50k characters)
	if len(cleaned) > 50000 {
		cleaned = cleaned[:50000]
	}

	return &ScrapedContent{
		Title:   title,
		Content: cleaned,
	}, nil
}
