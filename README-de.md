# postctl

Social-Media- und Blog-Manager fürs Terminal. Schreibe Beiträge in Markdown, plane sie ein und veröffentliche sie auf Twitter/X, LinkedIn, Threads, Mastodon, Bluesky, Facebook, Telegram, Discord, Reddit, Dev.to, Hashnode und Medium — per Kommandozeile oder in einer vollständigen TUI.

**Unterstützte Plattformen:** Twitter/X · LinkedIn · Threads · Mastodon · Bluesky · Facebook · Telegram · Discord · Reddit · Dev.to · Hashnode · Medium

---

## Schnellstart

1. **Installieren**

   ```bash
   git clone https://github.com/aeon022/postctl && cd postctl
   ./setup.sh
   ```

2. **Bei einer Plattform anmelden**

   ```bash
   postctl auth --platform twitter
   ```

3. **Beitrag schreiben** — erstelle eine Markdown-Datei (siehe [Beitragsformat](#post-markdown-format)):

   ```markdown
   ---
   platform: twitter
   title: Mein erster Beitrag
   ---

   Hallo von postctl.
   ```

4. **Datei importieren**

   ```bash
   postctl import my-post.md
   ```

5. **Sofort veröffentlichen oder einplanen**

   ```bash
   postctl post <ID>
   postctl schedule <ID> --time 2026-07-10T09:00:00+02:00
   ```

6. **TUI öffnen, um alles zu verwalten**

   ```bash
   postctl tui
   ```

---

## Cheatsheet

```
postctl                                  TUI öffnen (Standard)
postctl tui                              TUI explizit öffnen

postctl auth --platform PLATFORM         Bei einer Plattform anmelden
postctl config [--show] [--set K V]      Konfiguration anzeigen oder setzen
postctl config test                      Verbindung zu konfigurierten Plattform-APIs testen
postctl rss add URL                      RSS-Feed-URL hinzufügen
postctl rss list                         Alle konfigurierten RSS-Feeds auflisten
postctl rss remove URL                   Konfigurierten RSS-Feed entfernen
postctl rss import                       Feeds abrufen und Artikel als Entwürfe importieren

postctl import FILE_OR_DIR               Markdown-Beitrag(e) importieren
postctl list [--platform P] [--status S] [--campaign C] [--format human|json]
postctl template --platform PLATFORM     Beitragsvorlage erzeugen

postctl post ID [--dry-run]              Beitrag sofort veröffentlichen (Alias: publish)
postctl publish ID [--dry-run]           Beitrag sofort veröffentlichen
postctl schedule ID [--time DATETIME] [--queue] Beitrag einplanen (RFC3339) oder in die Warteschlange
postctl cancel ID                        Geplanten Beitrag abbrechen
postctl delete ID                        Beitrag lokal und remote löschen
postctl campaign list                    Alle Kampagnen auflisten
postctl campaign post NAME [--dry-run]   Alle Beiträge einer Kampagne veröffentlichen

postctl generate URL                     Beitrag per KI aus einer URL generieren
postctl repurpose ID --platform TARGET [--tone TONE] Beitrag mit anderem Ton umformulieren

postctl git-hook install [--dir DIR]     Post-Commit-Git-Hook installieren
postctl git-hook uninstall               Git-Hook deinstallieren

postctl analytics [--platform P] [--format human|json]
postctl daemon [--dry-run]               Hintergrund-Scheduler ausführen
postctl mcp                              MCP-Server starten (stdio)
postctl version                          Version ausgeben
```

---

## Beitragsformat (Markdown)

### Frontmatter-Felder

| Feld       | Pflicht | Werte / Format                                                        | Beschreibung                       |
|------------|---------|-------------------------------------------------------------------------|------------------------------------|
| `platform` | Ja      | `twitter`, `linkedin`, `threads`, `mastodon`, `bluesky`                 | Zielplattform                      |
| `title`    | Nein    | String                                                                   | Interner Titel (wird nicht veröffentlicht) |
| `campaign` | Nein    | String-Slug                                                              | Gruppiert Beiträge in einer Kampagne |
| `schedule` | Nein    | RFC3339 oder `"queue"`                                                   | Geplante Veröffentlichungszeit oder Smart Queue |

### Body-Format

Schreibe den Beitragstext in reinem Markdown unterhalb des abschließenden `---` des Frontmatter-Blocks.

- **LinkedIn, Threads, Mastodon, Bluesky:** Ein zusammenhängender Text, keine Trenner.
- **Twitter/X-Threads:** Einzelne Tweets mit einer Zeile trennen, die nur `---` enthält. Jeder Abschnitt wird ein Tweet im Thread.

### Beispiel: Twitter-Thread

```markdown
---
platform: twitter
title: Launch-Ankündigung
campaign: product-launch
schedule: 2026-07-10T09:00:00+02:00
---

Das ist der erste Tweet. Maximal 280 Zeichen für Twitter/X.

---

Zweiter Tweet im Thread.

---

Dritter Tweet. Threads gibt es nur bei Twitter.
```

---

## CLI-Referenz

### Authentifizierung und Konfiguration

| Befehl | Beschreibung |
|--------|--------------|
| `postctl auth --platform PLATFORM` | Bei der angegebenen Plattform anmelden (OAuth-Flow) |
| `postctl config --show` | Aktuelle Konfiguration ausgeben |
| `postctl config --set KEY VALUE` | Konfigurationswert setzen |
| `postctl config test` | Verbindungsdiagnose für alle konfigurierten Plattform-APIs |

### RSS-Feed-Importer

| Befehl | Beschreibung |
|--------|--------------|
| `postctl rss add URL` | Neue RSS-Feed-URL zur Konfiguration hinzufügen |
| `postctl rss list` | Alle konfigurierten RSS-Feeds auflisten |
| `postctl rss remove URL` | Konfigurierten RSS-Feed entfernen |
| `postctl rss import` | RSS-Feeds abrufen und neue Artikel als Entwürfe importieren |

### Content-Verwaltung

| Befehl | Beschreibung |
|--------|--------------|
| `postctl import FILE_OR_DIR` | Eine Markdown-Datei oder ein Verzeichnis voller Dateien importieren |
| `postctl list` | Beiträge auflisten; filtern mit `--platform`, `--status`, `--campaign`; Format mit `--format human\|json` |
| `postctl template --platform PLATFORM` | Markdown-Vorlage für die angegebene Plattform ausgeben |
| `postctl generate URL` | KI-generierten Entwurf aus dem Artikel unter URL erzeugen |
| `postctl repurpose ID --platform TARGET [--tone TONE]` | Bestehenden Beitrag mit optionalem, angepasstem Ton umformulieren |

### Veröffentlichung

| Befehl | Beschreibung |
|--------|--------------|
| `postctl post ID` | Beitrag sofort veröffentlichen (Alias: `publish`) |
| `postctl publish ID` | Beitrag sofort veröffentlichen |
| `postctl post ID --dry-run` | Veröffentlichung simulieren, ohne zu senden |
| `postctl schedule ID --time DATETIME` | Geplante Veröffentlichungszeit setzen oder ändern |
| `postctl schedule ID --queue` | Beitrag für den nächsten freien Warteschlangen-Slot einplanen |
| `postctl cancel ID` | Geplanten Beitrag abbrechen (Status zurück auf Entwurf) |
| `postctl delete ID` | Beitrag aus der lokalen Datenbank löschen (und remote, falls veröffentlicht) |
| `postctl campaign list` | Alle Kampagnen mit Beitragsanzahl auflisten |
| `postctl campaign post NAME` | Alle Beiträge einer Kampagne veröffentlichen |
| `postctl campaign post NAME --dry-run` | Kampagnen-Veröffentlichung simulieren |
| `postctl git-hook install [--dir DIR]` | Lokalen Git-Post-Commit-Hook für Auto-Import installieren |
| `postctl git-hook uninstall` | Lokalen Git-Post-Commit-Hook entfernen |
| `postctl daemon` | Hintergrund-Scheduler-Daemon starten |
| `postctl daemon --dry-run` | Daemon im Simulationsmodus ausführen |

### Analytics

| Befehl | Beschreibung |
|--------|--------------|
| `postctl analytics` | Analytics über alle Plattformen anzeigen |
| `postctl analytics --platform PLATFORM` | Auf eine Plattform filtern |
| `postctl analytics --format json` | Als JSON ausgeben |

### MCP-Server

| Befehl | Beschreibung |
|--------|--------------|
| `postctl mcp` | MCP-Server auf stdio starten, zur Nutzung durch KI-Agenten |

---

## TUI-Anleitung

Starten mit `postctl` oder `postctl tui`.

### Ansichten

| Ansicht | Beschreibung |
|---------|--------------|
| Beitragsliste | Hauptansicht; zeigt alle Beiträge mit Status-Badges (Entwurf / geplant / veröffentlicht / fehlgeschlagen) |
| Detail | Vollständiger Beitragsinhalt und Metadaten |
| Editor | Beitragstext und Frontmatter-Felder schreiben oder bearbeiten |
| Zeitplan | Geplante Zeit für einen Beitrag setzen oder anpassen |
| Analytics | Plattformweite Kennzahlen-Übersicht |
| Verlauf | Protokoll vergangener Veröffentlichungen |
| Einstellungen | App-Konfiguration |
| Readme | Dokumentation direkt in der App |

Zwischen Ansichten wechseln über Tabs oder die untenstehenden Tastenkürzel.

### Tastenkürzel

**Beitragsliste**

| Taste | Aktion |
|-------|--------|
| `j` / `k` | Nach oben/unten navigieren |
| `Space` | Beitragsauswahl für Massenaktionen umschalten |
| `Enter` | Detailansicht öffnen |
| `n` | Neuer Beitrag |
| `e` | Ausgewählten Beitrag bearbeiten |
| `d` | Ausgewählte(n) Beitrag/Beiträge lokal und remote löschen (falls veröffentlicht) |
| `p` | Ausgewählte(n) Beitrag/Beiträge sofort veröffentlichen |
| `s` | Ausgewählte(n)/markierte(n) Beitrag/Beiträge auf Warteschlangen-Slots einplanen |
| `Esc` | Mehrfachauswahl aufheben (oder Kampagnenfilter zurücksetzen) |
| `Tab` | Tabs wechseln |
| `q` | Beenden |

**Detailansicht**

| Taste | Aktion |
|-------|--------|
| `Esc` | Zurück zur Liste |
| `e` | Beitrag bearbeiten |
| `p` | Beitrag sofort veröffentlichen |
| `d` | Beitrag lokal und remote löschen |
| `r` | Beitrag umformulieren |

**Editor**

| Taste | Aktion |
|-------|--------|
| `Ctrl+S` | Speichern |
| `Esc` | Abbrechen |
| `Tab` | Zwischen Feldern wechseln |

---

## MCP — KI-Integration

postctl bringt einen eingebauten MCP-Server mit, der alle Kernfunktionen für KI-Agenten bereitstellt. Damit können Tools wie Claude Desktop in deinem Namen Beiträge erstellen, einplanen und veröffentlichen.

### Claude-Desktop-Konfiguration

Füge Folgendes zu `~/Library/Application Support/Claude/claude_desktop_config.json` hinzu:

```json
{
  "mcpServers": {
    "postctl": {
      "command": "postctl",
      "args": ["mcp"]
    }
  }
}
```

Starte Claude Desktop nach dem Speichern neu. Das `postctl`-Binary muss in deinem `PATH` liegen.

### MCP-Tools

| Tool | Parameter | Beschreibung |
|------|-----------|--------------|
| `list_posts` | `platform`, `status`, `campaign` (alle optional) | Beiträge mit optionalen Filtern auflisten |
| `get_post` | `id` | Vollständigen Beitragsinhalt und Metadaten per ID abrufen |
| `create_post` | `platform`, `body`, `title`, `campaign`, `schedule` | Entwurf oder geplanten Beitrag erstellen |
| `publish_post` | `id`, `dry_run` | Beitrag sofort veröffentlichen |
| `schedule_post` | `id`, `schedule` (RFC3339) | Geplante Veröffentlichungszeit setzen oder ändern |
| `list_campaigns` | — | Alle Kampagnen mit Gesamtanzahl und Status-Aufschlüsselung auflisten |
| `get_campaign` | `name`, `status` (optionaler Filter) | Alle Beiträge einer Kampagne mit vollständigem Inhalt abrufen |

Für Twitter-Threads einzelne Tweets im `body`-Feld beim Aufruf von `create_post` mit `\n---\n` trennen.

Zeitplan-Werte müssen RFC3339 sein, z. B. `2026-07-10T09:00:00+02:00`.

### Beispiele für KI-Workflows

**Kampagne aus einem Artikel planen**

> "Lies den Artikel unter https://example.com/blog/launch und erstelle dann eine fünfteilige Kampagne namens `launch-week` mit je einem Beitrag pro Tag ab Montag. Nutze Twitter für drei Beiträge und LinkedIn für zwei."

Claude ruft `create_post` für jeden Beitrag mit passendem Inhalt, dem Kampagnennamen und gestaffelten `schedule`-Werten auf, abgeleitet aus dem Artikelinhalt.

**Geplante Beiträge prüfen und veröffentlichen**

> "Zeig mir alles, was diese Woche geplant ist, und veröffentliche alle Beiträge, die bereit aussehen."

Claude ruft `list_posts` mit `status: scheduled` auf, präsentiert die Ergebnisse zur Prüfung und ruft dann `publish_post` für jeden freigegebenen Beitrag auf — oder alle auf einmal, wenn du bestätigst.

**Blogbeitrag über Plattformen hinweg umformulieren**

> "Nimm Beitrag abc123 und erstelle angepasste Versionen für LinkedIn und Threads."

Claude ruft `get_post` auf, um das Original zu holen, und dann zweimal `create_post` — einmal für `linkedin` und einmal für `threads` — und passt dabei Ton und Länge automatisch pro Plattform an.

---

## Plattform-Hinweise

| Plattform | Zeichenlimit | Threads | Bilder |
|-----------|--------------|---------|--------|
| Twitter/X | 280 pro Tweet | Ja, getrennt mit `---` | Unterstützt |
| LinkedIn | ~3.000 empfohlen | Nein | Unterstützt |
| Threads | 500 | Nein | Mindestens eines empfohlen (Meta-Vorgabe) |
| Mastodon | 500 (Instanz-Standard) | Nein | Unterstützt |
| Bluesky | 300 | Nein | Unterstützt |
| Facebook | ~63.206 | Nein | Unterstützt |
| Telegram | 4.096 (1.024 für Bildunterschriften) | Nein | Unterstützt |
| Discord | 2.000 | Nein | Unterstützt |
| Reddit | 40.000 | Nein | Nicht unterstützt |
| Dev.to | ~100.000 | Nein | Nicht unterstützt |
| Hashnode | ~100.000 | Nein | Nicht unterstützt |
| Medium | ~100.000 | Nein | Nicht unterstützt |

Twitter-Threads haben kein hartes Limit für die Anzahl der Tweets, aber halte Threads fokussiert. Andere Plattformen unterstützen keine mehrteiligen Thread-Beiträge — nutze dort einen einzelnen zusammenhängenden Text.

> [!WARNING]
> **API-Rate-Limits & Massenveröffentlichung:** Mehrere Beiträge gleichzeitig oder in schneller Folge zu veröffentlichen kann zu API-Rate-Limits oder dauerhaften Kontosperrungen führen (besonders bei föderierten Netzwerken wie Mastodon). Verteile Beiträge immer über einen Zeitraum (z. B. mindestens 15–30 Minuten Abstand zwischen aufeinanderfolgenden Veröffentlichungen).

---

## Architektur

```
Markdown-Dateien
      |
   postctl import
      |
      v
SQLite  (~/.local/share/postctl/postctl.db)
      |
      +---> TUI (Bubbletea)    ---> Plattform-APIs  (Twitter, LinkedIn, Threads, Mastodon, Bluesky)
      |
      +---> MCP-Server (stdio) ---> KI-Agenten  (Claude Desktop, etc.)
      |
      +---> postctl daemon     ---> geplante Veröffentlichung über Plattform-APIs
```

**Voraussetzungen:** macOS oder Linux · Go 1.21+ · API-Zugangsdaten für jede genutzte Plattform

---

## Profile — getrennte Konten für Arbeit / Privat / pro Projekt

Standardmäßig nutzt postctl eine einzige Konfiguration und Datenbank. Um komplett getrennte Sätze an Zugangsdaten, Beiträgen und Zeitplänen zu führen — z. B. ein privates und ein geschäftliches Twitter-Konto, oder einen Satz pro Kundenprojekt — übergib `--profile <name>` (oder setze `POSTCTL_PROFILE`) bei jedem Befehl:

```bash
postctl --profile work config set twitter.client_id "..."
postctl --profile work config set twitter.client_secret "..."
postctl --profile work tui

postctl --profile privat config set twitter.client_id "..."
postctl --profile privat tui
```

Ein Profil wird automatisch beim ersten Gebrauch seines Namens angelegt — es gibt keinen separaten "Erstellen"-Schritt. Jedes Profil bekommt seine eigene Konfigurationsdatei (`~/.config/postctl/profiles/<name>/config.yaml`) und seine eigene Datenbank (`~/.local/share/postctl/profiles/<name>/postctl.db`), völlig unabhängig vom Standardprofil und voneinander — keine geteilte Zugangsdatendatei, kein vermischter Beitragsverlauf.

```bash
postctl profile list      # alle bisher genutzten Profile anzeigen, inkl. aktivem
postctl profile           # nur das aktuell aktive Profil anzeigen
```

Ohne `--profile` wird immer das ursprüngliche Standardprofil genutzt (`~/.config/postctl/config.yaml`) — bestehende Setups bleiben unberührt. Jedes Profil kann auch unabhängig über sein eigenes `data_dir` (siehe unten) geräteübergreifend synchronisiert werden — z. B. "work" mit einer geschäftlichen Dropbox, "privat" mit deiner persönlichen iCloud Drive.

---

## Datenverzeichnis geräteübergreifend teilen

Standardmäßig liegt die Datenbank von postctl unter `~/.local/share/postctl/postctl.db`, lokal auf diesem Rechner. Um dieselben Daten auf mehreren Geräten zu nutzen, setze `data_dir` (in `~/.config/postctl/config.yaml`) auf einen Ordner, den du bereits selbst synchronisierst — iCloud Drive, Dropbox, Syncthing, etc.:

```yaml
data_dir: "~/Library/Mobile Documents/com~apple~CloudDocs/postctl"
```

Sobald gesetzt, wechselt postctl seinen SQLite-Journal-Modus automatisch von WAL auf Rollback-Journal — WAL teilt den Zustand auf mehrere Dateien auf, die ein Sync-Client nicht garantiert atomar zusammen aktualisiert. Dieser Wechsel hält das Verzeichnis auf eine einzige konsistente Datei begrenzt, sobald postctl nicht gerade aktiv schreibt. Eine Sperre auf demselben Rechner verhindert außerdem, dass zwei postctl-Prozesse die Datenbank gleichzeitig öffnen (führe `postctl doctor` aus, um den aktuellen Modus und Pfad zu sehen). Das schützt nur vor Problemen auf demselben Rechner und veralteten Snapshots, nicht davor, dass zwei Geräte im exakt selben Moment schreiben; eine noch nicht heruntergeladene iCloud-Datei wird explizit gemeldet statt als bloßer Fehler.

(Das ist getrennt von `postctl config export`/`import`, was deine Konfiguration und Datenbank in eine einzige verschlüsselte Datei für eine manuelle einmalige Übertragung verpackt — die Einstellung oben ist dafür da, sie fortlaufend synchron zu halten.)

---

## Preise

Der Kern von postctl ist kostenlos und Open Source (MIT) — unbegrenzte Beiträge und Entwürfe für bis zu 2 verbundene Konten. Eine Pro-Lifetime-Lizenz hebt das Konto-Limit auf und unterstützt die Weiterentwicklung; kaufen auf [Polar.sh](https://polar.sh) und aktivieren mit:

```bash
postctl license activate <key>
```

**🎉 Launch-Special:** mit Code `POSTCTL2026` gibt's 37 % Rabatt — gültig bis 31. Oktober 2026.

## Lizenz

Siehe [LICENSE](LICENSE).
