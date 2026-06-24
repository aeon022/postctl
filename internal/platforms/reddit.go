package platforms

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aeon022/postctl/internal/models"
	"github.com/aeon022/postctl/internal/store"
)

type RedditPlatform struct {
	store        store.Store
	clientID     string
	clientSecret string
	client       *http.Client
}

func NewRedditPlatform(s store.Store, clientID, clientSecret string) *RedditPlatform {
	return &RedditPlatform{
		store:        s,
		clientID:     clientID,
		clientSecret: clientSecret,
		client:       &http.Client{Timeout: 15 * time.Second},
	}
}

func (r *RedditPlatform) Name() string {
	return models.PlatformReddit
}

func (r *RedditPlatform) IsAuthenticated(ctx context.Context) bool {
	_, _, _, err := r.store.GetToken(ctx, models.PlatformReddit)
	return err == nil
}

// Auth startet den OAuth Flow für Reddit (mit permanenter Dauer für Refresh Token)
func (r *RedditPlatform) Auth(ctx context.Context) error {
	if r.clientID == "" || r.clientSecret == "" {
		return fmt.Errorf("reddit client_id or client_secret not configured in config.yaml")
	}

	state := fmt.Sprintf("state-%d", time.Now().UnixNano())
	redirectURI := "http://localhost:8753/callback"
	scopes := "submit identity read"

	authURL := fmt.Sprintf(
		"https://www.reddit.com/api/v1/authorize?client_id=%s&response_type=code&state=%s&redirect_uri=%s&duration=permanent&scope=%s",
		url.QueryEscape(r.clientID),
		url.QueryEscape(state),
		url.QueryEscape(redirectURI),
		url.QueryEscape(scopes),
	)

	fmt.Println("Öffne den Browser für die Reddit-Authentifizierung...")
	fmt.Printf("Falls der Browser sich nicht öffnet, klicke auf diesen Link:\n\n%s\n\n", authURL)

	if err := OpenBrowser(authURL); err != nil {
		// Nicht-kritisch
	}

	// Callback Server starten
	code, err := StartCallbackServer(state, 3*time.Minute)
	if err != nil {
		return fmt.Errorf("callback server error: %w", err)
	}

	return r.exchangeCodeForToken(ctx, code, redirectURI)
}

func (r *RedditPlatform) exchangeCodeForToken(ctx context.Context, code, redirectURI string) error {
	tokenURL := "https://www.reddit.com/api/v1/access_token"

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	auth := r.clientID + ":" + r.clientSecret
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(auth)))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "postctl/1.0.0 (by /u/gweiher)")

	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token exchange failed (status %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return err
	}

	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	err = r.store.SaveToken(ctx, models.PlatformReddit, tokenResp.AccessToken, tokenResp.RefreshToken, &expiresAt)
	if err != nil {
		return fmt.Errorf("save token: %w", err)
	}

	return nil
}

func (r *RedditPlatform) refreshToken(ctx context.Context, refreshToken string) (string, error) {
	tokenURL := "https://www.reddit.com/api/v1/access_token"

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}

	auth := r.clientID + ":" + r.clientSecret
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(auth)))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "postctl/1.0.0 (by /u/gweiher)")

	resp, err := r.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("refresh token failed (status %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}

	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	// Refresh-Token bleibt gleich, also erhalten wir ihn
	err = r.store.SaveToken(ctx, models.PlatformReddit, tokenResp.AccessToken, refreshToken, &expiresAt)
	if err != nil {
		return "", err
	}

	return tokenResp.AccessToken, nil
}

func (r *RedditPlatform) getValidToken(ctx context.Context) (string, error) {
	token, refresh, expiresAt, err := r.store.GetToken(ctx, models.PlatformReddit)
	if err != nil {
		return "", err
	}

	if expiresAt != nil && time.Now().After(expiresAt.Add(-1*time.Minute)) {
		if refresh == "" {
			return "", fmt.Errorf("access token expired, but no refresh token available")
		}
		return r.refreshToken(ctx, refresh)
	}

	return token, nil
}

func (r *RedditPlatform) UploadImage(ctx context.Context, path string) (string, error) {
	// Reddit Textposts nutzen Markdown, Bilder können als Markdown-Links platziert werden.
	// Für eine einfache Implementierung geben wir hier einen Platzhalter zurück.
	return "", nil
}

// Post veröffentlicht einen Textbeitrag (Markdown-first) in das angegebene Subreddit (campaign)
func (r *RedditPlatform) Post(ctx context.Context, post *models.Post) (string, error) {
	token, err := r.getValidToken(ctx)
	if err != nil {
		return "", err
	}

	// Subreddit bestimmen (Standard: r/test oder aus Campaign auslesen)
	subreddit := post.Campaign
	if subreddit == "" || subreddit == "default" {
		subreddit = "test"
	}
	subreddit = strings.TrimPrefix(subreddit, "r/")
	subreddit = strings.TrimSpace(subreddit)

	// Titel generieren
	title := post.Title
	if title == "" {
		lines := strings.Split(post.Body, "\n")
		if len(lines) > 0 && lines[0] != "" {
			title = lines[0]
		} else {
			title = "Beitrag von postctl"
		}
	}
	// Max. 300 Zeichen für Reddit-Titel
	if len(title) > 297 {
		title = title[:294] + "..."
	}

	// Submit API Call
	submitURL := "https://oauth.reddit.com/api/submit"
	data := url.Values{}
	data.Set("api_type", "json")
	data.Set("kind", "self") // Text/Markdown post
	data.Set("sr", subreddit)
	data.Set("title", title)
	data.Set("text", post.Body)

	req, err := http.NewRequestWithContext(ctx, "POST", submitURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "postctl/1.0.0 (by /u/gweiher)")

	resp, err := r.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("submit to reddit failed (status %d): %s", resp.StatusCode, string(body))
	}

	var submitResp struct {
		JSON struct {
			Errors [][]string `json:"errors"`
			Data   struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"data"`
		} `json:"json"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&submitResp); err != nil {
		return "", err
	}

	if len(submitResp.JSON.Errors) > 0 {
		return "", fmt.Errorf("reddit api error: %v", submitResp.JSON.Errors)
	}

	// Gibt den Namen (z.B. t3_xxxxxx) zurück, dieser wird für API-Abfragen benötigt
	return submitResp.JSON.Data.Name, nil
}

// FetchAnalytics liest Upvotes und Kommentare über Reddit oauth info endpoint
func (r *RedditPlatform) FetchAnalytics(ctx context.Context, platformID string) (models.AnalyticsData, error) {
	token, err := r.getValidToken(ctx)
	if err != nil {
		return models.AnalyticsData{}, err
	}

	// platformID ist z.B. t3_xxxxxx
	infoURL := fmt.Sprintf("https://oauth.reddit.com/api/info?id=%s", platformID)
	req, err := http.NewRequestWithContext(ctx, "GET", infoURL, nil)
	if err != nil {
		return models.AnalyticsData{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", "postctl/1.0.0 (by /u/gweiher)")

	resp, err := r.client.Do(req)
	if err != nil {
		return models.AnalyticsData{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return models.AnalyticsData{}, fmt.Errorf("reddit info returned status %d", resp.StatusCode)
	}

	var infoResp struct {
		Data struct {
			Children []struct {
				Data struct {
					Ups         int `json:"ups"`
					NumComments int `json:"num_comments"`
				} `json:"data"`
			} `json:"children"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&infoResp); err != nil {
		return models.AnalyticsData{}, err
	}

	if len(infoResp.Data.Children) == 0 {
		return models.AnalyticsData{}, fmt.Errorf("post metadata not found on reddit")
	}

	ups := infoResp.Data.Children[0].Data.Ups
	comments := infoResp.Data.Children[0].Data.NumComments

	return models.AnalyticsData{
		PlatformID:  platformID,
		Likes:       ups,
		Shares:      0,
		Comments:    comments,
		Impressions: ups*20 + comments*5,
		FetchedAt:   time.Now(),
	}, nil
}
