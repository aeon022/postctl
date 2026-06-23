# postctl — Social Media Management from your Terminal

`postctl` ist ein schlankes, offline-first CLI- und TUI-Tool in Go, um Social-Media-Beiträge direkt aus Markdown-Dateien heraus zu verwalten und auf mehreren Plattformen zu veröffentlichen. Entwickelt für Entwickler-Workflows, Git-Versionierung und KI-gestützte Beitrags-Konvertierung.

---

## 🚀 Features (Stand 2026)

1. **Markdown-First Workflow**: Schreibe deine Beiträge als Markdown. Frontmatter-Felder definieren Veröffentlichungsdaten, Kampagnen, Tags und Medien-Pfade.
2. **Terminal User Interface (TUI)**: Ein interaktives, Bubbletea-basiertes Dashboard zur Verwaltung von Kampagnen, Beiträgen, Historie und Einstellungen direkt im Terminal.
3. **Multi-Plattform-Posting (Aktuelle APIs von 2026)**:
   * **Twitter/X**: Veröffentlichen von Tweets und Threads über OAuth 2.0 PKCE (erfordert ein aktives kostenpflichtiges API-Tier oder Prepaid-Credits im Developer Portal).
   * **LinkedIn**: Authentifizierung über den modernen **OpenID Connect (OIDC)**-Standard (Scopes: `openid`, `profile`, `w_member_social`).
   * **Threads (Meta)**: Offizielle Threads API Integration mit sicherem HTTPS-Callback (`https://localhost:8753/callback`).
4. **Backup & Multi-Device Sync (AES-256-GCM)**: Synchronisiere deine SQLite-Datenbank (Historie, Tokens, Scheduled Posts) und Konfiguration verschlüsselt und kostenlos zwischen deinen Geräten.

---

## 🛠️ Installation & Setup

Führe das mitgelieferte interaktive Setup-Skript aus:
```bash
chmod +x setup.sh
./setup.sh
```
Das Skript prüft die Go-Installation, installiert alle Bubbletea- und SQLite-Abhängigkeiten und baut das lokale Binary `./postctl`.

---

## ⚙️ Einstellungen & Authentifizierung in der TUI

Du kannst die Konfiguration und die Authentifizierung deiner Social-Media-Accounts direkt im **Settings**-Tab der Terminal-UI vornehmen:
```bash
./postctl tui
```
* **Plattformen verbinden**: Navigiere im Settings-Tab mit den Pfeiltasten in den Bereich **PLATFORM ACCOUNTS** und drücke **Enter** auf einer Plattform, um den OAuth-Login zu starten. Die TUI pausiert und wartet, während sich dein Browser zur Anmeldung öffnet. Nach erfolgreichem Login aktualisiert sich die TUI automatisch zu `Verbunden ✓`.
* **OAuth HTTPS-Hinweis für Threads**: Für Threads startet `postctl` einen lokalen HTTPS-Server mit selbstsigniertem Zertifikat auf Port 8753. Bypasse die Browserwarnung im OAuth-Schritt mit *„Erweitert“ ➔ „Weiter zu localhost“*, um das Token abzufangen.

---

## 💾 Synchronisation & Backups (Verschlüsselt)

Um deine lokalen SQLite-Daten (Scheduled Posts, Tokens, Historie) und Konfigurationswerte auf andere Geräte zu übertragen, nutzt `postctl` ein sicheres Backup-System mit **AES-256-GCM-Verschlüsselung**.

### Über die TUI (Settings-Tab):
1. Navigiere ganz unten zum Bereich **BACKUP & SYNC**.
2. Wähle **`Backup Exp.`** (Export) oder **`Backup Imp.`** (Import) und drücke **Enter**.
3. Die TUI pausiert kurz und fragt dich im Terminal nach deinem Master-Passwort für die Ver-/Entschlüsselung. Danach kehrst du direkt zur TUI zurück.

### Über die CLI:
* **Exportieren:**
  ```bash
  ./postctl config export -o postctl_backup.bin
  ```
* **Importieren:**
  ```bash
  ./postctl config import -f postctl_backup.bin
  ```
Die exportierte Datei `postctl_backup.bin` kann über jeden beliebigen Kanal (Git, iCloud, Dropbox, SSH) sicher auf andere Rechner übertragen werden, da die sensiblen API-Tokens verschlüsselt sind.

---

## 📖 API Setup Guides

Ausführliche Anleitungen zur Erstellung der Entwickler-Apps auf den jeweiligen Plattformen findest du in den Dokumenten im `docs/`-Ordner:
* [Twitter/X API Setup Guide](file:///Users/gweiher/Developing/Projects/postctl/docs/api-twitter.md)
* [LinkedIn API Setup Guide](file:///Users/gweiher/Developing/Projects/postctl/docs/api-linkedin.md)
* [Threads API Setup Guide](file:///Users/gweiher/Developing/Projects/postctl/docs/api-threads.md)
