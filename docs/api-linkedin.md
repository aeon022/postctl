# LinkedIn API Setup Guide für postctl

Dieses Dokument beschreibt Schritt für Schritt, wie du eine LinkedIn-Entwickler-App erstellst, um automatisiert Beiträge über `postctl` zu veröffentlichen.

---

## 1. Entwickler-App auf LinkedIn erstellen

1. Gehe zum [LinkedIn Developer Portal](https://linkedin.com/developers).
2. Melde dich mit deinem persönlichen LinkedIn-Konto an.
3. Klicke auf **Create App**:
   * **App Name**: Gib deiner App einen Namen (z. B. `postctl-publisher`).
   * **LinkedIn Page**: Verknüpfe die App mit deiner LinkedIn-Unternehmensseite (falls vorhanden) oder erstelle eine temporäre Seite.
   * **App Logo**: Lade ein beliebiges quadratisches Bild als Logo hoch.
   * **Legal Terms**: Stimme den Bedingungen zu und klicke auf **Create app**.

---

## 2. Produkte hinzufügen (Sehr wichtig!)

Standardmäßig hat eine neue LinkedIn-App keine Berechtigungen für Beitrags-Postings. Du musst die passenden API-Produkte aktivieren:

1. Gehe in deiner App zum Reiter **Products**.
2. Suche nach **Share on LinkedIn** und klicke auf **Request access** (dies wird in der Regel sofort und automatisch genehmigt).
3. Suche nach **Sign In with LinkedIn v2** (oder **Sign In with LinkedIn**) und aktiviere auch dieses Produkt, um die Profil-IDs abzurufen.

---

## 3. Authentifizierungs-Einstellungen konfigurieren

1. Gehe zum Reiter **Auth**.
2. Scrolle zum Bereich **OAuth 2.0 settings**:
   * Füge unter **Authorized Redirect URLs** folgenden Callback hinzu:
     `http://localhost:8753/callback`
   * Klicke auf **Update**.

---

## 4. Client ID & Client Secret kopieren

1. Bleibe im Reiter **Auth**.
2. Kopiere im Bereich **Application credentials** die **Client ID** und das **Client Secret**.

---

## 5. In `postctl` konfigurieren

Nutze das CLI, um die Zugangsdaten in deiner `config.yaml` zu speichern:

```bash
# Client ID konfigurieren
postctl config set linkedin.client_id "DEINE_CLIENT_ID"

# Client Secret konfigurieren
postctl config set linkedin.client_secret "DEIN_CLIENT_SECRET"
```

Überprüfe die Einstellungen:
```bash
postctl config show
```

---

## 6. Verbindung herstellen (Auth Flow)

Starte den Authentifizierungs-Flow:

```bash
postctl auth linkedin
```

1. Es öffnet sich automatisch ein Browserfenster, das dich auffordert, deiner App den Zugriff auf dein LinkedIn-Profil zu erlauben.
2. Nach Klick auf **Zulassen/Allow** wirst du zum lokalen Webserver weitergeleitet.
3. Das CLI fängt das Token ab, verschlüsselt es mit AES-256-GCM und speichert es in der lokalen SQLite-Datenbank.
4. Du bist nun bereit, Beiträge auf LinkedIn zu posten!
