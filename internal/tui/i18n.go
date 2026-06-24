package tui

import (
	"strings"

	"github.com/aeon022/postctl/internal/config"
)

// Tr gibt den übersetzten Text basierend auf der konfigurierten Sprache zurück
func Tr(key string) string {
	lang := strings.ToLower(config.ActiveConfig.Defaults.Language)
	if lang == "" {
		lang = "en"
	}

	if translations, ok := translationsMap[key]; ok {
		if val, exists := translations[lang]; exists {
			return val
		}
		// Fallback zu englisch
		if val, exists := translations["en"]; exists {
			return val
		}
	}
	return key
}

var translationsMap = map[string]map[string]string{
	// Tab headers
	"tab_dashboard": {
		"de": "● DASHBOARD",
		"en": "● DASHBOARD",
	},
	"tab_posts": {
		"de": "◷ BEITRÄGE",
		"en": "◷ POSTS",
	},
	"tab_schedule": {
		"de": "↺ TIMELINE",
		"en": "↺ SCHEDULE",
	},
	"tab_history": {
		"de": "📊 VERLAUF",
		"en": "📊 HISTORY",
	},
	"tab_settings": {
		"de": "⚙ EINSTELLUNGEN",
		"en": "⚙ SETTINGS",
	},

	// Common Headers
	"header_dashboard": {
		"de": "DASHBOARD",
		"en": "DASHBOARD",
	},
	"header_posts": {
		"de": "BEITRÄGE",
		"en": "POSTS",
	},
	"header_schedule": {
		"de": "TIMELINE (NÄCHSTE BEITRÄGE)",
		"en": "PUBLICATION TIMELINE (NEXT UP)",
	},
	"header_history": {
		"de": "POSTING HISTORY (VERLAUF)",
		"en": "POSTING HISTORY",
	},
	"header_settings": {
		"de": "EINSTELLUNGEN & VERBINDUNGEN",
		"en": "SETTINGS & CONNECTIONS",
	},

	// Editor
	"editor_title_create": {
		"de": " BEITRAG ERSTELLEN ",
		"en": " CREATE NEW POST ",
	},
	"editor_title_edit": {
		"de": " BEITRAG BEARBEITEN ",
		"en": " EDIT POST DRAFT ",
	},
	"editor_label_platform": {
		"de": "Plattform:  ",
		"en": "Platform:   ",
	},
	"editor_label_campaign": {
		"de": "Kampagne:   ",
		"en": "Campaign:   ",
	},
	"editor_label_schedule": {
		"de": "Geplant am: ",
		"en": "Scheduled:  ",
	},
	"editor_label_images": {
		"de": "Bilder:     ",
		"en": "Images:     ",
	},
	"editor_label_body": {
		"de": "Beitrag / Inhalt: ",
		"en": "Post / Body: ",
	},
	"editor_save": {
		"de": " [ SPEICHERN ] ",
		"en": " [ SAVE ] ",
	},
	"editor_cancel": {
		"de": " [ ABBRECHEN ] ",
		"en": " [ CANCEL ] ",
	},

	// Dashboard Content
	"dash_campaigns": {
		"de": "KAMPAGNEN",
		"en": "CAMPAIGNS",
	},
	"dash_next_up": {
		"de": "NÄCHSTE VERÖFFENTLICHUNGEN",
		"en": "NEXT UP",
	},
	"dash_stats": {
		"de": "STATISTIKEN",
		"en": "STATS",
	},
	"dash_platforms": {
		"de": "PLATTFORMEN",
		"en": "PLATFORMS",
	},
	"dash_connected": {
		"de": "Verbunden ✓",
		"en": "Connected ✓",
	},
	"dash_not_auth": {
		"de": "Nicht verbunden (Enter drücken)",
		"en": "Not connected (Press Enter)",
	},
	"dash_no_campaigns": {
		"de": "Keine Kampagnen gefunden.",
		"en": "No campaigns found.",
	},
	"dash_no_schedules": {
		"de": "Keine geplanten Beiträge.",
		"en": "No scheduled posts.",
	},
	"dash_campaign_format": {
		"de": "   %d Beiträge (%d gepostet, %d geplant)\n",
		"en": "   %d posts (%d posted, %d scheduled)\n",
	},

	// Settings Options
	"settings_ai_provider": {
		"de": "KI-Provider",
		"en": "AI Provider",
	},
	"settings_ai_model": {
		"de": "KI-Modell   ",
		"en": "AI Model    ",
	},
	"settings_dry_run": {
		"de": "Dry Run      ",
		"en": "Dry Run      ",
	},
	"settings_language": {
		"de": "Sprache      ",
		"en": "Language     ",
	},
	"settings_license": {
		"de": "Lizenztyp    ",
		"en": "License Type ",
	},
	"settings_auth_twitter": {
		"de": "Twitter/X    ",
		"en": "Twitter/X    ",
	},
	"settings_auth_linkedin": {
		"de": "LinkedIn     ",
		"en": "LinkedIn     ",
	},
	"settings_auth_threads": {
		"de": "Threads      ",
		"en": "Threads      ",
	},
	"settings_auth_mastodon": {
		"de": "Mastodon     ",
		"en": "Mastodon     ",
	},
	"settings_config_export": {
		"de": "Backup Exp.  ",
		"en": "Backup Exp.  ",
	},
	"settings_config_import": {
		"de": "Backup Imp.  ",
		"en": "Backup Imp.  ",
	},
	"license_core": {
		"de": "Core (Gratis)",
		"en": "Core (Free)",
	},
	"license_pro": {
		"de": "Pro (Aktiv ✅)",
		"en": "Pro (Active ✅)",
	},
	"settings_run_action": {
		"de": "Ausführen (Enter drücken)",
		"en": "Execute (Press Enter)",
	},
	"settings_help_footer": {
		"de": "←/→ / enter: Werte ändern / Verbinden  ·  Änderungen werden sofort gespeichert.\nPro-Lizenz über CLI aktivieren: postctl config set license_key <key>",
		"en": "←/→ / enter: Change values / Connect  ·  Changes are saved instantly.\nActivate Pro license via CLI: postctl config set license_key <key>",
	},

	// Posts View
	"posts_header_filtered": {
		"de": "BEITRÄGE (Filter: Kampagne = %s) [ESC zum Zurücksetzen]",
		"en": "POSTS (Filter: Campaign = %s) [ESC to clear]",
	},
	"posts_none_found": {
		"de": "Keine Beiträge gefunden. Verwende 'postctl import <pfad>' zum Importieren.\n",
		"en": "No posts found. Use 'postctl import <path>' to import markdown posts.\n",
	},
	"posts_none_found_campaign": {
		"de": "Keine Beiträge für Kampagne %q gefunden.\n",
		"en": "No posts found for campaign %q.\n",
	},
	"meta_thread": {
		"de": "Thread · %d Tweets",
		"en": "thread · %d tweets",
	},
	"meta_single": {
		"de": "Einzelbeitrag",
		"en": "single",
	},
	"meta_images": {
		"de": "📎 %d Bilder",
		"en": "📎 %d images",
	},

	// History View
	"history_none_found": {
		"de": "Kein Verlauf vorhanden.\n",
		"en": "No posting history found.\n",
	},

	// Help / Keyboard Guide View
	"help_title": {
		"de": "TASTATURBEFEHLE (HILFE)",
		"en": "KEYBOARD HELP",
	},
	"help_tab": {
		"de": "Nächster Tab",
		"en": "Next Tab",
	},
	"help_shifttab": {
		"de": "Vorheriger Tab",
		"en": "Previous Tab",
	},
	"help_up": {
		"de": "Nach oben navigieren",
		"en": "Move Up",
	},
	"help_down": {
		"de": "Nach unten navigieren",
		"en": "Move Down",
	},
	"help_enter": {
		"de": "Auswählen / Detailvorschau / Dashboard-Filter",
		"en": "Select / Open Preview / Filter by campaign",
	},
	"help_new_post": {
		"de": "Neuen Beitragsentwurf erstellen",
		"en": "Create a new post draft",
	},
	"help_edit_post": {
		"de": "Ausgewählten Entwurf bearbeiten",
		"en": "Edit selected post draft",
	},
	"help_import": {
		"de": "Beiträge aus Ordner importieren (Pausiert TUI)",
		"en": "Import posts from files/folders (pauses TUI)",
	},
	"help_delete": {
		"de": "Ausgewählten Beitrag löschen",
		"en": "Delete selected post",
	},
	"help_repurpose": {
		"de": "Inhalt via KI für andere Plattformen umschreiben",
		"en": "Repurpose selected post via AI to other platforms",
	},
	"help_esc": {
		"de": "Vorschau schließen / Filter zurücksetzen",
		"en": "Close Preview / Clear filter",
	},
	"help_readme": {
		"de": "Handbuch (README.md Browser) öffnen",
		"en": "Open complete README documentation with TOC",
	},
	"help_toggle": {
		"de": "Schnellhilfe ein-/ausblenden",
		"en": "Toggle Quick Help",
	},
	"help_quit": {
		"de": "Anwendung beenden",
		"en": "Quit application",
	},

	// Editor helper comments
	"editor_helper_title_twitter": {
		"de": " postctl Editor-Hilfe (Twitter / X)\n ==================================\n\n",
		"en": " postctl Editor Help (Twitter / X)\n ==================================\n\n",
	},
	"editor_helper_ruler_twitter": {
		"de": " [Zeichen-Lineal (Max. 280 Zeichen pro Tweet)]\n",
		"en": " [Character Ruler (Max 280 characters per tweet)]\n",
	},
	"editor_helper_status_thread": {
		"de": " Aktueller Thread-Status:\n",
		"en": " Current Thread Status:\n",
	},
	"editor_helper_tweet_format": {
		"de": "   Tweet %d: %d Zeichen (%d verbleibend) [%s]\n",
		"en": "   Tweet %d: %d chars (%d remaining) [%s]\n",
	},
	"editor_helper_status_single": {
		"de": "   Länge: %d Zeichen (%d verbleibend) [%s]\n",
		"en": "   Length: %d chars (%d remaining) [%s]\n",
	},
	"editor_helper_status_other": {
		"de": "   Länge: %d Zeichen\n",
		"en": "   Length: %d chars\n",
	},
	"editor_helper_note_strip": {
		"de": "\n HINWEIS: Dieser Hilfeblock wird beim Speichern automatisch gelöscht.\n Schreibe deinen Beitrag unter diesem Kommentar:\n",
		"en": "\n NOTE: This helper block will be stripped out automatically upon save.\n Write your post content below this comment:\n",
	},
	"editor_helper_title_other": {
		"de": " postctl Editor-Hilfe (%s)\n ==================================\n\n",
		"en": " postctl Editor Help (%s)\n ==================================\n\n",
	},
}
