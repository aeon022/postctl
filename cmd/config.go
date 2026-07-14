package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/models"
	"github.com/aeon022/postctl/internal/platforms"
	"github.com/aeon022/postctl/internal/store"
	"github.com/spf13/cobra"
)

var setupCookie string
var setupCt0 string

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
	case "facebook.app_id":
		config.ActiveConfig.Facebook.AppID = value
	case "facebook.app_secret":
		config.ActiveConfig.Facebook.AppSecret = value
	case "facebook.page_id":
		config.ActiveConfig.Facebook.PageID = value
	case "telegram.bot_token":
		config.ActiveConfig.Telegram.BotToken = value
	case "telegram.chat_id":
		config.ActiveConfig.Telegram.ChatID = value
	case "discord.webhook_url":
		config.ActiveConfig.Discord.WebhookURL = value
	case "reddit.client_id":
		config.ActiveConfig.Reddit.ClientID = value
	case "reddit.client_secret":
		config.ActiveConfig.Reddit.ClientSecret = value
	case "reddit.username":
		config.ActiveConfig.Reddit.Username = value
	case "reddit.password":
		config.ActiveConfig.Reddit.Password = value
	case "devto.api_token":
		config.ActiveConfig.DevTo.APIToken = value
	case "hashnode.api_token":
		config.ActiveConfig.Hashnode.APIToken = value
	case "hashnode.publication_id":
		config.ActiveConfig.Hashnode.PublicationID = value
	case "medium.integration_token":
		config.ActiveConfig.Medium.IntegrationToken = value
	case "instagram.access_token":
		config.ActiveConfig.Instagram.AccessToken = value
	case "instagram.account_id":
		config.ActiveConfig.Instagram.AccountID = value
	case "pinterest.access_token":
		config.ActiveConfig.Pinterest.AccessToken = value
	case "pinterest.board_id":
		config.ActiveConfig.Pinterest.BoardID = value
	case "youtube.client_id":
		config.ActiveConfig.YouTube.ClientID = value
	case "youtube.client_secret":
		config.ActiveConfig.YouTube.ClientSecret = value
	case "license_key", "licensekey":
		config.ActiveConfig.LicenseKey = value
	case "license_status", "licensestatus":
		config.ActiveConfig.LicenseStatus = value
	case "polar_org_id", "polarorgid":
		config.ActiveConfig.PolarOrgID = value
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
	masked.Facebook.AppSecret = maskSecret(masked.Facebook.AppSecret)
	masked.Telegram.BotToken = maskSecret(masked.Telegram.BotToken)
	masked.Discord.WebhookURL = maskSecret(masked.Discord.WebhookURL)
	masked.Reddit.ClientSecret = maskSecret(masked.Reddit.ClientSecret)
	masked.Reddit.Password = maskSecret(masked.Reddit.Password)
	masked.DevTo.APIToken = maskSecret(masked.DevTo.APIToken)
	masked.Hashnode.APIToken = maskSecret(masked.Hashnode.APIToken)
	masked.Medium.IntegrationToken = maskSecret(masked.Medium.IntegrationToken)
	masked.Instagram.AccessToken = maskSecret(masked.Instagram.AccessToken)
	masked.Pinterest.AccessToken = maskSecret(masked.Pinterest.AccessToken)
	masked.YouTube.ClientSecret = maskSecret(masked.YouTube.ClientSecret)
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
		fmt.Fprintln(out, "  facebook:")
		fmt.Fprintf(out, "    app_id:          %s\n", masked.Facebook.AppID)
		fmt.Fprintf(out, "    app_secret:      %s\n", masked.Facebook.AppSecret)
		fmt.Fprintf(out, "    page_id:         %s\n\n", masked.Facebook.PageID)
		
		statusStr := "CORE (MIT)"
		if config.IsPro() {
			statusStr = "PRO ACTIVE (Polar.sh)"
		}
		fmt.Fprintf(out, "  polar_org_id:      %s\n", masked.PolarOrgID)
		fmt.Fprintf(out, "  license_status:    %s\n", masked.LicenseStatus)
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

var configSetupCmd = &cobra.Command{
	Use:   "setup [platform]",
	Short: "Interactive platform configuration assistant",
	Long:  `Run an interactive step-by-step setup wizard to configure API credentials for a platform (Twitter, LinkedIn, Threads, Mastodon, Bluesky, Reddit, Facebook).`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("Bitte gib eine Plattform an (z. B. twitter, linkedin, threads, mastodon, bluesky, reddit, facebook).")
			return
		}

		platform := strings.ToLower(strings.TrimSpace(args[0]))

		if platform == "twitter" && setupCookie != "" && setupCt0 != "" {
			config.ActiveConfig.Twitter.AuthMode = "cookie"
			dbPath := config.ActiveConfig.DBPath
			s, err := store.NewSQLiteStore(dbPath)
			if err != nil {
				fmt.Printf("Fehler beim Öffnen der Datenbank: %v\n", err)
				return
			}
			err = s.SaveToken(context.Background(), models.PlatformTwitter, setupCookie, setupCt0, nil)
			if err != nil {
				fmt.Printf("Fehler beim Speichern der Cookies: %v\n", err)
				return
			}
			config.ActiveConfig.Twitter.ClientID = ""
			config.ActiveConfig.Twitter.ClientSecret = ""
			if err := config.SaveConfig(); err != nil {
				fmt.Printf("\n❌ Fehler beim Speichern der Konfiguration: %v\n", err)
				return
			}
			fmt.Println("⚠️ WARNUNG: Die Cookie-basierte Methode ist inoffiziell, fehleranfällig und kann zur Sperrung deines Kontos führen.")
			fmt.Println("✔ Twitter/X wurde erfolgreich über Flags im Cookie-Modus verbunden!")
			return
		}
		
		// Terminal komplett leeren
		fmt.Print("\033[H\033[2J\033[3J")
		
		// Titel ausgeben
		fmt.Printf("=== postctl CONFIGURATION ASSISTANT: %s ===\n", strings.ToUpper(platform))
		fmt.Println("Beschreibung: Dieser Assistent hilft dir, die nötigen API-Schlüssel für die")
		fmt.Printf("              Plattform %s zu hinterlegen.\n", strings.ToUpper(platform))
		fmt.Println("==========================================================================")
		fmt.Println()

		reader := bufio.NewReader(os.Stdin)

		switch platform {
		case "twitter":
			fmt.Println("Wähle die Verbindungsmethode für Twitter/X:")
			fmt.Println("  1) Offizielle API (Erfordert kostenpflichtigen API-Zugang/Credits) [Empfohlen/Sicher]")
			fmt.Println("  2) Cookie-basiert (Kostenloser inoffizieller Weg, aber fehleranfällig; Risiko von Kontosperrung!)")
			fmt.Println()
			fmt.Print("➔ Deine Wahl (1 oder 2, Standard: 1): ")
			choice, _ := reader.ReadString('\n')
			choice = strings.TrimSpace(choice)

			if choice == "2" {
				config.ActiveConfig.Twitter.AuthMode = "cookie"
				
				fmt.Println()
				fmt.Println("⚠️ WARNUNG & SICHERHEITSHINWEIS:")
				fmt.Println("Die Cookie-basierte Methode ist ein inoffizieller Umgehungsversuch. Sie simuliert")
				fmt.Println("eine Websitzung und kann fehlerhaft sein oder zur Sperrung deines X-Kontos führen.")
				fmt.Println("Offiziell wird nur der API-Zugang mit kostenpflichtigen Credits unterstützt.")
				fmt.Println()
				fmt.Println("Anleitung zur Cookie-Extraktion:")
				fmt.Println("  1. Öffne twitter.com im Browser und logge dich ein.")
				fmt.Println("  2. Drücke F12, um die Entwicklertools zu öffnen.")
				fmt.Println("  3. Gehe zu 'Application' (Chrome) oder 'Storage' (Firefox) -> 'Cookies'.")
				fmt.Println("  4. Kopiere die Werte für 'auth_token' und 'ct0'.")
				fmt.Println()

				fmt.Print("➔ auth_token Cookie: ")
				authCookie, _ := reader.ReadString('\n')
				authCookie = strings.TrimSpace(authCookie)

				fmt.Print("➔ ct0 Cookie (CSRF Token): ")
				csrfCookie, _ := reader.ReadString('\n')
				csrfCookie = strings.TrimSpace(csrfCookie)

				if authCookie == "" || csrfCookie == "" {
					fmt.Println("Fehler: Beide Cookie-Werte müssen ausgefüllt sein!")
					return
				}

				// DB initialisieren und Cookies speichern
				dbPath := config.ActiveConfig.DBPath
				s, err := store.NewSQLiteStore(dbPath)
				if err != nil {
					fmt.Printf("Fehler beim Öffnen der Datenbank: %v\n", err)
					return
				}

				// auth_token im token-Feld, ct0 im refresh-Feld speichern
				err = s.SaveToken(context.Background(), models.PlatformTwitter, authCookie, csrfCookie, nil)
				if err != nil {
					fmt.Printf("Fehler beim Speichern der Cookies: %v\n", err)
					return
				}

				// ClientID/ClientSecret leeren, da Cookie-basiert
				config.ActiveConfig.Twitter.ClientID = ""
				config.ActiveConfig.Twitter.ClientSecret = ""

				fmt.Println("\n⚠️ Die Cookie-basierte Methode ist inoffiziell, fehleranfällig und birgt Risiken.")
				fmt.Println("✔ Twitter/X wurde erfolgreich im Cookie-Modus verbunden!")
			} else {
				config.ActiveConfig.Twitter.AuthMode = "api"
				
				fmt.Println()
				fmt.Println("Schritt 1: Gehe zum Twitter Developer Portal unter https://developer.twitter.com")
				fmt.Println("Schritt 2: Erstelle ein Projekt und eine App mit OAuth 2.0 PKCE (App-Typ: Web/Native App)")
				fmt.Println("Schritt 3: Setze die Redirect-URI auf: http://localhost:8753/callback")
				fmt.Println("Schritt 4: Trage unten die erhaltene Client ID & Client Secret ein:")
				fmt.Println()

				fmt.Print("➔ Client ID: ")
				clientID, _ := reader.ReadString('\n')
				clientID = strings.TrimSpace(clientID)

				fmt.Print("➔ Client Secret: ")
				clientSecret, _ := reader.ReadString('\n')
				clientSecret = strings.TrimSpace(clientSecret)

				if clientID != "" {
					config.ActiveConfig.Twitter.ClientID = clientID
				}
				if clientSecret != "" {
					config.ActiveConfig.Twitter.ClientSecret = clientSecret
				}
				
				// Wenn API gewählt, Token löschen falls Cookie-basierte Reste vorhanden waren
				dbPath := config.ActiveConfig.DBPath
				s, err := store.NewSQLiteStore(dbPath)
				if err == nil {
					_ = s.DeleteToken(context.Background(), models.PlatformTwitter)
				}

				fmt.Println("\n✔ Client ID und Secret wurden konfiguriert. Du kannst dich jetzt im Hauptmenü verbinden!")
			}

		case "linkedin":
			fmt.Println("Schritt 1: Gehe zum LinkedIn Developer Portal unter https://linkedin.com/developers")
			fmt.Println("Schritt 2: Erstelle eine App und aktiviere unter 'Products':")
			fmt.Println("           - Share on LinkedIn")
			fmt.Println("           - Sign In with LinkedIn using OIDC")
			fmt.Println("Schritt 3: Setze die Redirect-URI unter 'Auth' auf: http://localhost:8753/callback")
			fmt.Println("Schritt 4: Trage unten die erhaltene Client ID & Client Secret ein:")
			fmt.Println()

			fmt.Print("➔ Client ID: ")
			clientID, _ := reader.ReadString('\n')
			clientID = strings.TrimSpace(clientID)

			fmt.Print("➔ Client Secret: ")
			clientSecret, _ := reader.ReadString('\n')
			clientSecret = strings.TrimSpace(clientSecret)

			if clientID != "" {
				config.ActiveConfig.LinkedIn.ClientID = clientID
			}
			if clientSecret != "" {
				config.ActiveConfig.LinkedIn.ClientSecret = clientSecret
			}

		case "threads":
			fmt.Println("Schritt 1: Gehe zum Meta Developer Portal unter https://developers.facebook.com")
			fmt.Println("Schritt 2: Erstelle eine App vom Typ 'Consumer' und füge 'Threads API' hinzu")
			fmt.Println("Schritt 3: Setze Redirect-URIs unter 'Threads API' ➔ 'Threads-Einstellungen' auf:")
			fmt.Println("           - Valid OAuth Redirect: https://localhost:8753/callback")
			fmt.Println("           - Deinstallations-URL:  https://localhost:8753/uninstall")
			fmt.Println("           - Datenlöschungs-URL:   https://localhost:8753/delete")
			fmt.Println("Schritt 4: Trage unten die erhaltene App-ID & App Secret ein:")
			fmt.Println()

			fmt.Print("➔ App-ID: ")
			appID, _ := reader.ReadString('\n')
			appID = strings.TrimSpace(appID)

			fmt.Print("➔ App Secret: ")
			appSecret, _ := reader.ReadString('\n')
			appSecret = strings.TrimSpace(appSecret)

			if appID != "" {
				config.ActiveConfig.Threads.AppID = appID
			}
			if appSecret != "" {
				config.ActiveConfig.Threads.AppSecret = appSecret
			}

		case "mastodon":
			fmt.Println("Schritt 1: Trage die Instanz-URL ein (z. B. https://mastodon.social)")
			fmt.Println("Schritt 2: (Optional) Trage Client ID und Client Secret ein, falls du bereits")
			fmt.Println("           eine registrierte App hast. Falls nicht, registriert postctl")
			fmt.Println("           die App beim Authentifizieren automatisch!")
			fmt.Println()

			fmt.Print("➔ Instanz-URL [Standard: https://mastodon.social]: ")
			instanceURL, _ := reader.ReadString('\n')
			instanceURL = strings.TrimSpace(instanceURL)
			if instanceURL == "" {
				instanceURL = "https://mastodon.social"
			}

			fmt.Print("➔ Client ID (Optional, Enter zum Überspringen): ")
			clientID, _ := reader.ReadString('\n')
			clientID = strings.TrimSpace(clientID)

			fmt.Print("➔ Client Secret (Optional, Enter zum Überspringen): ")
			clientSecret, _ := reader.ReadString('\n')
			clientSecret = strings.TrimSpace(clientSecret)

			config.ActiveConfig.Mastodon.InstanceURL = instanceURL
			if clientID != "" {
				config.ActiveConfig.Mastodon.ClientID = clientID
			}
			if clientSecret != "" {
				config.ActiveConfig.Mastodon.ClientSecret = clientSecret
			}

		case "bluesky":
			fmt.Println("Schritt 1: Logge dich bei Bluesky in deinem Profil ein")
			fmt.Println("Schritt 2: Gehe zu Einstellungen ➔ App-Passwörter")
			fmt.Println("Schritt 3: Erstelle ein neues App-Passwort (z. B. 'postctl')")
			fmt.Println("Schritt 4: Trage unten deinen Handle und das generierte App-Passwort ein:")
			fmt.Println()

			fmt.Print("➔ Bluesky Handle (z.B. deinname.bsky.social): ")
			handle, _ := reader.ReadString('\n')
			handle = strings.TrimSpace(handle)

			fmt.Print("➔ App-Passwort (z.B. xxxx-xxxx-xxxx-xxxx): ")
			appPassword, _ := reader.ReadString('\n')
			appPassword = strings.TrimSpace(appPassword)

			if handle != "" {
				config.ActiveConfig.Bluesky.Handle = handle
			}
			if appPassword != "" {
				config.ActiveConfig.Bluesky.AppPassword = appPassword
			}

		case "facebook":
			fmt.Println("Schritt 1: Gehe zum Meta Developer Portal unter https://developers.facebook.com")
			fmt.Println("Schritt 2: Erstelle eine App und füge das Produkt 'Facebook Login' hinzu")
			fmt.Println("           mit Redirect-URI: https://localhost:8753/callback")
			fmt.Println("Schritt 3: Finde deine Facebook-Page-ID auf der Info-Seite deiner Facebook-Seite")
			fmt.Println("Schritt 4: Trage unten App-ID, App Secret und die Page-ID ein:")
			fmt.Println()

			fmt.Print("➔ App-ID: ")
			appID, _ := reader.ReadString('\n')
			appID = strings.TrimSpace(appID)

			fmt.Print("➔ App Secret: ")
			appSecret, _ := reader.ReadString('\n')
			appSecret = strings.TrimSpace(appSecret)

			fmt.Print("➔ Facebook Page ID: ")
			pageID, _ := reader.ReadString('\n')
			pageID = strings.TrimSpace(pageID)

			if appID != "" {
				config.ActiveConfig.Facebook.AppID = appID
			}
			if appSecret != "" {
				config.ActiveConfig.Facebook.AppSecret = appSecret
			}
			if pageID != "" {
				config.ActiveConfig.Facebook.PageID = pageID
			}

		case "telegram":
			fmt.Println("Schritt 1: Erstelle einen Telegram Bot via @BotFather in Telegram.")
			fmt.Println("Schritt 2: Kopiere das erhaltene HTTP API Token.")
			fmt.Println("Schritt 3: Füge den Bot zu deinem Kanal oder deiner Gruppe hinzu und mache ihn zum Admin.")
			fmt.Println("Schritt 4: Trage unten das Bot Token und die Chat ID (z.B. @deinkanal oder Gruppen-ID) ein:")
			fmt.Println()

			fmt.Print("➔ Bot Token: ")
			botToken, _ := reader.ReadString('\n')
			botToken = strings.TrimSpace(botToken)

			fmt.Print("➔ Chat ID (z.B. @kanalname oder -100...): ")
			chatID, _ := reader.ReadString('\n')
			chatID = strings.TrimSpace(chatID)

			if botToken != "" {
				config.ActiveConfig.Telegram.BotToken = botToken
			}
			if chatID != "" {
				config.ActiveConfig.Telegram.ChatID = chatID
			}

		case "discord":
			fmt.Println("Schritt 1: Gehe in deinem Discord-Server zu den Kanaleinstellungen.")
			fmt.Println("Schritt 2: Gehe zu Integrationen ➔ Webhooks.")
			fmt.Println("Schritt 3: Erstelle einen neuen Webhook und kopiere die Webhook-URL.")
			fmt.Println("Schritt 4: Trage unten die Webhook-URL ein:")
			fmt.Println()

			fmt.Print("➔ Webhook URL: ")
			webhookURL, _ := reader.ReadString('\n')
			webhookURL = strings.TrimSpace(webhookURL)

			if webhookURL != "" {
				config.ActiveConfig.Discord.WebhookURL = webhookURL
			}

		case "devto":
			fmt.Println("Schritt 1: Logge dich auf dev.to ein.")
			fmt.Println("Schritt 2: Gehe zu Settings ➔ Extensions.")
			fmt.Println("Schritt 3: Scrolle nach unten zu 'DEV Community API Keys' und generiere einen neuen Key.")
			fmt.Println("Schritt 4: Trage unten den API-Key ein:")
			fmt.Println()

			fmt.Print("➔ API Token: ")
			apiToken, _ := reader.ReadString('\n')
			apiToken = strings.TrimSpace(apiToken)

			if apiToken != "" {
				config.ActiveConfig.DevTo.APIToken = apiToken
			}

		case "reddit":
			fmt.Println("Schritt 1: Gehe zu https://www.reddit.com/prefs/apps.")
			fmt.Println("Schritt 2: Erstelle eine neue App (App-Typ: script).")
			fmt.Println("Schritt 3: Trage unten Client-ID, Client-Secret, Reddit-Benutzername und Reddit-Passwort ein:")
			fmt.Println()

			fmt.Print("➔ Client ID: ")
			clientID, _ := reader.ReadString('\n')
			clientID = strings.TrimSpace(clientID)

			fmt.Print("➔ Client Secret: ")
			clientSecret, _ := reader.ReadString('\n')
			clientSecret = strings.TrimSpace(clientSecret)

			fmt.Print("➔ Reddit Username: ")
			username, _ := reader.ReadString('\n')
			username = strings.TrimSpace(username)

			fmt.Print("➔ Reddit Password: ")
			password, _ := reader.ReadString('\n')
			password = strings.TrimSpace(password)

			if clientID != "" {
				config.ActiveConfig.Reddit.ClientID = clientID
			}
			if clientSecret != "" {
				config.ActiveConfig.Reddit.ClientSecret = clientSecret
			}
			if username != "" {
				config.ActiveConfig.Reddit.Username = username
			}
			if password != "" {
				config.ActiveConfig.Reddit.Password = password
			}

		case "hashnode":
			fmt.Println("Schritt 1: Logge dich auf hashnode.com ein und gehe ins Blog Dashboard.")
			fmt.Println("Schritt 2: Gehe zu Account Settings ➔ Developer und erstelle einen Personal Access Token.")
			fmt.Println("Schritt 3: Kopiere deine Publication ID (zu finden in Blog Dashboard ➔ Settings).")
			fmt.Println("Schritt 4: Trage unten den Token und die Publication ID ein:")
			fmt.Println()

			fmt.Print("➔ API Token: ")
			apiToken, _ := reader.ReadString('\n')
			apiToken = strings.TrimSpace(apiToken)

			fmt.Print("➔ Publication ID: ")
			publicationID, _ := reader.ReadString('\n')
			publicationID = strings.TrimSpace(publicationID)

			if apiToken != "" {
				config.ActiveConfig.Hashnode.APIToken = apiToken
			}
			if publicationID != "" {
				config.ActiveConfig.Hashnode.PublicationID = publicationID
			}

		case "medium":
			fmt.Println("Schritt 1: Logge dich auf medium.com ein.")
			fmt.Println("Schritt 2: Gehe zu Settings ➔ Security and apps ➔ Integration tokens.")
			fmt.Println("Schritt 3: Generiere einen neuen Integration Token.")
			fmt.Println("Schritt 4: Trage unten den Token ein:")
			fmt.Println()

			fmt.Print("➔ Integration Token: ")
			integrationToken, _ := reader.ReadString('\n')
			integrationToken = strings.TrimSpace(integrationToken)

			if integrationToken != "" {
				config.ActiveConfig.Medium.IntegrationToken = integrationToken
			}

		default:
			fmt.Printf("Unbekannte Plattform: %s\n", platform)
			return
		}

		// Speichern
		if err := config.SaveConfig(); err != nil {
			fmt.Printf("\n❌ Fehler beim Speichern der Konfiguration: %v\n", err)
			return
		}

		fmt.Println()
		fmt.Println("✅ Konfiguration erfolgreich in config.yaml gespeichert!")
		fmt.Println("\nDrücke Enter, um fortzufahren...")
		_, _ = reader.ReadString('\n')
	},
}

var configTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Test connection to all configured platform APIs",
	Long:  `Test the API credentials and connections for all configured social media and blog platforms.`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		dbPath := config.GetDBPath()
		s, err := store.NewSQLiteStore(dbPath)
		if err != nil {
			cmd.Printf("❌ Fehler beim Öffnen der Datenbank: %v\n", err)
			return
		}
		defer s.Close()

		platformsList := []string{
			models.PlatformTwitter,
			models.PlatformLinkedIn,
			models.PlatformThreads,
			models.PlatformMastodon,
			models.PlatformBluesky,
			models.PlatformFacebook,
			models.PlatformTelegram,
			models.PlatformDiscord,
			models.PlatformDevTo,
			models.PlatformReddit,
			models.PlatformHashnode,
			models.PlatformMedium,
		}

		cmd.Println("=== postctl CONNECTION DIAGNOSTIC ===")
		cmd.Println("Teste Verbindungen zu den konfigurierten Plattform-APIs...")
		cmd.Println("=====================================")
		cmd.Println()

		for _, pName := range platformsList {
			isConfigured := false
			switch pName {
			case models.PlatformTwitter:
				isConfigured = config.ActiveConfig.Twitter.ClientID != "" || config.ActiveConfig.Twitter.AuthMode == "cookie"
			case models.PlatformLinkedIn:
				isConfigured = config.ActiveConfig.LinkedIn.ClientID != ""
			case models.PlatformThreads:
				isConfigured = config.ActiveConfig.Threads.AppID != ""
			case models.PlatformMastodon:
				isConfigured = config.ActiveConfig.Mastodon.InstanceURL != ""
			case models.PlatformBluesky:
				isConfigured = config.ActiveConfig.Bluesky.Handle != ""
			case models.PlatformFacebook:
				isConfigured = config.ActiveConfig.Facebook.AppID != ""
			case models.PlatformTelegram:
				isConfigured = config.ActiveConfig.Telegram.BotToken != ""
			case models.PlatformDiscord:
				isConfigured = config.ActiveConfig.Discord.WebhookURL != ""
			case models.PlatformDevTo:
				isConfigured = config.ActiveConfig.DevTo.APIToken != ""
			case models.PlatformReddit:
				isConfigured = config.ActiveConfig.Reddit.ClientID != ""
			case models.PlatformHashnode:
				isConfigured = config.ActiveConfig.Hashnode.APIToken != ""
			case models.PlatformMedium:
				isConfigured = config.ActiveConfig.Medium.IntegrationToken != ""
			}

			if !isConfigured {
				cmd.Printf("➖ %-10s: Nicht konfiguriert (Übersprungen)\n", strings.ToUpper(pName))
				continue
			}

			// Bei OAuth-Plattformen prüfen wir, ob ein Token in der DB liegt, um das blockierende Öffnen des Browsers zu vermeiden
			isOAuth := pName == models.PlatformLinkedIn || pName == models.PlatformThreads || pName == models.PlatformMastodon || pName == models.PlatformFacebook || (pName == models.PlatformTwitter && config.ActiveConfig.Twitter.AuthMode != "cookie")

			if isOAuth {
				token, _, expiry, err := s.GetToken(ctx, pName)
				if err == nil && token != "" {
					if expiry != nil && !expiry.IsZero() && expiry.Before(time.Now()) {
						cmd.Printf("⚠️  %-10s: Token abgelaufen (Re-Authentifizierung erforderlich)\n", strings.ToUpper(pName))
					} else {
						cmd.Printf("✅ %-10s: Verbindung aktiv (Token vorhanden)\n", strings.ToUpper(pName))
					}
				} else {
					cmd.Printf("❌ %-10s: Nicht verbunden (Bitte führe 'postctl config setup %s' aus)\n", strings.ToUpper(pName), pName)
				}
				continue
			}

			plat, err := platforms.GetPlatform(pName, s, false)
			if err != nil {
				cmd.Printf("❌ %-10s: Fehler beim Erstellen des API-Clients: %v\n", strings.ToUpper(pName), err)
				continue
			}

			cmd.Printf("⏳ %-10s: Teste Verbindung...", strings.ToUpper(pName))
			err = plat.Auth(ctx)
			// Text zurücksetzen
			cmd.Print("\r\033[K")
			if err != nil {
				cmd.Printf("❌ %-10s: Fehler bei Authentifizierung: %v\n", strings.ToUpper(pName), err)
			} else {
				cmd.Printf("✅ %-10s: Verbindung erfolgreich hergestellt!\n", strings.ToUpper(pName))
			}
		}
		cmd.Println()
		cmd.Println("=====================================")
	},
}

func init() {
	configExportCmd.Flags().StringVarP(&exportPassword, "password", "p", "", "Master password for encryption")
	configExportCmd.Flags().StringVarP(&exportOutputFile, "output", "o", "postctl_backup.bin", "Path to the output encrypted backup file")

	configImportCmd.Flags().StringVarP(&importPassword, "password", "p", "", "Master password for decryption")
	configImportCmd.Flags().StringVarP(&importInputFile, "file", "f", "postctl_backup.bin", "Path to the encrypted backup file to import")

	configSetupCmd.Flags().StringVar(&setupCookie, "cookie", "", "Twitter/X Cookie string (full header or auth_token)")
	configSetupCmd.Flags().StringVar(&setupCt0, "ct0", "", "Twitter/X ct0 (CSRF token)")

	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configExportCmd)
	configCmd.AddCommand(configImportCmd)
	configCmd.AddCommand(configSetupCmd)
	configCmd.AddCommand(configTestCmd)
	rootCmd.AddCommand(configCmd)
}
