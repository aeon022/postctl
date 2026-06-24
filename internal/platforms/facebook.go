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

type FacebookPlatform struct {
	store        store.Store
	appID        string
	appSecret    string
	pageID       string
	client       *http.Client
}

func NewFacebookPlatform(s store.Store, appID, appSecret, pageID string) *FacebookPlatform {
	return &FacebookPlatform{
		store:        s,
		appID:        appID,
		appSecret:    appSecret,
		pageID:       pageID,
		client:       &http.Client{Timeout: 20 * time.Second},
	}
}

func (f *FacebookPlatform) Name() string {
	return models.PlatformFacebook
}

func (f *FacebookPlatform) IsAuthenticated(ctx context.Context) bool {
	_, _, _, err := f.store.GetToken(ctx, models.PlatformFacebook)
	return err == nil
}

// Auth startet den OAuth Flow für Facebook via StartCallbackServerTLS (HTTPS)
func (f *FacebookPlatform) Auth(ctx context.Context) error {
	if f.appID == "" || f.appSecret == "" || f.pageID == "" {
		if config.ActiveConfig.Defaults.Language == "de" {
			return fmt.Errorf("Facebook-Konfiguration fehlt! Bitte folge diesen Schritten:\n" +
				"  1. Gehe zum Meta Developer Portal unter https://developers.facebook.com\n" +
				"  2. Erstelle eine App: Klicke \"App erstellen\" und wähle ganz unten die Option \"Eine App ohne Anwendungsfall erstellen\" (Create an app without a use case).\n" +
				"  3. Gehe im App-Dashboard links auf \"Anwendungsfälle\" (Use Cases) -> wähle den Anwendungsfall **\"Alles auf deiner Seite verwalten\"** (das mit dem Flaggen-Icon).\n" +
				"  4. Klicke dort auf **\"Anpassen\"** (Customize).\n" +
				"     -> WICHTIG: Scrolle in der Liste nach unten und klicke neben **\"pages_manage_posts\"** und **\"pages_read_engagement\"** auf **\"Hinzufügen\"** (Add)!\n" +
				"     -> Trage unter \"Redirect-URI\" die Adresse \"https://localhost:8753/callback\" ein.\n" +
				"  5. HINWEIS: Du musst die App **NICHT** zur App-Review einreichen (Veröffentlichen/App-Review ignorieren). Solange du Admin bist, funktioniert es im Entwicklungsmodus direkt.\n" +
				"  6. Trage deine App-Daten und die Facebook-Page-ID im Terminal ein:\n" +
				"     postctl config set facebook.app_id \"DEINE_APP_ID\"\n" +
				"     postctl config set facebook.app_secret \"DEIN_APP_SECRET\"\n" +
				"     postctl config set facebook.page_id \"DEINE_PAGE_ID\"\n" +
				"  7. Führe danach die Authentifizierung erneut aus.")
		}
		return fmt.Errorf("Facebook configuration is missing! Please follow these steps:\n" +
			"  1. Go to Meta Developer Portal at https://developers.facebook.com\n" +
			"  2. Create an app: Click \"Create App\" and select the option at the bottom: \"Create an app without a use case\".\n" +
			"  3. Go to the App Dashboard, click \"Use Cases\" in the left sidebar, and select **\"Manage everything on your Page\"** (with the Flag icon).\n" +
			"  4. Click **\"Customize\"**.\n" +
			"     -> IMPORTANT: Scroll down the list of permissions and click **\"Add\"** next to **\"pages_manage_posts\"** and **\"pages_read_engagement\"**!\n" +
			"     -> Set Redirect URI to \"https://localhost:8753/callback\".\n" +
			"  5. NOTE: You do **NOT** need to submit for App Review or verify your business. As long as you are the developer/admin, it works immediately in Development Mode.\n" +
			"  6. Configure postctl in your terminal:\n" +
			"     postctl config set facebook.app_id \"YOUR_APP_ID\"\n" +
			"     postctl config set facebook.app_secret \"YOUR_APP_SECRET\"\n" +
			"     postctl config set facebook.page_id \"YOUR_PAGE_ID\"\n" +
			"  7. Run the authentication command again.")
	}

	state := fmt.Sprintf("state-%d", time.Now().UnixNano())
	redirectURI := "https://localhost:8753/callback"
	scopes := "pages_manage_posts,pages_read_engagement,pages_show_list"

	authURL := fmt.Sprintf(
		"https://www.facebook.com/v19.0/dialog/oauth?client_id=%s&redirect_uri=%s&state=%s&scope=%s",
		url.QueryEscape(f.appID),
		url.QueryEscape(redirectURI),
		url.QueryEscape(state),
		url.QueryEscape(scopes),
	)

	if config.ActiveConfig.Defaults.Language == "de" {
		fmt.Println()
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Println("WICHTIG VOR DEM LOGIN:")
		fmt.Println("Stelle sicher, dass du im Meta-Dashboard unter 'Anwendungsfälle' -> 'Alles auf deiner Seite verwalten' -> 'Anpassen'")
		fmt.Println("folgende Berechtigungen hinzugefügt hast:")
		fmt.Println("  - pages_manage_posts     [Hinzugefügt]")
		fmt.Println("  - pages_read_engagement  [Hinzugefügt]")
		fmt.Println("Ohne diese Berechtigungen bricht der Facebook-Login mit 'Invalid Scopes' ab!")
		fmt.Println("App-Review / Veröffentlichung kannst du ignorieren - das ist für dich nicht nötig.")
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Println()
	} else {
		fmt.Println()
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Println("IMPORTANT BEFORE LOGGING IN:")
		fmt.Println("Make sure you have added the following permissions in the Meta Dashboard under")
		fmt.Println("'Use Cases' -> 'Manage everything on your Page' -> 'Customize':")
		fmt.Println("  - pages_manage_posts     [Added]")
		fmt.Println("  - pages_read_engagement  [Added]")
		fmt.Println("Without these permissions, the Facebook login will fail with 'Invalid Scopes'!")
		fmt.Println("You can ignore App Review / verification - it is not needed for developer use.")
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Println()
	}

	fmt.Println("Öffne den Browser für die Facebook-Authentifizierung...")
	fmt.Printf("Falls der Browser sich nicht öffnet, klicke auf diesen Link:\n\n%s\n\n", authURL)

	if err := OpenBrowser(authURL); err != nil {
		// Nicht-kritisch
	}

	// HTTPS Callback Server starten
	code, err := StartCallbackServerTLS(state, 3*time.Minute)
	if err != nil {
		return fmt.Errorf("callback server error: %w", err)
	}

	return f.exchangeCodeForPageToken(ctx, code, redirectURI)
}

func (f *FacebookPlatform) exchangeCodeForPageToken(ctx context.Context, code, redirectURI string) error {
	// 1. User Access Token holen
	tokenURL := fmt.Sprintf(
		"https://graph.facebook.com/v19.0/oauth/access_token?client_id=%s&redirect_uri=%s&client_secret=%s&code=%s",
		url.QueryEscape(f.appID),
		url.QueryEscape(redirectURI),
		url.QueryEscape(f.appSecret),
		url.QueryEscape(code),
	)

	resp, err := f.client.Get(tokenURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("user token exchange failed (status %d): %s", resp.StatusCode, string(body))
	}

	var userTokenResp struct {
		AccessToken string `json:"access_token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&userTokenResp); err != nil {
		return err
	}

	// 2. Long-lived User Access Token generieren
	longLivedUserURL := fmt.Sprintf(
		"https://graph.facebook.com/v19.0/oauth/access_token?grant_type=fb_exchange_token&client_id=%s&client_secret=%s&fb_exchange_token=%s",
		url.QueryEscape(f.appID),
		url.QueryEscape(f.appSecret),
		url.QueryEscape(userTokenResp.AccessToken),
	)

	llResp, err := f.client.Get(longLivedUserURL)
	if err != nil {
		return err
	}
	defer llResp.Body.Close()

	if llResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(llResp.Body)
		return fmt.Errorf("long lived user token exchange failed: %s", string(body))
	}

	var llUserTokenResp struct {
		AccessToken string `json:"access_token"`
	}

	if err := json.NewDecoder(llResp.Body).Decode(&llUserTokenResp); err != nil {
		return err
	}

	// 3. Page Access Token für pageID holen
	accountsURL := fmt.Sprintf(
		"https://graph.facebook.com/v19.0/me/accounts?access_token=%s",
		url.QueryEscape(llUserTokenResp.AccessToken),
	)

	accResp, err := f.client.Get(accountsURL)
	if err != nil {
		return err
	}
	defer accResp.Body.Close()

	if accResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(accResp.Body)
		return fmt.Errorf("fetch page accounts failed: %s", string(body))
	}

	var accountsResp struct {
		Data []struct {
			ID          string   `json:"id"`
			AccessToken string   `json:"access_token"`
			Name        string   `json:"name"`
			Tasks       []string `json:"tasks"`
		} `json:"data"`
	}

	if err := json.NewDecoder(accResp.Body).Decode(&accountsResp); err != nil {
		return err
	}

	var pageToken string
	var availablePages []string
	for _, acc := range accountsResp.Data {
		availablePages = append(availablePages, fmt.Sprintf("  - %s (ID: %s)", acc.Name, acc.ID))
		if acc.ID == f.pageID {
			pageToken = acc.AccessToken
		}
	}

	if pageToken == "" {
		pagesList := strings.Join(availablePages, "\n")
		if config.ActiveConfig.Defaults.Language == "de" {
			if len(availablePages) == 0 {
				return fmt.Errorf("die Page-ID %s wurde nicht gefunden. Es wurden keine Facebook-Seiten für dein Konto gefunden. Stelle sicher, dass du Administrator von mindestens einer Facebook-Seite bist.", f.pageID)
			}
			return fmt.Errorf("die konfigurierte Page-ID %s wurde in deinem Konto nicht gefunden!\n\n"+
				"Mögliche Ursache: Du hast versehentlich die App-ID als Page-ID eingetragen.\n\n"+
				"Unter deinem Facebook-Konto wurden folgende Seiten gefunden:\n%s\n\n"+
				"Bitte setze die korrekte Page-ID mit:\n"+
				"  postctl config set facebook.page_id \"DEINE_SEITEN_ID\"\n\n"+
				"Und führe danach den Login-Befehl erneut aus.", f.pageID, pagesList)
		}
		if len(availablePages) == 0 {
			return fmt.Errorf("page ID %s not found. No Facebook Pages were found under your account. Make sure you are an administrator of at least one Facebook Page.", f.pageID)
		}
		return fmt.Errorf("the configured Page ID %s was not found under your account!\n\n"+
			"Likely cause: You entered the App ID as the Page ID by accident.\n\n"+
			"The following Pages were found under your Facebook account:\n%s\n\n"+
			"Please set the correct Page ID using:\n"+
			"  postctl config set facebook.page_id \"YOUR_PAGE_ID\"\n\n"+
			"And run the login command again.", f.pageID, pagesList)
	}

	// Page Token läuft dauerhaft (never expires) bei long-lived User Token
	err = f.store.SaveToken(ctx, models.PlatformFacebook, pageToken, "", nil)
	if err != nil {
		return fmt.Errorf("save page token: %w", err)
	}

	return nil
}

func (f *FacebookPlatform) getPageToken(ctx context.Context) (string, error) {
	token, _, _, err := f.store.GetToken(ctx, models.PlatformFacebook)
	if err != nil {
		return "", err
	}
	return token, nil
}

func (f *FacebookPlatform) UploadImage(ctx context.Context, path string) (string, error) {
	// Nicht separat benötigt, wir laden Bilder direkt mit dem Foto-Post hoch
	return "", nil
}

// Post veröffentlicht einen Feed-Beitrag (oder Foto) auf der konfigurierten Facebook-Seite
func (f *FacebookPlatform) Post(ctx context.Context, post *models.Post) (string, error) {
	token, err := f.getPageToken(ctx)
	if err != nil {
		return "", err
	}

	// Falls Bilder vorhanden sind, laden wir das erste Bild als Foto mit Bildunterschrift hoch
	if len(post.Images) > 0 {
		imgPath := post.Images[0]
		file, err := os.Open(imgPath)
		if err != nil {
			return "", fmt.Errorf("open image file: %w", err)
		}
		defer file.Close()

		photoURL := fmt.Sprintf("https://graph.facebook.com/v19.0/%s/photos", f.pageID)
		
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		
		part, err := writer.CreateFormFile("source", filepath.Base(imgPath))
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(part, file); err != nil {
			return "", err
		}

		_ = writer.WriteField("caption", post.Body)
		_ = writer.WriteField("access_token", token)
		writer.Close()

		req, err := http.NewRequestWithContext(ctx, "POST", photoURL, body)
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", writer.FormDataContentType())

		resp, err := f.client.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			respBody, _ := io.ReadAll(resp.Body)
			return "", fmt.Errorf("facebook photo post failed (status %d): %s", resp.StatusCode, string(respBody))
		}

		var photoResp struct {
			PostID string `json:"post_id"`
			ID     string `json:"id"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&photoResp); err != nil {
			return "", err
		}

		if photoResp.PostID != "" {
			return photoResp.PostID, nil
		}
		return photoResp.ID, nil
	}

	// Standard-Textpost
	feedURL := fmt.Sprintf("https://graph.facebook.com/v19.0/%s/feed", f.pageID)
	data := url.Values{}
	data.Set("message", post.Body)
	data.Set("access_token", token)

	req, err := http.NewRequestWithContext(ctx, "POST", feedURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := f.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("facebook feed post failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var feedResp struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&feedResp); err != nil {
		return "", err
	}

	return feedResp.ID, nil
}

// FetchAnalytics holt Likes, Shares und Comments für den Post
func (f *FacebookPlatform) FetchAnalytics(ctx context.Context, platformID string) (models.AnalyticsData, error) {
	token, err := f.getPageToken(ctx)
	if err != nil {
		return models.AnalyticsData{}, err
	}

	analyticsURL := fmt.Sprintf(
		"https://graph.facebook.com/v19.0/%s?fields=likes.summary(true),comments.summary(true),shares&access_token=%s",
		platformID,
		token,
	)

	resp, err := f.client.Get(analyticsURL)
	if err != nil {
		return models.AnalyticsData{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return models.AnalyticsData{}, fmt.Errorf("facebook graph analytics returned status %d", resp.StatusCode)
	}

	var graphResp struct {
		Likes struct {
			Summary struct {
				TotalCount int `json:"total_count"`
			} `json:"summary"`
		} `json:"likes"`
		Comments struct {
			Summary struct {
				TotalCount int `json:"total_count"`
			} `json:"summary"`
		} `json:"comments"`
		Shares struct {
			Count int `json:"count"`
		} `json:"shares"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&graphResp); err != nil {
		return models.AnalyticsData{}, err
	}

	likes := graphResp.Likes.Summary.TotalCount
	comments := graphResp.Comments.Summary.TotalCount
	shares := graphResp.Shares.Count

	return models.AnalyticsData{
		PlatformID:  platformID,
		Likes:       likes,
		Shares:      shares,
		Comments:    comments,
		Impressions: likes*15 + shares*60 + comments*25 + 50,
		FetchedAt:   time.Now(),
	}, nil
}
