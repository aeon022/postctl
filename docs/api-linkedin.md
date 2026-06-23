# LinkedIn API Setup Guide für postctl

Dieses Dokument beschreibt Schritt für Schritt, wie du eine LinkedIn-Entwickler-App erstellst, diese auf das moderne OpenID Connect (OIDC) migrierst und deine Zugangsdaten für `postctl` einrichtest.

---

## 1. Entwickler-App auf LinkedIn erstellen

1. Gehe zum [LinkedIn Developer Portal](https://linkedin.com/developers).
2. Melde dich mit deinem persönlichen LinkedIn-Konto an.
3. Klicke auf **Create App**:
   * **App Name**: Gib deiner App einen Namen (z. B. `postctl-publisher`).
   * **LinkedIn Page**: Verknüpfe die App mit deiner LinkedIn-Unternehmensseite oder erstelle eine temporäre Seite (Pflichtfeld).
   * **App Logo**: Lade das generierte Logo aus `postctl/icons` (oder ein anderes quadratisches Bild) hoch.
   * **Legal Terms**: Stimme den Bedingungen zu und klicke auf **Create app**.

---

## 2. Produkte hinzufügen (Sehr wichtig!)

Standardmäßig hat eine neue LinkedIn-App keine Berechtigungen für Beitrags-Postings. Seit August 2023 hat LinkedIn die alten Berechtigungen (`r_liteprofile`) abgelöst. Du musst die folgenden Produkte aktivieren:

1. Gehe in deiner App zum Reiter **Products**.
2. Suche nach **Share on LinkedIn** und klicke auf **Request access** (dies wird in der Regel sofort genehmigt).
3. Suche nach **Sign In with LinkedIn using OpenID Connect** (nicht das veraltete *Sign In with LinkedIn*) und aktiviere dieses, um den modernen OIDC-Login freizuschalten.

---

## 3. Authentifizierungs-Einstellungen konfigurieren

1. Gehe zum Reiter **Auth**.
2. Scrolle zum Bereich **OAuth 2.0 settings**:
   * Füge unter **Authorized Redirect URLs** folgenden Callback hinzu:
     `http://localhost:8753/callback`
   * Klicke auf **Update**.
3. Überprüfe im Bereich **OAuth 2.0 scopes**, ob dir nun folgende Scopes angezeigt werden:
   * `openid`, `profile`, `w_member_social`, `email`.

---

## 4. Client ID & Client Secret kopieren

1. Bleibe im Reiter **Auth**.
2. Kopiere im Bereich **Application credentials** die **Client ID** und das **Client Secret**.

---

## 5. In `postctl` konfigurieren

Nutze das CLI, um die Zugangsdaten in deiner `config.yaml` zu speichern (nutze das lokale Binary `./postctl`):

```bash
# Client ID konfigurieren
./postctl config set linkedin.client_id "DEINE_CLIENT_ID"

# Client Secret konfigurieren
./postctl config set linkedin.client_secret "DEIN_CLIENT_SECRET"
```

Überprüfe die Einstellungen:
```bash
./postctl config show
```

---

## 6. Verbindung herstellen (Auth Flow)

Starte den Authentifizierungs-Flow:

```bash
./postctl auth linkedin
```

1. Es öffnet sich automatisch ein Browserfenster, das dich auffordert, deiner App den Zugriff auf dein LinkedIn-Profil zu erlauben.
2. Nach Klick auf **Zulassen/Allow** wirst du zum lokalen Webserver weitergeleitet.
3. Das CLI fängt das Token über den OIDC-Flow ab, ermittelt die Benutzer-ID (über `/v2/userinfo`), verschlüsselt das Access Token und speichert es in der lokalen SQLite-Datenbank.
4. Du bist nun bereit, Beiträge auf LinkedIn zu veröffentlichen!
