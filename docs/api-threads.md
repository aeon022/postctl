# Threads (Meta) API Setup Guide für postctl

Dieses Dokument beschreibt detailliert, wie du Zugriff auf die offizielle Threads API erhältst, deine App im Meta Developer Portal konfigurierst und die Authentifizierung für `postctl` einrichtest.

---

## 1. Meta Developer App erstellen

1. Gehe zum [Meta Developer Portal](https://developers.facebook.com) und melde dich an.
2. Klicke oben rechts auf **Meine Apps** und dann auf **App erstellen** (Create App):
   * Wähle als App-Typ **Anderes** (Other) und klicke auf *Weiter*.
   * Wähle als App-Typ **Verbraucher** (Consumer) oder einen Typ, der dir Zugriff auf die Threads/Instagram API ermöglicht.
   * Vergib einen Namen für deine Anwendung (z. B. `postctl-threads-app`).
   * Klicke auf **App erstellen** (Ggf. musst du dein Facebook-Passwort eingeben).

---

## 2. Threads API hinzufügen

1. Scrolle im Dashboard deiner erstellten App nach unten zu **Produkt hinzufügen** (Add Products to Your App).
2. Suche nach **Threads API** und klicke auf **Einrichten** (Set Up).
3. Die App befindet sich nun im **Entwicklungsmodus** (In Development) – das ist perfekt und wichtig für die lokale Entwicklung.

---

## 3. Callback-URLs konfigurieren (Wichtig: Pflichtfelder & HTTPS!)

Meta erzwingt für die Threads API eine sichere HTTPS-Verbindung. Gleichzeitig weigert sich das Dashboard, Änderungen zu speichern, wenn Deinstallations- oder Lösch-URLs fehlen. 

1. Klicke in der linken Navigationsleiste unter **Threads API** auf **Threads-Einstellungen** (Threads Settings) oder gehe zu **Anwendungsfälle** (Use Cases) ➔ **Auf Threads API zugreifen** ➔ **Anpassen / Einstellungen**.
2. Konfiguriere dort die folgenden drei Felder:
   * **Callback-URLs umleiten** (Valid OAuth Redirect URIs):
     `https://localhost:8753/callback` (Wichtig: **https** statt http!)
   * **Callback-URL deinstallieren** (Uninstall Callback URL):
     `https://localhost:8753/uninstall` (oder `https://example.com/uninstall` als Platzhalter)
   * **Callback-URL löschen** (Delete Callback URL):
     `https://localhost:8753/delete` (oder `https://example.com/delete` als Platzhalter)
3. Klicke unten rechts auf **Speichern** (Save).

---

## 4. App-Rollen & Tester-Einladung einrichten (Wichtig!)

Da sich deine App im Entwicklungsmodus befindet, kann sie nur von Benutzern authentifiziert werden, die explizit als Tester registriert sind.

### Schritt A: Tester im Meta-Dashboard hinzufügen
1. Klicke in der linken Navigationsleiste auf **App-Rollen** (App Roles / das Personen-Symbol) ➔ **Rollen** (Roles).
2. Scrolle ganz nach unten zum Bereich **Threads-Tester** (Threads Testers).
3. Klicke auf **Threads-Tester hinzufügen** (Add Threads Testers).
4. Gib den exakten Instagram/Threads-Benutzernamen des Kontos ein, mit dem du später posten willst, und klicke auf **Bestätigen**.

### Schritt B: Einladung in Instagram annehmen
Die Einladung muss von deinem Threads-Konto manuell bestätigt werden:
1. Logge dich im Desktop-Browser auf **[Instagram.com](https://www.instagram.com/)** mit dem entsprechenden Account ein.
2. Gehe auf dein Profil ➔ Klicke auf das **Zahnrad-Symbol** (Einstellungen).
3. Navigiere in der linken Leiste zu **Apps und Websites** (Apps and Websites).
4. Klicke oben auf den Reiter **Tester-Einladungen** (Tester Invites).
5. Dort wird dir die Einladung deiner App (z. B. `postctl-threads-app`) angezeigt. Klicke auf **Akzeptieren** (Accept).

---

## 5. App ID & App Secret abrufen

1. Gehe in der linken Navigationsleiste auf **App-Einstellungen** (App Settings) ➔ **Standard** (Basic).
2. Kopiere die **App-ID** (App ID).
3. Kopiere den **App-Geheimschlüssel** (App Secret / erfordert Klick auf *Anzeigen* und Passworteingabe).

---

## 6. In `postctl` konfigurieren

Speichere deine Threads-Zugangsdaten über dein Terminal ab (verwende `./postctl` für das lokale Binary):

```bash
# App-ID eintragen
./postctl config set threads.app_id "DEINE_APP_ID"

# App Secret eintragen
./postctl config set threads.app_secret "DEIN_APP_SECRET"
```

Überprüfe deine Konfiguration:
```bash
./postctl config show
```

---

## 7. Verbindung herstellen (Auth Flow)

Führe den Auth-Flow aus:

```bash
./postctl auth threads
```

1. Dein Standardbrowser öffnet sich und leitet dich zum Threads-Anmeldedialog weiter.
2. Melde dich mit deinem Threads-Account an und erlaube den Zugriff auf deine Profildaten und Medien (`threads_basic` und `threads_content_publish`).
3. Nach dem Login leitet dich Meta zurück zu `https://localhost:8753/callback`.
4. **Umgang mit der SSL-Warnung im Browser:**
   * Da der lokale Server von `postctl` zur Verschlüsselung ein temporäres, selbstsigniertes SSL-Zertifikat nutzt, wird dein Browser die Warnung **„Dies ist keine sichere Verbindung“** (oder `NET::ERR_CERT_AUTHORITY_INVALID`) anzeigen.
   * Das ist völlig normal und sicher, da die Kommunikation nur lokal auf deinem Mac stattfindet.
   * Klicke im Browser-Fenster auf **„Erweitert“** (oder *Details anzeigen*) und wähle **„Weiter zu localhost (unsicher)“**.
5. Sobald du darauf klickst, empfängt `postctl` das Zugriffstoken, tauscht es im Hintergrund in ein 60 Tage gültiges Long-Lived Token um und speichert es verschlüsselt in der SQLite-Datenbank.
6. Im Terminal erscheint die Erfolgsmeldung! Du bist nun bereit, Beiträge auf Threads zu posten.
