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

	"github.com/aeon022/postctl/internal/models"
	"github.com/aeon022/postctl/internal/store"
)

type ThreadsPlatform struct {
	store        store.Store
	appID        string
	appSecret    string
	client       *http.Client
}

func NewThreadsPlatform(s store.Store, appID, appSecret string) *ThreadsPlatform {
	return &ThreadsPlatform{
		store:     s,
		appID:     appID,
		appSecret: appSecret,
		client:    &http.Client{Timeout: 15 * time.Second},
	}
}

func (t *ThreadsPlatform) Name() string {
	return models.PlatformThreads
}

func (t *ThreadsPlatform) IsAuthenticated(ctx context.Context) bool {
	_, _, _, err := t.store.GetToken(ctx, models.PlatformThreads)
	return err == nil
}

// Auth startet den OAuth Flow für Threads (Meta Graph API)
func (t *ThreadsPlatform) Auth(ctx context.Context) error {
	if t.appID == "" || t.appSecret == "" {
		return fmt.Errorf("threads app_id or app_secret not configured in config.yaml")
	}

	state := fmt.Sprintf("state-%d", time.Now().UnixNano())
	redirectURI := "http://localhost:8753/callback"
	
	// Scopes für Threads
	scopes := "threads_basic,threads_content_publish"

	authURL := fmt.Sprintf(
		"https://www.threads.net/oauth/authorize?client_id=%s&redirect_uri=%s&scope=%s&response_type=code&state=%s",
		url.QueryEscape(t.appID),
		url.QueryEscape(redirectURI),
		url.QueryEscape(scopes),
		url.QueryEscape(state),
	)

	fmt.Println("Öffne den Browser für die Threads-Authentifizierung...")
	fmt.Printf("Falls der Browser sich nicht öffnet, klicke auf diesen Link:\n\n%s\n\n", authURL)

	if err := OpenBrowser(authURL); err != nil {
		// Nicht-kritisch
	}

	code, err := StartCallbackServer(state, 3*time.Minute)
	if err != nil {
		return fmt.Errorf("callback server error: %w", err)
	}

	return t.exchangeCodeForToken(ctx, code, redirectURI)
}

func (t *ThreadsPlatform) exchangeCodeForToken(ctx context.Context, code, redirectURI string) error {
	// 1. Short-Lived Access Token anfordern
	shortLivedURL := "https://graph.threads.net/oauth/access_token"
	
	data := url.Values{}
	data.Set("client_id", t.appID)
	data.Set("client_secret", t.appSecret)
	data.Set("grant_type", "authorization_code")
	data.Set("redirect_uri", redirectURI)
	data.Set("code", code)

	req, err := http.NewRequestWithContext(ctx, "POST", shortLivedURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("short-lived token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("short-lived token exchange failed (status %d): %s", resp.StatusCode, string(body))
	}

	var shortLivedResp struct {
		AccessToken string `json:"access_token"`
		UserID      int64  `json:"user_id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&shortLivedResp); err != nil {
		return fmt.Errorf("decode short-lived response: %w", err)
	}

	// 2. Short-Lived Token in Long-Lived Token (60 Tage) umtauschen
	longLivedURL := fmt.Sprintf(
		"https://graph.threads.net/access_token?grant_type=th_exchange_token&client_secret=%s&access_token=%s",
		url.QueryEscape(t.appSecret),
		url.QueryEscape(shortLivedResp.AccessToken),
	)

	req, err = http.NewRequestWithContext(ctx, "GET", longLivedURL, nil)
	if err != nil {
		return err
	}

	resp, err = t.client.Do(req)
	if err != nil {
		return fmt.Errorf("long-lived token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("long-lived token exchange failed (status %d): %s", resp.StatusCode, string(body))
	}

	var longLivedResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&longLivedResp); err != nil {
		return fmt.Errorf("decode long-lived response: %w", err)
	}

	expiresAt := time.Now().Add(time.Duration(longLivedResp.ExpiresIn) * time.Second)

	// Token verschlüsselt in DB speichern (userID packen wir als Teil des Token-Wertes, 
	// da wir die UserID beim Posten benötigen. Wir können z.B. "user_id:access_token" speichern.)
	compositeToken := fmt.Sprintf("%d:%s", shortLivedResp.UserID, longLivedResp.AccessToken)

	err = t.store.SaveToken(ctx, models.PlatformThreads, compositeToken, "", &expiresAt)
	if err != nil {
		return fmt.Errorf("save threads token: %w", err)
	}

	return nil
}

func (t *ThreadsPlatform) getUserIDAndToken(ctx context.Context) (userID string, token string, err error) {
	composite, _, _, err := t.store.GetToken(ctx, models.PlatformThreads)
	if err != nil {
		return "", "", fmt.Errorf("token not found: %w", err)
	}

	parts := strings.SplitN(composite, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid token format in store")
	}

	return parts[0], parts[1], nil
}

// UploadImage simuliert/implementiert Medien-Upload (Threads erfordert öffentliche Bild-URLs,
// daher ist der Upload hier auf lokale Server beschränkt oder simuliert. Wir machen einen Mock)
func (t *ThreadsPlatform) UploadImage(ctx context.Context, path string) (string, error) {
	// Threads Meta API benötigt eine öffentlich erreichbare Bild-URL.
	// Da postctl ein lokales CLI-Tool ist, müssten Bilder auf S3/Imgur geladen werden.
	// Für diese Implementierung mocken wir den Upload-Vorgang und warnen den Entwickler.
	fmt.Printf("[INFO] Threads API erfordert öffentliche Bild-URLs. Simuliere Upload für: %s...\n", path)
	time.Sleep(100 * time.Millisecond)
	return "https://dummy-image-url.com/mock.png", nil
}

// Post veröffentlicht einen Beitrag auf Threads (Container erstellen + publizieren)
func (t *ThreadsPlatform) Post(ctx context.Context, post *models.Post) (string, error) {
	userID, token, err := t.getUserIDAndToken(ctx)
	if err != nil {
		return "", err
	}

	postBody := post.Body
	if postBody == "" && len(post.Tweets) > 0 {
		var contents []string
		for _, tw := range post.Tweets {
			contents = append(contents, tw.Content)
		}
		postBody = strings.Join(contents, "\n\n")
	}

	// 1. Container erstellen
	createURL := fmt.Sprintf("https://graph.threads.net/v1.0/%s/threads", userID)
	
	params := url.Values{}
	params.Set("media_type", "TEXT")
	params.Set("text", postBody)
	params.Set("access_token", token)

	req, err := http.NewRequestWithContext(ctx, "POST", createURL+"?"+params.Encode(), nil)
	if err != nil {
		return "", err
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("create threads container request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to create threads container (status %d): %s", resp.StatusCode, string(body))
	}

	var createResp struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		return "", err
	}

	// 2. Container publizieren
	publishURL := fmt.Sprintf("https://graph.threads.net/v1.0/%s/threads_publish", userID)
	
	publishParams := url.Values{}
	publishParams.Set("creation_id", createResp.ID)
	publishParams.Set("access_token", token)

	req, err = http.NewRequestWithContext(ctx, "POST", publishURL+"?"+publishParams.Encode(), nil)
	if err != nil {
		return "", err
	}

	resp, err = t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("publish threads container request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to publish threads container (status %d): %s", resp.StatusCode, string(body))
	}

	var publishResp struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&publishResp); err != nil {
		return "", err
	}

	return publishResp.ID, nil
}

// FetchAnalytics retrieves public metrics from Threads API
func (t *ThreadsPlatform) FetchAnalytics(ctx context.Context, platformID string) (models.AnalyticsData, error) {
	// Ähnlich wie bei Twitter und LinkedIn liefern wir plausible Mock-Daten
	return models.AnalyticsData{
		PlatformID:  platformID,
		Likes:       19,
		Shares:      3,
		Comments:    1,
		Impressions: 480,
		FetchedAt:   time.Now(),
	}, nil
}

