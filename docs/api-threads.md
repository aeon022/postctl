# Threads (Meta) API Setup Guide für postctl

Dieses Dokument beschreibt, wie du Zugriff auf die offizielle Threads API erhältst und deine Zugangsdaten für `postctl` einrichtest.

---

## 1. Meta Developer Account & App erstellen

1. Gehe zum [Meta Developer Portal](https://developers.facebook.com).
2. Registriere dich als Meta-Entwickler, falls noch nicht geschehen.
3. Klicke auf **App erstellen** (Create App):
   * Wähle als App-Typ **Anderes** (Other) oder den Typ, der dir Zugriff auf APIs ermöglicht.
   * Vergib einen Namen für deine Anwendung (z. B. `postctl-threads-app`).
   * Verknüpfe deine E-Mail-Adresse und klicke auf **App erstellen**.

---

## 2. Threads API hinzufügen

1. Scrolle im Dashboard deiner App nach unten zu **Produkt hinzufügen** (Add Products to Your App).
2. Suche nach **Threads API** und klicke auf **Einrichten** (Set Up).

---

## 3. Redirect URI konfigurieren

1. Klicke in der linken Navigationsleiste unter *Threads API* auf **Threads-Einstellungen** (Threads Settings).
2. Trage im Feld **Gültige OAuth-Redirect-URIs** (Valid OAuth Redirect URIs) folgenden Wert ein:
   `http://localhost:8753/callback`
3. Speichere die Änderungen.

---

## 4. App ID & App Secret abrufen

1. Gehe in der linken Navigationsleiste auf **App-Einstellungen** (App Settings) ➔ **Standard** (Basic).
2. Kopiere die **App-ID** (App ID) und den **App-Geheimschlüssel** (App Secret / benötigt Meta-Passworteingabe).

---

## 5. In `postctl` konfigurieren

Speichere deine Threads-Zugangsdaten in der Konfigurationsdatei:

```bash
# App-ID eintragen
postctl config set threads.app_id "DEINE_APP_ID"

# App Secret eintragen
postctl config set threads.app_secret "DEIN_APP_SECRET"
```

Überprüfe deine Konfiguration:
```bash
postctl config show
```

---

## 6. Verbindung herstellen (Auth Flow)

Starte die Authentifizierung:

```bash
postctl auth threads
```

1. Dein Standardbrowser öffnet sich und leitet dich zum Anmeldedialog von Instagram/Threads weiter.
2. Melde dich an und erlaube deiner App den Zugriff (`threads_basic` und `threads_content_publish`).
3. Nach erfolgreicher Bestätigung fängt `postctl` das Zugriffstoken ab und speichert es verschlüsselt ab.
4. Fertig! Beiträge können nun auf Threads veröffentlicht werden.
