package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Config hält alle Konfigurationseinstellungen für postctl
type Config struct {
	Twitter struct {
		ClientID     string `mapstructure:"client_id" yaml:"client_id"`
		ClientSecret string `mapstructure:"client_secret" yaml:"client_secret"`
	} `mapstructure:"twitter" yaml:"twitter"`
	LinkedIn struct {
		ClientID     string `mapstructure:"client_id" yaml:"client_id"`
		ClientSecret string `mapstructure:"client_secret" yaml:"client_secret"`
	} `mapstructure:"linkedin" yaml:"linkedin"`
	Threads struct {
		AppID     string `mapstructure:"app_id" yaml:"app_id"`
		AppSecret string `mapstructure:"app_secret" yaml:"app_secret"`
	} `mapstructure:"threads" yaml:"threads"`
	Defaults struct {
		Timezone string `mapstructure:"timezone" yaml:"timezone"`
		DryRun   bool   `mapstructure:"dry_run" yaml:"dry_run"`
		ImageDir string `mapstructure:"image_dir" yaml:"image_dir"`
	} `mapstructure:"defaults" yaml:"defaults"`
	AI struct {
		Provider string `mapstructure:"provider" yaml:"provider"`
		APIKey   string `mapstructure:"api_key" yaml:"api_key"`
		Model    string `mapstructure:"model" yaml:"model"`
		BaseURL  string `mapstructure:"base_url" yaml:"base_url"`
	} `mapstructure:"ai" yaml:"ai"`
	DBPath string `mapstructure:"db_path" yaml:"db_path"`
}

// ActiveConfig stellt die geladene Konfiguration global zur Verfügung
var ActiveConfig Config

// LoadConfig lädt die Konfiguration aus ~/.config/postctl/config.yaml oder setzt Defaultwerte
func LoadConfig() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get user home dir: %w", err)
	}

	configDir := filepath.Join(home, ".config", "postctl")
	
	// Default-Werte setzen
	viper.SetDefault("db_path", "~/.config/postctl/postctl.db")
	viper.SetDefault("defaults.timezone", "Europe/Vienna")
	viper.SetDefault("defaults.dry_run", false)
	viper.SetDefault("defaults.image_dir", "./screenshots")
	viper.SetDefault("ai.provider", "openai")
	viper.SetDefault("ai.model", "gpt-4o-mini")

	viper.AddConfigPath(configDir)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// Falls die Konfigurationsdatei nicht existiert, erstellen wir sie mit Standardwerten
	configPath := filepath.Join(configDir, "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("create config dir: %w", err)
		}
		
		dummyContent := `# postctl configuration file
db_path: "~/.config/postctl/postctl.db"

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
`
		if err := os.WriteFile(configPath, []byte(dummyContent), 0644); err != nil {
			return fmt.Errorf("create default config file: %w", err)
		}
	}

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("read config file: %w", err)
	}

	if err := viper.Unmarshal(&ActiveConfig); err != nil {
		return fmt.Errorf("unmarshal config: %w", err)
	}

	return nil
}

// GetDBPath gibt den expandierten Pfad zur SQLite-Datenbank zurück
func GetDBPath() string {
	path := ActiveConfig.DBPath
	if path == "" {
		path = "~/.config/postctl/postctl.db"
	}
	if strings.HasPrefix(path, "~") {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, path[1:])
	}
	return filepath.Clean(path)
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

	// Viper Instanz ebenfalls aktualisieren
	viper.Set("db_path", ActiveConfig.DBPath)
	viper.Set("defaults.timezone", ActiveConfig.Defaults.Timezone)
	viper.Set("defaults.dry_run", ActiveConfig.Defaults.DryRun)
	viper.Set("defaults.image_dir", ActiveConfig.Defaults.ImageDir)
	viper.Set("ai.provider", ActiveConfig.AI.Provider)
	viper.Set("ai.model", ActiveConfig.AI.Model)
	viper.Set("ai.api_key", ActiveConfig.AI.APIKey)
	viper.Set("ai.base_url", ActiveConfig.AI.BaseURL)
	viper.Set("twitter.client_id", ActiveConfig.Twitter.ClientID)
	viper.Set("twitter.client_secret", ActiveConfig.Twitter.ClientSecret)
	viper.Set("linkedin.client_id", ActiveConfig.LinkedIn.ClientID)
	viper.Set("linkedin.client_secret", ActiveConfig.LinkedIn.ClientSecret)
	viper.Set("threads.app_id", ActiveConfig.Threads.AppID)
	viper.Set("threads.app_secret", ActiveConfig.Threads.AppSecret)

	return nil
}

