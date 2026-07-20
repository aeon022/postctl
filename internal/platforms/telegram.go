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

type TelegramPlatform struct {
	store    *store.SQLiteStore
	botToken string
	chatID   string
	client   *http.Client
	apiURL   string
}

func NewTelegramPlatform(s *store.SQLiteStore, botToken, chatID string) *TelegramPlatform {
	return &TelegramPlatform{
		store:    s,
		botToken: botToken,
		chatID:   chatID,
		client:   &http.Client{Timeout: 30 * time.Second},
		apiURL:   "https://api.telegram.org",
	}
}

func (t *TelegramPlatform) Name() string {
	return models.PlatformTelegram
}

func (t *TelegramPlatform) IsAuthenticated(ctx context.Context) bool {
	return t.botToken != "" && t.chatID != ""
}

func (t *TelegramPlatform) Auth(ctx context.Context) error {
	if t.botToken == "" || t.chatID == "" {
		if config.ActiveConfig.Defaults.Language == "de" {
			return fmt.Errorf("Telegram-Konfiguration fehlt! Bitte folge diesen Schritten:\n" +
				"  1. Erstelle einen Telegram Bot via @BotFather und kopiere den API Token.\n" +
				"  2. Füge den Bot zu deinem Kanal/Gruppe hinzu und mache ihn zum Admin.\n" +
				"  3. Ermittle die Chat-ID (z.B. @deinkanal oder eine ID wie -100123456789).\n" +
				"  4. Trage die Zugangsdaten im Terminal ein:\n" +
				"     postctl config set telegram.bot_token \"DEIN_BOT_TOKEN\"\n" +
				"     postctl config set telegram.chat_id \"DEINE_CHAT_ID\"")
		}
		return fmt.Errorf("Telegram configuration is missing! Please follow these steps:\n" +
			"  1. Create a Telegram Bot via @BotFather and copy the API Token.\n" +
			"  2. Add the bot to your channel/group and make it an admin.\n" +
			"  3. Get the Chat ID (e.g. @yourchannel or an ID like -100123456789).\n" +
			"  4. Configure postctl in your terminal:\n" +
			"     postctl config set telegram.bot_token \"YOUR_BOT_TOKEN\"\n" +
			"     postctl config set telegram.chat_id \"YOUR_CHAT_ID\"")
	}

	// Token testen
	url := fmt.Sprintf("%s/bot%s/getMe", t.apiURL, t.botToken)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to reach Telegram API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("invalid bot token (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

func (t *TelegramPlatform) UploadImage(ctx context.Context, path string) (string, error) {
	// Telegram lädt Bilder direkt beim Posten hoch, daher geben wir einfach den Pfad zurück
	return path, nil
}

func (t *TelegramPlatform) Post(ctx context.Context, post *models.Post) (string, error) {
	if len(post.Images) == 0 {
		return t.sendTextMessage(ctx, post.Body)
	} else if len(post.Images) == 1 {
		return t.sendPhotoMessage(ctx, post.Images[0], post.Body)
	} else {
		return t.sendMediaGroupMessage(ctx, post.Images, post.Body)
	}
}

func (t *TelegramPlatform) FetchAnalytics(ctx context.Context, platformID string) (models.AnalyticsData, error) {
	// Telegram API bietet standardmäßig keine programmgesteuerten Interaktionsdaten für Bots
	return models.AnalyticsData{}, nil
}

func (t *TelegramPlatform) sendTextMessage(ctx context.Context, text string) (string, error) {
	url := fmt.Sprintf("%s/bot%s/sendMessage", t.apiURL, t.botToken)
	
	payload := map[string]interface{}{
		"chat_id":    t.chatID,
		"text":       text,
		"parse_mode": "Markdown",
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	return parseTelegramResponse(resp)
}

func (t *TelegramPlatform) sendPhotoMessage(ctx context.Context, imgPath string, caption string) (string, error) {
	url := fmt.Sprintf("%s/bot%s/sendPhoto", t.apiURL, t.botToken)

	file, err := os.Open(imgPath)
	if err != nil {
		return "", fmt.Errorf("open image file: %w", err)
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	
	_ = writer.WriteField("chat_id", t.chatID)
	_ = writer.WriteField("caption", caption)
	_ = writer.WriteField("parse_mode", "Markdown")

	part, err := writer.CreateFormFile("photo", filepath.Base(imgPath))
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(part, file); err != nil {
		return "", err
	}
	writer.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := t.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	return parseTelegramResponse(resp)
}

func (t *TelegramPlatform) sendMediaGroupMessage(ctx context.Context, imgPaths []string, caption string) (string, error) {
	url := fmt.Sprintf("%s/bot%s/sendMediaGroup", t.apiURL, t.botToken)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("chat_id", t.chatID)

	type mediaInput struct {
		Type      string `json:"type"`
		Media     string `json:"media"`
		Caption   string `json:"caption,omitempty"`
		ParseMode string `json:"parse_mode,omitempty"`
	}

	var mediaList []mediaInput
	for i, path := range imgPaths {
		fieldName := fmt.Sprintf("photo_%d", i)
		part, err := writer.CreateFormFile(fieldName, filepath.Base(path))
		if err != nil {
			return "", err
		}
		
		file, err := os.Open(path)
		if err != nil {
			return "", err
		}
		_, _ = io.Copy(part, file)
		file.Close()

		item := mediaInput{
			Type:  "photo",
			Media: "attach://" + fieldName,
		}
		if i == 0 {
			item.Caption = caption
			item.ParseMode = "Markdown"
		}
		mediaList = append(mediaList, item)
	}

	mediaJson, _ := json.Marshal(mediaList)
	_ = writer.WriteField("media", string(mediaJson))
	writer.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := t.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	return parseTelegramResponse(resp)
}

func parseTelegramResponse(resp *http.Response) (string, error) {
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("telegram API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Ok     bool `json:"ok"`
		Result struct {
			MessageID int `json:"message_id"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	return fmt.Sprintf("%d", result.Result.MessageID), nil
}

// Delete löscht eine gesendete Nachricht aus dem Telegram Chat
func (t *TelegramPlatform) Delete(ctx context.Context, platformID string) error {
	url := fmt.Sprintf("%s/bot%s/deleteMessage", t.apiURL, t.botToken)
	
	var msgID int
	_, err := fmt.Sscanf(platformID, "%d", &msgID)
	if err != nil {
		return fmt.Errorf("invalid telegram message id: %w", err)
	}

	payload := map[string]interface{}{
		"chat_id":    t.chatID,
		"message_id": msgID,
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API error (status %d): %s", resp.StatusCode, string(body))
	}
	return nil
}
