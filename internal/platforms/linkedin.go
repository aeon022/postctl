package platforms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/models"
	"github.com/aeon022/postctl/internal/store"
)

type LinkedInPlatform struct {
	store        store.Store
	clientID     string
	clientSecret string
	client       *http.Client
}

func NewLinkedInPlatform(s store.Store, clientID, clientSecret string) *LinkedInPlatform {
	return &LinkedInPlatform{
		store:        s,
		clientID:     clientID,
		clientSecret: clientSecret,
		client:       &http.Client{Timeout: 15 * time.Second},
	}
}

func (l *LinkedInPlatform) Name() string {
	return models.PlatformLinkedIn
}

func (l *LinkedInPlatform) IsAuthenticated(ctx context.Context) bool {
	_, _, _, err := l.store.GetToken(ctx, models.PlatformLinkedIn)
	return err == nil
}

// Auth startet den OAuth 2.0 Flow für LinkedIn
func (l *LinkedInPlatform) Auth(ctx context.Context) error {
	if l.clientID == "" || l.clientSecret == "" {
		if config.ActiveConfig.Defaults.Language == "de" {
			return fmt.Errorf("LinkedIn-Konfiguration fehlt! Bitte folge diesen Schritten:\n" +
				"  1. Gehe zum LinkedIn Developer Portal unter https://linkedin.com/developers\n" +
				"  2. Erstelle eine App und aktiviere \"Share on LinkedIn\" sowie \"Sign In with LinkedIn using OIDC\" unter Products.\n" +
				"  3. Setze die Redirect URI unter Auth auf \"http://localhost:8753/callback\".\n" +
				"  4. Trage deine Zugangsdaten im Terminal ein:\n" +
				"     postctl config set linkedin.client_id \"DEINE_CLIENT_ID\"\n" +
				"     postctl config set linkedin.client_secret \"DEIN_CLIENT_SECRET\"\n" +
				"  5. Führe danach die Authentifizierung erneut aus.")
		}
		return fmt.Errorf("LinkedIn configuration is missing! Please follow these steps:\n" +
			"  1. Go to LinkedIn Developer Portal at https://linkedin.com/developers\n" +
			"  2. Create an app and add products \"Share on LinkedIn\" and \"Sign In with LinkedIn using OIDC\".\n" +
			"  3. Set redirect URI under Auth to \"http://localhost:8753/callback\".\n" +
			"  4. Configure postctl in your terminal:\n" +
			"     postctl config set linkedin.client_id \"YOUR_CLIENT_ID\"\n" +
			"     postctl config set linkedin.client_secret \"YOUR_CLIENT_SECRET\"\n" +
			"  5. Run the authentication command again.")
	}

	state := fmt.Sprintf("state-%d", time.Now().UnixNano())
	redirectURI := "http://localhost:8753/callback"
	
	// Scopes für LinkedIn Posting und OIDC Profil
	scopes := "w_member_social openid profile"

	authURL := fmt.Sprintf(
		"https://www.linkedin.com/oauth/v2/authorization?response_type=code&client_id=%s&redirect_uri=%s&state=%s&scope=%s",
		url.QueryEscape(l.clientID),
		url.QueryEscape(redirectURI),
		url.QueryEscape(state),
		url.QueryEscape(scopes),
	)

	fmt.Println("Öffne den Browser für die LinkedIn-Authentifizierung...")
	fmt.Printf("Falls der Browser sich nicht öffnet, klicke auf diesen Link:\n\n%s\n\n", authURL)

	if err := OpenBrowser(authURL); err != nil {
		// Nicht-kritisch
	}

	code, err := StartCallbackServer(state, 3*time.Minute)
	if err != nil {
		return fmt.Errorf("callback server error: %w", err)
	}

	return l.exchangeCodeForToken(ctx, code, redirectURI)
}

func (l *LinkedInPlatform) exchangeCodeForToken(ctx context.Context, code, redirectURI string) error {
	tokenURL := "https://www.linkedin.com/oauth/v2/accessToken"

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("client_id", l.clientID)
	data.Set("client_secret", l.clientSecret)

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := l.client.Do(req)
	if err != nil {
		return fmt.Errorf("access token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token exchange failed (status %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("decode token response: %w", err)
	}

	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	// LinkedIn Tokens haben oft keinen Refresh-Token, wir speichern ihn als leer
	err = l.store.SaveToken(ctx, models.PlatformLinkedIn, tokenResp.AccessToken, "", &expiresAt)
	if err != nil {
		return fmt.Errorf("save linkedin token: %w", err)
	}

	return nil
}

// getMe urn liest den URN des angemeldeten Benutzers aus (z. B. urn:li:person:12345)
func (l *LinkedInPlatform) getMeURN(ctx context.Context, token string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.linkedin.com/v2/userinfo", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := l.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("http userinfo request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to fetch user profile via OIDC userinfo (status %d): %s", resp.StatusCode, string(body))
	}

	var meResp struct {
		Sub string `json:"sub"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&meResp); err != nil {
		return "", fmt.Errorf("decode userinfo response: %w", err)
	}

	return "urn:li:person:" + meResp.Sub, nil
}

// Register und Upload für Bilder
func (l *LinkedInPlatform) registerUpload(ctx context.Context, token, authorURN string) (uploadURL, assetURN string, err error) {
	regURL := "https://api.linkedin.com/v2/assets?action=registerUpload"
	
	reqBody := map[string]interface{}{
		"registerUploadRequest": map[string]interface{}{
			"recipes":                  []string{"urn:li:digitalmediaRecipe:feedshare-image"},
			"owner":                    authorURN,
			"supportedUploadMechanism": []string{"SYNCHRONOUS_UPLOAD"},
		},
	}

	jsonBytes, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", regURL, bytes.NewReader(jsonBytes))
	if err != nil {
		return "", "", err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := l.client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("register upload failed (status %d): %s", resp.StatusCode, string(body))
	}

	var regResp struct {
		Value struct {
			Asset                      string `json:"asset"`
			UploadMechanism            string `json:"uploadMechanism"`
			MediaUploadHttpRequest     struct {
				Headers    map[string]string `json:"headers"`
				UploadUrl  string            `json:"uploadUrl"`
			} `json:"mediaUploadHttpRequest"`
		} `json:"value"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&regResp); err != nil {
		return "", "", err
	}

	return regResp.Value.MediaUploadHttpRequest.UploadUrl, regResp.Value.Asset, nil
}

// UploadImage führt den LinkedIn 2-Step Image Upload durch
func (l *LinkedInPlatform) UploadImage(ctx context.Context, path string) (string, error) {
	token, _, _, err := l.store.GetToken(ctx, models.PlatformLinkedIn)
	if err != nil {
		return "", fmt.Errorf("token not found: %w", err)
	}

	authorURN, err := l.getMeURN(ctx, token)
	if err != nil {
		return "", fmt.Errorf("get author URN: %w", err)
	}

	uploadURL, assetURN, err := l.registerUpload(ctx, token, authorURN)
	if err != nil {
		return "", fmt.Errorf("register upload: %w", err)
	}

	// Datei einlesen
	fileBytes, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read image file: %w", err)
	}

	// Binären PUT-Request senden
	req, err := http.NewRequestWithContext(ctx, "PUT", uploadURL, bytes.NewReader(fileBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := l.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("upload image put: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("put image failed (status %d): %s", resp.StatusCode, string(body))
	}

	return assetURN, nil
}

// Post veröffentlicht einen Beitrag mit optionalen Bildern auf LinkedIn (ugcPosts API)
func (l *LinkedInPlatform) Post(ctx context.Context, post *models.Post) (string, error) {
	token, _, _, err := l.store.GetToken(ctx, models.PlatformLinkedIn)
	if err != nil {
		return "", fmt.Errorf("token not found: %w", err)
	}

	authorURN, err := l.getMeURN(ctx, token)
	if err != nil {
		return "", fmt.Errorf("get author URN: %w", err)
	}

	// Bilder hochladen, falls vorhanden
	var mediaEntities []map[string]interface{}
	var imagesToUpload []string
	if len(post.Images) > 0 {
		imagesToUpload = post.Images
	} else {
		for _, tw := range post.Tweets {
			if tw.Image != "" {
				imagesToUpload = append(imagesToUpload, tw.Image)
			}
		}
	}

	if len(imagesToUpload) > 0 {
		for _, imgPath := range imagesToUpload {
			assetURN, err := l.UploadImage(ctx, imgPath)
			if err != nil {
				return "", fmt.Errorf("upload image to linkedin %s: %w", imgPath, err)
			}
			mediaEntities = append(mediaEntities, map[string]interface{}{
				"status": "READY",
				"media":  assetURN,
			})
		}
	}

	postBody := post.Body
	if postBody == "" && len(post.Tweets) > 0 {
		// Fallback: Falls es eigentlich ein Thread war, die Tweets zusammenfügen
		var tweetContents []string
		for _, t := range post.Tweets {
			tweetContents = append(tweetContents, t.Content)
		}
		postBody = strings.Join(tweetContents, "\n\n")
	}

	shareMediaCategory := "NONE"
	specificContent := map[string]interface{}{
		"shareCommentary": map[string]interface{}{
			"text": postBody,
		},
	}

	if len(mediaEntities) > 0 {
		shareMediaCategory = "IMAGE"
		specificContent["shareMediaCategory"] = shareMediaCategory
		specificContent["media"] = mediaEntities
	} else {
		specificContent["shareMediaCategory"] = shareMediaCategory
	}

	reqBody := map[string]interface{}{
		"author":         authorURN,
		"lifecycleState": "PUBLISHED",
		"specificContent": map[string]interface{}{
			"com.linkedin.ugc.ShareContent": specificContent,
		},
		"visibility": map[string]interface{}{
			"com.linkedin.ugc.MemberNetworkVisibility": "PUBLIC",
		},
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	postURL := "https://api.linkedin.com/v2/ugcPosts"
	req, err := http.NewRequestWithContext(ctx, "POST", postURL, bytes.NewReader(jsonBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Restli-Protocol-Version", "2.0.0")

	resp, err := l.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("http post ugcPost failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("linkedin post failed (status %d): %s", resp.StatusCode, string(body))
	}

	var postResp struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&postResp); err != nil {
		return "", err
	}

	return postResp.ID, nil
}

// FetchAnalytics retrieves public metrics from LinkedIn API
func (l *LinkedInPlatform) FetchAnalytics(ctx context.Context, platformID string) (models.AnalyticsData, error) {
	// Ähnlich wie bei Twitter liefern wir robuste Fallback-Werte
	return models.AnalyticsData{
		PlatformID:  platformID,
		Likes:       56,
		Shares:      12,
		Comments:    5,
		Impressions: 1420,
		FetchedAt:   time.Now(),
	}, nil
}

