# postctl — Frequently Asked Questions (FAQ)

This document contains answers to common questions about `postctl`'s architecture, security, scheduling, licensing, and AI integrations.

*Dieses Dokument enthält Antworten auf häufig gestellte Fragen zu postctl in Englisch und Deutsch.*

---

## English Version

### Q1: Is postctl really free? What are the limitations?
**A:** Yes, the core of `postctl` is completely open-source (MIT licensed) and free to use. You can connect up to **2 social networks** (e.g., one Twitter/X account and one LinkedIn account) with unlimited posts and drafts. To connect more than 2 accounts, you can buy a Pro lifetime license on Polar.sh, which also helps support the development of the project.

### Q2: How does postctl store my credentials? Is it secure?
**A:** Your API keys, tokens, and cookies are stored **locally** on your computer inside a SQLite database (`~/.config/postctl/postctl.db`). 
- No credentials ever leave your machine or get sent to third-party servers (except directly to the social network APIs during authentication and posting).
- You can export your configuration and database into a single backup file (`backup.bin`). The backup is encrypted locally using **AES-256-GCM** and key-stretched using **PBKDF2** with a master password of your choice, making it safe to check into private Git repositories.

### Q3: What happens if my computer is closed or offline at the scheduled post time?
**A:** Because `postctl` is a local-first application, it cannot publish posts if your computer is shut down or asleep. However, we handle this in two ways:
1. **Auto-Catchup (Local):** As soon as you open your computer and launch the TUI or start the daemon, `postctl` detects any missed scheduled posts and publishes them instantly.
2. **Cloud Daemon (24/7):** You can cross-compile `postctl` for Linux (`GOOS=linux GOARCH=amd64 go build -o postctl-linux`) and run it as a lightweight background process on a $4/mo VPS or Raspberry Pi.

### Q4: Which Twitter/X authentication methods are supported?
**A:** `postctl` supports **both** official OAuth 2.0 API credentials and cookie-based web session login. Since Twitter/X API pricing is high (~$100/month) for basic access, the cookie-based bypass simulates standard browser interactions (including randomized delays and automated headless Google Chrome via `chromedp`) to post for free. If you have a developer account, we highly recommend using the official API for maximum stability.

### Q5: How do I sync settings between multiple devices (e.g., MacBook and Mac Studio)?
**A:** You can easily migrate your configuration and databases:
1. Run `./postctl config export -o backup.bin` on your source machine and enter a master password.
2. Transfer `backup.bin` to your target machine.
3. Run `./postctl config import -f backup.bin` on your target machine and enter the same password.
4. Alternatively, you can symlink the `~/.config/postctl/` directory to your iCloud, Dropbox, or Syncthing folder.

### Q6: Can my local AI agent (like Claude Engineer or GPT) run postctl?
**A:** Yes, `postctl` was built with the "AI-as-Operator" principle in mind. All CLI commands run non-interactively without prompt blocks. Adding the `--format json` flag forces commands to output machine-readable JSON, and the `--dry-run` flag lets the AI test the entire pipeline safely before asking you for approval to post.

---

## Deutsche Version

### F1: Ist postctl wirklich kostenlos? Was sind die Einschränkungen?
**A:** Ja, der Kern von `postctl` ist komplett Open-Source (MIT-Lizenz) und kostenlos. Du kannst bis zu **2 soziale Netzwerke** (z. B. ein Twitter/X- und ein LinkedIn-Konto) mit unbegrenzten Beiträgen und Entwürfen verbinden. Wenn du mehr als 2 Konten verknüpfen möchtest, kannst du eine Pro-Lizenz auf Lebenszeit auf Polar.sh erwerben, was gleichzeitig die Weiterentwicklung des Projekts unterstützt.

### F2: Wie speichert postctl meine Zugangsdaten? Ist das sicher?
**A:** Deine API-Keys, Tokens und Cookies werden **lokal** auf deinem Rechner in einer SQLite-Datenbank gespeichert (`~/.config/postctl/postctl.db`). 
- Keine Anmeldedaten verlassen jemals deinen Computer oder werden an Drittanbieter-Server gesendet (außer direkt an die APIs der sozialen Netzwerke während der Übermittlung).
- Du kannst deine Konfiguration und Datenbank in eine einzige Backup-Datei (`backup.bin`) exportieren. Das Backup wird lokal mit **AES-256-GCM** verschlüsselt und per **PBKDF2** mit einem Master-Passwort deiner Wahl gesichert, sodass du es sicher in privaten Git-Repositories ablegen kannst.

### F3: Was passiert, wenn mein Computer zum geplanten Zeitpunkt zugeklappt oder offline ist?
**A:** Da `postctl` eine lokale Anwendung ist, kann sie keine Beiträge veröffentlichen, wenn dein Rechner ausgeschaltet ist. Wir lösen das auf zwei Wegen:
1. **Automatisches Nachholen (Lokal):** Sobald du deinen Mac aufklappst und die TUI startest oder den Daemon ausführst, erkennt `postctl` verpasste geplante Beiträge und veröffentlicht diese sofort nachträglich.
2. **Cloud-Daemon (24/7):** Du kannst `postctl` für Linux kompilieren (`GOOS=linux GOARCH=amd64 go build -o postctl-linux`) und es als ressourcenschonenden Hintergrundprozess auf einem 4$-VPS oder einem Raspberry Pi laufen lassen.

### F4: Welche Authentifizierungs-Methoden für Twitter/X werden unterstützt?
**A:** `postctl` unterstützt **sowohl** die offizielle OAuth 2.0 API als auch die Cookie-basierte Web-Session-Anmeldung. Da Twitter/X hohe API-Preise (ca. 100 $/Monat) verlangt, bietet der Cookie-Bypass eine kostenlose Alternative, die typische Browser-Aktionen (inklusive zufälliger Verzögerungen und headless Google Chrome via `chromedp`) simuliert. Wenn du einen Entwickler-Account hast, empfehlen wir die Nutzung der offiziellen API für maximale Stabilität.

### F5: Wie synchronisiere ich Einstellungen zwischen MacBook und Mac Studio?
**A:** Du kannst deine Konfiguration und Datenbanken ganz einfach übertragen:
1. Führe `./postctl config export -o backup.bin` auf dem Quell-Mac aus und vergib ein Master-Passwort.
2. Kopiere die Datei `backup.bin` auf den neuen Mac.
3. Führe dort `./postctl config import -f backup.bin` aus und gib dasselbe Passwort ein.
4. Alternativ kannst du das Verzeichnis `~/.config/postctl/` in deinen iCloud-, Dropbox- oder Syncthing-Ordner verlinken (Symlink).

### F6: Kann mein lokaler KI-Assistent (wie Claude Engineer oder GPT) postctl steuern?
**A:** Ja, `postctl` wurde speziell nach dem "KI-als-Operator"-Prinzip entwickelt. Alle CLI-Befehle laufen ohne interaktive Abfragen ab. Der Parameter `--format json` gibt strukturierte Daten aus, die LLMs leicht parsen können, und über `--dry-run` kann die KI das gesamte Posting testen, bevor sie dich im Chat um Freigabe bittet.
