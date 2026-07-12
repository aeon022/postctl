package platforms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/models"
	"github.com/aeon022/postctl/internal/store"
)

type DevToPlatform struct {
	store    *store.SQLiteStore
	apiToken string
	client   *http.Client
	apiURL   string // defaults to https://dev.to/api
}

func NewDevToPlatform(s *store.SQLiteStore, apiToken string) *DevToPlatform {
	return &DevToPlatform{
		store:    s,
		apiToken: apiToken,
		client:   &http.Client{Timeout: 30 * time.Second},
		apiURL:   "https://dev.to/api",
	}
}

func (d *DevToPlatform) Name() string {
	return models.PlatformDevTo
}

func (d *DevToPlatform) IsAuthenticated(ctx context.Context) bool {
	return d.apiToken != ""
}

func (d *DevToPlatform) Auth(ctx context.Context) error {
	if d.apiToken == "" {
		if config.ActiveConfig.Defaults.Language == "de" {
			return fmt.Errorf("Dev.to-Konfiguration fehlt! Bitte folge diesen Schritten:\n" +
				"  1. Gehe in deine Dev.to Einstellungen ➔ Extensions ➔ DEV Community API Keys.\n" +
				"  2. Erstelle einen API Key und kopiere ihn.\n" +
				"  3. Trage den API Key im Terminal ein:\n" +
				"     postctl config set devto.api_token \"DEIN_API_TOKEN\"")
		}
		return fmt.Errorf("Dev.to configuration is missing! Please follow these steps:\n" +
			"  1. Go to your Dev.to Settings ➔ Extensions ➔ DEV Community API Keys.\n" +
			"  2. Generate a new API Key and copy it.\n" +
			"  3. Configure postctl in your terminal:\n" +
			"     postctl config set devto.api_token \"YOUR_API_TOKEN\"")
	}

	// Token verifizieren
	req, err := http.NewRequestWithContext(ctx, "GET", d.apiURL+"/users/me", nil)
	if err != nil {
		return err
	}
	req.Header.Set("api-key", d.apiToken)

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to reach Dev.to API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("invalid Dev.to API Token (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

func (d *DevToPlatform) UploadImage(ctx context.Context, path string) (string, error) {
	// Dev.to nimmt Bilder als URLs im Markdown-Body auf
	return path, nil
}

func (d *DevToPlatform) Post(ctx context.Context, post *models.Post) (string, error) {
	published := true
	if post.ScheduledAt != nil {
		published = false
	}

	type Article struct {
		Title        string   `json:"title"`
		BodyMarkdown string   `json:"body_markdown"`
		Published    bool     `json:"published"`
		Tags         []string `json:"tags"`
	}

	type Payload struct {
		Article Article `json:"article"`
	}

	payload := Payload{
		Article: Article{
			Title:        post.Title,
			BodyMarkdown: post.Body,
			Published:    published,
			Tags:         post.Tags,
		},
	}

	// Dev.to erlaubt max 4 Tags
	if len(payload.Article.Tags) > 4 {
		payload.Article.Tags = payload.Article.Tags[:4]
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", d.apiURL+"/articles", bytes.NewBuffer(jsonBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("api-key", d.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("dev.to API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return "", err
	}

	return fmt.Sprintf("%d", result.ID), nil
}

func (d *DevToPlatform) FetchAnalytics(ctx context.Context, platformID string) (models.AnalyticsData, error) {
	return models.AnalyticsData{}, nil
}
