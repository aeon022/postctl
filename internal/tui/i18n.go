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
		"de": "◷ POSTS",
		"en": "◷ POSTS",
	},
	"tab_schedule": {
		"de": "↺ QUEUE",
		"en": "↺ QUEUE",
	},
	"tab_history": {
		"de": "▤ HISTORY",
		"en": "▤ HISTORY",
	},
	"tab_analytics": {
		"de": "⇗ STATS",
		"en": "⇗ STATS",
	},
	"tab_settings": {
		"de": "⚙ SETTINGS",
		"en": "⚙ SETTINGS",
	},
	"tab_logs": {
		"de": "▤ LOGS",
		"en": "▤ LOGS",
	},
	// Unified chrome: footer hints
	"hint_back":     {"de": "zurück", "en": "back"},
	"hint_scroll":   {"de": "scrollen", "en": "scroll"},
	"hint_save":     {"de": "speichern", "en": "save"},
	"hint_cancel":   {"de": "abbrechen", "en": "cancel"},
	"hint_next":     {"de": "nächstes Feld", "en": "next field"},
	"hint_prev":     {"de": "voriges Feld", "en": "prev field"},
	"hint_calendar": {"de": "Kalender", "en": "calendar"},
	"hint_nvim":     {"de": "Neovim", "en": "nvim"},
	"hint_day":      {"de": "Tag", "en": "day"},
	"hint_month":    {"de": "Monat", "en": "month"},
	"hint_pick":     {"de": "wählen", "en": "pick"},
	"hint_change":   {"de": "ändern", "en": "change"},
	"hint_reset":    {"de": "zurücksetzen", "en": "reset"},
	"hint_toc":      {"de": "Inhalt", "en": "contents"},
	"hint_jump":     {"de": "springen", "en": "jump"},
	"hint_move":     {"de": "bewegen", "en": "move"},
	"hint_reload":   {"de": "neu laden", "en": "reload"},

	// Unified chrome: panels and labels
	"panel_preview":          {"de": "Vorschau", "en": "Preview"},
	"panel_thread":           {"de": "Thread", "en": "Thread"},
	"panel_images":           {"de": "Bilder", "en": "Images"},
	"panel_calendar":         {"de": "Kalender", "en": "Calendar"},
	"panel_readme":           {"de": "Handbuch", "en": "Manual"},
	"panel_toc":              {"de": "Inhaltsverzeichnis", "en": "Table of contents"},
	"panel_slots":            {"de": "Queue-Slots bearbeiten", "en": "Edit queue slots"},
	"panel_logs":             {"de": "Hintergrund-Logs", "en": "Background logs"},
	"panel_history":          {"de": "Verlauf", "en": "Posting history"},
	"panel_settings":         {"de": "Einstellungen", "en": "Settings"},
	"panel_hdetail":          {"de": "Verlaufseintrag", "en": "History entry"},
	"panel_output":           {"de": "Ausgabe / Fehlermeldung", "en": "Output / error message"},
	"panel_summary":          {"de": "Zusammenfassung", "en": "Summary"},
	"panel_dist":             {"de": "Interaktions-Verteilung", "en": "Interaction split"},
	"panel_trend":            {"de": "Engagement-Trend (30 Tage)", "en": "Engagement trend (30 days)"},
	"panel_details":          {"de": "Plattform-Details", "en": "Platform details"},
	"detail_campaign":        {"de": "Kampagne", "en": "Campaign"},
	"detail_type":            {"de": "Typ", "en": "Type"},
	"detail_status":          {"de": "Status", "en": "Status"},
	"detail_sched_at":        {"de": "geplant für", "en": "scheduled for"},
	"detail_error":           {"de": "Fehler", "en": "Error"},
	"detail_file":            {"de": "Datei", "en": "File"},
	"detail_no_image":        {"de": "kein Bild", "en": "no image"},
	"detail_reply":           {"de": "Antwort", "en": "reply"},
	"detail_too_long":        {"de": "zu lang", "en": "too long"},
	"hd_timestamp":           {"de": "Zeitpunkt", "en": "Timestamp"},
	"hd_post_id":             {"de": "Post-ID", "en": "Post ID"},
	"hd_platform_id":         {"de": "Plattform-ID", "en": "Platform ID"},
	"hd_no_error":            {"de": "(Kein Fehler – Post erfolgreich veröffentlicht)", "en": "(No error recorded – post published successfully)"},
	"logs_empty":             {"de": "Keine Logs vorhanden. Hintergrund-Aktivitäten werden hier protokolliert.", "en": "No logs yet. Background activity is recorded here."},
	"an_loading":             {"de": "Lade Social Analytics & Engagement-Daten…", "en": "Loading social analytics & engagement data…"},
	"an_no_data":             {"de": "Keine Daten geladen.", "en": "No data loaded."},
	"an_load_error":          {"de": "Fehler beim Laden: %v", "en": "Failed to load: %v"},
	"an_info":                {"de": "Für Twitter/X, LinkedIn und Threads sind eigene API-Zugangsdaten nötig, um echte Interaktionsdaten abzurufen; sonst werden 0 angezeigt. Mastodon und Bluesky nutzen Live-Daten.", "en": "Twitter/X, LinkedIn and Threads need your own API credentials for real engagement data; otherwise 0 is shown. Mastodon and Bluesky use live data."},
	"an_posts":               {"de": "Veröffentlichte Beiträge", "en": "Published posts"},
	"an_likes":               {"de": "Likes", "en": "Likes"},
	"an_shares":              {"de": "Shares", "en": "Shares"},
	"an_comments":            {"de": "Kommentare", "en": "Comments"},
	"an_impressions":         {"de": "Impressions", "en": "Impressions"},
	"an_no_details":          {"de": "Keine Beitragsdetails vorhanden.", "en": "No post details yet."},
	"an_col_platform":        {"de": "Plattform", "en": "Platform"},
	"an_axis_start":          {"de": "vor 30 Tagen", "en": "30 days ago"},
	"an_axis_end":            {"de": "heute", "en": "today"},
	"settings_sec_platforms": {"de": "Plattform-Konten", "en": "Platform accounts"},
	"settings_sec_backup":    {"de": "Backup & Sync", "en": "Backup & sync"},
	"settings_sec_slots":     {"de": "Scheduler-Queue-Slots", "en": "Scheduler queue slots"},
	"settings_none":          {"de": "keine", "en": "none"},
	"slots_help":             {"de": "Slots kommagetrennt eingeben (Format: 'Tag HH:MM', z. B. 'Mon 09:00, Wed 14:00'):", "en": "Enter slots comma-separated (format: 'Day HH:MM', e.g. 'Mon 09:00, Wed 14:00'):"},
	"readme_back_to_top":     {"de": "▲ t: zurück zum Inhaltsverzeichnis", "en": "▲ t: back to contents"},
	"editor_tip_now":         {"de": "(Tipp: 'now' / 'jetzt' für sofortigen Versand oder ctrl+d)", "en": "(Tip: type 'now' for immediate publication or press ctrl+d)"},
	"editor_preview_label":   {"de": "Vorschau:", "en": "Preview:"},
	"cal_weekdays":           {"de": "Mo Di Mi Do Fr Sa So", "en": "Mo Tu We Th Fr Sa Su"},
	"picker_title":           {"de": "Profil wählen", "en": "Choose a profile"},
	"picker_default":         {"de": "default", "en": "default"},

	"hint_close": {
		"de": "schließen",
		"en": "close",
	},
	"hint_quit": {
		"de": "beenden",
		"en": "quit",
	},
	"hint_open": {
		"de": "öffnen",
		"en": "open",
	},
	"hint_new": {
		"de": "neu",
		"en": "new",
	},
	"hint_help": {
		"de": "Hilfe",
		"en": "help",
	},
	"hint_edit": {
		"de": "bearbeiten",
		"en": "edit",
	},
	"hint_delete": {
		"de": "löschen",
		"en": "delete",
	},
	"hint_schedule": {
		"de": "einplanen",
		"en": "schedule",
	},
	"hint_post": {
		"de": "jetzt posten",
		"en": "post now",
	},
	"hint_select": {
		"de": "markieren",
		"en": "select",
	},
	"hint_filter": {
		"de": "Filter",
		"en": "filter",
	},
	"hint_repurpose": {
		"de": "umschreiben",
		"en": "repurpose",
	},
	"hint_import": {
		"de": "Import",
		"en": "import",
	},
	"hint_tab": {
		"de": "nächster Tab",
		"en": "next tab",
	},
	"hint_manual": {
		"de": "Handbuch",
		"en": "manual",
	},
	"hint_clear": {
		"de": "Filter löschen",
		"en": "clear filter",
	},
	"hint_export": {
		"de": "exportieren",
		"en": "export",
	},
	"dash_campaign_counts": {
		"de": "%d Beiträge · %d gepostet · %d geplant",
		"en": "%d posts · %d posted · %d scheduled",
	},
	"dash_connect_hint": {
		"de": "Plattformen im Tab SETTINGS verbinden",
		"en": "Connect platforms in the SETTINGS tab",
	},
	"posts_empty_hint": {
		"de": "Mit n einen Beitrag anlegen oder mit i importieren",
		"en": "Press n to create a post, or i to import",
	},
	"queue_empty_hint": {
		"de": "Mit s einen Beitrag einplanen",
		"en": "Press s on a post to schedule it",
	},
	"header_logs": {
		"de": "HINTERGRUND-LOGS (SYSTEMVERLAUF)",
		"en": "BACKGROUND LOGS (SYSTEM ACTIVITY)",
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
	"editor_images_help": {
		"de": "     (Tipp: Pfad eintragen oder Bild per Finder Drag & Drop ins Terminal ziehen)",
		"en": "     (Tip: Enter path or drag & drop image from Finder into terminal)",
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
	// Dashboard stats labels (value follows directly, so each ends in padding)
	"stats_posted": {
		"de": "Veröffentlicht: ",
		"en": "Posted:       ",
	},
	"stats_scheduled": {
		"de": "Geplant:        ",
		"en": "Scheduled:    ",
	},
	"stats_drafts": {
		"de": "Entwürfe:       ",
		"en": "Drafts:       ",
	},
	"stats_failed": {
		"de": "Fehlgeschlagen: ",
		"en": "Failed:       ",
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
		"de": "Nicht verbunden",
		"en": "Not connected",
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
	"settings_auto_publish": {
		"de": "Auto-Publish ",
		"en": "Auto-Publish ",
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
	"settings_auth_bluesky": {
		"de": "Bluesky      ",
		"en": "Bluesky      ",
	},
	"settings_auth_facebook": {
		"de": "Facebook     ",
		"en": "Facebook     ",
	},
	"settings_auth_telegram": {
		"de": "Telegram     ",
		"en": "Telegram     ",
	},
	"settings_auth_discord": {
		"de": "Discord      ",
		"en": "Discord      ",
	},
	"settings_auth_devto": {
		"de": "Dev.to       ",
		"en": "Dev.to       ",
	},
	"settings_auth_reddit": {
		"de": "Reddit       ",
		"en": "Reddit       ",
	},
	"settings_auth_hashnode": {
		"de": "Hashnode     ",
		"en": "Hashnode     ",
	},
	"settings_auth_medium": {
		"de": "Medium       ",
		"en": "Medium       ",
	},
	"settings_config_export": {
		"de": "Backup Exp.  ",
		"en": "Backup Exp.  ",
	},
	"settings_config_import": {
		"de": "Backup Imp.  ",
		"en": "Backup Imp.  ",
	},
	"settings_edit_slots": {
		"de": "Queue bearb. ",
		"en": "Edit Queue   ",
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
		"de": "←/→ / enter: Ändern / Verbinden  ·  d: Reset/Disconnect  ·  Sofort gespeichert.\nSupport postctl (Spenden): https://buy.polar.sh/polar_cl_wQUHDTz9e8ZLhWS59jxxRaBGiNjdaWIHpwt4T4fyzhW\nPro-Lizenz kaufen (37% Launch-Special, Code postctl2026): https://buy.polar.sh/polar_cl_ookKcZLP6IjbeYc9inqKu3734J9WA8ssy5cL90jmJSs\nAktivieren: postctl config set license_key <key>",
		"en": "←/→ / enter: Change / Connect  ·  d: Reset/Disconnect  ·  Saved instantly.\nSupport postctl (Donate): https://buy.polar.sh/polar_cl_wQUHDTz9e8ZLhWS59jxxRaBGiNjdaWIHpwt4T4fyzhW\nBuy Pro (37% launch special, code postctl2026): https://buy.polar.sh/polar_cl_ookKcZLP6IjbeYc9inqKu3734J9WA8ssy5cL90jmJSs\nActivate: postctl config set license_key <key>",
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
	"help_filter": {
		"de": "Kampagnen-Filter umschalten",
		"en": "Toggle campaign filter",
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
	"help_export": {
		"de": "History exportieren (x)",
		"en": "Export history as JSON (x)",
	},
	"help_schedule": {
		"de": "Beitrag in Warteschlange einplanen",
		"en": "Schedule post (add to queue)",
	},
	"help_post": {
		"de": "Beitrag sofort veröffentlichen (p)",
		"en": "Publish post immediately (p)",
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
		"de": "\n HINWEIS: Dieser Hilfeblock wird beim Speichern automatisch gelöscht.\n Bilder hinzufügen:\n   - Einzelpost: Pfad im Feld 'images' oben eintragen oder Drag&Drop im TUI.\n   - Threads: Nutze inline <!-- image: /pfad/zum/bild.png --> (per Finder Drag&Drop ins Terminal ziehen).\n\n Schreibe deinen Beitrag unter diesem Kommentar:\n",
		"en": "\n NOTE: This helper block will be stripped out automatically upon save.\n Adding images:\n   - Single Post: Enter path in the 'images' field above or drag & drop in the TUI.\n   - Threads: Use inline <!-- image: /path/to/image.png --> (drag & drop from Finder into terminal).\n\n Write your post content below this comment:\n",
	},
	"editor_helper_title_other": {
		"de": " postctl Editor-Hilfe (%s)\n ==================================\n\n",
		"en": " postctl Editor Help (%s)\n ==================================\n\n",
	},
	"editor_twitter_thread_note": {
		"de": " (Nutze '---' für Thread-Teilung)",
		"en": " (Use '---' to split into thread)",
	},
	"editor_help_footer": {
		"de": "tab: Nächstes Feld  ·  shift+tab: Vorheriges Feld  ·  esc: Abbrechen  ·  ctrl+v: Neovim",
		"en": "tab: next field  ·  shift+tab: prev field  ·  esc: cancel  ·  ctrl+v: nvim",
	},
}
