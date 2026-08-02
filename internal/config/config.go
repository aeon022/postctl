package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	coreconfig "github.com/aeon022/missionctl-core/config"
	"gopkg.in/yaml.v3"
)

// Config hält alle Konfigurationseinstellungen für postctl
type Config struct {
	Twitter struct {
		ClientID     string `mapstructure:"client_id" yaml:"client_id"`
		ClientSecret string `mapstructure:"client_secret" yaml:"client_secret"`
		AuthMode     string `mapstructure:"auth_mode" yaml:"auth_mode"` // "api" oder "cookie"
	} `mapstructure:"twitter" yaml:"twitter"`
	LinkedIn struct {
		ClientID     string `mapstructure:"client_id" yaml:"client_id"`
		ClientSecret string `mapstructure:"client_secret" yaml:"client_secret"`
	} `mapstructure:"linkedin" yaml:"linkedin"`
	Threads struct {
		AppID     string `mapstructure:"app_id" yaml:"app_id"`
		AppSecret string `mapstructure:"app_secret" yaml:"app_secret"`
	} `mapstructure:"threads" yaml:"threads"`
	Mastodon struct {
		InstanceURL  string `mapstructure:"instance_url" yaml:"instance_url"`
		ClientID     string `mapstructure:"client_id" yaml:"client_id"`
		ClientSecret string `mapstructure:"client_secret" yaml:"client_secret"`
	} `mapstructure:"mastodon" yaml:"mastodon"`
	Bluesky struct {
		Handle      string `mapstructure:"handle" yaml:"handle"`
		AppPassword string `mapstructure:"app_password" yaml:"app_password"`
	} `mapstructure:"bluesky" yaml:"bluesky"`
	Facebook struct {
		AppID     string `mapstructure:"app_id" yaml:"app_id"`
		AppSecret string `mapstructure:"app_secret" yaml:"app_secret"`
		PageID    string `mapstructure:"page_id" yaml:"page_id"`
	} `mapstructure:"facebook" yaml:"facebook"`
	Telegram struct {
		BotToken string `mapstructure:"bot_token" yaml:"bot_token"`
		ChatID   string `mapstructure:"chat_id" yaml:"chat_id"`
	} `mapstructure:"telegram" yaml:"telegram"`
	Discord struct {
		WebhookURL string `mapstructure:"webhook_url" yaml:"webhook_url"`
	} `mapstructure:"discord" yaml:"discord"`
	Reddit struct {
		ClientID     string `mapstructure:"client_id" yaml:"client_id"`
		ClientSecret string `mapstructure:"client_secret" yaml:"client_secret"`
		Username     string `mapstructure:"username" yaml:"username"`
		Password     string `mapstructure:"password" yaml:"password"`
	} `mapstructure:"reddit" yaml:"reddit"`
	DevTo struct {
		APIToken string `mapstructure:"api_token" yaml:"api_token"`
	} `mapstructure:"devto" yaml:"devto"`
	Hashnode struct {
		APIToken      string `mapstructure:"api_token" yaml:"api_token"`
		PublicationID string `mapstructure:"publication_id" yaml:"publication_id"`
	} `mapstructure:"hashnode" yaml:"hashnode"`
	Medium struct {
		IntegrationToken string `mapstructure:"integration_token" yaml:"integration_token"`
	} `mapstructure:"medium" yaml:"medium"`
	Instagram struct {
		AccessToken string `mapstructure:"access_token" yaml:"access_token"`
		AccountID   string `mapstructure:"account_id" yaml:"account_id"`
	} `mapstructure:"instagram" yaml:"instagram"`
	Pinterest struct {
		AccessToken string `mapstructure:"access_token" yaml:"access_token"`
		BoardID     string `mapstructure:"board_id" yaml:"board_id"`
	} `mapstructure:"pinterest" yaml:"pinterest"`
	YouTube struct {
		ClientID     string `mapstructure:"client_id" yaml:"client_id"`
		ClientSecret string `mapstructure:"client_secret" yaml:"client_secret"`
	} `mapstructure:"youtube" yaml:"youtube"`
	Defaults struct {
		Timezone string `mapstructure:"timezone" yaml:"timezone"`
		DryRun   bool   `mapstructure:"dry_run" yaml:"dry_run"`
		ImageDir string `mapstructure:"image_dir" yaml:"image_dir"`
		Language string `mapstructure:"language" yaml:"language"`
	} `mapstructure:"defaults" yaml:"defaults"`
	AI struct {
		Provider string `mapstructure:"provider" yaml:"provider"`
		APIKey   string `mapstructure:"api_key" yaml:"api_key"`
		Model    string `mapstructure:"model" yaml:"model"`
		BaseURL  string `mapstructure:"base_url" yaml:"base_url"`
	} `mapstructure:"ai" yaml:"ai"`
	Scheduler struct {
		Slots []string `mapstructure:"slots" yaml:"slots"`
	} `mapstructure:"scheduler" yaml:"scheduler"`
	DBPath        string   `mapstructure:"db_path" yaml:"db_path"`
	DataDir       string   `mapstructure:"data_dir" yaml:"data_dir"`
	LicenseKey    string   `mapstructure:"license_key" yaml:"license_key"`
	LicenseStatus string   `mapstructure:"license_status" yaml:"license_status"`
	PolarOrgID    string   `mapstructure:"polar_org_id" yaml:"polar_org_id"`
	RSSFeeds      []string `mapstructure:"rss_feeds" yaml:"rss_feeds"`
}

// ActiveConfig stellt die geladene Konfiguration global zur Verfügung
var ActiveConfig Config

// IsPro prüft, ob eine gültige Pro-Lizenz aktiv ist
func IsPro() bool {
	key := strings.TrimSpace(ActiveConfig.LicenseKey)
	if key == "postctl-pro-dev" || key == "postctl-pro-family" {
		return true
	}
	if (strings.HasPrefix(key, "PCTL-PRO-") && len(key) >= 16) || (strings.HasPrefix(key, "PCTL-DEV-") && len(key) >= 12) || (strings.HasPrefix(key, "a83-postctl") && len(key) >= 15) {
		return true
	}
	return ActiveConfig.LicenseStatus == "active"
}

// ValidateLicenseKey prüft das Format und die Gültigkeit des Lizenzschlüssels.
func ValidateLicenseKey(key string) bool {
	key = strings.TrimSpace(key)
	if key == "" {
		return false
	}
	if key == "postctl-pro-dev" || key == "postctl-pro-family" {
		return true
	}
	if (strings.HasPrefix(key, "PCTL-PRO-") && len(key) >= 16) || (strings.HasPrefix(key, "PCTL-DEV-") && len(key) >= 12) || (strings.HasPrefix(key, "a83-postctl") && len(key) >= 15) {
		return true
	}
	return false
}

// LoadConfig lädt die Konfiguration aus ~/.config/postctl/config.yaml oder setzt Defaultwerte
func LoadConfig() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get user home dir: %w", err)
	}

	configDir := filepath.Join(home, ".config", "postctl")
	
	// Default-Werte setzen
	ActiveConfig.DBPath = "~/.config/postctl/postctl.db"
	ActiveConfig.Defaults.Timezone = "Europe/Vienna"
	ActiveConfig.Defaults.DryRun = false
	ActiveConfig.Defaults.ImageDir = "./screenshots"
	ActiveConfig.Defaults.Language = "en"
	ActiveConfig.AI.Provider = "openai"
	ActiveConfig.AI.Model = "gpt-4o-mini"
	ActiveConfig.LicenseKey = ""
	ActiveConfig.LicenseStatus = ""
	ActiveConfig.PolarOrgID = "aa792ea4-650e-492e-a955-9b3d564e943e"
	ActiveConfig.Mastodon.InstanceURL = "https://mastodon.social"
	ActiveConfig.Scheduler.Slots = []string{"Mon 09:00", "Wed 14:00", "Fri 17:30"}

	// Falls die Konfigurationsdatei nicht existiert, erstellen wir sie mit Standardwerten
	configPath := filepath.Join(configDir, "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("create config dir: %w", err)
		}
		
		dummyContent := `# postctl configuration file
db_path: "~/.config/postctl/postctl.db"

# Point postctl's database at a directory you sync yourself (iCloud
# Drive, Dropbox, ...) to share data across devices. Takes precedence
# over db_path above. See doctor's "Data directory" check.
# data_dir: "~/Library/Mobile Documents/com~apple~CloudDocs/postctl"

# License Key for Pro Features
license_key: ""

defaults:
  timezone: "Europe/Vienna"
  dry_run: false
  image_dir: "./screenshots"

# AI Generator settings (openai | claude | ollama)
ai:
  provider: "openai"
  model: "gpt-4o-mini"
  api_key: ""
  base_url: ""

# API Keys (Trage hier deine OAuth-Keys ein)
twitter:
  client_id: ""
  client_secret: ""

linkedin:
  client_id: ""
  client_secret: ""

threads:
  app_id: ""
  app_secret: ""

mastodon:
  instance_url: "https://mastodon.social"
  client_id: ""
  client_secret: ""

bluesky:
  handle: ""
  app_password: ""

facebook:
  app_id: ""
  app_secret: ""
  page_id: ""

telegram:
  bot_token: ""
  chat_id: ""

discord:
  webhook_url: ""

reddit:
  client_id: ""
  client_secret: ""
  username: ""
  password: ""

devto:
  api_token: ""

hashnode:
  api_token: ""
  publication_id: ""

medium:
  integration_token: ""

instagram:
  access_token: ""
  account_id: ""

pinterest:
  access_token: ""
  board_id: ""

youtube:
  client_id: ""
  client_secret: ""
`
		if err := os.WriteFile(configPath, []byte(dummyContent), 0644); err != nil {
			return fmt.Errorf("create default config file: %w", err)
		}
	}

	yamlBytes, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}

	if err := yaml.Unmarshal(yamlBytes, &ActiveConfig); err != nil {
		return fmt.Errorf("unmarshal config: %w", err)
	}

	return nil
}

// legacyDefaultDBPath is the hardcoded default this config used before
// data_dir/coreconfig.DataDir existed. Any db_path still equal to this
// (i.e. never customized by the user) is treated as unset, so those
// installs migrate to the new default below rather than being pinned to
// this path forever.
const legacyDefaultDBPath = "~/.config/postctl/postctl.db"

// GetDBPath returns the expanded path to the SQLite database, resolved
// with this precedence: data_dir (new, directory-shaped — the supported
// way to point postctl at a folder you sync yourself, e.g. iCloud Drive
// or Dropbox) > a customized legacy db_path (a full file path) > the
// migrated default (~/.local/share/postctl, matching the rest of the
// suite — see migratedDefaultDir for the one-time move of an existing
// database at the old ~/.config/postctl location).
func GetDBPath() string {
	if dir := strings.TrimSpace(ActiveConfig.DataDir); dir != "" {
		resolved, _ := coreconfig.ResolveDir("postctl", dir)
		return filepath.Join(resolved, "postctl.db")
	}

	path := ActiveConfig.DBPath
	if path != "" && path != legacyDefaultDBPath {
		if strings.HasPrefix(path, "~") {
			home, _ := os.UserHomeDir()
			path = filepath.Join(home, path[1:])
		}
		return filepath.Clean(path)
	}

	return filepath.Join(migratedDefaultDir(), "postctl.db")
}

// Shared reports whether GetDBPath currently resolves to a user-configured
// directory (data_dir) rather than postctl's own default/legacy path.
func Shared() bool {
	return strings.TrimSpace(ActiveConfig.DataDir) != ""
}

// migratedDefaultDir returns postctl's default data directory
// (~/.local/share/postctl, matching the rest of the suite — postctl
// previously used ~/.config/postctl for its database, an inconsistency
// with the rest of the suite that predates this package), migrating a
// pre-existing database from that old location the first time it's
// needed, so existing users don't lose data.
func migratedDefaultDir() string {
	dir := coreconfig.DataDir("postctl")
	newPath := filepath.Join(dir, "postctl.db")
	if _, err := os.Stat(newPath); err == nil {
		return dir // already migrated
	}
	home, _ := os.UserHomeDir()
	oldPath := filepath.Join(home, ".config", "postctl", "postctl.db")
	if _, err := os.Stat(oldPath); err == nil {
		if err := os.Rename(oldPath, newPath); err == nil {
			fmt.Printf("postctl: moved database from %s to %s (new default location, matching the rest of the suite)\n", oldPath, newPath)
		}
	}
	return dir
}

// SaveConfig schreibt die aktuelle ActiveConfig zurück in die ~/.config/postctl/config.yaml Datei
func SaveConfig() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get user home dir: %w", err)
	}

	configDir := filepath.Join(home, ".config", "postctl")
	configPath := filepath.Join(configDir, "config.yaml")

	yamlBytes, err := yaml.Marshal(ActiveConfig)
	if err != nil {
		return fmt.Errorf("marshal config to yaml: %w", err)
	}

	if err := os.WriteFile(configPath, yamlBytes, 0644); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}

