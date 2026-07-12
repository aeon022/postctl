package platforms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/generator"
	"github.com/aeon022/postctl/internal/models"
	"github.com/aeon022/postctl/internal/store"
)

type MastodonPlatform struct {
	store        *store.SQLiteStore
	instanceURL  string
	clientID     string
	clientSecret string
	client       *http.Client
}

func NewMastodonPlatform(s *store.SQLiteStore, instanceURL, clientID, clientSecret string) *MastodonPlatform {
	// Standard-Instanz falls nicht konfiguriert
	if instanceURL == "" {
		instanceURL = "https://mastodon.social"
	}
	// Slashes am Ende entfernen
	instanceURL = strings.TrimSuffix(instanceURL, "/")

	return &MastodonPlatform{
		store:        s,
		instanceURL:  instanceURL,
		clientID:     clientID,
		clientSecret: clientSecret,
		client:       &http.Client{Timeout: 20 * time.Second},
	}
}

func (m *MastodonPlatform) Name() string {
	return models.PlatformMastodon
}

func (m *MastodonPlatform) IsAuthenticated(ctx context.Context) bool {
	_, _, _, err := m.store.GetToken(ctx, models.PlatformMastodon)
	return err == nil
}

// Auth startet den OAuth Flow. Falls ClientID/ClientSecret fehlen, registrieren wir die App dynamisch.
func (m *MastodonPlatform) Auth(ctx context.Context) error {
	// 1. Dynamische App-Registrierung falls Credentials leer sind
	if m.clientID == "" || m.clientSecret == "" {
		fmt.Printf("Registriere Anwendung dynamisch bei Mastodon Instanz: %s...\n", m.instanceURL)
		if err := m.registerApp(ctx); err != nil {
			return fmt.Errorf("register mastodon app: %w", err)
		}
	}

	state := fmt.Sprintf("state-%d", time.Now().UnixNano())
	redirectURI := "http://127.0.0.1:8753/callback"
	scopes := "read write"

	authURL := fmt.Sprintf(
		"%s/oauth/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=%s&state=%s",
		m.instanceURL,
		url.QueryEscape(m.clientID),
		url.QueryEscape(redirectURI),
		url.QueryEscape(scopes),
		url.QueryEscape(state),
	)

	fmt.Println("Öffne den Browser für die Mastodon-Authentifizierung...")
	fmt.Printf("Falls der Browser sich nicht öffnet, klicke auf diesen Link:\n\n%s\n\n", authURL)

	if err := OpenBrowser(authURL); err != nil {
		// Nicht-kritisch, Link wurde gedruckt
	}

	// Callback Server starten
	code, err := StartCallbackServer(state, 3*time.Minute)
	if err != nil {
		return fmt.Errorf("callback server error: %w", err)
	}

	// Code gegen Token tauschen
	return m.exchangeCodeForToken(ctx, code, redirectURI)
}

func (m *MastodonPlatform) registerApp(ctx context.Context) error {
	regURL := m.instanceURL + "/api/v1/apps"
	redirectURI := "http://127.0.0.1:8753/callback"

	data := url.Values{}
	data.Set("client_name", "postctl")
	data.Set("redirect_uris", redirectURI)
	data.Set("scopes", "read write")
	data.Set("website", "https://github.com/aeon022/postctl")

	req, err := http.NewRequestWithContext(ctx, "POST", regURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := m.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("register app returned status %d: %s", resp.StatusCode, string(body))
	}

	var regResp struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&regResp); err != nil {
		return err
	}

	m.clientID = regResp.ClientID
	m.clientSecret = regResp.ClientSecret

	// In globaler Config persistieren
	config.ActiveConfig.Mastodon.ClientID = regResp.ClientID
	config.ActiveConfig.Mastodon.ClientSecret = regResp.ClientSecret
	config.ActiveConfig.Mastodon.InstanceURL = m.instanceURL
	if err := config.SaveConfig(); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	return nil
}

func (m *MastodonPlatform) exchangeCodeForToken(ctx context.Context, code, redirectURI string) error {
	tokenURL := m.instanceURL + "/oauth/token"

	data := url.Values{}
	data.Set("client_id", m.clientID)
	data.Set("client_secret", m.clientSecret)
	data.Set("redirect_uri", redirectURI)
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("scope", "read write")

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := m.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token exchange returned status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return err
	}

	// Mastodon Tokens laufen standardmäßig nicht ab. Keine Ablaufzeit speichern.
	err = m.store.SaveToken(ctx, models.PlatformMastodon, tokenResp.AccessToken, "", nil)
	if err != nil {
		return fmt.Errorf("save token in store: %w", err)
	}

	return nil
}

func (m *MastodonPlatform) getValidToken(ctx context.Context) (string, error) {
	token, _, _, err := m.store.GetToken(ctx, models.PlatformMastodon)
	if err != nil {
		return "", fmt.Errorf("token not found: %w", err)
	}
	return token, nil
}

// UploadImage lädt ein Bild hoch und gibt die Mastodon Media-ID zurück
func (m *MastodonPlatform) UploadImage(ctx context.Context, path string) (string, error) {
	token, err := m.getValidToken(ctx)
	if err != nil {
		return "", err
	}

	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open image file: %w", err)
	}
	defer file.Close()

	uploadURL := m.instanceURL + "/api/v1/media"

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(part, file); err != nil {
		return "", err
	}

	// Alt-Text automatisch generieren falls möglich
	aiCfg := generator.GeneratorConfig{
		Provider: config.ActiveConfig.AI.Provider,
		APIKey:   config.ActiveConfig.AI.APIKey,
		Model:    config.ActiveConfig.AI.Model,
		BaseURL:  config.ActiveConfig.AI.BaseURL,
	}
	if desc, err := generator.GenerateAltText(ctx, aiCfg, path); err == nil && desc != "" {
		_ = writer.WriteField("description", desc)
	}

	writer.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", uploadURL, body)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := m.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("media upload failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var uploadResp struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
		return "", err
	}

	return uploadResp.ID, nil
}

// Post veröffentlicht ein Status-Update oder einen verketteten Mastodon-Thread
func (m *MastodonPlatform) Post(ctx context.Context, post *models.Post) (string, error) {
	token, err := m.getValidToken(ctx)
	if err != nil {
		return "", err
	}

	var itemsToPost []string
	if post.Type == "thread" {
		for _, tweet := range post.Tweets {
			itemsToPost = append(itemsToPost, tweet.Content)
		}
	} else {
		itemsToPost = []string{post.Body}
	}

	if len(itemsToPost) == 0 {
		return "", fmt.Errorf("no content to post")
	}

	// 1. Bilder vorab hochladen bei Single Posts
	var uploadedMediaIDs []string
	if post.Type != "thread" && len(post.Images) > 0 {
		for _, imgPath := range post.Images {
			mediaID, err := m.UploadImage(ctx, imgPath)
			if err != nil {
				return "", fmt.Errorf("upload image %s: %w", imgPath, err)
			}
			uploadedMediaIDs = append(uploadedMediaIDs, mediaID)
		}
	}

	var firstPostID string
	var lastPostID string

	// 2. Sequentiell absenden
	for i, content := range itemsToPost {
		data := url.Values{}
		data.Set("status", content)
		data.Set("visibility", "public")

		// Bildzuweisung
		var currentMediaIDs []string
		if post.Type == "thread" {
			// Für Threads: Wenn der Einzelschnitt ein spezifisches Bild hat (analog zu Twitter.Image)
			if len(post.Tweets) > i && post.Tweets[i].Image != "" {
				mediaID, err := m.UploadImage(ctx, post.Tweets[i].Image)
				if err != nil {
					return "", fmt.Errorf("upload image %s for thread item %d: %w", post.Tweets[i].Image, i+1, err)
				}
				currentMediaIDs = []string{mediaID}
			}
		} else {
			currentMediaIDs = uploadedMediaIDs
		}

		for _, mediaID := range currentMediaIDs {
			data.Add("media_ids[]", mediaID)
		}

		// Thread-Verkettung
		if i > 0 && lastPostID != "" {
			data.Set("in_reply_to_id", lastPostID)
		}

		postURL := m.instanceURL + "/api/v1/statuses"
		req, err := http.NewRequestWithContext(ctx, "POST", postURL, strings.NewReader(data.Encode()))
		if err != nil {
			return "", err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := m.client.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			respBody, _ := io.ReadAll(resp.Body)
			return "", fmt.Errorf("post status item %d failed (status %d): %s", i+1, resp.StatusCode, string(respBody))
		}

		var statusResp struct {
			ID string `json:"id"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
			return "", err
		}

		lastPostID = statusResp.ID
		if i == 0 {
			firstPostID = lastPostID
		}
	}

	return firstPostID, nil
}

// FetchAnalytics liest echte Interaktionen für den Beitrag direkt via Mastodon API
func (m *MastodonPlatform) FetchAnalytics(ctx context.Context, platformID string) (models.AnalyticsData, error) {
	token, err := m.getValidToken(ctx)
	if err != nil {
		return models.AnalyticsData{}, err
	}

	statusURL := fmt.Sprintf("%s/api/v1/statuses/%s", m.instanceURL, platformID)
	req, err := http.NewRequestWithContext(ctx, "GET", statusURL, nil)
	if err != nil {
		return models.AnalyticsData{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := m.client.Do(req)
	if err != nil {
		return models.AnalyticsData{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return models.AnalyticsData{}, fmt.Errorf("fetch status metadata returned status %d", resp.StatusCode)
	}

	var statusData struct {
		FavouritesCount int `json:"favourites_count"` // Likes
		ReblogsCount    int `json:"reblogs_count"`    // Shares (Retweets/Boosts)
		RepliesCount    int `json:"replies_count"`    // Comments
	}

	if err := json.NewDecoder(resp.Body).Decode(&statusData); err != nil {
		return models.AnalyticsData{}, err
	}

	return models.AnalyticsData{
		PlatformID:  platformID,
		Likes:       statusData.FavouritesCount,
		Shares:      statusData.ReblogsCount,
		Comments:    statusData.RepliesCount,
		Impressions: statusData.FavouritesCount*10 + statusData.ReblogsCount*50 + 20, // Mastodon trackt Impressions nicht direkt, daher berechnen wir einen plausiblen Schätzwert
		FetchedAt:   time.Now(),
	}, nil
}
