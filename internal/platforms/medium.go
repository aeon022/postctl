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

type MediumPlatform struct {
	store    *store.SQLiteStore
	token    string
	client   *http.Client
	apiURL   string // defaults to https://api.medium.com/v1
}

func NewMediumPlatform(s *store.SQLiteStore, token string) *MediumPlatform {
	return &MediumPlatform{
		store:  s,
		token:  token,
		client: &http.Client{Timeout: 30 * time.Second},
		apiURL: "https://api.medium.com/v1",
	}
}

func (m *MediumPlatform) Name() string {
	return models.PlatformMedium
}

func (m *MediumPlatform) IsAuthenticated(ctx context.Context) bool {
	return m.token != ""
}

func (m *MediumPlatform) Auth(ctx context.Context) error {
	if m.token == "" {
		if config.ActiveConfig.Defaults.Language == "de" {
			return fmt.Errorf("Medium-Konfiguration fehlt! Bitte folge diesen Schritten:\n" +
				"  1. Gehe in deine Medium Einstellungen ➔ Security and apps ➔ Integration tokens.\n" +
				"  2. Erstelle einen Token und kopiere ihn.\n" +
				"  3. Trage den Token im Terminal ein:\n" +
				"     postctl config set medium.integration_token \"DEIN_TOKEN\"")
		}
		return fmt.Errorf("Medium configuration is missing! Please follow these steps:\n" +
			"  1. Go to your Medium Settings ➔ Security and apps ➔ Integration tokens.\n" +
			"  2. Generate an Integration Token and copy it.\n" +
			"  3. Configure postctl in your terminal:\n" +
			"     postctl config set medium.integration_token \"YOUR_TOKEN\"")
	}

	// Token testen durch Aufruf von /me
	req, err := http.NewRequestWithContext(ctx, "GET", m.apiURL+"/me", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+m.token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Charset", "utf-8")

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to reach Medium API: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("medium auth failed (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

func (m *MediumPlatform) getUserId(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", m.apiURL+"/me", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+m.token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Charset", "utf-8")

	resp, err := m.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get user info (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return "", err
	}

	return result.Data.ID, nil
}

func (m *MediumPlatform) UploadImage(ctx context.Context, path string) (string, error) {
	return path, nil
}

func (m *MediumPlatform) Post(ctx context.Context, post *models.Post) (string, error) {
	userId, err := m.getUserId(ctx)
	if err != nil {
		return "", fmt.Errorf("get medium user ID: %w", err)
	}

	publishStatus := "public"
	if post.ScheduledAt != nil {
		publishStatus = "draft"
	}

	type payloadInput struct {
		Title         string   `json:"title"`
		ContentFormat string   `json:"contentFormat"`
		Content       string   `json:"content"`
		Tags          []string `json:"tags"`
		PublishStatus string   `json:"publishStatus"`
	}

	payload := payloadInput{
		Title:         post.Title,
		ContentFormat: "markdown",
		Content:       post.Body,
		Tags:          post.Tags,
		PublishStatus: publishStatus,
	}

	if len(payload.Tags) > 5 {
		payload.Tags = payload.Tags[:5]
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/users/%s/posts", m.apiURL, userId)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+m.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("medium API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return "", err
	}

	return result.Data.ID, nil
}

func (m *MediumPlatform) FetchAnalytics(ctx context.Context, platformID string) (models.AnalyticsData, error) {
	return models.AnalyticsData{}, nil
}

// Delete is a stub for Medium delete method
func (m *MediumPlatform) Delete(ctx context.Context, platformID string) error {
	return nil
}
