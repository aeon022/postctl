# TikTok (Entwurfskanal)

postctl legt Videos über die Content Posting API (Scope `video.upload`) in den **TikTok-Posteingang** des Kontos. Die Person öffnet die Benachrichtigung in der App, ergänzt Beschreibung und Ton und veröffentlicht selbst. Direct Post (automatisch live) erfordert eine von TikTok geprüfte App; bis dahin sind alle API-Posts ohnehin nur privat sichtbar.

## Einrichtung
1. App auf developers.tiktok.com anlegen, **Login Kit** und **Content Posting API** hinzufügen, Scope `video.upload`.
2. Redirect URI `http://127.0.0.1:8753/callback` eintragen (falls TikTok sie nicht zulässt, Fehlermeldung beim Login beachten).
3. Solange die App nicht geprüft ist: das TikTok-Konto als Target User der Sandbox eintragen.
4. `postctl config setup tiktok` (oder `config set tiktok.client_key …` / `tiktok.client_secret …`), dann `postctl auth tiktok`.

## Post-Datei
```
---
platform: tiktok
type: single
schedule: 2026-11-02 18:00
images:
  - https://cdn.jsdelivr.net/gh/aeon022/bruecke-media@<commit>/video/…/clip.mp4
---
Beschreibung (wird von der Inbox-API nicht übertragen, nur als Merkzettel gespeichert)
```
- Genau **ein** Video (`.mp4`/`.mov`, URL oder lokaler Pfad), bis 64 MB (ein Chunk).
- Fotos gehen nicht: die Foto-API verlangt eine bei TikTok verifizierte URL-Domain.
- Limits laut TikTok: 6 Requests/Minute je Token, höchstens 5 offene Entwürfe pro 24 h (`spam_risk_too_many_pending_share`).
- Löschen per API nicht möglich; Entwurf in der App verwerfen.
