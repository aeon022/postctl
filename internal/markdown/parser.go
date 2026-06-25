package markdown

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/aeon022/postctl/internal/models"
	"gopkg.in/yaml.v3"
)

// StringOrSlice repräsentiert entweder einen einzelnen String oder eine Liste von Strings in YAML
type StringOrSlice []string

func (s *StringOrSlice) UnmarshalYAML(value *yaml.Node) error {
	var str string
	if err := value.Decode(&str); err == nil {
		*s = []string{str}
		return nil
	}

	var slice []string
	if err := value.Decode(&slice); err == nil {
		*s = slice
		return nil
	}

	return nil
}

// Frontmatter entspricht der YAML-Struktur am Anfang der Datei
type Frontmatter struct {
	Platform string        `yaml:"platform"` // twitter | linkedin | threads | all
	Type     string        `yaml:"type"`     // thread | single | article
	Language string        `yaml:"language"`
	Campaign string        `yaml:"campaign"`
	Schedule string        `yaml:"schedule"` // ISO 8601 oder YYYY-MM-DD HH:MM
	Images   StringOrSlice `yaml:"images"`
	Tags     []string      `yaml:"tags"`
	Title    string        `yaml:"title"`
}

var (
	headerRegex = regexp.MustCompile(`(?m)^##\s+(Tweet\s+\d+|Reply)\s*$`)
	tweetNumRegex = regexp.MustCompile(`\d+`)
	inlineImageRegex = regexp.MustCompile(`<!--\s*image:\s*([^\s-]+.*?)\s*-->`)
	urlRegex = regexp.MustCompile(`https?://[^\s]+`)
)

// ParseFile liest eine Markdown-Datei ein und gibt eine Liste von Posts zurück
// (ein Post pro Plattform, falls Platform == "all" oder mehrere Plattformen)
func ParseFile(path string) ([]models.Post, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	return ParseContent(string(data), path)
}

// ParseContent parst den Inhalt einer Markdown-Datei
func ParseContent(content, sourcePath string) ([]models.Post, error) {
	filename := filepath.Base(sourcePath)
	baseID := strings.TrimSuffix(filename, filepath.Ext(filename))
	// Bereinigen der ID (nur Kleinbuchstaben, Zahlen und Bindestriche)
	baseID = cleanID(baseID)

	// Frontmatter und Body trennen
	parts := strings.SplitN(content, "---", 3)
	var yamlStr, bodyStr string

	if len(parts) == 3 {
		yamlStr = parts[1]
		bodyStr = parts[2]
	} else if len(parts) == 2 && strings.HasPrefix(content, "---") {
		// Nur Frontmatter, kein Body oder umgekehrt
		yamlStr = parts[1]
	} else {
		bodyStr = content
	}

	var fm Frontmatter
	if yamlStr != "" {
		if err := yaml.Unmarshal([]byte(yamlStr), &fm); err != nil {
			return nil, fmt.Errorf("parse frontmatter: %w", err)
		}
	}

	// Defaults setzen
	if fm.Language == "" {
		fm.Language = "en"
	}
	if fm.Type == "" {
		fm.Type = "single"
	}

	var scheduledAt *time.Time
	if fm.Schedule != "" {
		parsed, err := ParseScheduleTime(fm.Schedule)
		if err != nil {
			return nil, fmt.Errorf("invalid schedule time %q: %w", fm.Schedule, err)
		}
		scheduledAt = &parsed
	}

	// Plattformen ermitteln
	var targetPlatforms []string
	pLower := strings.ToLower(strings.TrimSpace(fm.Platform))
	if pLower == "all" || pLower == "" {
		targetPlatforms = []string{models.PlatformTwitter, models.PlatformLinkedIn, models.PlatformThreads, models.PlatformMastodon, models.PlatformBluesky, models.PlatformFacebook}
	} else {
		// Komma-separiert erlauben
		rawPlats := strings.Split(pLower, ",")
		for _, rp := range rawPlats {
			plat := strings.TrimSpace(rp)
			if plat != "" {
				targetPlatforms = append(targetPlatforms, plat)
			}
		}
	}

	// Body parsen
	var posts []models.Post
	for _, platform := range targetPlatforms {
		title := fm.Title
		if title == "" {
			title = models.DeriveTitle(bodyStr)
		}

		post := models.Post{
			Platform:    platform,
			Type:        fm.Type,
			Language:    fm.Language,
			Campaign:    fm.Campaign,
			Title:       title,
			Images:      []string(fm.Images),
			Tags:        fm.Tags,
			ScheduledAt: scheduledAt,
			SourceFile:  sourcePath,
			Status:      models.StatusDraft,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		if scheduledAt != nil {
			post.Status = models.StatusScheduled
		}

		// Generiere deterministische ID
		post.ID = fmt.Sprintf("%s-%s", baseID, platform)

		// Inhalt parsen
		if fm.Type == "thread" || platform == models.PlatformTwitter || platform == models.PlatformMastodon || platform == models.PlatformBluesky {
			// Für Twitter/Mastodon/Bluesky oder explizite Threads teilen wir in Tweets auf
			post.Tweets = parseTweets(bodyStr, []string(fm.Images))
			
			// Titel aus erstem Tweet generieren, falls leer
			if post.Title == "" && len(post.Tweets) > 0 {
				lines := strings.Split(post.Tweets[0].Content, "\n")
				if len(lines) > 0 {
					post.Title = lines[0]
					if len(post.Title) > 50 {
						post.Title = post.Title[:47] + "..."
					}
				}
			}
		} else {
			// Single Post (z.B. LinkedIn, Threads)
			post.Body = strings.TrimSpace(bodyStr)
			if post.Title == "" {
				lines := strings.Split(post.Body, "\n")
				if len(lines) > 0 {
					post.Title = lines[0]
					if len(post.Title) > 50 {
						post.Title = post.Title[:47] + "..."
					}
				}
			}
		}

		posts = append(posts, post)
	}

	return posts, nil
}

// cleanID bereinigt den Dateinamen für eine saubere ID
func cleanID(s string) string {
	s = strings.ToLower(s)
	reg := regexp.MustCompile(`[^a-z0-9-_]`)
	s = reg.ReplaceAllString(s, "-")
	// Doppelte Bindestriche entfernen
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return strings.Trim(s, "-")
}

// ParseScheduleTime parst verschiedene Datumsformate
func ParseScheduleTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	formats := []string{
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
	}

	for _, format := range formats {
		if t, err := time.ParseInLocation(format, s, time.Local); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("could not parse time with any supported format")
}

// parseTweets zerlegt den Body in einzelne Tweets und weist Bilder zu
func parseTweets(body string, images []string) []models.Tweet {
	// Suchen nach Überschriften
	matches := headerRegex.FindAllStringSubmatchIndex(body, -1)
	if len(matches) == 0 {
		// Keine Header vorhanden, behandle gesamten Body als einen Tweet
		content := strings.TrimSpace(body)
		if content == "" {
			return nil
		}
		
		// Inline Bild suchen
		var inlineImage string
		if imgMatch := inlineImageRegex.FindStringSubmatch(content); len(imgMatch) > 1 {
			inlineImage = strings.TrimSpace(imgMatch[1])
			content = inlineImageRegex.ReplaceAllString(content, "")
			content = strings.TrimSpace(content)
		}
		
		return []models.Tweet{
			{
				Index:   1,
				Content: content,
				Image:   inlineImage,
			},
		}
	}

	var tweets []models.Tweet
	var lastEnd int
	var tweetIndex = 1

	for i, match := range matches {
		headerStart, headerEnd := match[0], match[1]
		headerText := body[headerStart:headerEnd]

		// Wenn vor dem ersten Header Text stand, ignorieren wir ihn meistens (z.B. Einleitung),
		// es sei denn, es war relevanter Content. In postctl gehen wir davon aus,
		// dass der Thread mit "## Tweet 1" startet.

		// Den vorherigen Block verarbeiten
		if i > 0 {
			blockContent := strings.TrimSpace(body[lastEnd:headerStart])
			if blockContent != "" {
				tweets = append(tweets, createTweet(tweetIndex, blockContent, images))
				tweetIndex++
			}
		}

		lastEnd = headerEnd
		
		// Falls dies der letzte Header ist, müssen wir den Rest des Bodys verarbeiten
		if i == len(matches)-1 {
			blockContent := strings.TrimSpace(body[lastEnd:])
			// Prüfen, ob dieser Header ein Reply ist
			isReply := strings.Contains(strings.ToLower(headerText), "reply")
			t := createTweet(tweetIndex, blockContent, images)
			t.IsReply = isReply
			if t.Content != "" {
				tweets = append(tweets, t)
			}
		}
	}

	return tweets
}

// createTweet erstellt einen Tweet, extrahiert inline Bilder oder weist standardmäßig Bilder zu
func createTweet(index int, content string, images []string) models.Tweet {
	var inlineImage string
	if imgMatch := inlineImageRegex.FindStringSubmatch(content); len(imgMatch) > 1 {
		inlineImage = strings.TrimSpace(imgMatch[1])
		content = inlineImageRegex.ReplaceAllString(content, "")
		content = strings.TrimSpace(content)
	}

	// Falls kein inline Bild gefunden wurde und wir ein Standardbild aus Frontmatter haben:
	// Bild-Zuweisung: Tweet 2 (Index 2) erhält images[0], Tweet 3 erhält images[1] usw.
	if inlineImage == "" && index > 1 && len(images) >= index-1 {
		inlineImage = images[index-2]
	}

	return models.Tweet{
		Index:   index,
		Content: content,
		Image:   inlineImage,
	}
}

// TwitterLength berechnet die Twitter-Länge (inklusive 23 Zeichen für URLs)
func TwitterLength(text string) int {
	// URLs finden und durch 23-Zeichen-Platzhalter ersetzen
	processed := urlRegex.ReplaceAllString(text, "12345678901234567890123")
	return len([]rune(processed))
}
