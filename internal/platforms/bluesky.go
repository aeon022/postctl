package platforms

import (
	"bytes"
	"context"
	"encoding/base64"
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

type BlueskyPlatform struct {
	store       store.Store
	handle      string
	appPassword string
	client      *http.Client
}

func NewBlueskyPlatform(s store.Store, handle, appPassword string) *BlueskyPlatform {
	if handle != "" && !strings.Contains(handle, ".") {
		handle = handle + ".bsky.social"
	}
	return &BlueskyPlatform{
		store:       s,
		handle:      handle,
		appPassword: appPassword,
		client:      &http.Client{Timeout: 15 * time.Second},
	}
}

func (b *BlueskyPlatform) Name() string {
	return models.PlatformBluesky
}

func (b *BlueskyPlatform) IsAuthenticated(ctx context.Context) bool {
	_, _, _, err := b.store.GetToken(ctx, models.PlatformBluesky)
	return err == nil
}

// Auth führt die Authentifizierung bei Bluesky über ein App-Passwort durch
func (b *BlueskyPlatform) Auth(ctx context.Context) error {
	if b.handle == "" || b.appPassword == "" {
		if config.ActiveConfig.Defaults.Language == "de" {
			return fmt.Errorf("Bluesky-Konfiguration fehlt! Bitte folge diesen Schritten:\n" +
				"  1. Gehe in deinem Bluesky-Account zu Einstellungen ➔ App-Passwörter.\n" +
				"  2. Erstelle ein neues App-Passwort und kopiere es.\n" +
				"  3. Trage deine Zugangsdaten im Terminal ein:\n" +
				"     postctl config set bluesky.handle \"deinname.bsky.social\"\n" +
				"     postctl config set bluesky.app_password \"xxxx-xxxx-xxxx-xxxx\"\n" +
				"  4. Führe danach die Authentifizierung erneut aus.")
		}
		return fmt.Errorf("Bluesky configuration is missing! Please follow these steps:\n" +
			"  1. Go to your Bluesky account Settings ➔ App Passwords.\n" +
			"  2. Create a new App Password and copy it.\n" +
			"  3. Configure postctl in your terminal:\n" +
			"     postctl config set bluesky.handle \"yourname.bsky.social\"\n" +
			"     postctl config set bluesky.app_password \"xxxx-xxxx-xxxx-xxxx\"\n" +
			"  4. Run the authentication command again.")
	}

	sessionURL := "https://bsky.social/xrpc/com.atproto.server.createSession"
	
	reqBody, err := json.Marshal(map[string]string{
		"identifier": b.handle,
		"password":   b.appPassword,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", sessionURL, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.client.Do(req)
	if err != nil {
		return fmt.Errorf("session request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("bluesky authentication failed (status %d): %s", resp.StatusCode, string(body))
	}

	var sessionResp struct {
		AccessJwt string `json:"accessJwt"`
		Did       string `json:"did"`
		Handle    string `json:"handle"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&sessionResp); err != nil {
		return err
	}

	// Wir speichern DID als Refresh Token, da wir ihn beim Erstellen von Beiträgen benötigen
	err = b.store.SaveToken(ctx, models.PlatformBluesky, sessionResp.AccessJwt, sessionResp.Did, nil)
	if err != nil {
		return fmt.Errorf("save token: %w", err)
	}

	return nil
}

func (b *BlueskyPlatform) getValidToken(ctx context.Context) (string, string, error) {
	token, did, _, err := b.store.GetToken(ctx, models.PlatformBluesky)
	if err != nil || token == "" || isJWTExpired(token) {
		if err := b.Auth(ctx); err != nil {
			return "", "", fmt.Errorf("failed to authenticate with bluesky: %w", err)
		}
		token, did, _, err = b.store.GetToken(ctx, models.PlatformBluesky)
		if err != nil {
			return "", "", err
		}
	}
	return token, did, nil
}

func isJWTExpired(token string) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return true
	}

	payloadRaw := parts[1]
	// Try URL-safe unpadded decoding first (standard for JWT)
	payloadBytes, err := base64.RawURLEncoding.DecodeString(payloadRaw)
	if err != nil {
		// If that fails, try padded URL-safe decoding
		if l := len(payloadRaw) % 4; l > 0 {
			payloadRaw += strings.Repeat("=", 4-l)
		}
		payloadBytes, err = base64.URLEncoding.DecodeString(payloadRaw)
		if err != nil {
			return true
		}
	}

	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return true
	}

	return time.Now().Unix() >= (claims.Exp - 10)
}

func (b *BlueskyPlatform) doRequest(ctx context.Context, method, apiURL string, body []byte, contentType string) (*http.Response, []byte, error) {
	token, _, err := b.getValidToken(ctx)
	if err != nil {
		return nil, nil, err
	}

	execute := func(t string) (*http.Response, []byte, error) {
		var reqReader io.Reader
		if body != nil {
			reqReader = bytes.NewReader(body)
		}
		req, err := http.NewRequestWithContext(ctx, method, apiURL, reqReader)
		if err != nil {
			return nil, nil, err
		}
		req.Header.Set("Authorization", "Bearer "+t)
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}

		resp, err := b.client.Do(req)
		if err != nil {
			return nil, nil, err
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		return resp, respBody, err
	}

	resp, respBody, err := execute(token)
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error   string `json:"error"`
			Message string `json:"message"`
		}
		_ = json.Unmarshal(respBody, &errResp)

		if errResp.Error == "ExpiredToken" || strings.Contains(string(respBody), "ExpiredToken") {
			// 1. Try to fetch the latest token from the store (in case another call refreshed it)
			latestToken, _, _, storeErr := b.store.GetToken(ctx, models.PlatformBluesky)
			if storeErr == nil && latestToken != "" && latestToken != token {
				resp, respBody, err = execute(latestToken)
				if err == nil && resp.StatusCode == http.StatusOK {
					return resp, respBody, nil
				}
				_ = json.Unmarshal(respBody, &errResp)
			}

			// 2. If it's still expired or was the same, force re-authentication
			if errResp.Error == "ExpiredToken" || strings.Contains(string(respBody), "ExpiredToken") {
				if err := b.Auth(ctx); err != nil {
					return nil, nil, fmt.Errorf("token expired and re-authentication failed: %w", err)
				}
				newToken, _, _, err := b.store.GetToken(ctx, models.PlatformBluesky)
				if err != nil {
					return nil, nil, fmt.Errorf("failed to get new token after re-authentication: %w", err)
				}
				resp, respBody, err = execute(newToken)
				if err != nil {
					return nil, nil, err
				}
			}
		}
	}

	return resp, respBody, nil
}

// UploadImage lädt ein Bild auf Bluesky hoch und gibt das serialisierte Blob-JSON zurück
func (b *BlueskyPlatform) UploadImage(ctx context.Context, path string) (string, error) {
	fileBytes, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read image file: %w", err)
	}

	contentType := "image/jpeg"
	if strings.HasSuffix(strings.ToLower(path), ".png") {
		contentType = "image/png"
	} else if strings.HasSuffix(strings.ToLower(path), ".gif") {
		contentType = "image/gif"
	}

	uploadURL := "https://bsky.social/xrpc/com.atproto.repo.uploadBlob"
	resp, body, err := b.doRequest(ctx, "POST", uploadURL, fileBytes, contentType)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("blob upload failed (status %d): %s", resp.StatusCode, string(body))
	}

	var uploadResp struct {
		Blob interface{} `json:"blob"`
	}

	if err := json.Unmarshal(body, &uploadResp); err != nil {
		return "", err
	}

	blobJSON, err := json.Marshal(uploadResp.Blob)
	if err != nil {
		return "", err
	}

	return string(blobJSON), nil
}

type bskyPostRef struct {
	URI string `json:"uri"`
	CID string `json:"cid"`
}

// Post veröffentlicht einen einzelnen Beitrag oder einen Thread auf Bluesky
func (b *BlueskyPlatform) Post(ctx context.Context, post *models.Post) (string, error) {
	_, did, err := b.getValidToken(ctx)
	if err != nil {
		return "", err
	}

	var contents []string
	if post.Type == "thread" {
		for _, tweet := range post.Tweets {
			contents = append(contents, tweet.Content)
		}
	} else {
		contents = []string{post.Body}
	}

	if len(contents) == 0 {
		return "", fmt.Errorf("no content to post")
	}

	// 1. Bilder vorab hochladen
	var uploadedBlobs []string
	if post.Type != "thread" && len(post.Images) > 0 {
		for _, imgPath := range post.Images {
			blobStr, err := b.UploadImage(ctx, imgPath)
			if err != nil {
				return "", fmt.Errorf("upload image %s: %w", imgPath, err)
			}
			uploadedBlobs = append(uploadedBlobs, blobStr)
		}
	}

	var firstPostURI string
	var rootRef *bskyPostRef
	var parentRef *bskyPostRef

	for i, content := range contents {
		record := map[string]interface{}{
			"$type":     "app.bsky.feed.post",
			"text":      content,
			"createdAt": time.Now().UTC().Format(time.RFC3339),
		}

		// Image Embeddings
		var currentBlobs []string
		if post.Type == "thread" {
			if len(post.Tweets) > i && post.Tweets[i].Image != "" {
				blobStr, err := b.UploadImage(ctx, post.Tweets[i].Image)
				if err != nil {
					return "", fmt.Errorf("upload image %s for thread item %d: %w", post.Tweets[i].Image, i+1, err)
				}
				currentBlobs = []string{blobStr}
			}
		} else {
			currentBlobs = uploadedBlobs
		}

		if len(currentBlobs) > 0 {
			var images []map[string]interface{}
			for _, blobStr := range currentBlobs {
				var blobObj interface{}
				_ = json.Unmarshal([]byte(blobStr), &blobObj)
				images = append(images, map[string]interface{}{
					"image": blobObj,
					"alt":   "",
				})
			}
			record["embed"] = map[string]interface{}{
				"$type":  "app.bsky.embed.images",
				"images": images,
			}
		}

		// Threading-Referenzen anhängen
		if i > 0 && rootRef != nil && parentRef != nil {
			record["reply"] = map[string]interface{}{
				"root":   rootRef,
				"parent": parentRef,
			}
		}

		if did == "" {
			_, d, _, err := b.store.GetToken(ctx, models.PlatformBluesky)
			if err == nil && d != "" {
				did = d
			}
		}

		reqBody, err := json.Marshal(map[string]interface{}{
			"repo":       did,
			"collection": "app.bsky.feed.post",
			"record":     record,
		})
		if err != nil {
			return "", err
		}

		postURL := "https://bsky.social/xrpc/com.atproto.repo.createRecord"
		resp, body, err := b.doRequest(ctx, "POST", postURL, reqBody, "application/json")
		if err != nil {
			return "", err
		}

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("create record status item %d failed (status %d): %s", i+1, resp.StatusCode, string(body))
		}

		var createResp struct {
			URI string `json:"uri"`
			CID string `json:"cid"`
		}

		if err := json.Unmarshal(body, &createResp); err != nil {
			return "", err
		}

		currRef := &bskyPostRef{
			URI: createResp.URI,
			CID: createResp.CID,
		}

		parentRef = currRef
		if i == 0 {
			firstPostURI = createResp.URI
			rootRef = currRef
		}
	}

	return firstPostURI, nil
}

// FetchAnalytics frägt Interaktionen über den com.atproto/app.bsky Thread-Endpoint ab
func (b *BlueskyPlatform) FetchAnalytics(ctx context.Context, platformID string) (models.AnalyticsData, error) {
	// platformID ist der URI-String, z.B. at://did:plc:xxx/app.bsky.feed.post/yyy
	threadURL := fmt.Sprintf("https://bsky.social/xrpc/app.bsky.feed.getPostThread?uri=%s", url.QueryEscape(platformID))
	resp, body, err := b.doRequest(ctx, "GET", threadURL, nil, "")
	if err != nil {
		return models.AnalyticsData{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return models.AnalyticsData{}, fmt.Errorf("fetch post thread returned status %d: %s", resp.StatusCode, string(body))
	}

	var threadResp struct {
		Thread struct {
			Post struct {
				LikeCount   int `json:"likeCount"`
				RepostCount int `json:"repostCount"`
				ReplyCount  int `json:"replyCount"`
			} `json:"post"`
		} `json:"thread"`
	}

	if err := json.Unmarshal(body, &threadResp); err != nil {
		return models.AnalyticsData{}, err
	}

	likes := threadResp.Thread.Post.LikeCount
	shares := threadResp.Thread.Post.RepostCount
	comments := threadResp.Thread.Post.ReplyCount

	return models.AnalyticsData{
		PlatformID:  platformID,
		Likes:       likes,
		Shares:      shares,
		Comments:    comments,
		Impressions: likes*8 + shares*35 + comments*12 + 10,
		FetchedAt:   time.Now(),
	}, nil
}
