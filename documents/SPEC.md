---
joplin_id: 8b2dcaf743c64c17b95f710ea59f73f8
updated_at: '2026-08-07T19:51:55.745Z'
---
# postctl — Spezifikation

## Überblick

**postctl** ist ein in Go geschriebenes TUI-CLI-Tool, das Social-Media-Beiträge und Blogartikel über Twitter/X, LinkedIn, Threads, Mastodon, Bluesky, Facebook, Telegram, Discord, Reddit, Dev.to, Hashnode und Medium hinweg verwaltet. Beiträge werden als Markdown-Dateien verfasst, in eine lokale SQLite-Datenbank importiert, in einer Terminal-UI vorgeschaut und über Plattform-APIs veröffentlicht — sofort oder nach Zeitplan.

## User Stories

### Als Solo-Entwickler / Indie Hacker:
- Ich will Beiträge als Markdown-Dateien in meinem Repo schreiben
- Ich will vorschauen, wie ein Twitter-Thread aussehen wird, bevor ich poste
- Ich will Beiträge für optimales Timing über Zeitzonen hinweg einplanen
- Ich will denselben Inhalt mit einem Befehl auf mehreren Plattformen posten
- Ich will sehen, was ich wann gepostet habe
- Ich will eine Zeichenzahl-Prüfung vor dem Posten (280 Zeichen pro Tweet)

### Als KI-Assistent (Claude, GPT, Antigravity):
- Ich will korrekt formatierte Markdown-Beiträge erstellen
- Ich will das Posten über CLI-Befehle auslösen
- Ich will Beitragsstatus und -verlauf prüfen können
- Ich will den kompletten Workflow durchgängig ohne menschliches Eingreifen ausführen (außer Freigabe)
- Ich will strukturierte JSON-Ausgabe für alle Befehle, um Ergebnisse zu parsen
- Ich will einen Dry-Run-Modus, um alles zu prüfen, bevor es verbindlich wird

## Prinzip: KI als Operator

**postctl ist darauf ausgelegt, dass KI es bedient — nicht nur Text generiert.**

Die meisten Social-Media-Tools behandeln KI als Copywriting-Feature — "KI schreibt deine Bildunterschrift". postctl behandelt KI als Operator des gesamten Tools. Der Mensch legt die Strategie fest und gibt frei; die KI führt aus.

### Workflow: KI bedient, Mensch gibt frei

```
Mensch: "Poste den neuen Orbiter-v0.3.66-Release"
   ↓
KI (Claude/GPT):
   1. Schreibt Beiträge als Markdown-Dateien (alle Plattformen, DE+EN)
   2. postctl import ./posts/
   3. postctl list --format json          → zeigt Entwürfe
   4. "Hier sind 8 fertige Beiträge. Prüfen?"
   ↓
Mensch: "LinkedIn DE ändern, sonst passt es"
   ↓
KI:
   5. Bearbeitet die Datei, importiert neu
   6. postctl campaign post orbiter-v0366 --dry-run
   7. "Dry Run erfolgreich. Posten?"
   ↓
Mensch: "go"
   ↓
KI:
   8. postctl campaign post orbiter-v0366
   9. "Gepostet. 4/4 Twitter, 2/2 LinkedIn, 2/2 Threads. IDs: ..."
```

### Designanforderungen für den KI-Betrieb

1. **Alle Befehle müssen nicht-interaktiv funktionieren** — keine Prompts, kein "bist du sicher?", keine interaktiven Menüs. Flags steuern alles.
2. **`--format json`** bei allen Befehlen — strukturierte Ausgabe, die KI parsen kann. Standard ist menschenlesbar, `--format json` liefert maschinenlesbar.
3. **`--dry-run`** bei allen verändernden Befehlen — KI kann ohne Nebenwirkungen vorschauen. Mensch gibt frei, dann führt KI ohne `--dry-run` aus.
4. **Exit-Codes** — 0 = Erfolg, 1 = Validierungsfehler, 2 = API-Fehler, 3 = Auth-Fehler. KI liest Exit-Codes, nicht nur die Ausgabe.
5. **Idempotente Importe** — `postctl import` zweimal auszuführen dupliziert keine Beiträge. KI kann nach Bearbeitungen ohne Aufräumen neu importieren.
6. **Wiederherstellung nach Teilfehlern** — schlägt Tweet 3/5 fehl, setzt `postctl post <id> --resume` dort fort, wo es aufgehört hat. KI muss den Zustand nicht manuell verfolgen.
7. **Kein Browser nötig** — der OAuth-Flow nutzt einen Localhost-Callback, aber nach der Authentifizierung läuft alles rein über die CLI. KI muss nie einen Browser öffnen.
8. **Batch-Operationen** — `postctl campaign post <name>` postet alle Beiträge einer Kampagne. KI muss nicht über einzelne Beiträge loopen.

### Beispiel für JSON-Ausgabe

```bash
$ postctl list --format json
{
  "posts": [
    {
      "id": "orbiter-v0366-twitter-en",
      "platform": "twitter",
      "type": "thread",
      "status": "draft",
      "tweets": 5,
      "images": 2,
      "chars": [245, 220, 180, 260, 190],
      "valid": true
    }
  ],
  "total": 8,
  "by_status": {"draft": 8, "posted": 0, "scheduled": 0}
}
```

```bash
$ postctl post orbiter-v0366-twitter-en --format json
{
  "ok": true,
  "platform": "twitter",
  "tweets_posted": 5,
  "thread_id": "1234567890",
  "urls": ["https://x.com/gerwinweiher/status/1234567890"]
}
```

## Kern-Workflows

### 1. Import-Workflow
```
Markdown-Dateien → postctl import → SQLite-DB
                                    ↓
                              Beiträge mit Status "draft"
```

**Eingabe**: Verzeichnis mit `.md`-Dateien mit YAML-Frontmatter
**Verarbeitung**:
- Frontmatter parsen (platform, type, language, campaign, schedule, images)
- Body in Tweets zerlegen (Trennung bei `## Tweet N`-Überschriften)
- Reply-Abschnitt erkennen (`## Reply`)
- Validieren: Zeichenzahl pro Tweet (≤280), Bildpfade existieren
- Deterministische ID aus Dateiname + Plattform erzeugen
- In SQLite einfügen/aktualisieren (Upsert auf ID)

**Ausgabe**: Beiträge in der DB mit Status `draft` oder `scheduled` (falls `schedule:`-Frontmatter vorhanden)

### 2. Vorschau-Workflow
```
postctl (ohne Argumente) → TUI
                      ↓
              Dashboard → Beitragsliste → Detailansicht
                                        ↓
                                   Tweet-für-Tweet-Vorschau
                                   mit Zeichenzahl + Bildindikatoren
```

### 3. Post-Workflow
```
postctl post <id> → Aus DB laden → Validieren → API-Aufruf → Status aktualisieren
                                                ↓
                                          Zuerst Bilder hochladen
                                          Dann Text mit Media-IDs posten
                                          Bei Threads: sequenziell posten,
                                          als Antwort auf vorherige Tweet-ID
```

**Ablauf beim Thread-Posten**:
1. Alle Bilder hochladen → Media-IDs erhalten
2. Tweet 1 posten → Tweet-ID erhalten
3. Tweet 2 als Antwort auf Tweet 1 posten → Tweet-ID erhalten
4. Tweet 3 als Antwort auf Tweet 2 posten → ...
5. Reply-Tweet als Antwort auf den letzten Tweet posten
6. DB aktualisieren: status = "posted", platform_id = erste Tweet-ID

**Fehlerbehandlung**:
- Schlägt Tweet N fehl: Beitrag als "partial" markieren, letzte erfolgreiche Tweet-ID speichern
- Wiederholung: ab dem fehlgeschlagenen Tweet fortsetzen (erfolgreiche nicht erneut posten)
- Bei Rate-Limit: warten und mit exponentiellem Backoff wiederholen

### 4. Zeitplan-Workflow
```
postctl schedule <id> "2026-06-23 09:00"
       ↓
  DB aktualisieren: status = "scheduled", scheduled_at = Zeitstempel
       ↓
  Scheduler-Daemon greift zur richtigen Zeit
       ↓
  Wie der Post-Workflow
```

**Scheduler**:
- Läuft als Hintergrund-Goroutine, während die TUI offen ist
- Läuft auch als `postctl daemon` im Headless-Modus
- Prüft alle 30 Sekunden auf fällige Beiträge
- Postet in Reihenfolge von scheduled_at

### 5. Auth-Workflow
```
postctl auth twitter
       ↓
  Browser öffnen → Twitter-OAuth-Zustimmungsseite
       ↓
  Lokaler HTTP-Server auf :8753 fängt den Callback ab
       ↓
  Code gegen Token tauschen
       ↓
  Verschlüsseltes Token in SQLite speichern
```

## Markdown-Formatspezifikation

### Frontmatter-Felder

| Feld | Pflicht | Typ | Werte | Standard |
|------|---------|-----|-------|----------|
| `platform` | ja | String | `twitter`, `linkedin`, `threads`, `all` | — |
| `type` | ja | String | `thread`, `single`, `article` | — |
| `language` | nein | String | ISO 639-1 (`en`, `de`) | `en` |
| `campaign` | nein | String | freier Slug | — |
| `schedule` | nein | Datum/Zeit | ISO 8601 lokal | — |
| `images` | nein | Liste | relative Dateipfade | — |
| `tags` | nein | Liste | Hashtags ohne # | — |

### Body-Format

**Einzelbeitrag** (LinkedIn, Threads):
```markdown
---
platform: linkedin
type: single
---

Der gesamte Beitragstext kommt hierher.
Mehrere Absätze werden unterstützt.
```

**Thread** (Twitter):
```markdown
---
platform: twitter
type: thread
images:
  - screenshots/01-dashboard.png
---

## Tweet 1

Inhalt des ersten Tweets. Keine Links hier, wegen algorithmischer Reichweite.

## Tweet 2

Zweiter Tweet. Bild anhängen: screenshots/01-dashboard.png

## Tweet 3

Inhalt des dritten Tweets.

## Reply

Links und Hashtags kommen in den Selbst-Reply.
github.com/aeon022/orbiter

#opensource #webdev
```

**Regeln**:
- `## Tweet N` trennt in einzelne Tweets
- `## Reply` wird als Antwort auf den letzten Tweet gepostet
- Bildzuordnung: erstes Bild in der `images:`-Liste geht an Tweet 2, zweites an Tweet 3, usw. Alternativ inline mit `<!-- image: filename.png -->`
- Zeichenzahl: ≤280 pro Tweet (URLs zählen als 23 Zeichen, nach Twitters t.co)
- Leere Tweets werden übersprungen

## Plattform-API-Details

### Twitter/X v2

**Auth**: OAuth 2.0 mit PKCE
```
GET https://twitter.com/i/oauth2/authorize
  ?client_id=...
  &redirect_uri=http://localhost:8753/callback
  &scope=tweet.read+tweet.write+users.read+offline.access
  &response_type=code
  &code_challenge=...
  &code_challenge_method=S256
  &state=...
```

**Tweet posten**:
```
POST https://api.twitter.com/2/tweets
Authorization: Bearer <token>
Content-Type: application/json

{"text": "...", "media": {"media_ids": ["..."]}, "reply": {"in_reply_to_tweet_id": "..."}}
```

**Medien hochladen** (v1.1 — weiterhin erforderlich):
```
POST https://upload.twitter.com/1.1/media/upload.json
Content-Type: multipart/form-data

media_data=<base64>
```

### LinkedIn v2

**Beitrag**:
```
POST https://api.linkedin.com/v2/ugcPosts
Authorization: Bearer <token>

{
  "author": "urn:li:person:<id>",
  "lifecycleState": "PUBLISHED",
  "specificContent": {
    "com.linkedin.ugc.ShareContent": {
      "shareCommentary": {"text": "..."},
      "shareMediaCategory": "IMAGE",
      "media": [{"status": "READY", "media": "<asset-urn>"}]
    }
  },
  "visibility": {"com.linkedin.ugc.MemberNetworkVisibility": "PUBLIC"}
}
```

**Bild-Upload** (2 Schritte):
1. Registrieren: `POST /v2/assets?action=registerUpload` → liefert Upload-URL + Asset-URN
2. Hochladen: `PUT <upload-url>` mit binären Bilddaten

### Threads (Meta Graph API)

**Container erstellen**:
```
POST https://graph.threads.net/v1.0/<user_id>/threads
  ?media_type=TEXT
  &text=...
  &access_token=...
```

**Veröffentlichen**:
```
POST https://graph.threads.net/v1.0/<user_id>/threads_publish
  ?creation_id=<container_id>
  &access_token=...
```

## Fehlerbehandlungs-Strategie

| Fehler | Aktion |
|--------|--------|
| Rate-Limit (429) | `retry-after`-Header abwarten, dann wiederholen |
| Auth abgelaufen (401) | Token-Refresh versuchen, bei Fehlschlag erneute Anmeldung anfordern |
| Netzwerkfehler | 3x wiederholen mit exponentiellem Backoff (1s, 4s, 16s) |
| Teilweiser Thread | Als "partial" markieren, Fortschritt speichern, Fortsetzung erlauben |
| Ungültiger Inhalt | Validierungsfehler vor dem API-Aufruf, Anzeige in der TUI |
| Bild zu groß | Mit Go-Bildbibliothek vor dem Upload verkleinern |

## Nicht-Ziele (v1)

- Kein Web-Dashboard
- Keine Mehrbenutzer-/Team-Funktionen
- Keine eingebaute Bildgenerierung
- Kein automatisches Cross-Posting (explizit pro Plattform)

## Erfolgsmetriken

- 20 Markdown-Beiträge in <1 Sekunde importieren
- Einen 6-Tweet-Thread mit 2 Bildern in <10 Sekunden posten
- TUI rendert mit 60fps im Standard-Terminal
- Einzelnes Binary, <20MB, keine Laufzeit-Abhängigkeiten
- Läuft unter macOS, Linux, Windows

---

## Zukünftige Features (v2+)

> **Hinweis:** Alles in diesem Abschnitt ist eine zukunftsgerichtete Idee, keine Zusage und kein aktuell verkauftes Produkt. postctls tatsächliches Preismodell ist heute die einmalige missionctl-Bundle-Lizenz (Polar.sh), beschrieben auf der [Preise-Seite](/#pricing) — nicht die Abo-Stufen, die unten skizziert sind.

### `postctl generate`
KI generiert Beiträge aus einer URL oder einem Markdown-Artikel.
- Eingabe: URL, Markdown-Datei oder freier Text
- Ausgabe: Thread-Entwurf + LinkedIn-Beitrag + Threads-Beitrag als Markdown
- Nutzt die Claude-API, OpenAI-API oder Ollama (lokal)
- Nutzer prüft und bearbeitet vor dem Posten

### `postctl repurpose`
Nimmt einen bestehenden Beitrag und konvertiert ihn für andere Plattformen.
- Dev.to-Artikel → Twitter-Thread + LinkedIn + Threads
- Twitter-Thread → LinkedIn-Langbeitrag
- Passt Länge, Ton und Hashtags automatisch an

### `postctl analytics`
Holt Engagement-Daten von den APIs nach dem Posten.
- Likes, Retweets, Impressions (Twitter)
- Reaktionen, Kommentare (LinkedIn)
- Ermittelt beste Posting-Zeiten
- Terminal-Dashboard mit Sparklines

### `postctl template`
Vorgefertigte Beitragsstrukturen.
- `postctl template launch` — Produkt-Launch-Ankündigung
- `postctl template feature` — Feature-Update-Thread
- `postctl template thought` — Thought-Leadership-Beitrag
- Erzeugt eine Markdown-Datei mit Platzhaltern

---

## Ökosystem-Vision

postctl ist Teil eines Content-Loops:

```
Orbiter (Erstellen) → postctl (Verteilen) → Analytics (Lernen) → Orbiter (Verbessern)
```

1. Content in Orbiter schreiben (Blogbeiträge, Seiten)
2. Beiträge als Markdown exportieren/generieren
3. postctl verteilt auf Twitter, LinkedIn, Threads
4. Analytics zeigt, was funktioniert
5. Erkenntnisse fließen in den nächsten Content-Zyklus

Langfristig: `orbiter export --to-postctl` als Integration.

---

## Projekt-Setup

```
Repository: ~/Developing/Projects/postctl
Module:     github.com/aeon022/postctl
License:    MIT
```
