package platforms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/models"
	"github.com/aeon022/postctl/internal/store"
)

type HashnodePlatform struct {
	store         *store.SQLiteStore
	apiToken      string
	publicationID string
	client        *http.Client
	apiURL        string // defaults to https://gql.hashnode.com
}

func NewHashnodePlatform(s *store.SQLiteStore, apiToken, publicationID string) *HashnodePlatform {
	return &HashnodePlatform{
		store:         s,
		apiToken:      apiToken,
		publicationID: publicationID,
		client:        &http.Client{Timeout: 30 * time.Second},
		apiURL:        "https://gql.hashnode.com",
	}
}

func (h *HashnodePlatform) Name() string {
	return models.PlatformHashnode
}

func (h *HashnodePlatform) IsAuthenticated(ctx context.Context) bool {
	return h.apiToken != "" && h.publicationID != ""
}

func (h *HashnodePlatform) Auth(ctx context.Context) error {
	if h.apiToken == "" || h.publicationID == "" {
		if config.ActiveConfig.Defaults.Language == "de" {
			return fmt.Errorf("Hashnode-Konfiguration fehlt! Bitte folge diesen Schritten:\n" +
				"  1. Gehe in dein Hashnode Dashboard ➔ Account Settings ➔ Developer.\n" +
				"  2. Erstelle einen Personal Access Token und kopiere ihn.\n" +
				"  3. Ermittle deine Publication ID (im Dashboard unter Blog Dashboard ➔ Settings).\n" +
				"  4. Trage deine Zugangsdaten im Terminal ein:\n" +
				"     postctl config set hashnode.api_token \"DEIN_API_TOKEN\"\n" +
				"     postctl config set hashnode.publication_id \"DEINE_PUBLICATION_ID\"")
		}
		return fmt.Errorf("Hashnode configuration is missing! Please follow these steps:\n" +
			"  1. Go to your Hashnode Blog Dashboard ➔ Account Settings ➔ Developer.\n" +
			"  2. Generate a Personal Access Token and copy it.\n" +
			"  3. Get your Publication ID (found in Blog Dashboard ➔ Settings).\n" +
			"  4. Configure postctl in your terminal:\n" +
			"     postctl config set hashnode.api_token \"YOUR_API_TOKEN\"\n" +
			"     postctl config set hashnode.publication_id \"YOUR_PUBLICATION_ID\"")
	}

	// Token testen mit query { me { username } }
	payload := map[string]interface{}{
		"query": `query { me { username } }`,
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", h.apiURL, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", h.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to reach Hashnode API: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("hashnode auth request failed (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
		Data struct {
			Me struct {
				Username string `json:"username"`
			} `json:"me"`
		} `json:"data"`
	}

	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return err
	}

	if len(result.Errors) > 0 {
		return fmt.Errorf("hashnode auth error: %s", result.Errors[0].Message)
	}

	if result.Data.Me.Username == "" {
		return fmt.Errorf("invalid access token (no username returned)")
	}

	return nil
}

func (h *HashnodePlatform) UploadImage(ctx context.Context, path string) (string, error) {
	return path, nil
}

func (h *HashnodePlatform) Post(ctx context.Context, post *models.Post) (string, error) {
	type tagInput struct {
		Slug string `json:"slug"`
		Name string `json:"name"`
	}

	var tags []tagInput
	for _, tag := range post.Tags {
		tags = append(tags, tagInput{Slug: strings.ToLower(tag), Name: tag})
	}

	type publishInput struct {
		Title           string     `json:"title"`
		PublicationID   string     `json:"publicationId"`
		ContentMarkdown string     `json:"contentMarkdown"`
		Tags            []tagInput `json:"tags"`
	}

	payload := map[string]interface{}{
		"query": `mutation PublishPost($input: PublishPostInput!) {
			publishPost(input: $input) {
				post {
					id
					url
				}
			}
		}`,
		"variables": map[string]interface{}{
			"input": publishInput{
				Title:           post.Title,
				PublicationID:   h.publicationID,
				ContentMarkdown: post.Body,
				Tags:            tags,
			},
		},
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", h.apiURL, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", h.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("hashnode API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
		Data struct {
			PublishPost struct {
				Post struct {
					ID  string `json:"id"`
					URL string `json:"url"`
				} `json:"post"`
			} `json:"publishPost"`
		} `json:"data"`
	}

	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return "", err
	}

	if len(result.Errors) > 0 {
		return "", fmt.Errorf("hashnode publish error: %s", result.Errors[0].Message)
	}

	return result.Data.PublishPost.Post.ID, nil
}

func (h *HashnodePlatform) FetchAnalytics(ctx context.Context, platformID string) (models.AnalyticsData, error) {
	return models.AnalyticsData{}, nil
}
