# Brücke-Profil einrichten (Instagram, LinkedIn-Seite, X, Facebook)

Stand: 2026-10-16. Profil `bruecke` existiert (`postctl profile create bruecke`).
Alle Befehle mit `--profile bruecke`, sonst landen Zugangsdaten im Default-Profil.
**Token nie in den Chat oder in Repos einfügen** — nur über `config setup` ins Terminal.

Die Lizenz muss nicht neu eingerichtet werden: ein benanntes Profil ohne eigenen
`license_key` erbt den des Default-Profils.

---

## 1. Instagram

Vorher: Das Konto muss ein **Professional-Konto** sein (Business oder Creator) und mit
der **Facebook-Seite der Brücke verknüpft** sein.
- Instagram-App: *Einstellungen → Kontotyp und Tools → Zu Professional-Konto wechseln*
- Verknüpfen: Instagram *Konto → Verknüpfte Konten / Seite verbinden* oder in der
  Facebook-Seite *Einstellungen → Verknüpfte Konten → Instagram*

### App anlegen (developers.facebook.com → App erstellen → Anwendungsfälle)

Anwendungsfall **„Messaging und Content auf Instagram verwalten"** wählen — den braucht
postctl. **„Facebook-, Instagram- und Threads-Content auf anderen Websites einbetten"
(oEmbed) abwählen**: wird nicht gebraucht.
Für die Facebook-Seite der Brücke zusätzlich **„Alles auf deiner Seite verwalten"**.
Meta weist darauf hin, dass sich manche Anwendungsfälle nicht kombinieren lassen — wenn
das hier zutrifft, für Facebook eine zweite App verwenden.
Solange du Admin der App bist, ist kein App-Review nötig (Entwicklungsmodus).

### Variante A (empfohlen): Instagram-Login — keine Facebook-Seite, kein Graph Explorer

1. App-Dashboard → *Anwendungsfälle → Messaging und Content auf Instagram verwalten →
   Anpassen → API-Einrichtung mit Instagram-Login*.
2. Das Instagram-Konto der Brücke als **Instagram-Tester** zur App hinzufügen
   (*App-Rollen → Rollen*) und die Einladung in Instagram annehmen
   (*Einstellungen → Apps und Websites → Tester-Einladungen*).
3. Im Schritt *Zugriffstoken generieren* das Konto hinzufügen, bei Instagram einloggen
   und freigeben. Scopes: `instagram_business_basic`, `instagram_business_content_publish`.
   Das Dashboard zeigt den **Token (60 Tage gültig)** und die **Konto-ID**.
4. Token verlängern (frühestens 24 h nach Ausstellung, vor Ablauf):
   `https://graph.instagram.com/refresh_access_token?grant_type=ig_refresh_token&access_token=TOKEN`
5. Konto-ID alternativ: `https://graph.instagram.com/me?fields=user_id,username&access_token=TOKEN`

Die Menünamen bei Meta ändern sich häufig — die Reihenfolge stimmt, die Beschriftung kann abweichen.

### Variante B: Facebook-Login (Konto muss mit einer Facebook-Seite verknüpft sein)

1. developers.facebook.com/tools/explorer → App wählen, *User Token*. Berechtigungen:
   `instagram_basic`, `instagram_content_publish`, `pages_show_list`, `pages_read_engagement`.
   *Generate Access Token* → Brücke-Seite und Instagram-Konto wählen (gilt ca. 1 Stunde).
2. Langzeit-Token (60 Tage):
   `https://graph.facebook.com/v19.0/oauth/access_token?grant_type=fb_exchange_token&client_id=APP_ID&client_secret=APP_SECRET&fb_exchange_token=KURZZEIT_TOKEN`
3. Besser: mit dem Langzeit-Token `me/accounts` abfragen und das `access_token` der
   Seite nehmen (läuft nicht ab).
4. Account-ID: `me/accounts?fields=name,instagram_business_account` → `id` unter
   `instagram_business_account`.

### Wichtig beim Posten

- Jeder Instagram-Post braucht **ein Bild als öffentliche `https://`-URL**
  (`images:` im Frontmatter). Lokale Dateien nimmt die API nicht. Bilder z. B. auf der
  eigenen Webseite ablegen (`~/Sites`).
- Reine Textposts gehen nicht. Caption max. 2200 Zeichen.
- Löschen per API ist bei Instagram nicht möglich — Posts in der App löschen.
- Ein Token läuft nach 60 Tagen ab (Variante A: vorher verlängern, siehe oben) → sonst `config setup instagram` erneut.
- Die Einstellung steht in `instagram.api` (`instagram` = Variante A, sonst Variante B). Beide Varianten sind gegen Mocks getestet, nicht gegen die echte API.

---

## 2. LinkedIn

### Als Person posten
Funktioniert mit der normalen LinkedIn-App (Produkt *Share on LinkedIn*, Scope
`w_member_social`):

```sh
postctl --profile bruecke config setup linkedin    # Client-ID/-Secret der App
postctl --profile bruecke auth linkedin            # Browser-Login
```

### Als Unternehmensseite posten — Ergebnis der Recherche

postctl kann es (`linkedin.organization_id`, Commit `0d945f7`), **aber LinkedIn gibt
die API dafür nur eingeschränkt frei**. Seiten-Posts brauchen den Scope
`w_organization_social`, und der gehört zum Produkt **Community Management API**.

Voraussetzungen laut Microsoft/LinkedIn-Doku
([Increasing Access](https://learn.microsoft.com/en-us/linkedin/marketing/increasing-access?view=li-lms-2025-11),
[Migration Guide](https://learn.microsoft.com/en-us/linkedin/marketing/community-management/community-management-api-migration-guide?view=li-lms-2026-06)):

- **„At this time, our Community Management APIs are only available to registered legal
  organizations for commercial use cases only."** Private Projekte/Einzelpersonen ohne
  Firma werden voraussichtlich abgelehnt.
- Für den Antrag nötig: **verifizierte Geschäfts-E-Mail** (keine private Adresse),
  Firmenname, Firmenadresse, Website, Datenschutzerklärung.
- Ein **Super-Admin der LinkedIn-Seite** muss die App mit der Seite verknüpfen
  (*App → Settings → Verify*).
- Die App darf **kein anderes API-Produkt** haben — das Produkt wird auf einer
  **neuen, leeren App** beantragt (*Products → Community Management API → Request Access*).
  Für das Brücke-Profil also eine eigene LinkedIn-App mit eigener Client-ID/-Secret.
- Das eigene LinkedIn-Konto braucht auf der Seite die Rolle **Administrator**, **Content-Admin** oder
  **Direct Sponsored Content Poster** (laut Posts-API-Doku).
- Stufen: **Development** (Standard nach Freigabe): 500 Aufrufe/App und 100 Aufrufe/Mitglied
  pro 24 h, in 12 Monaten fertig integrieren. **Standard**: keine Limits, braucht
  zweites Formular + Screencast (< 5 min). Für wenige Posts pro Woche reicht Development.
- Ablehnung: kein erneuter Antrag mit derselben App — neue App + neuer Antrag.

### Falls die Freigabe kommt

```sh
postctl --profile bruecke config set linkedin.organization_id 123456   # Zahl aus linkedin.com/company/<ID>/admin
postctl --profile bruecke auth linkedin                                  # neu einloggen (neuer Scope)
```

**Technik (seit Commit "company page via Posts API"):** Mit gesetzter `linkedin.organization_id` postet
postctl über die **Posts API** (`POST /rest/posts`, Header `Linkedin-Version: 202606`), nicht mehr über das
abgelöste `/v2/ugcPosts` (laut LinkedIn-Doku „The Posts API replaces the ugcPosts API"). Bilder laufen über die
Images API (`/rest/images?action=initializeUpload` → Upload → `urn:li:image`). Unterstützt: **Text oder ein Bild**;
mehrere Bilder werden mit klarer Fehlermeldung abgelehnt (MultiImage-Format noch nicht umgesetzt). Bild-URLs
(`https://…`) werden heruntergeladen und hochgeladen. Löschen: `DELETE /rest/posts/{urn}`.
Erforderliche Seitenrolle laut Doku: **ADMINISTRATOR, DIRECT_SPONSORED_CONTENT_POSTER oder CONTENT_ADMIN**.
Gegen die echte API ist das **noch nicht getestet** (nur Mocks) — erster Testpost nach der Freigabe.

### Alternativen, falls der Antrag nicht durchgeht
- Auf der Seite von Hand posten (oder LinkedIns eigene Planung nutzen).
- Als Person posten und die Seite teilen lassen.

---

## 3. X und Facebook

```sh
postctl --profile bruecke config setup twitter
postctl --profile bruecke auth twitter
postctl --profile bruecke config setup facebook   # App-ID, App-Secret, Page-ID der Brücke-Seite
postctl --profile bruecke auth facebook
```

X: Löschen funktioniert nur im OAuth-Modus, nicht im Cookie-Modus.

---

### Wichtig: Meta-App auf „Live" stellen (sonst sieht nur das Team die Posts)
Eine Meta-App im **Entwicklungsmodus** darf zwar posten, die Beiträge sind aber nur für Personen mit Rolle in der App sichtbar (Admin,
Entwickler, Tester) — für alle anderen wirken sie, als gäbe es sie nicht, obwohl die API „öffentlich" meldet. Deshalb vor dem ersten
echten Post: im Dashboard oben den **App-Modus auf „Live"** schalten. Meta verlangt dafür u. a. Datenschutzerklärungs-URL, App-Symbol,
Kategorie (und je nach App eine Kontakt-E-Mail). Eine App-Prüfung ist für die eigene Seite nicht nötig. Danach den Post in einem
privaten Browserfenster (ausgeloggt) prüfen. Gilt für Facebook; bei Instagram und Threads ebenfalls prüfen.

### Facebook ohne Weiterleitungs-URL (Seiten-Token)
Findest du bei Meta kein Feld für `https://localhost:8753/callback`, geht es auch mit einem fertigen **Seiten-Token**:
1. developers.facebook.com/tools/explorer → deine App wählen, *User Token*, Berechtigungen `pages_show_list`,
   `pages_manage_posts`, `pages_read_engagement` → *Generate Access Token* (Seite im Dialog auswählen).
2. Optional (für einen Token ohne Ablauf): den Token im Browser gegen einen langlebigen tauschen:
   `https://graph.facebook.com/v19.0/oauth/access_token?grant_type=fb_exchange_token&client_id=APP_ID&client_secret=APP_SECRET&fb_exchange_token=TOKEN`
3. Im Explorer `me/accounts` abfragen → das `access_token` **der Brücke-Seite** kopieren (das ist der Seiten-Token).
4. `postctl --profile bruecke config setup facebook`: App-ID/Secret leer lassen oder eintragen, als Seite `brueckegraz`, beim
   Seiten-Access-Token den Token einfügen. Dann `postctl --profile bruecke auth facebook` — prüft den Token, löst die Seiten-ID auf, speichert.
Token nie in den Chat.

## 4. Scheduler

```sh
postctl --profile bruecke daemon install    # eigener Dienst für dieses Profil
postctl --profile bruecke daemon status
```

Log: `~/Library/Logs/postctl-bruecke.log`. Im TUI: Settings → *Scheduler (Hintergrund)*.
Solange das TUI offen ist, hält es die DB-Sperre — dann pausiert der Dienst; mit
`scheduler.auto_publish: true` veröffentlicht stattdessen das TUI.

## 5. Im TUI zwischen Profilen wechseln

Taste `P` → Profilauswahl.
