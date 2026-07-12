package platforms

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/models"
	"github.com/aeon022/postctl/internal/store"
)

type RedditPlatform struct {
	store        *store.SQLiteStore
	clientID     string
	clientSecret string
	username     string
	password     string
	client       *http.Client
	oauthURL     string // defaults to https://www.reddit.com/api/v1/access_token
	apiURL       string // defaults to https://oauth.reddit.com/api
}

func NewRedditPlatform(s *store.SQLiteStore, clientID, clientSecret, username, password string) *RedditPlatform {
	return &RedditPlatform{
		store:        s,
		clientID:     clientID,
		clientSecret: clientSecret,
		username:     username,
		password:     password,
		client:       &http.Client{Timeout: 30 * time.Second},
		oauthURL:     "https://www.reddit.com/api/v1/access_token",
		apiURL:       "https://oauth.reddit.com/api",
	}
}

func (r *RedditPlatform) Name() string {
	return models.PlatformReddit
}

func (r *RedditPlatform) IsAuthenticated(ctx context.Context) bool {
	return r.clientID != "" && r.clientSecret != "" && r.username != "" && r.password != ""
}

func (r *RedditPlatform) Auth(ctx context.Context) error {
	if r.clientID == "" || r.clientSecret == "" || r.username == "" || r.password == "" {
		if config.ActiveConfig.Defaults.Language == "de" {
			return fmt.Errorf("Reddit-Konfiguration fehlt! Bitte folge diesen Schritten:\n" +
				"  1. Erstelle eine Reddit App (Typ: script) unter https://www.reddit.com/prefs/apps.\n" +
				"  2. Kopiere die Client-ID (unter dem App-Namen) und das Client-Secret.\n" +
				"  3. Trage deine Zugangsdaten im Terminal ein:\n" +
				"     postctl config set reddit.client_id \"DEINE_CLIENT_ID\"\n" +
				"     postctl config set reddit.client_secret \"DEIN_CLIENT_SECRET\"\n" +
				"     postctl config set reddit.username \"DEIN_BENUTZERNAME\"\n" +
				"     postctl config set reddit.password \"DEIN_PASSWORT\"")
		}
		return fmt.Errorf("Reddit configuration is missing! Please follow these steps:\n" +
			"  1. Create a Reddit App (type: script) under https://www.reddit.com/prefs/apps.\n" +
			"  2. Copy the Client ID (under the app name) and the Client Secret.\n" +
			"  3. Configure postctl in your terminal:\n" +
			"     postctl config set reddit.client_id \"YOUR_CLIENT_ID\"\n" +
			"     postctl config set reddit.client_secret \"YOUR_CLIENT_SECRET\"\n" +
			"     postctl config set reddit.username \"YOUR_USERNAME\"\n" +
			"     postctl config set reddit.password \"YOUR_PASSWORD\"")
	}

	_, err := r.getAccessToken(ctx)
	return err
}

func (r *RedditPlatform) getAccessToken(ctx context.Context) (string, error) {
	data := url.Values{}
	data.Set("grant_type", "password")
	data.Set("username", r.username)
	data.Set("password", r.password)

	req, err := http.NewRequestWithContext(ctx, "POST", r.oauthURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(r.clientID, r.clientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", fmt.Sprintf("postctl/1.0.0 (by /u/%s)", r.username))

	resp, err := r.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("reddit oauth failed (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if result.Error != "" {
		return "", fmt.Errorf("reddit oauth error: %s", result.Error)
	}

	return result.AccessToken, nil
}

func (r *RedditPlatform) UploadImage(ctx context.Context, path string) (string, error) {
	return path, nil
}

func (r *RedditPlatform) Post(ctx context.Context, post *models.Post) (string, error) {
	token, err := r.getAccessToken(ctx)
	if err != nil {
		return "", err
	}

	sr := "test"
	if len(post.Tags) > 0 {
		sr = post.Tags[0]
	} else if post.Campaign != "" && post.Campaign != "default" {
		sr = post.Campaign
	}

	data := url.Values{}
	data.Set("sr", sr)
	data.Set("title", post.Title)
	data.Set("kind", "self")
	data.Set("text", post.Body)

	req, err := http.NewRequestWithContext(ctx, "POST", r.apiURL+"/submit", strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "bearer "+token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", fmt.Sprintf("postctl/1.0.0 (by /u/%s)", r.username))

	resp, err := r.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("reddit API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		JSON struct {
			Errors [][]interface{} `json:"errors"`
			Data   struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"data"`
		} `json:"json"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if len(result.JSON.Errors) > 0 {
		return "", fmt.Errorf("reddit submit error: %v", result.JSON.Errors)
	}

	return result.JSON.Data.Name, nil
}

func (r *RedditPlatform) FetchAnalytics(ctx context.Context, platformID string) (models.AnalyticsData, error) {
	return models.AnalyticsData{}, nil
}
