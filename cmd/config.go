package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aeon022/postctl/internal/config"
	"github.com/spf13/cobra"
)

// configCmd repräsentiert das Hauptkommando config
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration settings",
	Long:  `Show or set configuration options for postctl.`,
}

// configShowCmd repräsentiert config show
var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration settings",
	Long:  `Display the active configuration settings, with API secrets masked.`,
	Run: func(cmd *cobra.Command, args []string) {
		reportConfigShow(cmd)
	},
}

// configSetCmd repräsentiert config set
var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration setting value",
	Long:  `Update a configuration setting in config.yaml.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		key := args[0]
		value := args[1]

		if err := setConfigValue(key, value); err != nil {
			reportConfigError(cmd, err, 1)
			return
		}

		if !DryRunFlag {
			if err := config.SaveConfig(); err != nil {
				reportConfigError(cmd, fmt.Errorf("failed to save config: %w", err), 2)
				return
			}
		}

		reportConfigSuccess(cmd, key, value)
	},
}

func maskSecret(s string) string {
	if s == "" {
		return ""
	}
	if len(s) <= 4 {
		return "****"
	}
	return s[:4] + strings.Repeat("*", len(s)-4)
}

func setConfigValue(key, value string) error {
	switch strings.ToLower(key) {
	case "db_path":
		config.ActiveConfig.DBPath = value
	case "defaults.timezone":
		config.ActiveConfig.Defaults.Timezone = value
	case "defaults.dry_run":
		config.ActiveConfig.Defaults.DryRun = (value == "true" || value == "1")
	case "defaults.image_dir":
		config.ActiveConfig.Defaults.ImageDir = value
	case "ai.provider":
		config.ActiveConfig.AI.Provider = value
	case "ai.model":
		config.ActiveConfig.AI.Model = value
	case "ai.api_key":
		config.ActiveConfig.AI.APIKey = value
	case "ai.base_url":
		config.ActiveConfig.AI.BaseURL = value
	case "twitter.client_id":
		config.ActiveConfig.Twitter.ClientID = value
	case "twitter.client_secret":
		config.ActiveConfig.Twitter.ClientSecret = value
	case "linkedin.client_id":
		config.ActiveConfig.LinkedIn.ClientID = value
	case "linkedin.client_secret":
		config.ActiveConfig.LinkedIn.ClientSecret = value
	case "threads.app_id":
		config.ActiveConfig.Threads.AppID = value
	case "threads.app_secret":
		config.ActiveConfig.Threads.AppSecret = value
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}
	return nil
}

type configShowJSON struct {
	OK     bool          `json:"ok"`
	Config config.Config `json:"config"`
}

type configSuccessJSON struct {
	OK    bool   `json:"ok"`
	Key   string `json:"key"`
	Value string `json:"value"`
}

type configErrorJSON struct {
	OK     bool     `json:"ok"`
	Code   int      `json:"code"`
	Errors []string `json:"errors"`
}

func reportConfigShow(cmd *cobra.Command) {
	// Maskierte Kopie erstellen
	masked := config.ActiveConfig
	masked.AI.APIKey = maskSecret(masked.AI.APIKey)
	masked.Twitter.ClientSecret = maskSecret(masked.Twitter.ClientSecret)
	masked.LinkedIn.ClientSecret = maskSecret(masked.LinkedIn.ClientSecret)
	masked.Threads.AppSecret = maskSecret(masked.Threads.AppSecret)

	out := cmd.OutOrStdout()

	if FormatFlag == "json" {
		outJSON := configShowJSON{
			OK:     true,
			Config: masked,
		}
		jsonBytes, _ := json.MarshalIndent(outJSON, "", "  ")
		fmt.Fprintln(out, string(jsonBytes))
	} else {
		fmt.Fprintln(out, "=== postctl CONFIGURATION ===")
		fmt.Fprintf(out, "  db_path:           %s\n\n", masked.DBPath)
		fmt.Fprintln(out, "  defaults:")
		fmt.Fprintf(out, "    timezone:        %s\n", masked.Defaults.Timezone)
		fmt.Fprintf(out, "    dry_run:         %t\n", masked.Defaults.DryRun)
		fmt.Fprintf(out, "    image_dir:       %s\n\n", masked.Defaults.ImageDir)
		fmt.Fprintln(out, "  ai:")
		fmt.Fprintf(out, "    provider:        %s\n", masked.AI.Provider)
		fmt.Fprintf(out, "    model:           %s\n", masked.AI.Model)
		fmt.Fprintf(out, "    api_key:         %s\n", masked.AI.APIKey)
		fmt.Fprintf(out, "    base_url:        %s\n\n", masked.AI.BaseURL)
		fmt.Fprintln(out, "  twitter:")
		fmt.Fprintf(out, "    client_id:       %s\n", masked.Twitter.ClientID)
		fmt.Fprintf(out, "    client_secret:   %s\n\n", masked.Twitter.ClientSecret)
		fmt.Fprintln(out, "  linkedin:")
		fmt.Fprintf(out, "    client_id:       %s\n", masked.LinkedIn.ClientID)
		fmt.Fprintf(out, "    client_secret:   %s\n\n", masked.LinkedIn.ClientSecret)
		fmt.Fprintln(out, "  threads:")
		fmt.Fprintf(out, "    app_id:          %s\n", masked.Threads.AppID)
		fmt.Fprintf(out, "    app_secret:      %s\n", masked.Threads.AppSecret)
	}
}

func reportConfigSuccess(cmd *cobra.Command, key, value string) {
	out := cmd.OutOrStdout()
	if FormatFlag == "json" {
		outJSON := configSuccessJSON{
			OK:    true,
			Key:   key,
			Value: value,
		}
		jsonBytes, _ := json.MarshalIndent(outJSON, "", "  ")
		fmt.Fprintln(out, string(jsonBytes))
	} else {
		prefix := ""
		if DryRunFlag {
			prefix = "[DRY RUN] "
		}
		fmt.Fprintf(out, "%sSuccessfully set %q to %q\n", prefix, key, value)
	}
}

func reportConfigError(cmd *cobra.Command, err error, exitCode int) {
	outErr := cmd.ErrOrStderr()
	if FormatFlag == "json" {
		outJSON := configErrorJSON{
			OK:     false,
			Code:   exitCode,
			Errors: []string{err.Error()},
		}
		jsonBytes, _ := json.MarshalIndent(outJSON, "", "  ")
		fmt.Fprintln(outErr, string(jsonBytes))
	} else {
		fmt.Fprintf(outErr, "Config Error: %v\n", err)
	}
	exitFunc(exitCode)
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
	rootCmd.AddCommand(configCmd)
}
