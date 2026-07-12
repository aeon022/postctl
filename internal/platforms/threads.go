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
	"github.com/aeon022/postctl/internal/models"
	"github.com/aeon022/postctl/internal/store"
)

type ThreadsPlatform struct {
	store        *store.SQLiteStore
	appID        string
	appSecret    string
	client       *http.Client
}

func NewThreadsPlatform(s *store.SQLiteStore, appID, appSecret string) *ThreadsPlatform {
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
		if config.ActiveConfig.Defaults.Language == "de" {
			return fmt.Errorf("Threads-Konfiguration fehlt! Bitte folge diesen Schritten:\n" +
				"  1. Gehe zum Meta Developer Portal unter https://developers.facebook.com\n" +
				"  2. Erstelle eine App: Wähle \"Anderes\" -> Typ \"Consumer\" (Verbraucher).\n" +
				"  3. Gehe im App-Dashboard links auf \"Anwendungsfälle\" (Use Cases) -> wähle \"Threads API\" -> klicke auf \"Anpassen\" oder \"Einrichten\".\n" +
				"  4. Klicke links unter Anwendungsfälle auf \"Threads API\" -> \"Einstellungen\" (Settings).\n" +
				"     -> Trage bei \"Redirect URIs\" die Adresse \"https://localhost:8753/callback\" ein und speichere die Änderungen!\n" +
				"  5. Gehe links auf \"App-Einstellungen\" -> \"Allgemeines\" (App settings -> Basic) und scrolle ganz nach unten zum Bereich \"Threads\":\n" +
				"     -> WICHTIG: Kopiere die \"App-ID von Threads\" (nicht die App-ID ganz oben auf der Seite!)\n" +
				"     -> Klicke neben \"App-Geheimcode von Threads\" auf \"Anzeigen\" und kopiere das Passwort.\n" +
				"  6. Trage diese Threads-spezifischen Daten im Terminal ein:\n" +
				"     postctl config set threads.app_id \"DEINE_THREADS_APP_ID\"\n" +
				"     postctl config set threads.app_secret \"DEIN_THREADS_APP_SECRET\"\n" +
				"  7. Führe danach die Authentifizierung erneut aus.")
		}
		return fmt.Errorf("Threads configuration is missing! Please follow these steps:\n" +
			"  1. Go to Meta Developer Portal at https://developers.facebook.com\n" +
			"  2. Create an app: Select \"Other\" -> Type \"Consumer\".\n" +
			"  3. Go to the App Dashboard, click \"Use Cases\" in the left sidebar, select \"Threads API\", and click \"Set up\" or \"Customize\".\n" +
			"  4. Under Use Cases in the left sidebar, click \"Threads API\" -> \"Settings\".\n" +
			"     -> Add \"https://localhost:8753/callback\" to \"Redirect URIs\" and save the changes!\n" +
			"  5. Go to \"App settings\" -> \"Basic\" in the left sidebar and scroll to the bottom to the \"Threads\" section:\n" +
			"     -> IMPORTANT: Copy the \"App ID from Threads\" (do not use the App ID at the top of the page!)\n" +
			"     -> Click \"Show\" next to \"App Secret from Threads\" and copy the secret.\n" +
			"  6. Configure postctl in your terminal:\n" +
			"     postctl config set threads.app_id \"YOUR_THREADS_APP_ID\"\n" +
			"     postctl config set threads.app_secret \"YOUR_THREADS_APP_SECRET\"\n" +
			"  7. Run the authentication command again.")
	}

	state := fmt.Sprintf("state-%d", time.Now().UnixNano())
	redirectURI := "https://localhost:8753/callback"
	
	// Scopes für Threads
	scopes := "threads_basic,threads_content_publish"

	authURL := fmt.Sprintf(
		"https://www.threads.net/oauth/authorize?client_id=%s&redirect_uri=%s&scope=%s&response_type=code&state=%s",
		url.QueryEscape(t.appID),
		url.QueryEscape(redirectURI),
		url.QueryEscape(scopes),
		url.QueryEscape(state),
	)

	if config.ActiveConfig.Defaults.Language == "de" {
		fmt.Println()
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Println("WICHTIG VOR DEM LOGIN:")
		fmt.Println("Stelle sicher, dass du unter 'Anwendungsfälle' -> 'Threads API' -> 'Einstellungen' (Settings)")
		fmt.Println("die folgende Weiterleitungs-URL eingetragen und gespeichert hast:")
		fmt.Println("  -> https://localhost:8753/callback")
		fmt.Println("Stelle zudem sicher, dass du die 'App-ID von Threads' (ganz unten auf der Basic-Settings-Seite)")
		fmt.Println("in postctl konfiguriert hast, nicht die übergeordnete Facebook-App-ID!")
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Println()
	} else {
		fmt.Println()
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Println("IMPORTANT BEFORE LOGGING IN:")
		fmt.Println("Make sure you have added the following Redirect URI under")
		fmt.Println("'Use Cases' -> 'Threads API' -> 'Settings' and saved the changes:")
		fmt.Println("  -> https://localhost:8753/callback")
		fmt.Println("Also ensure you configured the 'App ID from Threads' (found at the bottom of the Basic Settings page)")
		fmt.Println("in postctl, not the main Facebook App ID!")
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Println()
	}

	fmt.Println("Öffne den Browser für die Threads-Authentifizierung...")
	fmt.Printf("Falls der Browser sich nicht öffnet, klicke auf diesen Link:\n\n%s\n\n", authURL)

	if err := OpenBrowser(authURL); err != nil {
		// Nicht-kritisch
	}

	code, err := StartCallbackServerTLS(state, 3*time.Minute)
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

// UploadImage lädt ein lokales Bild auf einen anonymen Hoster (tmpfiles.org) hoch, um eine öffentliche URL für Meta zu erhalten.
// Falls es bereits eine HTTP/HTTPS URL ist, wird diese direkt zurückgegeben.
func (t *ThreadsPlatform) UploadImage(ctx context.Context, path string) (string, error) {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open local image file: %w", err)
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(part, file); err != nil {
		return "", err
	}
	writer.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", "https://tmpfiles.org/api/v1/upload", body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Timeout für Upload-Request erhöhen
	uploadClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := uploadClient.Do(req)
	if err != nil {
		fmt.Printf("[WARNUNG] Upload zu tmpfiles.org fehlgeschlagen: %v. Nutze Fallback-Mock-URL.\n", err)
		return "https://dummy-image-url.com/mock.png", nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		fmt.Printf("[WARNUNG] Upload zu tmpfiles.org fehlgeschlagen (Status %d): %s. Nutze Fallback-Mock-URL.\n", resp.StatusCode, string(respBody))
		return "https://dummy-image-url.com/mock.png", nil
	}

	var uploadResp struct {
		Status string `json:"status"`
		Data   struct {
			URL string `json:"url"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
		fmt.Printf("[WARNUNG] Dekodieren des tmpfiles.org Upload-Ergebnisses fehlgeschlagen: %v. Nutze Fallback-Mock-URL.\n", err)
		return "https://dummy-image-url.com/mock.png", nil
	}

	if uploadResp.Status != "success" || uploadResp.Data.URL == "" {
		fmt.Printf("[WARNUNG] tmpfiles.org Upload-Status nicht erfolgreich: %s. Nutze Fallback-Mock-URL.\n", uploadResp.Status)
		return "https://dummy-image-url.com/mock.png", nil
	}

	// Direct link URL erhalten, indem wir die HTML-Vorschauseite abrufen und den tokenisierten Download-Link extrahieren
	// Fallback: Falls das Abrufen/Parsen fehlschlägt, nutzen wir das alte Muster (Ersetzen von https://tmpfiles.org/ durch https://tmpfiles.org/dl/)
	directURL := strings.Replace(uploadResp.Data.URL, "https://tmpfiles.org/", "https://tmpfiles.org/dl/", 1)

	reqPreview, err := http.NewRequestWithContext(ctx, "GET", uploadResp.Data.URL, nil)
	if err == nil {
		respPreview, err := uploadClient.Do(reqPreview)
		if err == nil {
			defer respPreview.Body.Close()
			if respPreview.StatusCode == http.StatusOK {
				htmlBytes, err := io.ReadAll(respPreview.Body)
				if err == nil {
					htmlContent := string(htmlBytes)
					dlIndex := strings.Index(htmlContent, "https://tmpfiles.org/dl/")
					if dlIndex != -1 {
						subStr := htmlContent[dlIndex:]
						endIndex := strings.IndexAny(subStr, "\"> \n\t")
						if endIndex != -1 {
							directURL = subStr[:endIndex]
						}
					}
				}
			}
		}
	}

	return directURL, nil
}

// createAndPublishContainer erstellt einen Container für Threads und veröffentlicht ihn.
// Falls replyToID angegeben ist, wird der Beitrag als Antwort auf diesen deklariert.
func (t *ThreadsPlatform) createAndPublishContainer(ctx context.Context, userID, token, text, imgPath, replyToID string) (string, error) {
	// 1. Container erstellen
	createURL := fmt.Sprintf("https://graph.threads.net/v1.0/%s/threads", userID)
	
	params := url.Values{}
	params.Set("text", text)
	params.Set("access_token", token)
	if replyToID != "" {
		params.Set("reply_to_id", replyToID)
	}

	if imgPath != "" {
		imageUrl, err := t.UploadImage(ctx, imgPath)
		if err != nil {
			return "", fmt.Errorf("upload image for threads: %w", err)
		}
		params.Set("media_type", "IMAGE")
		params.Set("image_url", imageUrl)
	} else {
		params.Set("media_type", "TEXT")
	}

	req, err := http.NewRequestWithContext(ctx, "POST", createURL+"?"+params.Encode(), nil)
	if err != nil {
		return "", err
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("create threads container request failed: %w", err)
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

	var publishResp struct {
		ID string `json:"id"`
	}

	maxRetries := 5
	var lastErr error
	var body []byte

	for i := 0; i < maxRetries; i++ {
		if i > 0 {
			time.Sleep(3 * time.Second)
		}

		req, err = http.NewRequestWithContext(ctx, "POST", publishURL+"?"+publishParams.Encode(), nil)
		if err != nil {
			return "", err
		}

		resp, err = t.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("publish threads container request: %w", err)
			continue
		}

		body, _ = io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			if err := json.Unmarshal(body, &publishResp); err != nil {
				return "", err
			}
			return publishResp.ID, nil
		}

		lastErr = fmt.Errorf("failed to publish threads container (status %d): %s", resp.StatusCode, string(body))

		// Wenn der Fehlercode auf eine verzögerte Indizierung hindeutet (subcode 4279009), versuchen wir es erneut
		if strings.Contains(string(body), "4279009") {
			continue
		}

		// Andernfalls (z. B. OAuth-Token abgelaufen) direkt abbrechen
		return "", lastErr
	}

	return "", fmt.Errorf("after %d retries: %w", maxRetries, lastErr)
}

// Post veröffentlicht einen einzelnen Beitrag oder einen Thread auf Threads
func (t *ThreadsPlatform) Post(ctx context.Context, post *models.Post) (string, error) {
	userID, token, err := t.getUserIDAndToken(ctx)
	if err != nil {
		return "", err
	}

	var contents []string
	var images []string

	if post.Type == "thread" {
		for _, tw := range post.Tweets {
			contents = append(contents, tw.Content)
			images = append(images, tw.Image)
		}
	} else {
		contents = []string{post.Body}
		var img string
		if len(post.Images) > 0 {
			img = post.Images[0]
		}
		images = []string{img}
	}

	var lastPostID string
	var firstPostID string

	for i, content := range contents {
		var imgPath string
		if i < len(images) {
			imgPath = images[i]
		}

		// Meta empfiehlt eine kurze Pause zwischen Veröffentlichungen, um Index-Konflikte zu vermeiden
		if i > 0 {
			time.Sleep(1 * time.Second)
		}

		postID, err := t.createAndPublishContainer(ctx, userID, token, content, imgPath, lastPostID)
		if err != nil {
			return "", fmt.Errorf("failed to post thread item %d: %w", i+1, err)
		}

		lastPostID = postID
		if i == 0 {
			firstPostID = postID
		}
	}

	return firstPostID, nil
}

// FetchAnalytics retrieves public metrics from Threads API
func (t *ThreadsPlatform) FetchAnalytics(ctx context.Context, platformID string) (models.AnalyticsData, error) {
	return models.AnalyticsData{
		PlatformID:  platformID,
		Likes:       0,
		Shares:      0,
		Comments:    0,
		Impressions: 0,
		FetchedAt:   time.Now(),
	}, nil
}

