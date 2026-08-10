package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

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
		// AutoPublish gates the TUI's background scheduler (the 10s tick
		// that checks for and publishes due posts). Defaults to false —
		// with data_dir pointed at a Dropbox-synced folder, two machines
		// each running their own local SQLite copy can both see the same
		// post as still "scheduled" before Dropbox has synced the other
		// machine's "posted" update, and both publish it. Enable this on
		// exactly one machine (the one that should actually publish);
		// leave it off everywhere else so opening the TUI there is always
		// safe to just browse/edit.
		AutoPublish bool `mapstructure:"auto_publish" yaml:"auto_publish"`
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

// ActiveProfile is the currently loaded profile name ("" = the original,
// unprofiled default). Set once at startup by LoadConfigProfile (from the
// --profile flag / POSTCTL_PROFILE env var) and left untouched by
// LoadConfig()'s no-arg reload calls, so config changes written mid-session
// (config set, config import) get picked back up for the same profile.
var ActiveProfile string

// IsPro prüft, ob eine gültige Pro-Lizenz aktiv ist. Polars echte
// License-Key-Ressource nutzt "granted" als Status (gegen die Live-API
// bestätigt) — "active" wird zusätzlich akzeptiert für den lokalen
// Dev/Family-Bypass, der diesen Wert direkt in die Config schreibt.
func IsPro() bool {
	return ActiveConfig.LicenseStatus == "granted" || ActiveConfig.LicenseStatus == "active"
}

// AutoPublishEnabled reports whether this machine is allowed to
// automatically publish due posts in the background (see Scheduler.AutoPublish).
func AutoPublishEnabled() bool {
	return ActiveConfig.Scheduler.AutoPublish
}

// DBSettled reports whether the SQLite database at GetDBPath hasn't been
// written to in at least quiet — i.e. it looks like a Dropbox/iCloud sync
// in progress has finished, not still mid-flight. Used to hold off
// auto-publishing until the local copy is likely caught up with any other
// machine's more recent writes, rather than trusting a possibly-stale
// "scheduled" status. Best-effort: a sync that's paused, offline, or just
// slower than quiet will still look settled — this narrows the race, it
// doesn't close it.
func DBSettled(quiet time.Duration) bool {
	info, err := os.Stat(GetDBPath())
	if err != nil {
		return false
	}
	return time.Since(info.ModTime()) >= quiet
}

// configDir returns the config directory for the given profile: the
// original ~/.config/postctl for "" (the default/unprofiled case, for
// backward compatibility with existing installs), or
// ~/.config/postctl/profiles/<name> for a named one.
func configDir(profile string) string {
	home, _ := os.UserHomeDir()
	if profile == "" {
		return filepath.Join(home, ".config", "postctl")
	}
	return filepath.Join(home, ".config", "postctl", "profiles", profile)
}

// ProfilesDir is where named profiles live — ~/.config/postctl/profiles.
func ProfilesDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "postctl", "profiles")
}

// ListProfiles returns the names of every profile that has been used at
// least once (i.e. has a config directory under ProfilesDir), sorted
// alphabetically. The unprofiled default is not included — callers that
// want to mention it do so explicitly.
func ListProfiles() ([]string, error) {
	entries, err := os.ReadDir(ProfilesDir())
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

// LoadConfig (re)loads the config for the currently active profile — use
// this to pick up changes written mid-session (config set, config
// import). To switch profiles, use LoadConfigProfile instead.
func LoadConfig() error {
	return loadConfig(ActiveProfile)
}

// LoadConfigProfile switches to and loads the named profile ("" = the
// original default). Called once at startup from the --profile flag /
// POSTCTL_PROFILE env var.
func LoadConfigProfile(profile string) error {
	return loadConfig(strings.TrimSpace(profile))
}

func loadConfig(profile string) error {
	ActiveProfile = profile
	dir := configDir(profile)

	// Default-Werte setzen (reset first: LoadConfig can be called more than
	// once per process — config import reloads after restoring a backup —
	// and a stale value from a previous load must not survive that).
	ActiveConfig = Config{}
	if profile == "" {
		// Only the original default profile ever had data at this legacy
		// path — see GetDBPath's legacyDefaultDBPath comment. A new named
		// profile has no such legacy to carry.
		ActiveConfig.DBPath = legacyDefaultDBPath
	}
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
	configPath := filepath.Join(dir, "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create config dir: %w", err)
		}
		if err := os.WriteFile(configPath, []byte(defaultConfigYAML(profile)), 0644); err != nil {
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

// defaultConfigYAML is the dummy config content written the first time a
// profile ("" = default) is loaded. The credential/defaults body is the
// same for every profile; only the header differs, since db_path's legacy
// hint only ever applied to the original default.
func defaultConfigYAML(profile string) string {
	header := `# postctl configuration file
db_path: "~/.config/postctl/postctl.db"

# Point postctl's database at a directory you sync yourself (iCloud
# Drive, Dropbox, ...) to share data across devices. Takes precedence
# over db_path above. See doctor's "Data directory" check.
# data_dir: "~/Library/Mobile Documents/com~apple~CloudDocs/postctl"

# License Key for Pro Features
license_key: ""
`
	if profile != "" {
		header = fmt.Sprintf(`# postctl configuration file — profile %q
# This profile has its own database, independent of the default profile
# and any other profile (see "postctl profile list"). Point it at a
# directory you sync yourself (iCloud Drive, Dropbox, ...) to share just
# this profile's data across devices — see doctor's "Data directory" check.
# data_dir: "~/Library/Mobile Documents/com~apple~CloudDocs/postctl-%s"

# License Key for Pro Features
license_key: ""
`, profile, profile)
	}
	return header + defaultConfigBody
}

const defaultConfigBody = `
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

// legacyDefaultDBPath is the hardcoded default this config used before
// data_dir/coreconfig.DataDir existed. Any db_path still equal to this
// (i.e. never customized by the user) is treated as unset, so those
// installs migrate to the new default below rather than being pinned to
// this path forever.
const legacyDefaultDBPath = "~/.config/postctl/postctl.db"

// GetDBPath returns the expanded path to the SQLite database, resolved
// with this precedence: data_dir (new, directory-shaped — the supported
// way to point this profile at a folder you sync yourself, e.g. iCloud
// Drive or Dropbox) > a customized legacy db_path (a full file path,
// meaningful only for the original default profile) > this profile's own
// default data directory (see defaultDataDir).
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

	return filepath.Join(defaultDataDir(), "postctl.db")
}

// Shared reports whether GetDBPath currently resolves to a user-configured
// directory (data_dir) rather than this profile's own default/legacy path.
func Shared() bool {
	return strings.TrimSpace(ActiveConfig.DataDir) != ""
}

// defaultDataDir returns the active profile's own private data directory.
// For the original unprofiled default it's ~/.local/share/postctl, with a
// one-time migration of a pre-existing database from the old
// ~/.config/postctl location (postctl previously stored its DB there, an
// inconsistency with the rest of the suite that predates this package).
// Named profiles are new and never had data at that old location, so no
// migration applies — they get their own subdirectory directly.
func defaultDataDir() string {
	base := coreconfig.DataDir("postctl")
	if ActiveProfile != "" {
		dir := filepath.Join(base, "profiles", ActiveProfile)
		_ = os.MkdirAll(dir, 0o755)
		return dir
	}

	newPath := filepath.Join(base, "postctl.db")
	if _, err := os.Stat(newPath); err == nil {
		return base // already migrated
	}
	home, _ := os.UserHomeDir()
	oldPath := filepath.Join(home, ".config", "postctl", "postctl.db")
	if _, err := os.Stat(oldPath); err == nil {
		if err := os.Rename(oldPath, newPath); err == nil {
			fmt.Printf("postctl: moved database from %s to %s (new default location, matching the rest of the suite)\n", oldPath, newPath)
		}
	}
	return base
}

// SaveConfig schreibt die aktuelle ActiveConfig zurück in config.yaml des aktiven Profils
func SaveConfig() error {
	dir := configDir(ActiveProfile)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	configPath := filepath.Join(dir, "config.yaml")

	yamlBytes, err := yaml.Marshal(ActiveConfig)
	if err != nil {
		return fmt.Errorf("marshal config to yaml: %w", err)
	}

	if err := os.WriteFile(configPath, yamlBytes, 0644); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}

