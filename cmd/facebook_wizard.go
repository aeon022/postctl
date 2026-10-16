package cmd

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/aeon022/postctl/internal/config"
	"golang.org/x/term"
)

// wl picks the German or English text according to the configured language.
func wl(de, en string) string {
	if strings.EqualFold(config.ActiveConfig.Defaults.Language, "de") {
		return de
	}
	return en
}

// normalizePageRef turns what people paste for "my page" into what the Graph API accepts:
// a facebook.com URL, @name, vanity name or numeric id → vanity name or id.
func normalizePageRef(s string) string {
	s = strings.TrimSpace(strings.Trim(s, `"'`))
	s = strings.TrimPrefix(s, "@")
	if !strings.Contains(s, "facebook.com") && !strings.Contains(s, "fb.com") && !strings.Contains(s, "/") {
		return s
	}
	if !strings.HasPrefix(s, "http") {
		s = "https://" + s
	}
	u, err := url.Parse(s)
	if err != nil {
		return strings.TrimSpace(s)
	}
	if id := u.Query().Get("id"); id != "" { // profile.php?id=123
		return id
	}
	path := strings.Trim(u.Path, "/")
	if i := strings.Index(path, "/"); i >= 0 { // pages/Name/123 → keep the numeric id if present
		parts := strings.Split(path, "/")
		for _, p := range parts {
			if p != "" && strings.Trim(p, "0123456789") == "" {
				return p
			}
		}
		return parts[0]
	}
	return path
}

// readSecret reads a secret without echo when stdin is a terminal.
func readSecret(prompt string, reader *bufio.Reader) string {
	fmt.Print(prompt)
	if fd := int(os.Stdin.Fd()); term.IsTerminal(fd) {
		b, err := term.ReadPassword(fd)
		fmt.Println()
		if err == nil {
			return strings.TrimSpace(strings.Trim(string(b), `"'`))
		}
	}
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(strings.Trim(line, `"'`))
}

func readLine(prompt string, reader *bufio.Reader) string {
	fmt.Print(prompt)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

// facebookWizard connects a Facebook Page. Facebook only lets people log in, never a Page,
// so the user pastes a token from the Graph API Explorer; postctl turns it into a permanent
// Page token at login (see FacebookPlatform.useStaticPageToken).
func facebookWizard(reader *bufio.Reader) {
	fmt.Println(wl(
		"Gepostet wird als Seite. Meta erlaubt aber nur Anmeldungen als Benutzer – daher so:\n",
		"Posts go out as the Page. Meta only allows logging in as a user, so it works like this:\n"))
	fmt.Println(wl(
		"  1. developers.facebook.com → Meine Apps → deine App öffnen (oder neu erstellen mit dem\n"+
			"     Anwendungsfall „Alles auf deiner Seite verwalten“).\n"+
			"  2. App-Einstellungen → Allgemein: App-ID und App-Secret notieren.\n"+
			"  3. developers.facebook.com/tools/explorer öffnen, oben deine App wählen, Berechtigungen\n"+
			"     pages_show_list, pages_manage_posts und pages_read_engagement hinzufügen,\n"+
			"     „Generate Access Token“ klicken und im Dialog DEINE SEITE auswählen.\n"+
			"  4. Den angezeigten Access Token unten einfügen. Ein Benutzer-Token genügt: postctl macht\n"+
			"     daraus selbst einen dauerhaften Seiten-Token und speichert den Eingabe-Token nicht.\n",
		"  1. developers.facebook.com → My Apps → open your app (or create one with the use case\n"+
			"     \"Manage everything on your Page\").\n"+
			"  2. App settings → Basic: note the App ID and App Secret.\n"+
			"  3. Open developers.facebook.com/tools/explorer, pick your app at the top, add the permissions\n"+
			"     pages_show_list, pages_manage_posts and pages_read_engagement, click\n"+
			"     \"Generate Access Token\" and select YOUR PAGE in the dialog.\n"+
			"  4. Paste the access token shown there below. A user token is enough: postctl turns it into\n"+
			"     a permanent Page token itself and does not keep the token you pasted.\n"))

	page := normalizePageRef(readLine(wl(
		"➔ Deine Facebook-Seite (Adresse, Name oder ID, z. B. facebook.com/meineseite): ",
		"➔ Your Facebook Page (URL, name or ID, e.g. facebook.com/mypage): "), reader))
	appID := readLine(wl("➔ App-ID: ", "➔ App ID: "), reader)
	appSecret := readSecret(wl("➔ App-Secret (Eingabe bleibt unsichtbar): ", "➔ App Secret (input stays hidden): "), reader)
	if appID == "" || appSecret == "" {
		fmt.Println(wl(
			"  Hinweis: Ohne App-ID und App-Secret hält der Zugang nur etwa eine Stunde.",
			"  Note: without App ID and App Secret the connection only lasts about an hour."))
	}
	token := readSecret(wl(
		"➔ Access Token aus dem Graph API Explorer (leer lassen für den Browser-Login, braucht eine\n   Weiterleitungs-URL https://localhost:8753/callback in der App; Eingabe unsichtbar): ",
		"➔ Access token from the Graph API Explorer (leave empty for browser login, which needs the redirect\n   URL https://localhost:8753/callback in the app; input stays hidden): "), reader)

	fb := &config.ActiveConfig.Facebook
	if page != "" {
		fb.PageID = page
	}
	if appID != "" {
		fb.AppID = appID
	}
	if appSecret != "" {
		fb.AppSecret = appSecret
	}
	if token != "" {
		fb.PageToken = token
	}
	fmt.Println()
	fmt.Println(wl("Als Nächstes die Verbindung herstellen: postctl auth facebook (im TUI geschieht das automatisch).",
		"Next, connect: postctl auth facebook (the TUI does this automatically)."))
}
