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
	case "mastodon.instance_url":
		config.ActiveConfig.Mastodon.InstanceURL = value
	case "mastodon.client_id":
		config.ActiveConfig.Mastodon.ClientID = value
	case "mastodon.client_secret":
		config.ActiveConfig.Mastodon.ClientSecret = value
	case "bluesky.handle":
		config.ActiveConfig.Bluesky.Handle = value
	case "bluesky.app_password":
		config.ActiveConfig.Bluesky.AppPassword = value
	case "reddit.client_id":
		config.ActiveConfig.Reddit.ClientID = value
	case "reddit.client_secret":
		config.ActiveConfig.Reddit.ClientSecret = value
	case "facebook.app_id":
		config.ActiveConfig.Facebook.AppID = value
	case "facebook.app_secret":
		config.ActiveConfig.Facebook.AppSecret = value
	case "facebook.page_id":
		config.ActiveConfig.Facebook.PageID = value
	case "license_key", "licensekey":
		config.ActiveConfig.LicenseKey = value
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}
	return nil
}

type configShowJSON struct {
	OK     bool          `json:"ok"`
	Config config.Config `json:"config"`
	IsPro  bool          `json:"is_pro"`
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
	masked.Mastodon.ClientSecret = maskSecret(masked.Mastodon.ClientSecret)
	masked.Bluesky.AppPassword = maskSecret(masked.Bluesky.AppPassword)
	masked.Reddit.ClientSecret = maskSecret(masked.Reddit.ClientSecret)
	masked.Facebook.AppSecret = maskSecret(masked.Facebook.AppSecret)
	masked.LicenseKey = maskSecret(masked.LicenseKey)

	out := cmd.OutOrStdout()

	if FormatFlag == "json" {
		outJSON := configShowJSON{
			OK:     true,
			Config: masked,
			IsPro:  config.IsPro(),
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
		fmt.Fprintf(out, "    app_secret:      %s\n\n", masked.Threads.AppSecret)
		fmt.Fprintln(out, "  mastodon:")
		fmt.Fprintf(out, "    instance_url:    %s\n", masked.Mastodon.InstanceURL)
		fmt.Fprintf(out, "    client_id:       %s\n", masked.Mastodon.ClientID)
		fmt.Fprintf(out, "    client_secret:   %s\n\n", masked.Mastodon.ClientSecret)
		fmt.Fprintln(out, "  bluesky:")
		fmt.Fprintf(out, "    handle:          %s\n", masked.Bluesky.Handle)
		fmt.Fprintf(out, "    app_password:    %s\n\n", masked.Bluesky.AppPassword)
		fmt.Fprintln(out, "  reddit:")
		fmt.Fprintf(out, "    client_id:       %s\n", masked.Reddit.ClientID)
		fmt.Fprintf(out, "    client_secret:   %s\n\n", masked.Reddit.ClientSecret)
		fmt.Fprintln(out, "  facebook:")
		fmt.Fprintf(out, "    app_id:          %s\n", masked.Facebook.AppID)
		fmt.Fprintf(out, "    app_secret:      %s\n", masked.Facebook.AppSecret)
		fmt.Fprintf(out, "    page_id:         %s\n\n", masked.Facebook.PageID)
		
		statusStr := "CORE"
		if config.IsPro() {
			statusStr = "PRO ACTIVE"
		}
		fmt.Fprintf(out, "  license_key:       %s [%s]\n", masked.LicenseKey, statusStr)
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

var isInteractiveConfig bool

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
	if isInteractiveConfig {
		fmt.Println("\nDrücke Enter, um zur TUI zurückzukehren...")
		var wait string
		fmt.Scanln(&wait)
	}
	exitFunc(exitCode)
}

var exportPassword string
var exportOutputFile string

var configExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export configuration and database",
	Long:  `Export the active configuration and SQLite database into a secure, encrypted backup package.`,
	Run: func(cmd *cobra.Command, args []string) {
		isInteractiveConfig = exportPassword == ""

		if isInteractiveConfig {
			// Terminal komplett leeren (wie clear)
			fmt.Print("\033[H\033[2J\033[3J")
			fmt.Println("=== postctl KONFIGURATIONS-EXPORT / BACKUP EXPORT ===")
			fmt.Println("Beschreibung: Sichert und verschlüsselt deine Konfiguration und die SQLite-Datenbank.")
			fmt.Println("Hilfe: Gib ein Master-Passwort für die AES-Verschlüsselung an. Dieses Passwort")
			fmt.Println("       wird auf dem anderen Gerät benötigt, um das Backup wiederherzustellen.")
			fmt.Println("       Du kannst den Ziel-Dateipfad per Drag & Drop in dieses Terminal schieben/ziehen.")
			fmt.Println("==========================================================================")
			fmt.Println()

			fmt.Print("➔ Gib ein Master-Passwort für die Verschlüsselung ein: ")
			fmt.Scanln(&exportPassword)
			if exportPassword == "" {
				reportConfigError(cmd, fmt.Errorf("Passwort darf nicht leer sein"), 1)
				return
			}

			if !cmd.Flags().Changed("output") {
				fmt.Printf("➔ Ziel-Dateipfad für das Backup [Standard: %s]: ", exportOutputFile)
				var inputPath string
				fmt.Scanln(&inputPath)
				inputPath = strings.TrimSpace(inputPath)
				inputPath = strings.ReplaceAll(inputPath, "\"", "")
				inputPath = strings.ReplaceAll(inputPath, "'", "")
				if inputPath != "" {
					exportOutputFile = inputPath
				}
			}
		}

		err := config.ExportConfig(exportPassword, exportOutputFile)
		if err != nil {
			reportConfigError(cmd, fmt.Errorf("Export fehlgeschlagen: %w", err), 2)
			return
		}

		if isInteractiveConfig {
			fmt.Println()
			fmt.Printf("✅ Konfiguration und Datenbank erfolgreich verschlüsselt exportiert nach:\n   %s\n", exportOutputFile)
			fmt.Println("\nDrücke Enter, um zur TUI zurückzukehren...")
			var wait string
			fmt.Scanln(&wait)
		} else {
			fmt.Printf("Konfiguration und Datenbank erfolgreich verschlüsselt exportiert nach: %s\n", exportOutputFile)
		}
	},
}

var importPassword string
var importInputFile string

var configImportCmd = &cobra.Command{
	Use:   "import",
	Short: "Import configuration and database",
	Long:  `Import and decrypt a postctl backup package to restore your configuration and database.`,
	Run: func(cmd *cobra.Command, args []string) {
		isInteractiveConfig = importPassword == ""

		if isInteractiveConfig {
			// Terminal komplett leeren (wie clear)
			fmt.Print("\033[H\033[2J\033[3J")
			fmt.Println("=== postctl KONFIGURATIONS-IMPORT / BACKUP IMPORT ===")
			fmt.Println("Beschreibung: Stellt deine Konfiguration und SQLite-Datenbank aus einem Backup wieder her.")
			fmt.Println("Hilfe: Gib das Master-Passwort an, das beim Exportieren verwendet wurde.")
			fmt.Println("       Du kannst die Backup-Datei (.bin) einfach per Drag & Drop")
			fmt.Println("       aus dem Finder direkt in dieses Terminalfenster schieben/ziehen!")
			fmt.Println("==========================================================================")
			fmt.Println()

			fmt.Print("➔ Gib das Master-Passwort zur Entschlüsselung ein: ")
			fmt.Scanln(&importPassword)
			if importPassword == "" {
				reportConfigError(cmd, fmt.Errorf("Passwort darf nicht leer sein"), 1)
				return
			}

			if !cmd.Flags().Changed("file") {
				fmt.Printf("➔ Dateipfad der Backup-Datei [Standard: %s]: ", importInputFile)
				var inputPath string
				fmt.Scanln(&inputPath)
				inputPath = strings.TrimSpace(inputPath)
				inputPath = strings.ReplaceAll(inputPath, "\"", "")
				inputPath = strings.ReplaceAll(inputPath, "'", "")
				if inputPath != "" {
					importInputFile = inputPath
				}
			}
		}

		err := config.ImportConfig(importPassword, importInputFile)
		if err != nil {
			reportConfigError(cmd, fmt.Errorf("Import fehlgeschlagen: %w", err), 2)
			return
		}

		if isInteractiveConfig {
			fmt.Println()
			fmt.Println("✅ Konfiguration und Datenbank erfolgreich entschlüsselt und wiederhergestellt!")
			fmt.Println("\nDrücke Enter, um zur TUI zurückzukehren...")
			var wait string
			fmt.Scanln(&wait)
		} else {
			fmt.Println("Konfiguration und Datenbank erfolgreich entschlüsselt und wiederhergestellt!")
		}
	},
}

func init() {
	configExportCmd.Flags().StringVarP(&exportPassword, "password", "p", "", "Master password for encryption")
	configExportCmd.Flags().StringVarP(&exportOutputFile, "output", "o", "postctl_backup.bin", "Path to the output encrypted backup file")

	configImportCmd.Flags().StringVarP(&importPassword, "password", "p", "", "Master password for decryption")
	configImportCmd.Flags().StringVarP(&importInputFile, "file", "f", "postctl_backup.bin", "Path to the encrypted backup file to import")

	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configExportCmd)
	configCmd.AddCommand(configImportCmd)
	rootCmd.AddCommand(configCmd)
}
