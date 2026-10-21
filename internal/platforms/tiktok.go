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

	"github.com/aeon022/postctl/internal/models"
	"github.com/aeon022/postctl/internal/store"
)

const (
	tiktokAPI      = "https://open.tiktokapis.com"
	tiktokAuthURL  = "https://www.tiktok.com/v2/auth/authorize/"
	tiktokScope    = "video.upload"
	tiktokMaxChunk = 64 << 20 // one chunk up to 64 MB; larger files would need chunked upload
)

// TikTokPlatform sends videos to the creator's TikTok inbox (Content Posting API,
// scope video.upload). That is a draft: the creator opens the in-app notification,
// adds caption and sound and publishes. Direct Post needs an audited app; photo
// posts need a verified URL prefix, so only video uploads are supported here.
type TikTokPlatform struct {
	store        *store.SQLiteStore
	clientKey    string
	clientSecret string
	client       *http.Client
	apiBase      string
	authURL      string
}

func NewTikTokPlatform(s *store.SQLiteStore, clientKey, clientSecret string) *TikTokPlatform {
	return &TikTokPlatform{
		store:        s,
		clientKey:    clientKey,
		clientSecret: clientSecret,
		client:       &http.Client{Timeout: 2 * time.Minute},
		apiBase:      tiktokAPI,
		authURL:      tiktokAuthURL,
	}
}

func (t *TikTokPlatform) Name() string { return models.PlatformTikTok }

func (t *TikTokPlatform) IsAuthenticated(ctx context.Context) bool {
	_, _, _, err := t.store.GetToken(ctx, models.PlatformTikTok)
	return err == nil
}

// Auth runs the OAuth authorization-code flow with PKCE (loopback redirect).
func (t *TikTokPlatform) Auth(ctx context.Context) error {
	if t.clientKey == "" || t.clientSecret == "" {
		return fmt.Errorf("tiktok: client_key und client_secret fehlen (postctl config set tiktok.client_key …)")
	}
	verifier, err := GenerateVerifier()
	if err != nil {
		return err
	}
	state := GenerateState()
	redirectURI := "http://127.0.0.1:8753/callback"
	q := url.Values{}
	q.Set("client_key", t.clientKey)
	q.Set("scope", tiktokScope)
	q.Set("response_type", "code")
	q.Set("redirect_uri", redirectURI)
	q.Set("state", state)
	q.Set("code_challenge", GenerateChallenge(verifier))
	q.Set("code_challenge_method", "S256")
	authURL := t.authURL + "?" + q.Encode()

	fmt.Println("Öffne den Browser für die TikTok-Authentifizierung...")
	fmt.Printf("Falls der Browser sich nicht öffnet, klicke auf diesen Link:\n\n%s\n\n", authURL)
	_ = OpenBrowser(authURL)

	code, err := StartCallbackServer(state, 3*time.Minute)
	if err != nil {
		return fmt.Errorf("callback server error: %w", err)
	}
	return t.tokenRequest(ctx, url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"code_verifier": {verifier},
	})
}

// tokenRequest posts to the token endpoint and stores access and refresh token.
func (t *TikTokPlatform) tokenRequest(ctx context.Context, form url.Values) error {
	form.Set("client_key", t.clientKey)
	form.Set("client_secret", t.clientSecret)
	req, err := http.NewRequestWithContext(ctx, "POST", t.apiBase+"/v2/oauth/token/", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var tr struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		Error        string `json:"error"`
		Description  string `json:"error_description"`
	}
	_ = json.Unmarshal(body, &tr)
	if resp.StatusCode != http.StatusOK || tr.AccessToken == "" {
		return fmt.Errorf("tiktok: token request failed (status %d): %s %s", resp.StatusCode, tr.Error, tr.Description)
	}
	exp := time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)
	return t.store.SaveToken(ctx, models.PlatformTikTok, tr.AccessToken, tr.RefreshToken, &exp)
}

// getValidToken returns the access token, refreshing it (24 h lifetime) when it is about to expire.
func (t *TikTokPlatform) getValidToken(ctx context.Context) (string, error) {
	tok, refresh, exp, err := t.store.GetToken(ctx, models.PlatformTikTok)
	if err != nil {
		return "", fmt.Errorf("tiktok: nicht angemeldet (postctl auth tiktok): %w", err)
	}
	if exp != nil && time.Until(*exp) < 5*time.Minute {
		if refresh == "" {
			return "", fmt.Errorf("tiktok: Token abgelaufen, bitte neu anmelden (postctl auth tiktok)")
		}
		if err := t.tokenRequest(ctx, url.Values{"grant_type": {"refresh_token"}, "refresh_token": {refresh}}); err != nil {
			return "", err
		}
		tok, _, _, err = t.store.GetToken(ctx, models.PlatformTikTok)
		if err != nil {
			return "", err
		}
	}
	return tok, nil
}

// Post uploads post.Images[0] (an .mp4/.mov URL or local path) to the TikTok inbox
// and returns the publish_id. The caption in post.Body cannot be sent by the inbox
// API; the creator types it in the app.
func (t *TikTokPlatform) Post(ctx context.Context, post *models.Post) (string, error) {
	if len(post.Images) != 1 || !isVideo(post.Images[0]) {
		return "", fmt.Errorf("tiktok: ein Entwurf braucht genau ein Video (.mp4/.mov) unter images (Foto-Posts erfordern eine verifizierte URL-Domain)")
	}
	token, err := t.getValidToken(ctx)
	if err != nil {
		return "", err
	}
	video, err := t.readVideo(ctx, post.Images[0])
	if err != nil {
		return "", err
	}
	if len(video) > tiktokMaxChunk {
		return "", fmt.Errorf("tiktok: Video %d MB, mehr als %d MB brauchen Chunk-Upload (nicht implementiert)", len(video)>>20, tiktokMaxChunk>>20)
	}

	initBody, _ := json.Marshal(map[string]any{"source_info": map[string]any{
		"source": "FILE_UPLOAD", "video_size": len(video), "chunk_size": len(video), "total_chunk_count": 1,
	}})
	req, err := http.NewRequestWithContext(ctx, "POST", t.apiBase+"/v2/post/publish/inbox/video/init/", bytes.NewReader(initBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("tiktok: init request failed: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var ir struct {
		Data struct {
			PublishID string `json:"publish_id"`
			UploadURL string `json:"upload_url"`
		} `json:"data"`
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	_ = json.Unmarshal(raw, &ir)
	if resp.StatusCode != http.StatusOK || (ir.Error.Code != "" && ir.Error.Code != "ok") || ir.Data.UploadURL == "" {
		return "", fmt.Errorf("tiktok: init failed (status %d): %s %s", resp.StatusCode, ir.Error.Code, ir.Error.Message)
	}

	put, err := http.NewRequestWithContext(ctx, "PUT", ir.Data.UploadURL, bytes.NewReader(video))
	if err != nil {
		return "", err
	}
	put.Header.Set("Content-Type", videoMIME(post.Images[0]))
	put.Header.Set("Content-Range", fmt.Sprintf("bytes 0-%d/%d", len(video)-1, len(video)))
	presp, err := t.client.Do(put)
	if err != nil {
		return "", fmt.Errorf("tiktok: upload failed: %w", err)
	}
	defer presp.Body.Close()
	if presp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(io.LimitReader(presp.Body, 1024))
		return "", fmt.Errorf("tiktok: upload failed (status %d): %s", presp.StatusCode, b)
	}
	// ponytail: kein Status-Polling (/publish/status/fetch); das 2xx des Uploads gilt als „im Posteingang“. Polling ergänzen, falls Uploads stumm scheitern.
	return ir.Data.PublishID, nil
}

func videoMIME(u string) string {
	if strings.HasSuffix(strings.ToLower(strings.SplitN(u, "?", 2)[0]), ".mov") {
		return "video/quicktime"
	}
	return "video/mp4"
}

func (t *TikTokPlatform) readVideo(ctx context.Context, src string) ([]byte, error) {
	if !strings.HasPrefix(src, "http://") && !strings.HasPrefix(src, "https://") {
		return os.ReadFile(src)
	}
	req, err := http.NewRequestWithContext(ctx, "GET", src, nil)
	if err != nil {
		return nil, err
	}
	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tiktok: download video: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tiktok: download video: status %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, tiktokMaxChunk+1))
}

func (t *TikTokPlatform) UploadImage(ctx context.Context, path string) (string, error) {
	return "", fmt.Errorf("tiktok: Bild-Upload nicht unterstützt (nur Video-Entwürfe)")
}

func (t *TikTokPlatform) FetchAnalytics(ctx context.Context, platformID string) (models.AnalyticsData, error) {
	return models.AnalyticsData{}, nil
}

// Delete: an inbox upload can only be discarded in the TikTok app.
func (t *TikTokPlatform) Delete(ctx context.Context, platformID string) error {
	return fmt.Errorf("tiktok: die API kann Entwürfe nicht löschen — publish_id %s in der TikTok-App verwerfen", platformID)
}
