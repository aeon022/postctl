package platforms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/models"
	"github.com/aeon022/postctl/internal/store"
)

type DiscordPlatform struct {
	store      *store.SQLiteStore
	webhookURL string
	client     *http.Client
}

func NewDiscordPlatform(s *store.SQLiteStore, webhookURL string) *DiscordPlatform {
	return &DiscordPlatform{
		store:      s,
		webhookURL: webhookURL,
		client:     &http.Client{Timeout: 30 * time.Second},
	}
}

func (d *DiscordPlatform) Name() string {
	return models.PlatformDiscord
}

func (d *DiscordPlatform) IsAuthenticated(ctx context.Context) bool {
	return d.webhookURL != ""
}

func (d *DiscordPlatform) Auth(ctx context.Context) error {
	if d.webhookURL == "" {
		if config.ActiveConfig.Defaults.Language == "de" {
			return fmt.Errorf("Discord-Konfiguration fehlt! Bitte folge diesen Schritten:\n" +
				"  1. Gehe in deine Discord Server-Einstellungen ➔ Integrationen ➔ Webhooks.\n" +
				"  2. Erstelle einen neuen Webhook und kopiere die Webhook-URL.\n" +
				"  3. Trage die Webhook-URL im Terminal ein:\n" +
				"     postctl config set discord.webhook_url \"DEINE_WEBHOOK_URL\"")
		}
		return fmt.Errorf("Discord configuration is missing! Please follow these steps:\n" +
			"  1. Go to your Discord Server Settings ➔ Integrations ➔ Webhooks.\n" +
			"  2. Create a new webhook and copy the Webhook URL.\n" +
			"  3. Configure postctl in your terminal:\n" +
			"     postctl config set discord.webhook_url \"YOUR_WEBHOOK_URL\"")
	}

	// Webhook testen
	req, err := http.NewRequestWithContext(ctx, "GET", d.webhookURL, nil)
	if err != nil {
		return err
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to reach Discord API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("invalid Discord Webhook URL (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

func (d *DiscordPlatform) UploadImage(ctx context.Context, path string) (string, error) {
	// Discord lädt Bilder direkt beim Posten hoch, daher geben wir einfach den Pfad zurück
	return path, nil
}

func (d *DiscordPlatform) Post(ctx context.Context, post *models.Post) (string, error) {
	if len(post.Images) == 0 {
		return d.sendTextMessage(ctx, post.Body)
	}
	return d.sendMultipartMessage(ctx, post.Images, post.Body)
}

func (d *DiscordPlatform) FetchAnalytics(ctx context.Context, platformID string) (models.AnalyticsData, error) {
	// Discord Webhooks unterstützen standardmäßig keine Statistiken/Analytics
	return models.AnalyticsData{}, nil
}

func (d *DiscordPlatform) sendTextMessage(ctx context.Context, text string) (string, error) {
	payload := map[string]interface{}{
		"content": text,
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", d.webhookURL, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	return parseDiscordResponse(resp)
}

func (d *DiscordPlatform) sendMultipartMessage(ctx context.Context, imgPaths []string, text string) (string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Payload JSON-Teil anhängen
	payload := map[string]interface{}{
		"content": text,
	}
	payloadJson, _ := json.Marshal(payload)
	_ = writer.WriteField("payload_json", string(payloadJson))

	// Dateien anhängen
	for i, path := range imgPaths {
		file, err := os.Open(path)
		if err != nil {
			return "", fmt.Errorf("open image file %s: %w", path, err)
		}
		
		part, err := writer.CreateFormFile(fmt.Sprintf("files[%d]", i), filepath.Base(path))
		if err != nil {
			file.Close()
			return "", err
		}
		_, _ = io.Copy(part, file)
		file.Close()
	}
	writer.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", d.webhookURL, body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := d.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	return parseDiscordResponse(resp)
}

func parseDiscordResponse(resp *http.Response) (string, error) {
	body, _ := io.ReadAll(resp.Body)
	// Discord Webhook gibt Status 200 (OK) oder 204 (No Content) bei Erfolg zurück
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return "", fmt.Errorf("discord API error (status %d): %s", resp.StatusCode, string(body))
	}

	if resp.StatusCode == http.StatusNoContent {
		return "webhook-posted", nil
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &result); err == nil && result.ID != "" {
		return result.ID, nil
	}

	return "webhook-posted", nil
}
