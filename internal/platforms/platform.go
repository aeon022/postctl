package platforms

import (
	"context"
	"fmt"

	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/models"
	"github.com/aeon022/postctl/internal/store"
)

// Platform definiert die Schnittstelle für alle Social Media Kanäle
type Platform interface {
	Name() string
	Auth(ctx context.Context) error
	Post(ctx context.Context, post *models.Post) (string, error)  // Gibt die Plattform-Post-ID zurück
	UploadImage(ctx context.Context, path string) (string, error) // Gibt die URN / Media-ID zurück
	IsAuthenticated(ctx context.Context) bool
	FetchAnalytics(ctx context.Context, platformID string) (models.AnalyticsData, error)
	Delete(ctx context.Context, platformID string) error
}

// GetPlatform liefert die passende Platform-Instanz. Im Simulationsmodus (dryRun) wird ein Mock zurückgegeben.
func GetPlatform(name string, s *store.SQLiteStore, dryRun bool) (Platform, error) {
	if dryRun {
		return NewDryRunPlatform(name), nil
	}

	switch name {
	case models.PlatformTwitter:
		return NewTwitterPlatform(s, config.ActiveConfig.Twitter.ClientID, config.ActiveConfig.Twitter.ClientSecret), nil
	case models.PlatformLinkedIn:
		return NewLinkedInPlatform(s, config.ActiveConfig.LinkedIn.ClientID, config.ActiveConfig.LinkedIn.ClientSecret), nil
	case models.PlatformThreads:
		return NewThreadsPlatform(s, config.ActiveConfig.Threads.AppID, config.ActiveConfig.Threads.AppSecret), nil
	case models.PlatformMastodon:
		return NewMastodonPlatform(s, config.ActiveConfig.Mastodon.InstanceURL, config.ActiveConfig.Mastodon.ClientID, config.ActiveConfig.Mastodon.ClientSecret), nil
	case models.PlatformBluesky:
		return NewBlueskyPlatform(s, config.ActiveConfig.Bluesky.Handle, config.ActiveConfig.Bluesky.AppPassword), nil
	case models.PlatformFacebook:
		return NewFacebookPlatform(s, config.ActiveConfig.Facebook.AppID, config.ActiveConfig.Facebook.AppSecret, config.ActiveConfig.Facebook.PageID), nil
	case models.PlatformTelegram:
		return NewTelegramPlatform(s, config.ActiveConfig.Telegram.BotToken, config.ActiveConfig.Telegram.ChatID), nil
	case models.PlatformDiscord:
		return NewDiscordPlatform(s, config.ActiveConfig.Discord.WebhookURL), nil
	case models.PlatformDevTo:
		return NewDevToPlatform(s, config.ActiveConfig.DevTo.APIToken), nil
	case models.PlatformReddit:
		return NewRedditPlatform(s, config.ActiveConfig.Reddit.ClientID, config.ActiveConfig.Reddit.ClientSecret, config.ActiveConfig.Reddit.Username, config.ActiveConfig.Reddit.Password), nil
	case models.PlatformHashnode:
		return NewHashnodePlatform(s, config.ActiveConfig.Hashnode.APIToken, config.ActiveConfig.Hashnode.PublicationID), nil
	case models.PlatformMedium:
		return NewMediumPlatform(s, config.ActiveConfig.Medium.IntegrationToken), nil
	default:
		return nil, fmt.Errorf("unknown platform %q", name)
	}
}
