# Rezeptplaner – Projektplanung

Stand: Oktober 2026

Dieses Dokument fasst alle Entscheidungen der Planungsphase zusammen. Es dient als Referenz für die Entwicklung und wird bei Änderungen fortgeschrieben.

**Repository (öffentlich):** https://github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan

---

## 1. Ziel

Ein selbst gebautes, schlankes System zur Verwaltung von Kochrezepten und zur Wochenplanung des Abendessens für die Familie:

- Rezepte per URL (z. B. chefkoch.de) oder per Foto importieren, manuell anlegen und frei bearbeiten
- Essensplan mit frei scrollbaren Tagen, Planung beliebig weit im Voraus
- Einkaufsliste für einen festen Einkaufszeitraum erzeugen und an die Bring-App übertragen
- Anzeige des aktuellen Essensplans auf dem Home-Assistant-Dashboard
- Zentrale Datenhaltung als Add-on auf einem Home Assistant Green, dazu Weboberfläche und Android-App

**Grundsatz:** Alles wird selbst gebaut. Das System muss so schlank sein, dass es den Home Assistant Green neben seiner eigentlichen Aufgabe kaum belastet.

---

## 2. Architektur

```
┌──────────────────────────┐       ┌───────────────────────────────┐
│ Android-App (Kotlin)     │       │ Home Assistant Green          │
│ - ML Kit Texterkennung   │ HTTPS │ ┌───────────────────────────┐ │
│ - Offline-Speicher (Room)├──────►│ │ Add-on: Rezeptplaner      │ │
│ - API-Schlüssel          │ über  │ │ - Go-Server + REST-API    │ │
└──────────────────────────┘ Cloud-│ │ - SQLite + Bilder (/data) │ │
                             flare │ │ - Weboberfläche (eingeb.) │ │
┌──────────────────────────┐ Tunnel│ └──────────┬────────────────┘ │
│ Browser                  ├──────►│   Ingress  │  Supervisor-API  │
│ (über HA-Seitenleiste)   │       │            ▼                  │
└──────────────────────────┘       │ ┌───────────────────────────┐ │
                                   │ │ HA Core                   │ │
                                   │ │ - Bring-Integration (todo)│ │
                                   │ │ - Rezeptplaner-Integration│ │
                                   │ │ - Dashboard-Karte         │ │
                                   │ └───────────────────────────┘ │
                                   └───────────────────────────────┘
```

### Bausteine und Technik

| Baustein | Technik | Bemerkung |
|---|---|---|
| Server | Go, SQLite (reine Go-Variante, ohne cgo) | Eine kompilierte Datei, sehr geringer RAM-Bedarf, ARM64 |
| Weboberfläche | Svelte 5 (JavaScript) mit Vite, Drag-and-Drop-Bibliothek | Wird statisch gebaut und in die Go-Datei eingebettet; kein Node.js auf dem Green |
| Add-on | Docker-Image für Home Assistant | Ingress für die Seitenleiste, eigener Port für die App, Zugriff auf HA-API über Supervisor |
| Android-App | Kotlin, Jetpack Compose, Room, ML Kit | Offline-Lesen, Teilen-Funktion, Foto-Import |
| HA-Integration | Python (von Home Assistant vorgegeben) | Installierbar über HACS |
| Dashboard-Karte | JavaScript (Lovelace Custom Card) | Installierbar über HACS |

### Zugriff und Sicherheit

- **Weboberfläche:** über Ingress in der HA-Seitenleiste; Anmeldung erfolgt über Home Assistant.
- **Zwei getrennte Zugänge** (seit App 0.4.1): Port 8099 für Ingress nimmt nur Verbindungen vom Ingress-Vermittler (172.30.32.2) an – entscheidend ist die Gegenstelle der TCP-Verbindung, nicht Kopfzeilen wie `X-Ingress-Path`. Port 8100 ist der **App-Zugang**: jede Anfrage nur mit gültigem API-Schlüssel (sonst 401, auch für Health und Bilder), nur `/api/v1/…`, keine Weboberfläche, keine Schlüsselverwaltung. Ein Test durchläuft alle angemeldeten Endpunkte.
- **App:** über einen zusätzlichen Hostnamen im bestehenden Cloudflare-Tunnel (z. B. `rezepte.eure-domain.de`), der auf Port 8100 der App zeigt (intern über den Hostnamen der App, ohne Freigabe nach außen). Anleitung: [APP-ZUGANG.md](APP-ZUGANG.md).
- **API-Schlüssel:** einer pro Gerät (`rp_` + 32 Byte Zufall), nur als SHA-256-Hash gespeichert, einzeln sperrbar; Verwaltung unter Einstellungen → App-Zugang, nur über Ingress. Einrichtung der App per QR-Code (enthält Serveradresse und Schlüssel).
- **Integration:** eigener Schlüssel aus der Discovery, nur GET auf Essensplan, einzelne Rezepte und Bilder, nur im internen Netz (Port 8099).
- **Weboberfläche:** läuft im Ursprung von Home Assistant (Ingress) – ein XSS-Fehler wäre ein Zugriff auf Home Assistant. Daher Content-Security-Policy (Skripte nur vom eigenen Server), `nosniff`, `Referrer-Policy: no-referrer`; Quell-URLs nur `http`/`https`.
- **Import fremder Seiten:** Der Server ruft Rezeptseiten und Bilder nur per http/https und **nur über IPv4** ab; private, lokale und Tailscale-Adressen (100.64.0.0/10) sind gesperrt. So lässt sich der Import nicht nutzen, um Home Assistant oder andere Geräte im Heimnetz anzusprechen (Geräte haben dort oft zusätzlich eine weltweit gültige IPv6-Adresse). Zeitlimit 20 s, Seiten höchstens 5 MB.
- **Cloudflare Access mit Service Token** für den App-Hostnamen empfohlen (Anleitung in [APP-ZUGANG.md](APP-ZUGANG.md)); die Android-App schickt dafür optional `CF-Access-Client-Id` und `CF-Access-Client-Secret` mit.
- **Bilder:** `Cache-Control: private`, damit kein Zwischenspeicher (Cloudflare) sie ohne Schlüssel ausliefert.
- **Laufende Pflege:** GitHub Action „Sicherheit“ (govulncheck, npm audit) bei jedem Push und wöchentlich; Dependabot für Go-Module, npm, GitHub Actions und Docker-Basis-Images; Go-Version in `go.mod` (`toolchain`) und Dockerfile fest.
- **Datei-Import (ZIP/JSON, ab App 0.6.0, Ablauf G):** Pfade beim Entpacken prüfen (kein `..`, keine absoluten Pfade, keine Symlinks – Zip Slip), nur im Speicher entpacken und dabei mitzählen; Grenzen: Upload 30 MB, 200 Einträge, `recipes.json` 2 MB, je Bild entpackt 20 MB, insgesamt entpackt 150 MB, 100 Rezepte (ZIP-Bombe). Bildtyp am Inhalt erkennen, Bilder wie beim Upload neu kodieren (entfernt EXIF/GPS). Hochgeladene Dateien liegen bis zur Bestätigung höchstens 1 Stunde in einem eigenen Ordner.

### Ressourcen auf dem Green

- Server läuft im Leerlauf praktisch ohne CPU-Last.
- Bilder werden einmalig beim Hochladen verkleinert: großes Bild (längste Seite 1600 px) und Vorschaubild (480 px), beide JPEG. Handyfotos werden nach ihrer EXIF-Ausrichtung gedreht. Verkleinert wird durch schrittweises Halbieren, damit ein 12-Megapixel-Foto nur kurz rund 60 MB belegt; es wird immer nur ein Bild gleichzeitig verarbeitet. Bibliothek: `golang.org/x/image` (vom Go-Team, reines Go; Skalieren und WebP).
- Statistik und Statushinweise werden bei Bedarf per Datenbankabfrage berechnet, nicht gespeichert.
- Daten liegen im Datenordner des Add-ons und sind damit Teil der Home-Assistant-Backups.

**Messung nach Phase 1 (2026-10-20, Go-Server ohne Weboberfläche):**

| Messgröße | Wert |
|---|---|
| Programmdatei Linux/ARM64 (`-trimpath -ldflags="-s -w"`) | 10,2 MB (gzip: 4,2 MB) |
| Arbeitsspeicher nach dem Start | 12 MB |
| Arbeitsspeicher mit 500 Rezepten, nach 4000 Abfragen | 24 MB (Spitze 25 MB) |
| CPU im Leerlauf (20 s beobachtet) | 0 |
| Rezeptliste mit 500 Einträgen / Suche mit 3 Kategorien / Einzelrezept | ca. 8 ms / 10 ms / 1,5 ms |
| Datenbank mit 500 Rezepten (je 10 Zutaten, 6 Schritte) | ca. 1 MB |

Die ARM64-Datei wurde nur gebaut; Speicher und Zeiten stammen vom Windows-Build auf dem Entwicklungsrechner. In Phase 3 auf dem Green nachmessen.

**Messung nach Phase 2 (2026-10-21, Server mit eingebetteter Weboberfläche, Bildverarbeitung und URL-Import):**

| Messgröße | Wert |
|---|---|
| Programmdatei Linux/ARM64 (`-trimpath -ldflags="-s -w"`) | 11,4 MB (gzip: 4,7 MB), +1,2 MB gegenüber Phase 1 |
| Weboberfläche (eingebettet) | 145 KB; beim ersten Aufruf 28 KB gzip (JS + CSS), Bearbeiten-Seite wird mit 21 KB gzip nachgeladen |
| Arbeitsspeicher nach dem Start | 13 MB |
| Arbeitsspeicher mit 500 Rezepten, nach 4000 Abfragen | 26 MB |
| Arbeitsspeicher-Spitze beim Hochladen eines 12-Megapixel-Fotos (4,9 MB) | 71 MB, danach wieder 23 MB |
| Arbeitsspeicher nach 3 URL-Importen mit Bild, nach 2 min Leerlauf | 28 MB |
| CPU im Leerlauf (20 s beobachtet) | 0 |
| Rezeptliste mit 500 Einträgen / Suche mit 3 Kategorien / Einzelrezept (curl) | ca. 2,3 ms / 4,2 ms / 2,8 ms |
| Foto hochladen und verkleinern (12 MP) | ca. 290 ms |
| URL-Import Chefkoch / Bild der Quellseite übernehmen | 150–250 ms / 130–180 ms |
| Bilder je Rezept (großes Bild + Vorschaubild) | ca. 300 KB |

Gemessen wie in Phase 1 auf dem Entwicklungsrechner (Windows). Das Testfoto war künstlich erzeugt (Verlauf mit Rauschen) und lässt sich schlechter komprimieren als ein echtes Foto; echte Bilder werden eher kleiner. Die Spitze beim Hochladen passt zur Schätzung oben (rund 60 MB für das Bild). Gemessen wurden Uploads nacheinander; gleichzeitige Uploads wartet eine Sperre ab (immer nur ein Bild in Arbeit), die Spitze sollte also auch dann nicht steigen.

**Messung nach Phase 3 (2026-10-21, App 0.1.1 auf dem Home Assistant Green):**

| Messgröße | Wert |
|---|---|
| Arbeitsspeicher nach 10 min mit Anlegen, Foto-Upload und URL-Import | 15,6 MB |
| Arbeitsspeicher-Spitze seit dem Start (Foto vom Handy hochgeladen) | 66,8 MB |
| Datenbank mit wenigen Testrezepten | 152 KB |
| Bilder für 3 Rezepte (je großes Bild + Vorschaubild) | 558 KB, also ca. 190 KB je Rezept |
| CPU im Leerlauf (Infoseite der App) | 0,1 % |
| RAM-Nutzung laut Infoseite der App | 0,2 % |
| Foto vom Handy hochladen | gefühlt sofort |

Speicherwerte abgelesen in der Systeminfo der Einstellungsseite (VmRSS/VmHWM aus `/proc/self/status`; MB = MiB), CPU und RAM in Prozent auf der Infoseite der App in Home Assistant. Die Dauer des URL-Imports wurde auf dem Green nicht gesondert gemessen. Auf dem Green belegt der Server im Betrieb weniger als auf dem Entwicklungsrechner; die Spitze beim Foto-Upload entspricht der Schätzung. Echte Fotos werden kleiner gespeichert als das künstliche Testfoto der Phase-2-Messung.

**Messung nach Phase 4 (2026-10-21, Version 0.2.0 mit Essensplan):**

| Messgröße | Wert |
|---|---|
| Rezeptliste mit 500 Einträgen (mit Statushinweisen aus dem Plan) | ca. 4,7 ms |
| Essensplan über 5 Wochen, bei 730 Planeinträgen insgesamt | ca. 2,5 ms |
| Arbeitsspeicher mit 500 Rezepten und 730 Planeinträgen | 23 MB |
| Auf dem Green (App 0.2.0, inkl. Ziehen am Handy) | unauffällig |

Zeiten und Speicher auf dem Entwicklungsrechner (Windows) gemessen. Die Statushinweise werden bei jeder Abfrage berechnet; die Rezeptliste ist dadurch etwas langsamer als nach Phase 2, bleibt aber weit unter spürbaren Werten. Auf dem Green wurden keine einzelnen Zahlen erhoben.

**Messung nach Phase 5 (2026-10-21, Version 0.3.0 mit Einkaufsliste und Bring):**

| Messgröße | Wert |
|---|---|
| Programmdatei Linux/ARM64 (`-trimpath -ldflags="-s -w"`) | 12,3 MB (gzip: 5,1 MB) |
| Einkaufsliste mit Differenzvergleich (500 Rezepte, 730 Planeinträge, 7 Gerichte im Zeitraum) | ca. 1,8 ms |
| An Bring senden (Testmodus, 21 Artikel) | ca. 4 ms zuzüglich der Antwortzeit von Home Assistant je Artikel |
| Rezeptliste mit 500 Einträgen / Essensplan über 5 Wochen | ca. 2,7 ms / 1,5 ms |
| Arbeitsspeicher nach 600 Abfragen (Spitze) | 24,3 MB (24,7 MB) |
| CPU im Leerlauf (20 s beobachtet) | 0 |
| Datenbank mit 500 Rezepten und 730 Planeinträgen | 4,8 MB |

Gemessen auf dem Entwicklungsrechner (Windows, Arbeitsspeicher über den Task-Manager-Wert „Arbeitssatz“). Die Einkaufsliste wird bei jedem Abruf neu berechnet; das bleibt weit unter spürbaren Zeiten. Auf dem Green (App 0.3.0) liefen Senden an Bring und Nachsenden ohne spürbare Wartezeit.

**Test nach Phase 6 (2026-10-21, App 0.4.0, Integration 0.4.2 über HACS auf dem Green):** Discovery und Einrichtung per „Entdeckt“ ok; Kalender „Essensplan“ und Sensor „Heute“ zeigen die richtigen Gerichte; Karte mit Bildern und Kochansicht ok; „Öffnen“ führt zur Weboberfläche, auch ohne Eintrag in der Seitenleiste. Nach der Einrichtung die Seite einmal neu laden (F5). Die Integration fragt den Server nur alle 15 Minuten ab; zusätzliche Last auf dem Green ist nicht spürbar.

**Korrektur Karte (2026-10-22, Integration 0.4.3):** Über den Cloudflare-Tunnel verschwand die Karte zeitweise („Konfigurationsfehler“), lokal nie. Ursache: Während eines Neustarts von Home Assistant ist der Pfad der Kartendatei noch nicht angemeldet und liefert 404. Cloudflare speicherte diesen 404 zwischen und gab Browsern ohne eigenen Header die Vorgabe „4 Stunden behalten“ mit. Behoben durch eine Cloudflare-Cache-Regel (Bypass für den HA-Host, Browser-Cache-TTL „Respect Existing Headers“, Anleitung im README), eine Inhalts-Prüfsumme im Versionsparameter (`?v=0.4.3-<hash>`), eigenes Cache-Control für die Kartendatei und eine Karte, die bei Fehlern „Rezeptplaner nicht erreichbar“ zeigt und nach 30 Sekunden neu lädt. Test im Browser und in der Companion-App über den Tunnel erfolgreich, auch nach HA-Neustart.

**Zweite Korrektur Karte (2026-10-22, Integration 0.4.4):** Danach fehlte die Karte weiterhin nach Strg+F5 und in der Companion-App, nach F5 war sie wieder da; die Datei kam jedes Mal fehlerfrei an, `customElements.get('rezeptplaner-card')` war trotzdem `undefined`. Eigentliche Ursache: Das Frontend von Home Assistant (`app.js`) bringt einen Polyfill für „scoped custom element registries“ mit und ersetzt beim Start `window.customElements` und `window.HTMLElement`. Über `add_extra_js_url` lief die kleine Kartendatei bei einem harten Neuladen vor dem großen `app.js` und registrierte sich in der alten Registry, die das Dashboard danach nicht mehr fragt (bei F5 kam `app.js` aus dem Service Worker und lief zuerst). Behoben: Die Integration trägt die Karte als **Lovelace-Ressource** ein (wie HACS; legt sie einmalig an bzw. aktualisiert sie, entfernt Doppelte; im YAML-Modus Ersatz über `add_extra_js_url` mit Hinweis im Protokoll); die Karte meldet sich erst an, wenn `<home-assistant>` definiert ist, bildet ihre Klasse erst dann (gegen das ersetzte `HTMLElement`) und gibt ein Banner mit Version in der Konsole aus. Lokal mit dem echten Polyfill nachgestellt: alte Karte bei „Karte zuerst“ `undefined`, neue in beiden Reihenfolgen registriert.

Build-Befehl (so auch im Dockerfile, dort zusätzlich `-tags timetzdata` und die Version per `-X main.buildVersion`):

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" ./cmd/rezeptplaner
```

---

## 3. Fachliche Regeln

### Rezepte

- Importierte Rezepte werden zu einer normalen, vollständig bearbeitbaren Kopie. Gespeichert wird nur die Quell-URL, kein Originalzustand.
- Beim URL-Import wird auf Dubletten geprüft (gleiche Quell-URL).
- Jedes Rezept hat ein Bild (vom Handy, von der Quellseite oder nachträglich ersetzt).
- Favoriten per Herzchen, setzbar im Rezept und direkt im Essensplan.
- Ein Rezept kann beliebig vielen Kategorien angehören.
- **Varianten:** „Variante speichern“ legt eine Kopie als eigenständiges Rezept an (Titel „<Titel> (Variante)“, nicht doppelt) und öffnet sie zum Bearbeiten; Abbrechen direkt danach verwirft sie wieder. Kopiert werden alle Felder, Zutaten (Reihenfolge, Einkaufsrelevanz), Schritte, Kategorien, Quell-URL und das Bild als eigene Datei – nicht der Favoritenstatus, keine Planeinträge (Statistik daher je Variante). Varianten hängen immer am Ursprungsrezept, auch eine Variante einer Variante (nur eine Ebene, „Rezeptfamilie“). Wird das Ursprungsrezept gelöscht, bleiben die Varianten; die älteste wird neues Ursprungsrezept (vorher Hinweis).

### Zutaten

- Jede Zutat ist ein eigener Baustein: Reihenfolge per Drag and Drop, Menge/Einheit/Name/Notiz bearbeitbar, Häkchen „einkaufsrelevant“.
- Optionale Gruppenüberschriften (z. B. „Für die Soße“).
- Eine leere Menge bedeutet „nicht umrechnen“ (z. B. „1 Prise“, „nach Geschmack“).
- **Globale Vorratsliste** (mit Synonymen): Zutaten darauf werden beim Import/Anlegen automatisch als nicht einkaufsrelevant markiert. Pro Rezept überschreibbar.
  - Die Vorratsliste setzt nur den Vorschlag für neue Zutaten. Die Einkaufsliste richtet sich allein nach dem Häkchen „einkaufsrelevant“; eine Änderung der Vorratsliste ändert bestehende Rezepte nicht von selbst (entschieden 2026-10-21).
  - Ein Knopf „Auf alle Rezepte übertragen“ gleicht bei Bedarf alle Zutaten mit der Vorratsliste ab: im Vorrat → abgehakt, sonst → einkaufsrelevant. Vorher zeigt eine Vorschau jede betroffene Zutat mit Rezept, denn von Hand gesetzte Abweichungen gehen dabei verloren.
  - Verglichen wird der Zutatenname ohne Klammerzusätze und ohne alles nach dem ersten Komma, unabhängig von Groß-/Kleinschreibung und Umlauten („Olivenöl, nativ“ → Öl). Aufzählungen wie „Salz und Pfeffer“ passen, wenn alle Teile im Vorrat sind. Teilwörter zählen nicht („Salzbrezel“ ist kein Salz).
  - Startbestand: Salz, Pfeffer, Öl, Zucker, Wasser.

### Portionen

- Jedes Rezept hat Grundportionen; jeder Planeintrag hat eigene Portionen (Standard: Familiengröße aus den Einstellungen).
- Die Kochansicht rechnet Mengen per Plus/Minus um, ohne das Rezept zu ändern.
- Umgerechnete Mengen werden sinnvoll gerundet (z. B. keine 1,33 Eier).

### Essensplan

- Einträge mit Datum und Slot statt fester Wochen.
- Slots: *Abendessen* (Standard) und optional *Mittagessen* (z. B. für Ferien). Pro Datum und Slot sind mehrere Einträge möglich (z. B. Hauptgericht und Nachtisch); ihre Reihenfolge ist frei wählbar.
- Wird ein Rezept gelöscht, bleiben seine Planeinträge erhalten; der Rezepttitel wird zur Notiz des Eintrags.
- Einträge ohne Rezept sind möglich (z. B. „Essen gehen“, „Reste“).
- Planung beliebig weit im Voraus.

### Einkaufszeitraum

- Ein fester Einkaufstag in den Einstellungen bestimmt den Einkaufszeitraum (z. B. Samstag bis Freitag).
- Die Einkaufsliste umfasst immer nur diesen einen Zeitraum.

### Status eines Planeintrags

| Status | Bedeutung |
|---|---|
| `planned` | Geplant, Zutaten noch nicht übertragen |
| `transferred` | An Bring übertragen; gilt als eingekauft |
| `have_ingredients` | Zutaten sind laut Bestätigung vorhanden |

Es wird nicht ausgelesen, was in Bring tatsächlich abgehakt wurde.

### Verschieben und Tauschen (Drag and Drop)

- Ziehen auf ein Gericht tauscht die beiden Gerichte; Ziehen auf die freie Fläche eines Tages hängt das Gericht dort an.
- Die Prüfung der Warnregeln erfolgt auf dem Server; erst nach Bestätigung wird gespeichert.

| Situation | Reaktion |
|---|---|
| Tausch zweier übertragener Gerichte innerhalb des Einkaufszeitraums | Keine Meldung |
| Gericht mit Status `planned` rutscht in den Einkaufszeitraum | Dialog: „Trotzdem verschieben“ / „Zutaten sind vorhanden“ / „Fehlende Zutaten jetzt an Bring senden“ |
| Übertragenes Gericht wandert aus dem Einkaufszeitraum | Hinweis: Zutaten bereits eingekauft, evtl. verderblich |
| Portionen oder Rezept eines übertragenen Eintrags ändern sich | Hinweis mit Angebot, nur die Differenz nachzusenden |

### Einkaufsliste und Bring

- Grundlage: alle Einträge im Einkaufszeitraum mit Status `planned`.
- Mengen werden nach Portionen skaliert, nicht einkaufsrelevante Zutaten (darunter die vorgeschlagenen Vorratsartikel) entfallen.
- Gleiche Zutaten werden zusammengefasst. Unterschiedliche Einheiten derselben Zutat werden zunächst nicht umgerechnet, sondern gemeinsam angegeben.
  - Gleich heißt: gleicher Name ohne Klammerzusätze, unabhängig von Groß-/Kleinschreibung, Umlauten und einer Pluralendung am letzten Wort („Zwiebel“ = „Zwiebeln“, „Ei“ = „Eier“). Angezeigt wird die längere Schreibweise.
  - Mengen derselben Einheit werden addiert, verschiedene Einheiten mit „+“ verbunden („500 g + 2 EL“). Zutaten ohne Menge stehen nur mit Namen auf der Liste.
  - Stück-Einheiten (ohne Einheit, Stück, Dose, Packung, Bund, Zehe, Glas usw.) werden nach dem Addieren auf ganze Zahlen aufgerundet; andere Mengen werden wie in der Kochansicht gerundet dargestellt.
- Tage vor heute zählen nicht mit (schon gegessen). Neben dem aktuellen lässt sich auch der folgende Einkaufszeitraum abrufen, z. B. am Abend vor dem Einkaufstag (ergänzt 2026-10-21).
- Vor dem Senden lassen sich einzelne Artikel abwählen.
- Übertragung über die Bring-Integration von Home Assistant (To-do-Entität) per Supervisor-API; Artikelname und Mengenangabe getrennt, soweit möglich.
  - Die App hat `homeassistant_api: true` und ruft mit `SUPERVISOR_TOKEN` die Dienste `todo.get_items`, `todo.add_item` und `todo.update_item` unter `http://supervisor/core/api/` auf. Die To-do-Entität wird in den Einstellungen gewählt (`bring_entity`).
  - Die Bring-Integration übernimmt die Beschreibung eines To-do-Eintrags als Spezifikation, also die Mengenangabe unter dem Artikel.
  - Lokal ohne Supervisor: mit `REZEPTPLANER_HA_URL` und `REZEPTPLANER_HA_TOKEN` (langlebiger Token) gegen eine echte Instanz, sonst Testmodus mit einer Liste im Arbeitsspeicher.
- Übertragene Artikel werden protokolliert, um Differenzen nachsenden zu können. Zu viel Gekauftes wird nur angezeigt, nicht aus Bring gelöscht.

### Statistik und Statushinweis

- Statistik: am häufigsten eingeplante Rezepte, filterbar nach Zeitraum (z. B. letzte 3 Monate, gesamt).
- Statushinweis an jedem Rezept:
  - „eingeplant in 5 Tagen“, wenn es in der Zukunft eingeplant ist (hat Vorrang)
  - sonst „zuletzt vor 2 Wochen“
  - sonst „noch nie eingeplant“
- Sichtbar in Rezeptliste, Rezeptansicht und Rezeptauswahl beim Planen.

---

## 4. Datenmodell

Alle Tabellen erhalten zusätzlich `updated_at` für den Abgleich mit der App.

Die einzeln abgeglichenen Tabellen (`recipes`, `meal_plan_entries`, `pantry_items`, `tags`) vergeben IDs mit `AUTOINCREMENT` (seit Migration 008, App 0.5.1): Eine gelöschte ID kommt nie wieder, so können sich Löschung und Neuanlage beim Abgleich nicht in die Quere kommen.

### `recipes`

| Feld | Typ | Bemerkung |
|---|---|---|
| id | Integer | Primärschlüssel |
| title | Text | |
| description | Text | optional |
| base_servings | Integer | Grundportionen |
| prep_minutes | Integer | optional |
| total_minutes | Integer | optional |
| source_url | Text | optional, nur `http`/`https`, für Dublettenprüfung |
| source_name | Text | optional, Quelle ohne Adresse, z. B. „HelloFresh-Karte“, „Kochbuch“ (Migration 009) |
| image | Text | Dateiname großes Bild |
| thumbnail | Text | Dateiname Vorschaubild |
| is_favorite | Boolean | |
| notes | Text | eigene Notizen |
| parent_recipe_id | Integer | optional; gesetzt = Variante dieses Ursprungsrezepts (nur eine Ebene) |
| created_at / updated_at | Zeitstempel | |

### `recipe_ingredients`

| Feld | Typ | Bemerkung |
|---|---|---|
| id | Integer | |
| recipe_id | Integer | Verweis auf `recipes` |
| position | Integer | Reihenfolge |
| group_name | Text | optional |
| quantity | Dezimalzahl | leer = nicht umrechnen |
| unit | Text | |
| name | Text | |
| note | Text | optional |
| shopping_relevant | Boolean | |

### `recipe_steps`

| Feld | Typ |
|---|---|
| id | Integer |
| recipe_id | Integer |
| position | Integer |
| text | Text |

### `tags` und `recipe_tags`

| `tags` | Typ | Bemerkung |
|---|---|---|
| id | Integer | |
| name | Text | |
| tag_group | Text | Gruppe für die Filteranzeige |
| position | Integer | Reihenfolge in der Gruppe |

`recipe_tags`: `recipe_id`, `tag_id` (Verknüpfungstabelle)

**Startkategorien:**

- **Hauptzutat:** Vegetarisch, Vegan, Rind, Schwein, Geflügel, Hackfleisch, Fisch, Meeresfrüchte
- **Geschmack:** Herzhaft, Süß
- **Art:** Pasta, Reis, Kartoffeln, Suppe/Eintopf, Salat, Auflauf/Ofen, Pfanne, Grill
- **Eigenschaften:** Schnell (unter 30 Minuten), Kinderliebling, Gut vorzubereiten, Resteverwertung, Für Gäste

### `pantry_items`

| Feld | Typ | Bemerkung |
|---|---|---|
| id | Integer | |
| name | Text | z. B. „Öl“ |
| synonyms | Text | kommagetrennt, z. B. „Olivenöl, Rapsöl, Sonnenblumenöl“; in der API eine Liste |
| updated_at | Zeitstempel | für den Abgleich mit der App |

### `meal_plan_entries`

| Feld | Typ | Bemerkung |
|---|---|---|
| id | Integer | |
| date | Datum | |
| slot | Text | `dinner` (Standard) oder `lunch` |
| position | Integer | Reihenfolge innerhalb von Datum und Slot |
| recipe_id | Integer | optional (Einträge ohne Rezept möglich) |
| servings | Integer | |
| status | Text | `planned`, `transferred`, `have_ingredients` |
| transferred_at | Zeitstempel | optional |
| note | Text | optional, z. B. „Essen gehen“ |

Keine Eindeutigkeit über (`date`, `slot`): Mehrere Einträge pro Slot sind erlaubt. Ein Eintrag braucht ein Rezept oder eine Notiz.

### `transferred_items`

| Feld | Typ | Bemerkung |
|---|---|---|
| id | Integer | |
| entry_id | Integer | optional; wird beim Löschen des Planeintrags leer, die Zeile bleibt |
| period_from | Datum | Beginn des Einkaufszeitraums, für den übertragen wurde |
| item_key | Text | Vergleichsschlüssel des Artikels (Name ohne Pluralendung usw.) |
| ingredient_name | Text | |
| quantity | Dezimalzahl | auf die Portionen umgerechnet, ungerundet; leer = ohne Menge |
| unit | Text | |
| sent | Boolean | gesendet bzw. beim Senden abgewählt (gilt als erledigt) |
| transferred_at | Zeitstempel | |

Eine Zeile je Planeintrag und Zutat, damit sich Änderungen am Plan später genau mit dem Übertragenen vergleichen lassen.

### `api_keys`

| Feld | Typ | Bemerkung |
|---|---|---|
| id | Integer | |
| name | Text | z. B. „Handy Papa“ |
| key_hash | Text | nur Hash, nie der Schlüssel |
| created_at | Zeitstempel | |
| last_used_at | Zeitstempel | |

### `deletions`

Schmale Tabelle für gelöschte Datensätze, damit die App beim Abgleich erfährt, was sie entfernen muss: `table_name`, `record_id`, `deleted_at`.

### `settings`

Schlüssel-Wert-Speicher, u. a.:

- `family_size`
- `shopping_day` (englischer Wochentag, z. B. `saturday`)
- `bring_entity` (To-do-Entität der Bring-Liste in Home Assistant)
- `app_url` (Adresse, unter der die Android-App den App-Zugang erreicht, z. B. `https://rezepte.example.org`; steht im QR-Code)

### Nicht gespeichert (wird berechnet)

Statistik, Statushinweise, aktueller Einkaufszeitraum.

---

## 5. Bildschirme

Weboberfläche und App haben dieselbe Struktur. Navigation über die vier Hauptbereiche **Essensplan** (Start), **Rezepte**, **Einkauf** und **Einstellungen**: auf dem Handy als Leiste unten, auf großen Bildschirmen als Seitenleiste.

### 5.1 Essensplan (Startseite)

- Horizontal endlos scrollbar; 7 Tage nebeneinander auf großen Bildschirmen, 1–3 Tage auf dem Handy.
- Button „Heute“; aktueller Einkaufszeitraum farbig hinterlegt.
- Pro Eintrag: Vorschaubild, Titel, Portionen, Herzchen, Status-Symbol.
- Leere Tage mit Plus (öffnet die Rezeptauswahl), belegte Tage mit kleinem Plus für ein zweites Gericht.
- Antippen öffnet das Rezept, langes Drücken startet Drag and Drop.
- Menü am Eintrag: Portionen ändern, Ersetzen, Entfernen, Notiz.

### 5.2 Rezeptauswahl (beim Planen)

Eigene Seite mit Suche, Kategorie-Filtern, Favoriten und Statushinweis an jedem Rezept.

### 5.3 Rezeptübersicht

- Suchfeld (Volltextsuche über Titel, Zutaten, optional Zubereitung).
- Kategorie-Filter als Chips nach Gruppen, Herzchen-Filter.
- Kacheln mit Bild, Titel, Zeit, Statushinweis; Varianten mit Kennzeichen „Variante“ (Suche, Filter, Favoriten, Statushinweis je Variante).
- Sortierung: Name, zuletzt eingeplant, am häufigsten eingeplant, neu hinzugefügt.
- Plus-Button für Import/Anlegen; Zugang zur Statistik.

### 5.4 Rezeptansicht (Kochansicht)

- Großes Bild, Titel, Herzchen, Kategorien, Zeiten, Statushinweis.
- Portionen-Zähler mit sofortiger Umrechnung.
- Zutaten mit Gruppenüberschriften, nummerierte Schritte.
- Buttons „Zum Essensplan hinzufügen“ (Dialog mit den nächsten Tagen und deren Belegung; bei belegtem Tag: Ersetzen oder als zweites Gericht) und „Bearbeiten“.
- Bildschirm bleibt während der Kochansicht eingeschaltet.
- Button „Variante speichern“; bei einer Variante „Variante von <Titel>“; direkt unter dem Titel eine Leiste mit Mini-Kacheln (Vorschaubild, Titel, „Ursprungsrezept“/„Variante“) für alle anderen Rezepte der Familie, seitwärts scrollbar, ohne Familie nicht sichtbar.

### 5.5 Rezept bearbeiten

- Titel, Beschreibung, Bild, Grundportionen, Zeiten, Kategorien, Quell-URL, Notizen.
- Zutaten und Schritte als verschiebbare Kärtchen; Gruppenüberschriften als eigene Kärtchen.

### 5.6 Einkauf

- Gerichte im aktuellen Einkaufszeitraum; bereits übertragene ausgegraut.
- Zusammengefasste Einkaufsliste mit Abwahl einzelner Artikel.
- Button „An Bring senden“.

### 5.7 Einstellungen

Familiengröße, Einkaufstag, Bring-Liste, Vorratsliste, Kategorien, API-Schlüssel (mit QR-Code).

### 5.8 Home-Assistant-Dashboard

- Karte mit dem aktuellen Einkaufszeitraum: Tag, kleines Bild, Gericht; heutiger Tag hervorgehoben.
- Antippen öffnet einen großen Dialog mit der Kochansicht (Portionen aus dem Plan).
- Kein Bearbeiten auf der Karte; Link zur Weboberfläche (`/app/<App-ID>`, funktioniert auch ohne Eintrag in der Seitenleiste).
- Die Integration stellt zusätzlich bereit: Kalender „Essensplan“ (jedes Gericht als ganztägiger Termin, Mittagessen mit „Mittag:“) und Sensor „Heute“ (Abendgerichte von heute; Mittagessen und Portionen als Attribute). Sie fragt den Server alle 15 Minuten ab; die Karte und die Kalenderansicht holen ihre Daten beim Öffnen direkt.

**Verbindung Integration – App:** Die App meldet sich per Supervisor-Discovery (`discovery: [rezeptplaner]` in `config.yaml`) mit Hostname, Port und einem zufällig erzeugten Schlüssel (Datei `integration_token` im Datenordner). Die Integration richtet sich damit nach einer Bestätigung selbst ein. Der Server lässt neben Ingress nur lesende Anfragen (GET) mit diesem Schlüssel zu; nach außen bleibt kein Port offen. Die Karte erreicht den Server über Endpunkte der Integration (`/api/rezeptplaner/…`, mit Anmeldung bei Home Assistant; Bilder über signierte Adressen).

---

## 6. Abläufe

### A. Import per URL

1. URL einfügen oder auf dem Handy über „Teilen“ aus Browser/Chefkoch-App an die App senden.
2. Server lädt die Seite, liest die schema.org-Rezeptdaten, lädt das Bild, zerlegt Zutaten, gleicht mit der Vorratsliste ab.
3. Dublettenhinweis bei bekannter Quell-URL.
4. Ergebnis als Entwurf im Bearbeiten-Bildschirm; nach Prüfung speichern.

### B. Import per Foto (App)

1. Eine oder mehrere Seiten fotografieren.
2. ML Kit erkennt den Text auf dem Handy.
3. Zeilen zuordnen: *Titel*, *Zutat*, *Zubereitung*, *ignorieren* (einzeln antippen oder Bereich markieren).
4. Zutatenzeilen werden vom Server zerlegt (gemeinsamer Parser mit dem URL-Import) und mit der Vorratsliste abgeglichen.
5. Optional Foto vom fertigen Gericht als Rezeptbild.
6. Weiter im Bearbeiten-Bildschirm.

### C. Planen

Über das Plus im Plan (Rezeptauswahl) oder über „Zum Essensplan hinzufügen“ im Rezept (Tagauswahl). Portionen aus der Familiengröße, pro Eintrag änderbar.

### D. Verschieben und Tauschen

Siehe Warnregeln in Abschnitt 3. Ringtausche über mehrere Tage erfolgen als aufeinanderfolgende Einzeltausche.

### E. An Bring senden

1. Server berechnet die Liste für den Einkaufszeitraum.
2. Abwahl einzelner Artikel in der Vorschau.
3. Übertragung an die Bring-Liste in Home Assistant. Steht ein Artikel gleichen Namens schon offen auf der Liste, wird seine Mengenangabe ergänzt („1 l + 250 ml“) statt einen zweiten Eintrag anzulegen.
4. Einträge erhalten Status `transferred`, Artikel werden in `transferred_items` protokolliert – auch abgewählte (als erledigt). Lehnt Home Assistant einzelne Artikel ab, geht es mit den übrigen weiter; die abgelehnten werden nicht protokolliert und erscheinen dadurch als nachzusenden.

### G. Import aus einer Datei (z. B. von der Claude-Skill)

Format und Schema: [importformat.md](importformat.md). Die Claude-Skill `skills/rezeptplaner-import/` erzeugt aus Fotos von Rezeptkarten oder Kochbuchseiten genau dieses Format.

1. Datei wählen: Weboberfläche „Datei importieren“ in der Rezeptübersicht; App über Teilen, „Öffnen mit“ oder die Dateiauswahl. Eine ZIP-Datei (`recipes.json` + `images/`) oder eine einzelne `.json`, auch mehrere Dateien auf einmal.
2. Der Server prüft alles und liefert eine **Vorschau**, gespeichert wird noch nichts: je Rezept Bild, Titel, Zutatenanzahl, Fehler (Rezept nicht wählbar) und Hinweise – Dublette (gleicher Titel ohne Groß/klein und Umlaute oder gleiche Quell-URL), unbekannte Kategorie (wird **nicht** angelegt, sondern weggelassen), fehlendes Bild.
3. Zutaten auf der Vorratsliste werden wie beim URL-Import als nicht einkaufsrelevant markiert.
4. Rezepte einzeln abwählen, **Importieren** – erst jetzt wird gespeichert. Die App erkennt das Format am Inhalt (ZIP-Kennung oder Formatkennung im JSON), nicht am Dateityp; Import nur online.

### F. Nachträgliche Änderungen

Vergleich mit den übertragenen Artikeln; Angebot, nur die Differenz nachzusenden.

- Verglichen wird je Artikel und Einheit der Bedarf aller übertragenen Gerichte des Einkaufszeitraums (auch vergangener Tage) mit dem Protokoll in `transferred_items`. Stück-Einheiten werden dabei gerundet verglichen, so wie sie gesendet wurden.
- **Fehlt noch:** z. B. nach mehr Portionen, einem ersetzten Rezept oder einem Artikel, der beim Senden nicht ankam. Einzelne Artikel lassen sich abwählen (gelten dann als erledigt); „Differenz nachsenden“ ergänzt offene Bring-Einträge bzw. legt neue an.
- **Zu viel gesendet:** nur Anzeige, nichts wird aus Bring gelöscht. Abgewählte Artikel zählen nicht als zu viel, sie wurden ja nicht gekauft.
- Der Dialog beim Verschieben bietet „Zutaten an Bring senden“: Das Gericht wird verschoben und seine Zutaten werden sofort übertragen.

---

## 7. Android-App: Offline-Verhalten

- Lokaler Stand mit allen Rezepten, dem Essensplan (14 Tage zurück bis Ende der Planung) und den Einstellungen – als eine JSON-Datei, beim Start in den Speicher geladen (geändert 2026-10-23: statt Room, weil die App nur liest und bei einigen hundert Rezepten Suchen im Speicher genügt).
- Abgleich nur der Änderungen seit dem letzten Abgleich (über `updated_at` und `deletions`).
- Vorschaubilder immer lokal; große Bilder für den aktuellen Einkaufszeitraum vorab, sonst beim ersten Ansehen.
- Abgleich bei App-Start, beim Herunterziehen und regelmäßig im Hintergrund (alle paar Stunden).
- Offline: Banner „Offline – Stand von …“, Lesen und Kochansicht funktionieren, Bearbeiten, Foto- und Datei-Import sind deaktiviert.
- Ändern geht nur online und direkt über den Server; die App hat keine Warteschlange für Änderungen und damit keine Konflikte.
- Zielgeräte: Samsung Galaxy A54 mit Android 16 (One UI). `targetSdk` 36 (Verhalten von Android 16), `minSdk` 29 (Android 10, kostet keinen Mehraufwand, getestet wird nur auf 16). `compileSdk` 37, weil die aktuellen AndroidX-Bibliotheken (Compose, Navigation, Core) dagegen übersetzt werden müssen; das ändert am Verhalten der App nichts (geändert 2026-10-23). Darstellung von Rand zu Rand (`enableEdgeToEdge`, Insets in jedem Bildschirm) und vorausschauende Zurück-Geste (Compose-Navigation, `PredictiveBackHandler`, kein `onBackPressed`).
- Samsung schränkt Hintergrund-Apps stark ein: WorkManager alle 4 Stunden mit Netz; jeder Abgleich wird mit Uhrzeit und Anlass (Start, Herunterziehen, Hintergrund) gespeichert. Liegt der letzte Hintergrund-Abgleich über 12 Stunden zurück und ist die App akku-optimiert, zeigt die App einen Hinweis mit Knopf „Akku-Optimierung ausschalten“ und dem One-UI-Weg (App-Info → Akku → „Nicht eingeschränkt“; Akku → Hintergrundnutzungsgrenzen → „Nie in Standby versetzte Apps“).
- Verteilung: signierte APK in den GitHub-Releases (Tag `android-v…`, löst den Add-on-Build nicht aus), Updates auf dem Handy über Obtainium.

---

## 8. API-Übersicht

Alle Endpunkte unter `/api/v1`. Authentifizierung per API-Schlüssel (App-Zugang, Port 8100) bzw. über Ingress (Weboberfläche, Port 8099).

| Bereich | Funktionen |
|---|---|
| Rezepte | Liste mit Suche/Filtern, Einzelabruf (mit `parent_recipe_id` und `family`), Anlegen, Ändern, Löschen, Favorit setzen, Bild hochladen, Variante anlegen (`POST /recipes/{id}/variants`, der Server kopiert) |
| Import | URL einlesen (liefert Entwurf, Bildadresse und Dubletten), Zutatenzeilen zerlegen; Bild der Quellseite beim Speichern übernehmen; Datei-Import: `POST /import/file` (eine oder mehrere Dateien, liefert Vorschau und Kennung), `POST /import/file/{kennung}` (ausgewählte Rezepte speichern) |
| Essensplan | Einträge eines Zeitraums, Anlegen, Ändern, Löschen, Verschieben/Tauschen mit Warnprüfung (erst prüfen, dann bestätigen) |
| Einkauf | Vorschau der Einkaufsliste (aktueller oder folgender Einkaufszeitraum), An Bring senden, Differenz nachsenden |
| Stammdaten | Kategorien, Vorratsliste (inkl. Prüfung, ob Zutatennamen im Vorrat sind), Einstellungen |
| Statistik | Häufigkeiten nach Zeitraum |
| Abgleich | `GET /sync?since=…`: Änderungen seit Zeitpunkt X (Rezepte vollständig, Planeinträge im Zeitfenster, Vorrat, Kategorien, Einstellungen, Löschungen) und die Serverzeit als nächster Startpunkt; Rezepte mit `parent_recipe_id` (beim Umhängen nach dem Löschen eines Ursprungsrezepts ändert sich `updated_at` der Varianten) |
| Verwaltung | API-Schlüssel anlegen (mit QR-Code), sperren – nur über Ingress, nicht über den App-Zugang |
| Bilder | Bild und Vorschaubild abrufen |

---

## 9. Projektstruktur

Das Repository ist gleichzeitig Add-on-Repository für Home Assistant und HACS-Repository für die Integration. Beides stellt Anforderungen an die Struktur:

```
Rezepteverwaltung_Essenplan/
├── repository.yaml                  Kennzeichnet das Repo als Add-on-Repository
├── hacs.json                        HACS-Metadaten für die Integration
├── rezeptplaner/                    Add-on: config.yaml, Doku, Icon
├── custom_components/rezeptplaner/  HA-Integration (Pfad von HACS vorgegeben), Karte in frontend/
├── server/                          Go-Backend
├── web/                             Weboberfläche (wird in den Server eingebettet)
├── android/                         Android-App
├── skills/rezeptplaner-import/      Claude-Skill: Rezeptfotos → Importdatei (Format in docs/importformat.md)
├── .github/workflows/               Automatische Builds
└── docs/                            Planung und Notizen
```

### Builds und Installation

- **Add-on-Image wird nicht auf dem Green gebaut.** Ohne vorgefertigtes Image würde der Green bei jeder Installation und jedem Update Go und die Weboberfläche selbst kompilieren. Stattdessen baut eine GitHub Action das ARM64-Image und legt es in der GitHub Container Registry (ghcr.io) ab; die `config.yaml` des Add-ons verweist darauf. Der Green lädt nur noch das fertige Image herunter.
- **Dashboard-Karte wird von der Integration mitgeliefert.** Die JavaScript-Datei der Karte liegt in der Integration (`custom_components/rezeptplaner/frontend/`), die sie beim Start registriert (`add_extra_js_url`). So genügt eine einzige HACS-Installation für Integration und Karte. Die Karte ist eine schlichte Web Component ohne Bibliotheken und braucht keinen Build-Schritt; einen eigenen Ordner `ha-card/` gibt es deshalb nicht (geändert 2026-10-21).
- **Installation zu Hause:** Repository-URL im Add-on-Store von Home Assistant hinzufügen; dieselbe URL in HACS als benutzerdefiniertes Repository vom Typ „Integration“.

### Öffentliches Repository – Regeln

- Keine Zugangsdaten, API-Schlüssel, Domains oder Tunnel-Konfiguration im Code; alles Persönliche kommt in die Add-on-Konfiguration oder die Einstellungen.
- `.gitignore` von Anfang an für Datenbanken, Bilder, Build-Ergebnisse und lokale Konfigurationsdateien.
- Lizenz: **MIT** (Datei `LICENSE` im Hauptverzeichnis).

---

## 10. Projektphasen

| Phase | Inhalt | Ergebnis |
|---|---|---|
| 1. Grundgerüst | Go-Server, Datenbank, Rezept-Endpunkte, lokal lauffähig | Einstieg in Go an überschaubarer Aufgabe |
| 2. Weboberfläche Rezepte | Übersicht, Suche, Kochansicht, Bearbeiten mit Drag and Drop, URL-Import | |
| 3. Add-on | Paketierung für den Green, Ingress | Nutzbares Rezeptbuch zu Hause |
| 4. Essensplan | Planansicht, Planen in beide Richtungen, Verschieben mit Warnregeln, Statushinweise, Favoriten | |
| 5. Einkauf und Bring | Einkaufsliste, Vorratsliste, Übertragung, Differenzen | Kompletter Kern-Workflow im Browser |
| 6. Dashboard | HA-Integration und Dashboard-Karte | |
| 7. Android-App | QR-Einrichtung, Offline-Speicher, Teilen-Funktion, Datei-Import, Foto-Import mit ML Kit; optional `CF-Access-Client-Id`/`-Secret` (Cloudflare Access) in den Einstellungen der App, bei jeder Anfrage mitgeschickt | App-Zugang (Port 8100, API-Schlüssel) steht seit 0.4.1; Schritte siehe unten |

**Schritte der Phase 7** (festgelegt 2026-10-23):

| Schritt | Inhalt |
|---|---|
| 7.1 | Server: `GET /sync`, Einstellung `app_url`, QR-Code beim Anlegen eines Schlüssels (optional mit Cloudflare-Access-Daten, nicht gespeichert), Version der Schnittstelle in `/system` |
| 7.1b | Importformat (`docs/importformat.md`, JSON-Schema), Spalte `source_name` (Migration 009), Datei-Import im Server mit Vorschau |
| 7.1c | Weboberfläche: „Datei importieren“ in der Rezeptübersicht mit Vorschau; Quelle anzeigen und bearbeiten |
| 7.1d | Claude-Skill `skills/rezeptplaner-import/` – 7.1 bis 7.1c zusammen als App 0.6.0 |
| 7.2b | Weboberfläche „Einstellungen → App installieren“: Schritt 1 QR-Codes (Obtainium, Rezeptplaner in Obtainium, APK `releases/latest/download/rezeptplaner.apk`) mit Anleitung für das A54, Repository aus der Option `app_repository`; Schritt 2 „Neues Gerät verbinden“ erzeugt erst dann einen Schlüssel – App 0.6.1 |
| 7.2 | Android-Grundgerüst (Kotlin, Compose, Material 3, Navigation, kotlinx.serialization; Room, OkHttp, Coil und WorkManager kommen mit den Schritten, die sie brauchen), GitHub Action baut signierte APK; Anleitung [ANDROID.md](ANDROID.md) |
| 7.3 | Einrichtung per QR-Code (Google-Code-Scanner ohne Kamera-Berechtigung; auch Direktlink `rezeptplaner://setup` aus der Kamera-App, immer mit Bestätigung der Adresse) oder von Hand, Verbindungstest über `/system` (`api_version`), Einstellungen (Verbindung testen, Cloudflare Access, trennen), Verbindungsdaten verschlüsselt mit Schlüssel im Android-Keystore – App 0.2.0 |
| 7.4 | Abgleich und Offline-Lesen: Rezepte (Suche, Kategorien, Favoriten, Sortierung), Kochansicht (Portionen, Bildschirm bleibt an, Varianten), Plan ansehen, Offline-Hinweis, Abgleich bei Start/Herunterziehen/alle 4 h (WorkManager), Hinweis Akku-Optimierung – App 0.3.0. Lokaler Speicher als eine JSON-Datei im Speicher statt Room (die App liest nur, geändert wird online; spart Room/KSP) |
| 7.5 | Planen: hinzufügen („+“ je Tag mit Rezeptauswahl, „Zum Essensplan hinzufügen“ in der Kochansicht), Portionen, Notiz, „Zutaten im Haus“, ersetzen, entfernen, verschieben/tauschen mit den Warnregeln des Servers – App 0.4.0. Verschieben über eine Tagesauswahl statt Ziehen mit dem Finger (in der langen Tagesliste auf dem Handy zuverlässiger; Ziehen bei Bedarf später) |
| 7.6 | Teilen einer URL, Rezept bearbeiten, Bild per Kamera/Galerie, Variante speichern |
| 7.6b | Datei-Import: Teilen (`ACTION_SEND`/`_MULTIPLE`, großzügige Typen, Erkennung am Inhalt), „Öffnen mit“ (`ACTION_VIEW`), Knopf „Datei importieren“ (Dateiauswahl), Vorschau, nur online |
| 7.7 | Foto-Import mit ML Kit |
| 7.8 | Einkauf und Bring |
| 7.9 | Messungen, Doku |
| 8. Feinschliff | Statistik, Kategorien-Verwaltung, Erkenntnisse aus dem Alltag | |

---

## 11. Offene Punkte

- ~~**Bring-Mengenangabe**~~ – geklärt 2026-10-21: Die Beschreibung (`description`) eines To-do-Eintrags wird in Bring zur Spezifikation. Senden (18 Artikel in die gewählte Liste) und Nachsenden der Differenz auf dem Green mit App 0.3.0 getestet: ok.

- ~~**IDs nach dem Löschen**~~ – geklärt 2026-10-23: `AUTOINCREMENT` für die abgeglichenen Tabellen (Migration 008, App 0.5.1). Der Zähler startet hinter der größten ID, auch hinter bereits gelöschten aus `deletions`.

- **Googles Entwickler-Verifizierung (vor 2027 erledigen):** Ab 2027 sollen auf zertifizierten Geräten weltweit nur noch Apps registrierter Entwickler installierbar sein, auch außerhalb des Play Store (Details und Quellen in [ANDROID.md](ANDROID.md), Abschnitt 6). Plan: kostenloses Konto für eingeschränkten Vertrieb (bis 20 Geräte), Paketname mit dem Signaturzertifikat registrieren. Termin für Deutschland beobachten.

## 12. Mögliche spätere Erweiterungen

- Optionale KI-Strukturierung beim Foto-Import
- Umrechnung unterschiedlicher Einheiten derselben Zutat
- Auslesen des tatsächlichen Abhak-Status aus Bring
- Cloudflare Access mit Service Token
