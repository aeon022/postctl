# postctl — Social Media Management from your Terminal

`postctl` is a lightweight, offline-first CLI and TUI tool written in Go to manage and schedule social media posts directly from Markdown files. It is built for developer workflows, Git versioning, and AI-assisted post conversion.

*Scrollen Sie nach unten für die deutsche Version.*

---

## English Version

### 🚀 Features (2026 Standards)

1. **Markdown-First Workflow**: Write your posts as Markdown. Frontmatter fields define platforms, campaigns, schedule times, and image paths.
2. **Terminal User Interface (TUI)**: Interactive Bubble Tea-based dashboard to manage campaigns, posts, history, and settings.
3. **Modern API Integrations**:
   * **Twitter/X**: Post tweets and threads via OAuth 2.0 PKCE (requires a paid API tier / Prepaid credits in 2026).
   * **LinkedIn**: Authenticate via modern **OpenID Connect (OIDC)** (Scopes: `openid`, `profile`, `w_member_social`).
   * **Threads (Meta)**: Official Threads API integration with secure HTTPS callbacks (`https://localhost:8753/callback`).
4. **Encrypted Backup & Multi-Device Sync (AES-256-GCM)**: Synchronize your database (tokens, history, drafts) across devices securely and for free.

---

### 💾 Posting & Image Import/Export (Deep Dive)

#### 1. Importing Posts (`postctl import <path>`)
When you run `./postctl import [file.md or directory]`, the CLI performs the following steps:
* **Scan & Parse:** It parses all `.md` and `.markdown` files. The frontmatter properties are converted into SQLite database models.
* **Validation:** Before saving, `postctl` validates:
  * **Char Limits:** Twitter/X tweets are verified to be under 280 characters.
  * **Image Existence:** Every referenced image path is checked on your local disk. If any validation fails, the entire import is aborted.
* **Save:** Validated posts are saved to the local SQLite database (`postctl.db`) as `draft` or `scheduled`.

#### 2. Image Path Resolution
When you reference an image in your frontmatter (e.g., `images: ["logo.png"]`), `postctl` searches for it in this exact order:
1. **Absolute Path:** Check the literal absolute path (e.g., `/Users/gweiher/Images/logo.png`).
2. **Relative to Markdown File:** Check the directory containing the imported Markdown file (best practice for Git repositories).
3. **Relative to Working Directory (CWD):** Check the directory where you executed the command.
4. **Relative to configured Image Dir:** Check the path configured under `defaults.image_dir` in `config.yaml`.

#### 3. Image Exporting (Publishing to Platforms)
When a post containing images is published, `postctl` uploads the local images first:
* **LinkedIn (Native Host):** `postctl` registers the upload, performs a binary `PUT` request to LinkedIn's media servers, and attaches the resulting URN (e.g., `urn:li:digitalmediaAsset:...`) to the post.
* **Twitter/X (Native Host):** Uploads the image using the Twitter v1.1 Media Upload API to receive a `media_id`, which is then attached to the v2 tweet.
* **Threads (Meta HTTPS Requirement):** The Threads API *does not* allow binary uploads from local machines. It requires a **publicly accessible HTTPS URL** from which Meta downloads the image. 
  * *Production Tip:* For Threads, host your images on a public cloud storage (e.g., AWS S3, Cloudflare R2, Imgur) and use these URLs directly in the Markdown frontmatter.

---

### 🛠️ Installation & Setup
Run the interactive setup script:
```bash
chmod +x setup.sh
./setup.sh
```

---

### ⚙️ Settings & Auth in TUI
Run `./postctl tui` and navigate to the **Settings** tab:
* **Account Auth:** Highlight an account under **PLATFORM ACCOUNTS** and press **Enter** to start the OAuth login. The TUI pauses, opens your standard browser, and displays a status message upon completion.
* **Interactive Post Import:** In any main tab (Dashboard, Posts, Schedule, History), you can press **`i`** to trigger an interactive post import. The TUI will temporarily pause, clear the terminal, and prompt you for the Markdown file/directory path. **Tip: You can simply drag and drop the file or folder directly from your Finder into the terminal window.** The tool automatically cleans quote characters, validates the posts/images, imports them, and returns to the TUI.
* **Backup & Sync:** Go to **BACKUP & SYNC** at the bottom, select `Backup Exp.` (Export) or `Backup Imp.` (Import), and press **Enter** to enter your master password.
  * CLI Commands: `./postctl config export -o backup.bin` and `./postctl config import -f backup.bin`.

---
---

## Deutsche Version

### 🚀 Features (Stand 2026)

1. **Markdown-First Workflow**: Schreibe Beiträge als Markdown. Frontmatter-Felder definieren Plattformen, Kampagnen, Zeiten und Bildpfade.
2. **Terminal User Interface (TUI)**: Interaktives Bubble-Tea-Dashboard zur Verwaltung von Kampagnen, Beiträgen, Historie und Einstellungen.
3. **Moderne API-Integrationen**:
   * **Twitter/X**: Veröffentliche Tweets/Threads über OAuth 2.0 PKCE (erfordert im Jahr 2026 ein kostenpflichtiges API-Tier oder Prepaid-Credits).
   * **LinkedIn**: Authentifizierung über den modernen **OpenID Connect (OIDC)**-Standard (Scopes: `openid`, `profile`, `w_member_social`).
   * **Threads (Meta)**: Offizielle Threads API mit sicherem HTTPS-Callback (`https://localhost:8753/callback`).
4. **Backup & Multi-Device Sync (AES-256-GCM)**: Synchronisiere deine Datenbank (Tokens, Historie, Entwürfe) sicher und kostenlos zwischen Geräten.

---

### 💾 Import/Export von Beiträgen & Bildern (Detailerklärung)

#### 1. Importieren von Beiträgen (`postctl import <pfad>`)
Wenn du `./postctl import [datei.md oder ordner/]` ausführst, führt das CLI folgende Schritte aus:
* **Scan & Parse:** Alle `.md` und `.markdown`-Dateien werden analysiert. Die Frontmatter-Eigenschaften werden in SQLite-Datenbankmodelle umgewandelt.
* **Validierung:** Vor dem Speichern prüft `postctl`:
  * **Zeichenlimits:** Twitter/X-Tweets werden auf die Grenze von 280 Zeichen validiert.
  * **Bild-Existenz:** Jeder verlinkte Bildpfad wird auf der lokalen Festplatte geprüft. Schlägt die Validierung auch nur einer Datei fehl, wird der gesamte Import abgebrochen.
* **Speichern:** Validierte Beiträge werden in der lokalen SQLite-Datenbank (`postctl.db`) als `draft` (Entwurf) oder `scheduled` (Geplant) abgelegt.

#### 2. Auflösung von Bildpfaden
Wenn du ein Bild im Frontmatter angibst (z. B. `images: ["logo.png"]`), sucht `postctl` dieses in folgender Reihenfolge:
1. **Absoluter Pfad:** Direkte Prüfung des absoluten Pfads (z. B. `/Users/gweiher/Bilder/logo.png`).
2. **Relativ zur Markdown-Datei:** Prüfung im Ordner der importierten Markdown-Datei (Best Practice für Git-Repositories).
3. **Relativ zum Arbeitsverzeichnis (CWD):** Prüfung im aktuellen Ordner, in dem der Befehl ausgeführt wird.
4. **Relativ zum Standard-Bildordner:** Prüfung im unter `defaults.image_dir` in der `config.yaml` definierten Pfad.

#### 3. Bild-Export (Veröffentlichung auf den Plattformen)
Wird ein Beitrag mit Bildern veröffentlicht, lädt `postctl` die lokalen Bilder zuerst hoch:
* **LinkedIn (Nativer Upload):** `postctl` registriert den Upload, führt einen binären `PUT`-Request auf die LinkedIn-Server aus und hängt den erhaltenen URN (z. B. `urn:li:digitalmediaAsset:...`) an das Posting an.
* **Twitter/X (Nativer Upload):** Das Bild wird über die v1.1 Media Upload API von X hochgeladen, um eine `media_id` zu erhalten. Diese wird an den v2-Tweet angehängt.
* **Threads (Meta HTTPS-Pflicht):** Die Threads API erlaubt *keine* binären Uploads von lokalen Rechnern. Sie erfordert eine **öffentlich erreichbare HTTPS-URL**, von der Meta das Bild herunterlädt.
  * *Praxis-Tipp:* Hoste deine Bilder für Threads auf einem öffentlichen Cloud-Speicher (z. B. AWS S3, Cloudflare R2, Imgur) und verwende diese URLs direkt im Markdown-Frontmatter.

---

### 🛠️ Installation & Setup
Starte das interaktive Setup-Skript:
```bash
chmod +x setup.sh
./setup.sh
```

---

### ⚙️ Einstellungen & Auth in der TUI
Führe `./postctl tui` aus und wechsle in den **Settings**-Tab:
* **Account Auth:** Wähle einen Account unter **PLATFORM ACCOUNTS** und drücke **Enter**. Die TUI pausiert, öffnet deinen Browser und aktualisiert den Status nach erfolgreichem Login.
* **Interaktiver Beitrags-Import:** In jedem Haupt-Tab (Dashboard, Posts, Schedule, History) kannst du die Taste **`i`** drücken, um einen interaktiven Import zu starten. Die TUI pausiert kurz, leert das Terminal und bittet dich um den Pfad zur Markdown-Datei oder zum Ordner. **Tipp: Du kannst die Datei oder den Ordner einfach per Drag & Drop aus dem Finder direkt in das Terminalfenster ziehen.** Das Tool entfernt automatisch störende Anführungszeichen, validiert die Beiträge/Bilder und kehrt direkt wieder zur TUI zurück.
* **Backup & Sync:** Wähle unten im Bereich **BACKUP & SYNC** entweder `Backup Exp.` (Export) oder `Backup Imp.` (Import) und drücke **Enter**, um dein Master-Passwort einzugeben.
  * CLI-Befehle: `./postctl config export -o backup.bin` und `./postctl config import -f backup.bin`.
