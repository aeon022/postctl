package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/aeon022/postctl/internal/config"
	"github.com/aeon022/postctl/internal/models"
	"github.com/aeon022/postctl/internal/store"
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
			fmt.Println("  1) Offizielle API (erfordert Client ID & Client Secret)")
			fmt.Println("  2) Cookie-basiert (ohne API-Schlüssel, erfordert auth_token & ct0)")
			fmt.Println()
			fmt.Print("➔ Deine Wahl (1 oder 2, Standard: 1): ")
			choice, _ := reader.ReadString('\n')
			choice = strings.TrimSpace(choice)

			if choice == "2" {
				config.ActiveConfig.Twitter.AuthMode = "cookie"
				
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

				fmt.Println("\n✔ Twitter/X wurde erfolgreich im Cookie-Modus verbunden!")
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

func init() {
	configExportCmd.Flags().StringVarP(&exportPassword, "password", "p", "", "Master password for encryption")
	configExportCmd.Flags().StringVarP(&exportOutputFile, "output", "o", "postctl_backup.bin", "Path to the output encrypted backup file")

	configImportCmd.Flags().StringVarP(&importPassword, "password", "p", "", "Master password for decryption")
	configImportCmd.Flags().StringVarP(&importInputFile, "file", "f", "postctl_backup.bin", "Path to the encrypted backup file to import")

	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configExportCmd)
	configCmd.AddCommand(configImportCmd)
	configCmd.AddCommand(configSetupCmd)
	rootCmd.AddCommand(configCmd)
}
