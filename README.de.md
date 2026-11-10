# Rezeptplaner

[English](README.md) · **Deutsch**

Rezepte, Essensplan und Einkaufsliste für Home Assistant. Der Rezeptplaner läuft als App (Add-on) auf deinem Home Assistant, hält deine Rezepte an einem Ort, plant das Essen der Woche und schickt die Einkaufsliste an Bring. Er ist so schlank gebaut, dass ein Home Assistant Green ihn kaum bemerkt. Android-App, Dashboard-Karte und eine Claude-Skill für fotografierte Rezepte gibt es optional dazu.

Der Rezeptplaner ist ein Hobbyprojekt und kostenlos.

<!--
  Screenshots: Block einkommentieren, sobald die Dateien in docs/screenshots/
  liegen (Liste: docs/SCREENSHOTS.md). Die englischen Bilder gelten auch hier.

<p>
  <img src="docs/screenshots/web-meal-plan.png" alt="Essensplan in der Weboberfläche" width="49%">
  <img src="docs/screenshots/web-recipe.png" alt="Rezept mit Portionsrechner" width="49%">
</p>
<p>
  <img src="docs/screenshots/web-shopping.png" alt="Einkaufsliste vor dem Senden an Bring" width="49%">
  <img src="docs/screenshots/web-import-preview.png" alt="Import-Vorschau einer Datei der Claude-Skill" width="49%">
</p>
<p>
  <img src="docs/screenshots/android-meal-plan.png" alt="Android-App: Essensplan" width="24%">
  <img src="docs/screenshots/android-recipe.png" alt="Android-App: Kochansicht" width="24%">
  <img src="docs/screenshots/android-shopping.png" alt="Android-App: Einkauf" width="24%">
  <img src="docs/screenshots/android-photo-import.png" alt="Android-App: Foto-Import" width="24%">
</p>
-->

**Videos:** Ein deutsches und ein englisches Erklärvideo folgen in Kürze.
<!-- TODO Videos: Zeile oben ersetzen durch
**Videos:** [Erklärvideo auf Deutsch](https://…) · [Walkthrough in English](https://…)
-->

## Funktionen

- **Rezepte:** von einer Rezeptseite importieren (schema.org-Rezeptdaten, z. B. chefkoch.de, auch englische, französische, spanische und italienische Seiten), per Foto (Android-App) oder aus einer Datei; frei anlegen, bearbeiten und abwandeln; Portionsrechner und Kochansicht
- **Essensplan:** frei scrollbare Tage, Planung beliebig weit im Voraus, Mittag, Abend und optional Kaffee, Verschieben und Tauschen per Drag and Drop, Menüs mit mehreren Gängen
- **Einkaufsliste** für die aktuelle Einkaufswoche, Übertragung an **Bring** über Home Assistant; Vorräte bleiben automatisch weg
- **Fünf Sprachen:** Deutsch, Englisch, Französisch, Spanisch und Italienisch
- **Home-Assistant-Integration (optional):** Essensplan-Kalender, Sensor „Heute“ und Dashboard-Karte – eigenes Repository [rezeptplaner-ha](https://github.com/Thomas1607Haja/rezeptplaner-ha)
- **Android-App (optional):** Rezepte und Plan auch offline lesen, planen, Rezepte anlegen und bearbeiten, Foto-Import mit Texterkennung auf dem Handy, Rezepte als Link oder PDF teilen, Einkauf an Bring senden
- **Claude-Skill (optional):** macht aus Fotos von Rezeptkarten, Kochbuchseiten oder Zetteln eine Importdatei; fehlt ein Foto vom Gericht, zeichnet sie eine Illustration

## Voraussetzungen

| Was | Wofür | Hinweis |
|---|---|---|
| **Home Assistant OS** oder **Home Assistant Supervised** | alles | Apps (Add-ons) brauchen den Supervisor. Home Assistant Container und Core können keine Apps ausführen. |
| **ARM64-Hardware** | die App | z. B. Home Assistant Green, Raspberry Pi 4 oder 5. x86-64 (Mini-PCs, die meisten virtuellen Maschinen) wird noch nicht unterstützt. |
| Home Assistant **2026.11** oder neuer | empfohlen | Ältere Versionen gehen auch; die Integration kommt dann über HACS statt über den Marketplace. |
| **Android 10** oder neuer | Android-App (optional) | Installation und Updates mit [Obtainium](https://github.com/ImranR98/Obtainium) aus den GitHub-Releases. |
| **Bring**-Konto | Einkaufsliste (optional) | Über die Bring-Integration von Home Assistant. |
| **Claude**-Konto mit Skills | Claude-Skill (optional) | Um Rezepte aus Fotos zu erfassen. |

## Installation

### 1. App installieren

[![Home Assistant öffnen und die Rezeptplaner-App anzeigen (fügt bei Bedarf das Repository hinzu).](https://my.home-assistant.io/badges/supervisor_app.svg)](https://my.home-assistant.io/redirect/supervisor_app/?app=45a92d34_rezeptplaner&repository_url=https%3A%2F%2Fgithub.com%2FThomas1607Haja%2FRezepteverwaltung_Essenplan)

1. Auf den Button klicken. Beim ersten Mal fragt my.home-assistant.io nach der Adresse deines Home Assistant. Das Hinzufügen des Repositorys bestätigen.
2. „Rezeptplaner“ **installieren**, dann **starten** und **In Seitenleiste anzeigen** einschalten.

Erscheint die App nicht, zuerst [das Repository hinzufügen](https://my.home-assistant.io/redirect/supervisor_add_addon_repository/?repository_url=https%3A%2F%2Fgithub.com%2FThomas1607Haja%2FRezepteverwaltung_Essenplan) – oder unter **Einstellungen → Apps → App-Store → ⋮ → Repositories** `https://github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan` eintragen.

### 2. Erste Schritte

**Rezeptplaner** in der Seitenleiste öffnen. Unter **Einstellungen → Haushalt** Haushaltsgröße, Einkaufstag und auf Wunsch eine feste Sprache einstellen (sonst gilt die Sprache von Home Assistant). Dann unter **Rezepte** das erste Rezept anlegen – zum Beispiel, indem du die Adresse einer Rezeptseite einfügst.

### 3. Einkaufsliste an Bring senden (optional)

1. In Home Assistant die Bring-Integration einrichten:

   [![Home Assistant öffnen und die Einrichtung der Bring-Integration starten.](https://my.home-assistant.io/badges/config_flow_start.svg)](https://my.home-assistant.io/redirect/config_flow_start/?domain=bring)

2. Im Rezeptplaner unter **Einstellungen → Haushalt → Bring-Liste** die Liste wählen und speichern.
3. Gerichte planen, **Einkauf** öffnen und **An Bring senden** wählen.

### 4. Kalender, Sensor und Dashboard-Karte (optional)

1. Die Integration im Marketplace öffnen und **Rezeptplaner** herunterladen:

   [![Home Assistant öffnen und dieses Repository im Marketplace anzeigen.](https://my.home-assistant.io/badges/marketplace_repository.svg)](https://my.home-assistant.io/redirect/marketplace_repository/?owner=Thomas1607Haja&repository=rezeptplaner-ha&category=integration)

   Vor Home Assistant 2026.11 öffnet der Button stattdessen HACS.
2. Home Assistant neu starten.
3. „Rezeptplaner“ erscheint unter **Einstellungen → Geräte & Dienste** als **Entdeckt** – bestätigen oder die Einrichtung hier starten:

   [![Home Assistant öffnen und die Einrichtung der Rezeptplaner-Integration starten.](https://my.home-assistant.io/badges/config_flow_start.svg)](https://my.home-assistant.io/redirect/config_flow_start/?domain=rezeptplaner)

4. Die Seite im Browser neu laden und die Karte **Essensplan** zu einem Dashboard hinzufügen.

Einstellungen der Karte und Details: [rezeptplaner-ha](https://github.com/Thomas1607Haja/rezeptplaner-ha/blob/main/README.de.md). Dieselben Links stehen in der Weboberfläche unter **Einstellungen → Home Assistant**.

### 5. Android-App (optional)

In der Weboberfläche **Einstellungen → App installieren** öffnen:

- **Schritt 1** zeigt QR-Codes, um [Obtainium](https://github.com/ImranR98/Obtainium) zu installieren und den Rezeptplaner dort einzutragen. Obtainium installiert die App aus den GitHub-Releases und hält sie aktuell. Android fragt einmal, ob Obtainium Apps installieren darf.
- **Schritt 2** verbindet das Handy: die Adresse eintragen, unter der das Handy den App-Zugang erreicht, einen Schlüssel für das Gerät erzeugen und den QR-Code in der App scannen.

Die App spricht mit einem eigenen **App-Zugang** (Port 8100), nicht mit Home Assistant. Für unterwegs zuerst [Sicherheit](#sicherheit) lesen.

### 6. Claude-Skill (optional)

Die Skill unter **Einstellungen → App installieren → Claude-Skill** herunterladen und in Claude unter **Einstellungen → Fähigkeiten → Skills** hochladen. Dann Fotos von Rezepten in einen Chat ziehen. Die erzeugte ZIP-Datei per Teilen an die Android-App schicken oder in der Weboberfläche unter **Rezepte → Datei importieren** hochladen. Gespeichert wird erst nach einer Vorschau mit Dubletten, unbekannten Kategorien und Hinweisen. Stammt die Datei von einer älteren Version der Skill, bietet die Vorschau die neue an.

- Die Skill erfasst Rezepte **in der Sprache der Vorlage** und übersetzt oder rechnet keine Einheiten um.
- Die Kategorien müssen zum Rezeptplaner passen. Nach eigenen Änderungen die Liste kopieren (**Einstellungen → Kategorien → Kategorieliste für Claude kopieren**) und in den Chat einfügen.
- Format: [docs/importformat.md](docs/importformat.md).

## Sicherheit

- **Weboberfläche:** nur über Home Assistant erreichbar (Ingress). Die Anmeldung übernimmt Home Assistant; sonst ist nichts offen.
- **App-Zugang (Port 8100):** nur für die Android-App. Jede Anfrage braucht einen **API-Schlüssel**. Schlüssel entstehen je Gerät in der Weboberfläche, werden einmal als QR-Code gezeigt, gespeichert wird nur ein Hash. Ein verlorenes Handy sperrst du jederzeit mit **Sperren** unter **Einstellungen → App installieren**.
- **Port 8100 (und überhaupt keinen Port von Home Assistant) nie direkt ins Internet freigeben.** Für unterwegs einen **Tunnel** (z. B. die Cloudflared-App) oder ein **VPN** (z. B. WireGuard oder Tailscale) ins Heimnetz nutzen.
- **Optional zusätzlich Cloudflare Access:** Mit einem Tunnel kann Cloudflare ein Service Token verlangen, bevor eine Anfrage überhaupt bei dir zu Hause ankommt. Ein gestohlener API-Schlüssel allein nützt dann nichts. Die Android-App kann das Token mitschicken.
- Schritt für Schritt zu Tunnel, Cache-Regel und Cloudflare Access: [docs/APP-ZUGANG.md](docs/APP-ZUGANG.md).
- Die Integration bekommt über die Discovery von Home Assistant einen eigenen Schlüssel, der nur lesen darf und nur im lokalen Netz funktioniert.
- Beim Import von einer Rezeptseite lädt der Rezeptplaner Seite und Bild selbst; Adressen im Heimnetz sind gesperrt.

## Sprachen

Weboberfläche, Android-App, Integration und Karte gibt es auf **Deutsch, Englisch, Französisch, Spanisch und Italienisch**. Fehlt eine Sprache oder ein einzelner Text, gilt Englisch.

- **Weboberfläche:** folgt der Systemsprache von Home Assistant (Einstellungen → System → Allgemein), sonst der Sprache des Browsers. **Einstellungen → Haushalt → Sprache** legt eine feste Sprache für den ganzen Haushalt fest.
- **Android-App:** folgt der Sprache des Handys. Ab Android 13 lässt sich die Sprache nur für diese App wählen (Android-Einstellungen → Apps → Rezeptplaner → Sprache).
- **Integration und Karte:** Die Integration nimmt die Systemsprache von Home Assistant, die Karte die Sprache des angemeldeten Nutzers.
- **Inhalte werden nie übersetzt:** Rezepte, Zutaten, Notizen und eigene Kategorien bleiben, wie sie eingegeben wurden. Eine neue Installation legt Startkategorien und Vorratsliste in der Sprache von Home Assistant an.

Französisch, Spanisch und Italienisch hat noch kein Muttersprachler geprüft – manches kann holpern. **Korrekturen und neue Sprachen sind sehr willkommen:** siehe [CONTRIBUTING.md](CONTRIBUTING.md).

## Hilfe, Ideen und Mitmachen

- **Fehler und Wünsche:** [Issues](https://github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/issues) – zur Integration oder Karte bitte bei [rezeptplaner-ha](https://github.com/Thomas1607Haja/rezeptplaner-ha/issues).
- **Fragen und Ideen:** [Discussions](https://github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/discussions)
- **Übersetzungen und Code:** [CONTRIBUTING.md](CONTRIBUTING.md)
- **Neuigkeiten:** [CHANGELOG.md](CHANGELOG.md)

Issues und Discussions dürfen gern auf Deutsch sein.

## Technik

| Teil | Technik |
|---|---|
| Server | Go mit SQLite, läuft als Home-Assistant-App |
| Weboberfläche | Svelte, im Server eingebettet, über die Seitenleiste von Home Assistant |
| Integration und Dashboard-Karte | Python und JavaScript, eigenes Repository [rezeptplaner-ha](https://github.com/Thomas1607Haja/rezeptplaner-ha) |
| Android-App | Kotlin, Jetpack Compose |
| Claude-Skill | Python-Skripte und Anleitung in `skills/rezeptplaner-import` |

```
rezeptplaner/   Home-Assistant-App (Konfiguration, Changelog, Doku in Home Assistant)
server/         Go-Backend
web/            Weboberfläche
android/        Android-App
skills/         Claude-Skill „rezeptplaner-import“ (mit Tests)
docs/           Planung, Importformat, Übersetzungen, App-Zugang
```

Die vollständige Planung – Architektur, Fachregeln, Datenmodell, Oberflächen und Projektphasen – steht in [docs/PLANUNG.md](docs/PLANUNG.md).

## Unterstützen

Der Rezeptplaner ist kostenlos. Wenn er dir gefällt, freue ich mich über einen Kaffee.

<a href="https://buymeacoffee.com/rezeptplaner"><img src="https://cdn.buymeacoffee.com/buttons/v2/default-yellow.png" alt="Buy Me a Coffee" height="40"></a>

## Lizenz

[MIT](LICENSE)
