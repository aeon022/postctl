# Twitter/X API Setup Guide für postctl

Dieses Dokument beschreibt Schritt für Schritt, wie du die Authentifizierung für Twitter/X in `postctl` einrichtest.

Es stehen dir zwei Optionen zur Verfügung:
* **Option A: Cookie-basierte Authentifizierung (Kostenlos, aber inoffiziell)** – Nutzt deine bestehende Browsersitzung. Kostenlos, aber fehleranfällig und mit Risiko einer Kontosperrung.
* **Option B: Offizielle API (Kostenpflichtig & Empfohlen)** – Nutzt die offizielle Twitter API (erfordert ein bezahltes Abonnement ab ca. $100/Monat oder prepaid API-Credits). Der sichere, stabile Weg.

---

## Option A: Cookie-basierte Authentifizierung (Kostenlos, aber inoffiziell)

Diese Methode simuliert eine echte Browser-Sitzung, indem sie deine Anmelde-Cookies verwendet. Sie ist vollkommen kostenlos und erfordert keine Einrichtung im Twitter Developer Portal.

> [!WARNING]
> **Inoffizielle Umgehungsmethode (Risiko von Kontosperrung):**
> Die Cookie-basierte Authentifizierung simuliert eine Web-Sitzung. Diese Methode ist fehleranfällig, verstößt gegen die Nutzungsbedingungen (ToS) von X/Twitter und kann zur Sperrung deines Kontos führen. Der einzig offizielle und sichere Weg zum Posten ist die Verwendung der kostenpflichtigen API (Option B).
> 
> * `postctl` versucht, durch Header-Imitation und künstliche Pausen (5 Sekunden zwischen Beiträgen) das Risiko zu minimieren, bietet aber keine Garantie.
> * X verlangt zwingend auch das **`twid`**-Cookie (deine User-ID). Trage daher stets den **kompletten Cookie-String** oder beide Cookies (`auth_token` und `ct0`) ein.

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

### 🛠️ Fehlerbehebung bei Cookie-Fehlern / Automatisierungswarnungen

Sollte beim Veröffentlichen eines Tweets ein GraphQL-Fehler wie `empty tweet ID returned...` oder der Fehler **226** (`This request looks like it might be automated...`) auftreten, greift `postctl` automatisch auf einen **Headless-Browser-Fallback** zurück:
1. **Automatischer Browser-Start**: `postctl` startet im Hintergrund unsichtbar Google Chrome (mittels `chromedp`), lädt deine Cookies (`auth_token` & `ct0`), navigiert zur Web-Oberfläche von X, befüllt den Composer (inklusive Threads und Medien-Uploads) und klickt auf "Posten".
2. **Voraussetzung**: Google Chrome muss auf deinem System installiert sein (wird auf macOS standardmäßig in `/Applications` gesucht).
3. **Abgelaufene Session-Cookies**: Wenn auch der Headless-Browser scheitert, sind in der Regel deine Cookies abgelaufen. Wiederhole einfach **Schritt 1** und trage die neuen Cookies ein.

---

## Option B: Offizielle API (Kostenpflichtig & Empfohlen)

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
