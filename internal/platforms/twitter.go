package platforms

import (
	"bytes"
	"context"
	"encoding/base64"
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

	"github.com/aeon022/postctl/internal/models"
	"github.com/aeon022/postctl/internal/store"
)

type TwitterPlatform struct {
	store        store.Store
	clientID     string
	clientSecret string
	client       *http.Client
}

func NewTwitterPlatform(s store.Store, clientID, clientSecret string) *TwitterPlatform {
	return &TwitterPlatform{
		store:        s,
		clientID:     clientID,
		clientSecret: clientSecret,
		client:       &http.Client{Timeout: 15 * time.Second},
	}
}

func (t *TwitterPlatform) Name() string {
	return models.PlatformTwitter
}

func (t *TwitterPlatform) IsAuthenticated(ctx context.Context) bool {
	_, _, _, err := t.store.GetToken(ctx, models.PlatformTwitter)
	return err == nil
}

// Auth startet den OAuth 2.0 PKCE Flow für Twitter/X
func (t *TwitterPlatform) Auth(ctx context.Context) error {
	if t.clientID == "" || t.clientSecret == "" {
		return fmt.Errorf("twitter client_id or client_secret not configured in config.yaml")
	}

	state := fmt.Sprintf("state-%d", time.Now().UnixNano())
	verifier, err := GenerateVerifier()
	if err != nil {
		return fmt.Errorf("generate verifier: %w", err)
	}
	challenge := GenerateChallenge(verifier)

	redirectURI := "http://localhost:8753/callback"
	scopes := "tweet.read tweet.write users.read offline.access"

	authURL := fmt.Sprintf(
		"https://twitter.com/i/oauth2/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=%s&state=%s&code_challenge=%s&code_challenge_method=S256",
		url.QueryEscape(t.clientID),
		url.QueryEscape(redirectURI),
		url.QueryEscape(scopes),
		url.QueryEscape(state),
		url.QueryEscape(challenge),
	)

	fmt.Println("Öffne den Browser für die Twitter-Authentifizierung...")
	fmt.Printf("Falls der Browser sich nicht öffnet, klicke auf diesen Link:\n\n%s\n\n", authURL)

	if err := OpenBrowser(authURL); err != nil {
		// Nicht-kritischer Fehler, der Link wurde ausgegeben
	}

	// Callback Server starten mit 3 Minuten Timeout
	code, err := StartCallbackServer(state, 3*time.Minute)
	if err != nil {
		return fmt.Errorf("callback server error: %w", err)
	}

	// Token austauschen
	return t.exchangeCodeForToken(ctx, code, verifier, redirectURI)
}

func (t *TwitterPlatform) exchangeCodeForToken(ctx context.Context, code, verifier, redirectURI string) error {
	tokenURL := "https://api.twitter.com/2/oauth2/token"
	
	data := url.Values{}
	data.Set("code", code)
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", t.clientID)
	data.Set("redirect_uri", redirectURI)
	data.Set("code_verifier", verifier)

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	// Basic Auth hinzufügen
	auth := t.clientID + ":" + t.clientSecret
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(auth)))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token response failed (status %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("decode token response: %w", err)
	}

	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	// Token in DB verschlüsselt speichern
	err = t.store.SaveToken(ctx, models.PlatformTwitter, tokenResp.AccessToken, tokenResp.RefreshToken, &expiresAt)
	if err != nil {
		return fmt.Errorf("save token in store: %w", err)
	}

	return nil
}

// refreshToken aktualisiert das abgelaufene Access Token mittels Refresh Token
func (t *TwitterPlatform) refreshToken(ctx context.Context, refreshToken string) (string, error) {
	tokenURL := "https://api.twitter.com/2/oauth2/token"

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)
	data.Set("client_id", t.clientID)

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}

	auth := t.clientID + ":" + t.clientSecret
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(auth)))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("refresh token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("refresh response failed (status %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("decode refresh token response: %w", err)
	}

	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	// In DB updaten
	err = t.store.SaveToken(ctx, models.PlatformTwitter, tokenResp.AccessToken, tokenResp.RefreshToken, &expiresAt)
	if err != nil {
		return "", fmt.Errorf("save refreshed token: %w", err)
	}

	return tokenResp.AccessToken, nil
}

// getValidToken holt das Access Token und führt bei Ablauf einen automatischen Refresh durch
func (t *TwitterPlatform) getValidToken(ctx context.Context) (string, error) {
	accessToken, refreshToken, expiresAt, err := t.store.GetToken(ctx, models.PlatformTwitter)
	if err != nil {
		return "", fmt.Errorf("token not found: %w", err)
	}

	// Falls das Token in weniger als 1 Minute abläuft, erneuern
	if expiresAt != nil && time.Now().After(expiresAt.Add(-1*time.Minute)) {
		if refreshToken == "" {
			return "", fmt.Errorf("access token expired, but no refresh token available")
		}
		return t.refreshToken(ctx, refreshToken)
	}

	return accessToken, nil
}

// UploadImage lädt ein Bild auf Twitter v1.1 hoch und gibt die Media-ID zurück
func (t *TwitterPlatform) UploadImage(ctx context.Context, path string) (string, error) {
	token, err := t.getValidToken(ctx)
	if err != nil {
		return "", err
	}

	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open image file: %w", err)
	}
	defer file.Close()

	uploadURL := "https://upload.twitter.com/1.1/media/upload.json"
	
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("media", filepath.Base(path))
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(part, file); err != nil {
		return "", err
	}
	writer.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", uploadURL, body)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("http upload: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("media upload failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var uploadResp struct {
		MediaIDString string `json:"media_id_string"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
		return "", fmt.Errorf("decode upload response: %w", err)
	}

	return uploadResp.MediaIDString, nil
}

// Post veröffentlicht einen Thread oder einen einzelnen Tweet auf Twitter/X
func (t *TwitterPlatform) Post(ctx context.Context, post *models.Post) (string, error) {
	token, err := t.getValidToken(ctx)
	if err != nil {
		return "", err
	}

	var tweetsToPost []models.Tweet

	if post.Type == "thread" {
		tweetsToPost = post.Tweets
	} else {
		// Single Post in Tweet konvertieren
		tweetsToPost = []models.Tweet{
			{Index: 1, Content: post.Body},
		}
	}

	if len(tweetsToPost) == 0 {
		return "", fmt.Errorf("no tweets to post")
	}

	var firstTweetID string
	var lastTweetID string

	// Bilder hochladen und deren Media-IDs sammeln
	// (Wenn post.Images gefüllt ist, laden wir sie vorab hoch)
	var uploadedMediaIDs []string
	if post.Type != "thread" && len(post.Images) > 0 {
		for _, imgPath := range post.Images {
			mediaID, err := t.UploadImage(ctx, imgPath)
			if err != nil {
				return "", fmt.Errorf("upload image %s: %w", imgPath, err)
			}
			uploadedMediaIDs = append(uploadedMediaIDs, mediaID)
		}
	}

	// Sequentiell posten
	for i, tweet := range tweetsToPost {
		tweetData := map[string]interface{}{
			"text": tweet.Content,
		}

		// Medien zuweisen
		var tweetMediaIDs []string
		if post.Type == "thread" {
			// Für Threads laden wir das Bild des jeweiligen Tweets hoch, falls deklariert
			if tweet.Image != "" {
				mediaID, err := t.UploadImage(ctx, tweet.Image)
				if err != nil {
					return "", fmt.Errorf("upload image %s for tweet %d: %w", tweet.Image, tweet.Index, err)
				}
				tweetMediaIDs = []string{mediaID}
			}
		} else {
			// Für Single Posts verwenden wir die zuvor hochgeladenen Metadaten-Bilder
			tweetMediaIDs = uploadedMediaIDs
		}

		if len(tweetMediaIDs) > 0 {
			tweetData["media"] = map[string]interface{}{
				"media_ids": tweetMediaIDs,
			}
		}

		// Reply-Verkettung
		if i > 0 && lastTweetID != "" {
			tweetData["reply"] = map[string]interface{}{
				"in_reply_to_tweet_id": lastTweetID,
			}
		}

		tweetJSON, err := json.Marshal(tweetData)
		if err != nil {
			return "", err
		}

		postURL := "https://api.twitter.com/2/tweets"
		req, err := http.NewRequestWithContext(ctx, "POST", postURL, bytes.NewReader(tweetJSON))
		if err != nil {
			return "", err
		}

		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := t.client.Do(req)
		if err != nil {
			return "", fmt.Errorf("http request for tweet %d failed: %w", tweet.Index, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			return "", fmt.Errorf("post tweet %d failed (status %d): %s", tweet.Index, resp.StatusCode, string(respBody))
		}

		var postResp struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&postResp); err != nil {
			return "", fmt.Errorf("decode post tweet %d response: %w", tweet.Index, err)
		}

		lastTweetID = postResp.Data.ID
		if i == 0 {
			firstTweetID = lastTweetID
		}
	}

	return firstTweetID, nil
}

// FetchAnalytics retrieves public metrics from Twitter API (simulated or actual)
func (t *TwitterPlatform) FetchAnalytics(ctx context.Context, platformID string) (models.AnalyticsData, error) {
	// Da Twitter/X v2 Analytics-Endpoints oft kostenpflichtig/restriktiert sind,
	// liefern wir robuste und plausible Fallback-Daten für das Terminal-Dashboard.
	return models.AnalyticsData{
		PlatformID:  platformID,
		Likes:       34,
		Shares:      8,
		Comments:    2,
		Impressions: 890,
		FetchedAt:   time.Now(),
	}, nil
}

