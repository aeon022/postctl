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
> `postctl` verwendet eine optimierte Mobile-App-Imitation (Nokia G20 User-Agent & GraphQL-Schnittstelle `a1p9RWpkYKBjWv_I3WzS-A`). Dadurch entfallen zusätzliche Sicherheits-Header wie `x-client-transaction-id` und deine Beiträge werden ohne Bot-Erkennungsfehler (`226 automated request`) oder Captchas veröffentlicht. Zudem schützt eine automatische Pause von 5 Sekunden zwischen Thread-Tweets dein Konto vor Spam-Flags.

### Schritt 1: Cookies aus dem Browser auslesen

1. Öffne deinen Browser, gehe auf [x.com](https://x.com) und stelle sicher, dass du eingeloggt bist.
2. Öffne die Entwicklertools des Browsers:
   * Drücke **F12** oder **Cmd + Option + I** (Mac).
3. Wechsle auf den Reiter für Speicherdaten:
   * **Chrome/Edge/Brave:** Gehe auf **Application** (Anwendung) ➔ **Cookies** ➔ `https://x.com`.
   * **Firefox:** Gehe auf **Storage** (Web-Speicher) ➔ **Cookies** ➔ `https://x.com`.
   * **Safari:** Gehe auf **Storage** (Speicher) ➔ **Cookies** ➔ `https://x.com`.
4. Suche und kopiere die Werte für die folgenden zwei Cookies:
   * **`auth_token`**: Ein ca. 40-stelliger Hex-Wert (z. B. `a066c826c71d97...`).
   * **`ct0`**: Ein ca. 160-stelliger Hex-Wert (das ist dein CSRF-Token, z. B. `9b53579534...`).

### Schritt 2: In `postctl` einrichten

Du kannst das Setup interaktiv starten:
```bash
./postctl config setup twitter
```
Wähle Option **`2`** (Cookie-basierte Authentifizierung) und füge nacheinander dein `auth_token` und dein `ct0` ein.

#### Alternative: Schnelleinrichtung per Einzeiler
Um das Einfügen langer Strings im Terminal zu vereinfachen, kannst du die Konfiguration direkt über Flags übergeben:
```bash
./postctl config setup twitter --cookie "DEIN_AUTH_TOKEN" --ct0 "DEIN_CT0_WERT"
```

*Hinweis: Wenn du statt dem reinen `auth_token` den gesamten Cookie-String deines Browsers kopiert hast (inklusive `twid`, `kdt` etc.), kannst du diesen ebenfalls einfach bei `--cookie` einfügen. `postctl` filtert die relevanten Felder automatisch heraus.*

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
