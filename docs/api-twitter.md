# Twitter/X API Setup Guide für postctl

Dieses Dokument beschreibt Schritt für Schritt, wie du deine eigenen API-Anmeldedaten für Twitter/X erstellst und in `postctl` einrichtest.

---

## 1. Twitter Developer Account & Kostenpflichtiges API-Tier einrichten

> [!IMPORTANT]
> **Wichtiger Hinweis zum Preismodell (Stand 2026):**
> Twitter/X hat alle kostenlosen Tarife („Free Tier“) für den Schreibzugriff (Veröffentlichen von Tweets) vollständig eingestellt. 
> Um Beiträge über `postctl` posten zu können, benötigst du einen kostenpflichtigen Zugang:
> * **Prepaid / Pay-Per-Use (Pay-as-you-go):** Abrechnung nach tatsächlicher Nutzung.
> * **Basic Tier (ca. $100/Monat):** Ermöglicht das automatisierte Veröffentlichen von Beiträgen sowie grundlegende Lesezugriffe über die v2 API.
> 
> Ohne ein aktives, bezahltes Abonnement oder aufgeladenes Prepaid-Guthaben im Twitter Developer Portal wird jeder Versuch, einen Beitrag zu posten, mit einem API-Fehler (`403 Forbidden`) abgebrochen.

1. Gehe auf das [Twitter Developer Portal](https://developer.twitter.com).
2. Melde dich mit deinem Twitter/X-Konto an.
3. Richte dein API-Abonnement (Basic Tier oder Prepaid API Credits) ein, um Schreibrechte zu erhalten.

---

## 2. App im Developer Portal konfigurieren

1. Erstelle ein neues **Projekt** und eine neue **App** in deinem Portal-Dashboard.
2. Navigiere in den **App Settings** zum Bereich **User authentication settings** und klicke auf **Set up** oder **Edit**:
   * **App Type**: Wähle **Web App, Automated App or Bot**.
   * **App Permissions**: Wähle **Read and Write** (wichtig, damit `postctl` Tweets veröffentlichen kann).
   * **Type of App**: Wähle **Native App** oder **Single Page App** (damit der OAuth 2.0 PKCE Flow unterstützt wird).
   * **Callback Image / Redirect URI**: Trage exakt `http://localhost:8753/callback` ein. (Das CLI startet einen lokalen Webserver unter diesem Port, um den Autorisierungscode abzufangen).
   * **Website URL**: Trage deine eigene Website oder `https://github.com/aeon022/postctl` ein.

---

## 3. Client ID & Client Secret erhalten

Nach dem Speichern der Authentifizierungseinstellungen zeigt dir das Developer Portal deine **OAuth 2.0 Client ID** und dein **Client Secret** an.
> [!IMPORTANT]
> Sichere diese Werte sofort. Das Client Secret wird nur einmalig angezeigt.

---

## 4. In `postctl` eintragen

Verwende das `postctl config set` Kommando, um deine Schlüssel zu speichern (nutze das lokale Binary `./postctl`):

```bash
# Client ID eintragen
./postctl config set twitter.client_id "DEINE_CLIENT_ID"

# Client Secret eintragen
./postctl config set twitter.client_secret "DEIN_CLIENT_SECRET"
```

Du kannst die Konfiguration mit folgendem Befehl überprüfen:
```bash
./postctl config show
```

---

## 5. Authentifizierung starten

Führe danach den OAuth-Flow aus:

```bash
./postctl auth twitter
```

1. Es öffnet sich automatisch ein Browserfenster mit dem Twitter-Autorisierungsdialog.
2. Bestätige den Zugriff.
3. Nach erfolgreicher Bestätigung speichert `postctl` dein verschlüsseltes Access- und Refresh-Token in der SQLite-Datenbank.
4. Du bist nun bereit, Tweets mit `./postctl post` oder `./postctl campaign post` zu veröffentlichen (sofern ein bezahltes API-Tier aktiv ist).
