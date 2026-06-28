# Twitter/X API Setup Guide für postctl

Dieses Dokument beschreibt Schritt für Schritt, wie du die Authentifizierung für Twitter/X in `postctl` einrichtest.

Es stehen dir zwei Optionen zur Verfügung:
* **Option A: Cookie-basierte Authentifizierung (Kostenlos & Empfohlen)** – Nutzt deine bestehende Browsersitzung. Komplett kostenlos und ohne Entwickler-Account.
* **Option B: Offizielle API (Kostenpflichtig)** – Nutzt die offizielle Twitter API (erfordert ein bezahltes Abonnement ab ca. $100/Monat oder prepaid API-Credits).

---

## Option A: Cookie-basierte Authentifizierung (Kostenlos & Empfohlen)

Diese Methode simuliert eine echte Browser-Sitzung, indem sie deine Anmelde-Cookies verwendet. Sie ist vollkommen kostenlos und erfordert keine Einrichtung im Twitter Developer Portal.

> [!TIP]
> **Warum diese Methode stabil läuft:**
> `postctl` verwendet eine optimierte Desktop-Web-Imitation (Google Chrome auf macOS User-Agent, passende `Sec-Ch-Ua`-Verifikations-Header & GraphQL-Schnittstelle `SiM_cAu83R0wnrpmKQQSEw`). Dadurch entfallen zusätzliche Sicherheits-Header wie `x-client-transaction-id` und deine Beiträge werden ohne Bot-Erkennungsfehler (`226 automated request`) oder Captchas veröffentlicht. Zudem schützt eine automatische Pause von 5 Sekunden zwischen Thread-Tweets dein Konto vor Spam-Flags.

> [!IMPORTANT]
> **Kompletten Cookie-String kopieren (WICHTIG):**
> X (Twitter) verlangt für Beitragsveröffentlichungen zwingend auch das **`twid`**-Cookie (deine User-ID) und Session-Details. Kopiere daher immer den **kompletten Cookie-String** deines Browsers und füge ihn bei `--cookie` ein, um den Spam-Erkennungsfehler `226` zu vermeiden.

### Schritt 1: Komplette Browser-Cookies auslesen

Der einfachste Weg ist, den gesamten Cookie-Header einer beliebigen Anfrage zu kopieren:

1. Öffne deinen Webbrowser, gehe auf [x.com](https://x.com) und stelle sicher, dass du eingeloggt bist. (Am besten einmal aus- und wieder einloggen, um die Sitzung frisch zu starten).
2. Öffne die Entwicklertools (**F12** oder **Cmd + Option + I**).
3. Wechsle auf den Reiter **Network** (Netzwerk).
4. Lade die Seite einmal neu (**F5** oder **Cmd + R**).
5. Klicke in der Liste der Netzwerkanfragen auf eine beliebige Anfrage zu `x.com` (z. B. `home` oder einen GraphQL-Request).
6. Suche im rechten Bereich unter **Request Headers** (Anfrage-Header) nach der Zeile **`cookie:`**.
7. Kopiere den **gesamten langen Wert** (er fängt meist mit `guest_id=...` oder `kdt=...` an und enthält alle Cookies).
8. Suche zusätzlich in den Cookies den Wert für **`ct0`** (dein ca. 160-stelliger CSRF-Token) heraus und kopiere ihn ebenfalls.

### Schritt 2: In `postctl` einrichten

Wir empfehlen die Schnelleinrichtung per Einzeiler im Terminal. Ersetze die Platzhalter durch deine kopierten Werte:

```bash
./postctl config setup twitter --cookie "HIER_DER_GESAMTE_KOPIERTE_COOKIE_STRING" --ct0 "HIER_NUR_DER_CT0_WERT"
```

*(Hinweis: Falls du das interaktive Setup über `./postctl config setup twitter` startest und Option `2` wählst, kannst du bei der Abfrage nach dem `auth_token` ebenfalls den gesamten langen Cookie-String einfügen).*

### 🛠️ Fehlerbehebung bei Cookie-Fehlern (`empty tweet ID`)

Sollte beim Veröffentlichen eines Tweets der Fehler `empty tweet ID returned in cookie mode` auftreten, liegt das an einer von zwei Ursachen:
1. **Abgelaufene Session-Cookies:** Twitter/X beendet Browsersitzungen nach einiger Zeit. Wiederhole einfach **Schritt 1** und trage die neuen Cookies ein.
2. **Rotierte GraphQL Query-ID:** X ändert im Web-Frontend regelmäßig die interne ID für die `CreateTweet`-Mutation. In diesem Fall passt `postctl` die IDs in einem Update an. Stelle sicher, dass du die aktuellste Version von `postctl` nutzt.

---

## Option B: Offizielle API (Kostenpflichtig)

Wenn du ein offizielles Entwickler-Konto besitzt und die monatlichen API-Kosten tragen möchtest, kannst du die Standard-OAuth-Authentifizierung nutzen.

> [!IMPORTANT]
> **Kostenhinweis (Stand 2026):**
> Twitter/X bietet für neu erstellte Developer-Accounts keinen kostenlosen Schreibzugriff (Free Tier) mehr an. Um über die offizielle Schnittstelle zu posten, ist ein kostenpflichtiger API-Zugang (z. B. Basic Tier für ca. $100/Monat oder prepaid Credits) im Developer Portal erforderlich.

### Schritt 1: App im Developer Portal konfigurieren
1. Gehe auf das [Twitter Developer Portal](https://developer.twitter.com) und melde dich an.
2. Erstelle ein neues **Projekt** und eine neue **App** in deinem Portal-Dashboard.
3. Navigiere in den **App Settings** zu **User authentication settings** und klicke auf **Set up**:
   * **App Type**: Wähle **Web App, Automated App or Bot**.
   * **App Permissions**: Wähle **Read and Write** (wichtig für Schreibrechte).
   * **Type of App**: Wähle **Native App** oder **Single Page App** (für OAuth 2.0 PKCE).
   * **Callback URI / Redirect URL**: Trage exakt `http://localhost:8753/callback` ein.
   * **Website URL**: Trage deine eigene Website oder `https://github.com/aeon022/postctl` ein.
4. Speichere die Einstellungen und kopiere die angezeigte **Client ID** und das **Client Secret** an einen sicheren Ort.

### Schritt 2: Schlüssel hinterlegen
```bash
# Client ID eintragen
./postctl config set twitter.client_id "DEINE_CLIENT_ID"

# Client Secret eintragen
./postctl config set twitter.client_secret "DEIN_CLIENT_SECRET"
```

### Schritt 3: Authentifizierung durchführen
```bash
./postctl auth twitter
```
Es öffnet sich ein Browserfenster, in dem du der App den Zugriff erlaubst. Nach erfolgreichem Login speichert `postctl` dein verschlüsseltes Access- und Refresh-Token.
