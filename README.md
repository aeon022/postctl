# postctl — Social Media Management from your Terminal

`postctl` is a lightweight, offline-first CLI and TUI tool written in Go to manage and schedule social media posts directly from Markdown files. It is built for developer workflows, Git versioning, and AI-assisted post conversion.

*Scrollen Sie nach unten für die deutsche Version.*

---

## English Version

### 🚀 Features (2026 Standards)

1. **Markdown-First Workflow**: Write your posts as Markdown. Frontmatter fields define platforms, campaigns, schedule times, and image paths.
2. **Terminal User Interface (TUI)**: Interactive Bubble Tea-based dashboard to manage campaigns, posts, history, and settings.
3. **Modern API Integrations**:
   * **Twitter/X**: Post tweets and threads via the official OAuth 2.0 API (recommended, requires paid API credits) or a free but unofficial Cookie-based bypass (risk of account suspension).
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

For platform-specific setup details and authentication instructions, see the guides:
* 🐦 [Twitter/X API & Cookie Setup Guide](docs/api-twitter.md) (Official paid API setup or free unofficial cookie bypass, troubleshooting `empty tweet ID` errors)

---

### ⚙️ Settings & Auth in TUI
Run `./postctl tui` and navigate to the **Settings** tab:
* **Account Auth:** Highlight an account under **PLATFORM ACCOUNTS** and press **Enter** to start the OAuth login. The TUI pauses, opens your standard browser, and displays a status message upon completion.
* **Interactive Post Import:** In any main tab (Dashboard, Posts, Schedule, History), you can press **`i`** to trigger an interactive post import. The TUI will temporarily pause, clear the terminal, and prompt you for the Markdown file/directory path. **Tip: You can simply drag and drop the file or folder directly from your Finder into the terminal window.** The tool automatically cleans quote characters, validates the posts/images, imports them, and returns to the TUI.
* **Backup & Sync:** Go to **BACKUP & SYNC** at the bottom, select `Backup Exp.` (Export) or `Backup Imp.` (Import), and press **Enter** to enter your master password.
  * CLI Commands: `./postctl config export -o backup.bin` and `./postctl config import -f backup.bin`.

---

### 🕒 Running the Scheduler
Since `postctl` is a local-first application, scheduled posts are published using one of the following two modes:

1. **Interactive Mode (Automatic TUI Background Goroutine):**
   When the interactive TUI is open (`./postctl tui` or just `./postctl`), a background scheduler goroutine runs automatically in the background. It polls the database every 10 seconds and publishes any due posts. No separate setup or terminal tab is needed as long as the TUI is open.

2. **Headless Mode (Scheduler Daemon):**
   If you want to run `postctl` headless (e.g., on a server or without keeping the interactive terminal TUI open), you can start the scheduler daemon:
   ```bash
   ./postctl daemon
   ```
   To run the daemon silently in the background:
   ```bash
   nohup ./postctl daemon > daemon.log 2>&1 &
   ```

---

### 📝 Vim / Neovim External Editor Flow (Step-by-Step)

When editing or creating a post within the TUI, you can spawn your favorite terminal text editor (like Vim or Neovim) for a rich, distraction-free editing environment.

#### Step 1: Open the External Editor
1. In the TUI, highlight or select a post and press **`e`** to open the built-in edit form.
2. Press **`ctrl+v`** from any focused form field.
3. The TUI will suspend itself and open Neovim/Vim (it uses your `$EDITOR` environment variable, falling back to `nvim` or `vim`).

#### Step 2: Understand the Template Structure
The temporary file opened in Vim contains three main parts:
1. **Interactive Helper Block (`<!-- ... -->`)**:
   - Contains a character ruler showing the maximum length for the current platform (e.g., 280 for Twitter, 300 for Bluesky, 500 for Mastodon).
   - Shows live counts (at the time of launch) for single posts or thread segments (delimited by `---`).
   - *Note:* This entire HTML comment block is automatically removed upon saving.
2. **YAML Frontmatter Block**:
   - Defined between the `---` delimiters:
     ```yaml
     ---
     platform: twitter
     campaign: launch-2026
     schedule: 2026-06-25 15:00:00
     images: ["logo.png"]
     ---
     ```
   - **Bidirectional Sync:** You can edit these metadata values directly inside Vim! When you exit, `postctl` parses this YAML header and automatically updates the corresponding input fields back in the TUI form.
3. **Post Body Content**:
   - The actual content of your post starts directly below the closing `---` frontmatter delimiter.

#### Step 3: Write Your Post / Thread
- Write your post content.
- If you are writing a thread (supported on platforms like Twitter/X), separate each post using a single line with `---`.
- *Example:*
  ```text
  This is the first tweet of my thread.
  ---
  This is the second tweet of my thread.
  ```

#### Step 4: Save & Sync Back
1. Save and exit the editor by typing **`:wq`** or **`:x`** (or press **`ZZ`** in normal mode).
2. The TUI will resume instantly. The body text is updated, the helper comments are stripped, and the edited frontmatter values are synced back into the TUI fields.
3. Review your changes in the TUI and navigate to the **`Save`** button (or press `Enter` when focused) to write the changes to the database.

---
---

## Deutsche Version

### 🚀 Features (Stand 2026)

1. **Markdown-First Workflow**: Schreibe Beiträge als Markdown. Frontmatter-Felder definieren Plattformen, Kampagnen, Zeiten und Bildpfade.
2. **Terminal User Interface (TUI)**: Interaktives Bubble-Tea-Dashboard zur Verwaltung von Kampagnen, Beiträgen, Historie und Einstellungen.
3. **Moderne API-Integrationen**:
    * **Twitter/X**: Veröffentliche Tweets/Threads über die offizielle API (empfohlen, erfordert kostenpflichtige API-Credits) oder einen kostenlosen, aber inoffiziellen Cookie-Bypass (Risiko von Kontosperrung).
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

Für plattformspezifische Details und Zugangsdaten findest du hier die passenden Anleitungen:
* 🐦 [Twitter/X API & Cookie Setup Guide](docs/api-twitter.md) (Nutzung des kostenlosen Cookie-Bypasses oder der offiziellen API, sowie Fehlerbehebung für `empty tweet ID`)

---

### ⚙️ Einstellungen & Auth in der TUI
Führe `./postctl tui` aus und wechsle in den **Settings**-Tab:
* **Account Auth:** Wähle einen Account unter **PLATFORM ACCOUNTS** und drücke **Enter**. Die TUI pausiert, öffnet deinen Browser und aktualisiert den Status nach erfolgreichem Login.
* **Interaktiver Beitrags-Import:** In jedem Haupt-Tab (Dashboard, Posts, Schedule, History) kannst du die Taste **`i`** drücken, um einen interaktiven Import zu starten. Die TUI pausiert kurz, leert das Terminal und bittet dich um den Pfad zur Markdown-Datei oder zum Ordner. **Tipp: Du kannst die Datei oder den Ordner einfach per Drag & Drop aus dem Finder direkt in das Terminalfenster ziehen.** Das Tool entfernt automatisch störende Anführungszeichen, validiert die Beiträge/Bilder und kehrt direkt wieder zur TUI zurück.
* **Backup & Sync:** Wähle unten im Bereich **BACKUP & SYNC** entweder `Backup Exp.` (Export) oder `Backup Imp.` (Import) und drücke **Enter**, um dein Master-Passwort einzugeben.
  * CLI-Befehle: `./postctl config export -o backup.bin` und `./postctl config import -f backup.bin`.

---

### 🕒 Veröffentlichen geplanter Beiträge (Scheduler)
Da `postctl` eine lokale Anwendung ist, werden geplante Beiträge über einen der folgenden zwei Modi veröffentlicht:

1. **Interaktiver Modus (Automatische TUI-Hintergrund-Goroutine):**
   Sobald die interaktive TUI geöffnet ist (`./postctl tui` oder einfach `./postctl`), läuft automatisch eine Hintergrund-Goroutine mit. Diese prüft die Datenbank alle 10 Sekunden und veröffentlicht fällige Beiträge. Solange die TUI geöffnet ist, musst du nichts weiter tun.

2. **Headless-Modus (Scheduler-Daemon):**
   Wenn du `postctl` headless (z. B. auf einem Server oder ohne die TUI geöffnet zu lassen) betreiben möchtest, kannst du den Scheduler-Daemon starten:
   ```bash
   ./postctl daemon
   ```
   Um den Daemon geräuschlos im Hintergrund laufen zu lassen:
   ```bash
   nohup ./postctl daemon > daemon.log 2>&1 &
   ```

---

### 📝 Vim- / Neovim-Bearbeitungsflow (Schritt für Schritt)

Beim Bearbeiten oder Erstellen eines Beitrags in der TUI kannst du deinen bevorzugten Terminal-Texteditor (wie Vim oder Neovim) starten, um eine ablenkungsfreie und flexible Schreibumgebung zu nutzen.

#### Schritt 1: Externen Editor öffnen
1. Wähle in der TUI einen Beitrag aus und drücke **`e`**, um das Bearbeitungsformular zu öffnen.
2. Drücke die Tastenkombination **`ctrl+v`** in einem beliebigen Eingabefeld.
3. Die TUI pausiert und öffnet Neovim/Vim (das Tool nutzt die Umgebungsvariable `$EDITOR` und weicht bei Bedarf auf `nvim` oder `vim` aus).

#### Schritt 2: Aufbau der Vorlage verstehen
Die in Vim geöffnete temporäre Datei besteht aus drei Abschnitten:
1. **Interaktiver Hilfeblock (`<!-- ... -->`)**:
   - Zeigt ein Zeichenlineal passend zur aktuellen Plattform (z. B. 280 Zeichen für Twitter, 300 für Bluesky, 500 für Mastodon).
   - Zeigt die Zeichenanzahl zum Zeitpunkt des Aufrufs für Einzelbeiträge oder Threads (getrennt durch `---`).
   - *Hinweis:* Dieser HTML-Kommentarblock wird beim Speichern automatisch entfernt.
2. **YAML-Frontmatter-Block**:
   - Eingegrenzt durch die `---` Linien:
     ```yaml
     ---
     platform: twitter
     campaign: launch-2026
     schedule: 2026-06-25 15:00:00
     images: ["logo.png"]
     ---
     ```
   - **Bidirektionale Synchronisierung:** Du kannst diese Metadaten direkt in Vim ändern! Nach dem Schließen parst `postctl` diesen YAML-Header und aktualisiert automatisch die Eingabefelder in der TUI.
3. **Beitragsinhalt (Body)**:
   - Der eigentliche Inhalt deines Beitrags beginnt direkt unter der schließenden `---`-Linie des Frontmatter-Blocks.

#### Schritt 3: Beitrag oder Thread schreiben
- Schreibe deinen gewünschten Text.
- Wenn du einen mehrteiligen Thread verfasst (z. B. für Twitter/X), trenne die einzelnen Beiträge mit einer Zeile, die nur aus **`---`** besteht.
- *Beispiel:*
  ```text
  Das ist der erste Tweet meines Threads.
  ---
  Das ist der zweite Tweet meines Threads.
  ```

#### Schritt 4: Speichern & Zurückkehren
1. Speichere und schließe den Editor mit **`:wq`** oder **`:x`** (oder drücke **`ZZ`** im Normal-Modus).
2. Die TUI wird sofort fortgesetzt. Der Textinhalt wird aktualisiert, die Hilfskommentare entfernt und die editierten Frontmatter-Metadaten werden direkt in die TUI-Formularfelder übernommen.
3. Überprüfe die Änderungen in der TUI, navigiere auf den **`Save`**-Button und bestätige mit **`Enter`**, um den Beitrag in der Datenbank zu speichern.
