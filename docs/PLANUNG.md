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

**Messung nach Phase 7 (2026-10-23, Server 0.7.1, Android-App 0.8.1):**

| Messgröße | Wert |
|---|---|
| APK (signiert, R8) | 5,1 MB (0.1.0: 1,7 MB; Sprünge durch QR-Scanner in 0.2.x und Texterkennung in 0.7.0, beide über die Play-Dienste) |
| Programmdatei Server Linux/ARM64 | 12,3 MB (gzip: 5,1 MB) |
| Voller Abgleich, 500 Rezepte und 730 Planeinträge (Server) | ca. 22 ms; 1,4 MB JSON, mit gzip 126 KB (gzip seit 0.7.1, vorher unkomprimiert) |
| Abgleich ohne Änderungen | ca. 2,5 ms; 4,5 KB mit gzip |
| Voller Abgleich in der App (Daten einlesen und speichern) | ca. 0,2 s |
| Bilder beim ersten Abgleich (500 Vorschaubilder + 14 große der beiden Einkaufswochen) | ca. 14 s im Emulator über das lokale Netz |
| Daten auf dem Handy: `snapshot.json` / Bilder | 0,97 MB / ca. 26 MB (Vorschaubild je Rezept ca. 37 KB) |
| Kaltstart der App (R8, 500 Rezepte) | ca. 0,4 s |
| Arbeitsspeicher der App nach Blättern in Rezepten und Plan (PSS) | 79 MB |
| Texterkennung einer Kochbuchseite (1200 × 1700 px, 17 Zeilen) | 1,7–2,7 s |
| Arbeitsspeicher Server nach 50 vollen Abgleichen (Spitze inkl. 500 Bild-Uploads) | 43 MB (75 MB) |
| Hintergrund-Abgleich | alle 4 Stunden (WorkManager), nur mit Netz |

App-Werte im Emulator (Android 16, Profil Galaxy A54, x86_64 mit WHPX) mit dem qa-Build (R8 wie das Release), Speicherplatz mit dem Debug-Build; Server auf dem Entwicklungsrechner (Windows). Die Testdaten erzeugt ein Skript: 500 Rezepte mit je 10 Zutaten, 6 Schritten, 3 Kategorien und einem echten Foto, dazu 730 Planeinträge. Die Dauer stammt aus den Logzeilen der App („Abgleich (…): … Daten … ms, … Bilder in … ms“ und „Texterkennung: …“, sichtbar mit `adb logcat -s Rezeptplaner`). Der erste Abgleich ist der teure: Danach kommen nur Änderungen und neue Bilder. Auf dem A54 über den Cloudflare-Tunnel sind andere Zeiten zu erwarten; Cloudflare komprimiert JSON auf dem Weg zum Handy ohnehin, gzip im Server spart zusätzlich den Upload aus dem Heimnetz und hilft bei direkter Verbindung.

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
- **Eingekaufte Woche und nächster Einkauf** (geändert 2026-10-23): Die eingekaufte Woche beginnt am letzten Einkaufstag, und zwar erst ab der **Uhrzeit des Einkaufs** (Einstellung `shopping_time`, Standard 00:00). Beispiel Donnerstag 18:00: Bis Donnerstag 17:59 gilt die Vorwoche als eingekauft, Donnerstag bis Mittwoch ist der **nächste Einkauf**, den man gerade plant. Ab 18:00 wechselt beides.
- Plan in Web und App: eingekaufte Woche warm hervorgehoben, nächster Einkauf hellblau, mit Legende. Die App berechnet beide Zeiträume selbst aus Einkaufstag und Uhrzeit, damit der Wechsel auch ohne Verbindung pünktlich passiert.
- Einkaufsseite: Schalter „Eingekaufte Woche / Nächster Einkauf“. Sie öffnet „Nächster Einkauf“, wenn für die eingekaufte Woche nichts mehr zu senden ist.
- Warnregeln beim Verschieben beziehen sich auf die eingekaufte Woche.

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
  - Stück-Einheiten (ohne Einheit, Stück, Dose, Packung, Bund, Zehe, Glas usw.) werden nach dem Addieren auf ganze Zahlen aufgerundet. Gewicht und Volumen werden auf glatte Schritte aufgerundet (g/ml: bis 10 auf 1, bis 100 auf 5, bis 1000 auf 10, darüber auf 50; kg/l: bis 1 auf 0,05, darüber auf 0,1), EL/TL/Msp. auf ½ – aus 4/6 von 800 g werden 540 g statt 533 g (seit 0.7.2). Andere Einheiten („Prise“) bleiben. Die Kochansicht zeigt weiter die genaue Menge.
- Tage vor heute zählen nicht mit (schon gegessen). Neben dem aktuellen lässt sich auch der folgende Einkaufszeitraum abrufen, z. B. am Abend vor dem Einkaufstag (ergänzt 2026-10-21).
- Vor dem Senden lassen sich einzelne Artikel abwählen.
- **Zutaten je Eintrag abwählen (Schritt 8.E):** Beim Einplanen (Plus → Rezept, „Zum Essensplan hinzufügen“, Ersetzen) erscheint eine Kurzvorschau: Titel, Portionen (Mengen sofort umgerechnet), einkaufsrelevante Zutaten mit Häkchen (alle gesetzt), Vorratszutaten ausgegraut darunter, „Alles vorhanden“ (Status `have_ingredients`) und „Hinzufügen“. Abgewählte Zutaten („habe ich noch“) gelten nur für diesen Eintrag und gehen nicht in Einkaufsliste und Bring ein. Nachträglich änderbar über „Zutaten vorhanden …“ im Menü des Eintrags; nach der Übertragung erscheinen Änderungen als Differenz (nachsenden bzw. zu viel). Gespeichert wird der Zutatenschlüssel (wie beim Zusammenfassen, `item_key`), nicht die Zutaten-ID – die ändert sich bei jedem Speichern des Rezepts. Gelöschte Zutaten fallen damit heraus, neue sind eingeschlossen. Beim Verschieben und Tauschen wandert die Abwahl mit, beim Ersetzen durch ein anderes Rezept verfällt sie. Im Plan steht am Eintrag „2 Zutaten vorhanden“. Abschaltbar in den Einstellungen („Kurzvorschau beim Einplanen“, Standard an).
- Übertragung über die Bring-Integration von Home Assistant (To-do-Entität) per Supervisor-API; Artikelname und Mengenangabe getrennt, soweit möglich.
  - Die App hat `homeassistant_api: true` und ruft mit `SUPERVISOR_TOKEN` die Dienste `todo.get_items`, `todo.add_item` und `todo.update_item` unter `http://supervisor/core/api/` auf. Die To-do-Entität wird in den Einstellungen gewählt (`bring_entity`).
  - Die Bring-Integration übernimmt die Beschreibung eines To-do-Eintrags als Spezifikation, also die Mengenangabe unter dem Artikel.
  - Lokal ohne Supervisor: mit `REZEPTPLANER_HA_URL` und `REZEPTPLANER_HA_TOKEN` (langlebiger Token) gegen eine echte Instanz, sonst Testmodus mit einer Liste im Arbeitsspeicher.
- Übertragene Artikel werden protokolliert, um Differenzen nachsenden zu können. Zu viel Gekauftes wird nur angezeigt, nicht aus Bring gelöscht.

### Statistik und Statushinweis

- Statistik (eigene Seite in Web und App, Schritt 8.D): Anzahl Rezepte; schon eingeplant bzw. noch nie eingeplant (Anzahl, Prozent, Link auf den Filter „Nie eingeplant“); mit Illustration (Link auf den Filter); Favoriten; Top 20 der am häufigsten gegessenen Rezepte (letzte 3 Monate, letztes Jahr, gesamt); Einträge ohne Rezept im Zeitraum („2× Essen gehen“); Lieblingsgerichte, die lange nicht dran waren (Favoriten ohne Termin in den letzten 8 Wochen und ohne geplanten Termin); zuletzt hinzugefügt (5); Rezepte je Kategorie und ohne Kategorie.
- **Zählregeln:** Häufigkeiten zählen Planeinträge bis einschließlich heute – Geplantes zählt erst, wenn der Tag da ist. Dieselbe Regel gilt für `plan_count` und die Sortierung „Am häufigsten eingeplant“. „Noch nie eingeplant“ heißt: kein einziger Eintrag, auch kein zukünftiger. Varianten zählen als eigene Rezepte, Einträge ohne Rezept nicht als Rezept. Alles wird bei Bedarf berechnet, nichts gespeichert.
- Die App zeigt die Statistik online vom Server; ohne Verbindung rechnet sie „gesamt“ aus den Daten des Abgleichs (`plan_info`), ohne Zeitraum und ohne Einträge ohne Rezept.
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

### `tag_groups`, `tags` und `recipe_tags`

| `tag_groups` | Typ | Bemerkung |
|---|---|---|
| id | Integer | AUTOINCREMENT (seit Migration 011) |
| name | Text | z. B. „Hauptzutat“ |
| position | Integer | Reihenfolge der Gruppen |
| updated_at | Zeitstempel | |

| `tags` | Typ | Bemerkung |
|---|---|---|
| id | Integer | AUTOINCREMENT |
| name | Text | über alle Gruppen eindeutig, ohne Rücksicht auf Groß-/Kleinschreibung und Umlaute (prüft der Server) |
| group_id | Integer | Gruppe (bis Migration 010 Text `tag_group`) |
| position | Integer | Reihenfolge in der Gruppe |
| updated_at | Zeitstempel | |

`recipe_tags`: `recipe_id`, `tag_id` (Verknüpfungstabelle)

Verwaltet werden Gruppen und Kategorien nur in der Weboberfläche (Einstellungen → Kategorien, Schritt 8.C). Die App liest sie beim Abgleich vollständig (leere Gruppen fehlen dort). Weil die Rezepte ihre Kategorien mit Name und Gruppe mitbringen, bekommen alle betroffenen Rezepte bei Umbenennen, Verschieben in eine andere Gruppe, Löschen und Zusammenführen ein neues `updated_at`; gelöschte Kategorien stehen zusätzlich in `deletions`. Reines Umsortieren ändert die Rezepte nicht.

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
| slot | Text | `dinner` (Standard), `lunch` oder `coffee` (Kaffee & Kuchen, seit Migration 015) |
| position | Integer | Reihenfolge innerhalb von Datum und Slot |
| recipe_id | Integer | optional (Einträge ohne Rezept möglich) |
| servings | Integer | |
| status | Text | `planned`, `transferred`, `have_ingredients` |
| transferred_at | Zeitstempel | optional |
| note | Text | optional, z. B. „Essen gehen“ |
| excluded_ingredients | Text (JSON-Liste) | abgewählte Zutaten als `item_key` (Migration 012), Standard `[]` |
| menu_id | Integer | optional: Gang dieses Menüs (Migration 016); beim Löschen des Menüs leer |
| course_tag_id | Integer | optional: Gang (Kategorie der Gang-Gruppe), nur im Menü sinnvoll |

Keine Eindeutigkeit über (`date`, `slot`): Mehrere Einträge pro Slot sind erlaubt. Ein Eintrag braucht ein Rezept oder eine Notiz.

### `meal_plan_menus`

Menüs (Schritt 9.4, Migration 016): eine Klammer um normale Planeinträge. Datum und Slot der Gänge sind immer die des Menüs; verschoben wird nur der ganze Block.

| Feld | Typ | Bemerkung |
|---|---|---|
| id | Integer | `AUTOINCREMENT`, Löschungen in `deletions` (`meal_plan_menus`) |
| date, slot | Datum, Text | wie bei `meal_plan_entries` |
| position | Integer | Reihenfolge zusammen mit den einzelnen Einträgen im Slot |
| title | Text | leer = automatisch: Hauptgericht, sonst erstes Gericht (`display_title` in der API) |
| note | Text | z. B. „Suppe am Vortag kochen“ |
| servings | Integer | Personen; beim Ändern auf alle Gänge übertragbar (Rezepte mit Ergebnis-Einheit wie Kuchen bleiben) |
| created_at, updated_at | Zeitstempel | |

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
- `shopping_time` (Uhrzeit des Einkaufs, z. B. `18:00`; ab dann gilt am Einkaufstag eingekauft)
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

- Suchfeld (Volltextsuche über Titel, Beschreibung, Zutaten und – seit 0.12.1, nur Weboberfläche – Zubereitung; die App sucht offline weiter in Titel, Beschreibung und Zutaten).
- Kategorie-Filter als Chips nach Gruppen, Herzchen-Filter, Filter „Illustrationen“ (gezeichnete Bilder, die ein Foto ersetzen sollte; Kachel mit Kennzeichen „Illustration“).
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

**Per QR-Code (Schritt 8.F):** Viele Ausdrucke (z. B. Chefkoch) tragen die Adresse als QR-Code. App: Rezepte → Plus → „Rezept aus QR-Code“ öffnet direkt den Google-Code-Scanner (Play-Dienste, ohne Kamera-Berechtigung, wie bei der Einrichtung); dazu ein Scan-Knopf im Adressfeld von „Rezept aus Internetadresse“. Nach dem Scan startet der URL-Import sofort. Enthält der Code keine http(s)-Adresse, erscheint eine Meldung; fehlt das Scanner-Modul der Play-Dienste noch, stößt die App das Nachladen an und meldet „wird nachgeladen, gleich erneut versuchen“. Weboberfläche: Knopf „QR-Code“ neben dem Adressfeld, nur wenn der Browser QR-Erkennung (BarcodeDetector) und Kamera bietet (z. B. Chrome auf Android über https); sonst ausgeblendet. In der Home-Assistant-App ist die Kamera im Ingress-Fenster oft gesperrt – dann erklärt eine Meldung, die Adresse einzufügen.

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

**Titel und Illustration (Skill 2.0, Format Version 2):** Hat die Vorlage keinen Titel (Zettel, Kochbuchausschnitt), vergibt die Skill einen beschreibenden Titel nach den Hauptbestandteilen und vermerkt das in den Notizen und im Chat. Fehlt ein brauchbares Foto vom Gericht (nur Text, schwarz-weiß, zu klein, unscharf), zeichnet `scripts/illustrate.py` eine Illustration im einheitlichen flachen Stil (nur Pillow, kein Netz – die Code-Ausführung von claude.ai hat kein Bildmodell und oft kein Internet). Das Feld `image_kind` (`photo`/`illustration`) wird gespeichert (Spalte `recipes.image_kind`); die Rezeptübersicht filtert mit „Illustrationen“ nach gezeichneten Bildern, ein hochgeladenes oder von einer Webadresse geholtes Bild gilt wieder als Foto. Weil der Server unbekannte Felder ablehnt, bekam das Format dafür die Version 2; Dateien der Version 1 bleiben gültig.

### F. Nachträgliche Änderungen

Vergleich mit den übertragenen Artikeln; Angebot, nur die Differenz nachzusenden.

- Verglichen wird je Artikel und Einheit der Bedarf aller übertragenen Gerichte des Einkaufszeitraums (auch vergangener Tage) mit dem Protokoll in `transferred_items`. Beide Seiten werden dabei gerundet verglichen, so wie sie gesendet werden (Protokoll mit genauen Werten); erhöht man z. B. von 4 auf 6 Portionen, fehlen 260 g statt 267 g.
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

### Datei-Import in der App (7.6b)

**Wege in die App**

| Weg | Android | Hinweise |
|---|---|---|
| Antippen einer Datei (WhatsApp, „Eigene Dateien“, Download-Benachrichtigung) | `ACTION_VIEW` | Rezeptplaner erscheint in „Öffnen mit“. Hat jemand für ZIP eine andere Standard-App festgelegt, öffnet WhatsApp diese – dann über Teilen |
| Lange drücken → Teilen (WhatsApp, „Eigene Dateien“, Claude-App) | `ACTION_SEND`, `ACTION_SEND_MULTIPLE` | Datei kommt als `EXTRA_STREAM`; ein geteilter Text (`EXTRA_TEXT`) ist dagegen eine Rezept-URL (7.6) |
| Knopf „Datei importieren“ in der App | Dateiauswahl (`ACTION_OPEN_DOCUMENT`, mehrere Dateien) | Ausweichweg; WhatsApp-Dokumente liegen unter *Interner Speicher → Android → media → com.whatsapp → WhatsApp → Media → WhatsApp Documents*, meist auch unter „Zuletzt“ |

**Dateitypen:** Intent-Filter für `application/zip`, `application/x-zip-compressed`, `application/json` und `application/octet-stream` (WhatsApp und andere geben oft diesen allgemeinen Typ an). Nicht `*/*`, damit der Rezeptplaner nicht bei jedem Foto oder Dokument im Teilen-Menü steht. `text/plain` nur für geteilte URLs (7.6) und für `.json`, die manche Apps als Text weitergeben – dann entscheidet der Inhalt.

**Einlesen:** Andere Apps (WhatsApp) geben Dateien über eigene `content://`-Adressen mit befristeter Leseberechtigung und oft geändertem Namen weiter. Die App kopiert die Datei deshalb **sofort beim Empfang** in ihren Cache (höchstens 30 MB, wie die Grenze des Servers) und erkennt das Format **am Inhalt**: ZIP-Kennung `PK\x03\x04` bzw. `"format": "rezeptplaner-import"` im JSON. Andere Dateien lehnt sie sofort mit klarer Meldung ab („Keine Rezeptplaner-Importdatei“). Der Dateiname dient nur zur Anzeige.

**Ablauf:** Import-Bildschirm (erreichbar über Rezepte → „Datei importieren“ und über jeden der Wege oben) → Upload an `POST /api/v1/import/file` → Vorschau wie in der Weboberfläche (Bild, Titel, Portionen, Zutaten/Schritte, Hinweise zu Dubletten, unbekannten Kategorien und unsicheren Stellen; Rezepte einzeln abwählbar, Dubletten nicht vorausgewählt) → „Importieren“ → Abgleich. Nur online; ohne Verbindung „Import geht nur mit Verbindung zum Rezeptplaner“. Ist die App noch nicht verbunden, erst die Einrichtung, danach geht es mit der Datei weiter.

**Hinweis im Import-Bildschirm:** „Aus WhatsApp: Datei lange drücken → Teilen → Rezeptplaner“.

**Testfälle** (Samsung Galaxy A54, Android 16; vorher im Emulator, soweit dort möglich):

1. ZIP in WhatsApp antippen → „Öffnen mit“ → Rezeptplaner → Vorschau → Importieren
2. ZIP in WhatsApp lange drücken → Teilen → Rezeptplaner
3. einzelne `.json`-Datei auf beiden Wegen (1 und 2)
4. ZIP über „Datei importieren“ aus dem WhatsApp-Dokumentordner
5. Datei in der Samsung-App „Eigene Dateien“ antippen (Öffnen mit) und teilen
6. Datei direkt aus der Claude-App teilen; kann die Claude-App nur herunterladen: Download-Benachrichtigung antippen bzw. aus „Eigene Dateien → Downloads“ öffnen oder teilen (genauen Weg nach dem Test hier festhalten)
7. Gegenproben: ein Foto bzw. eine fremde ZIP teilen → klare Ablehnung; ohne Verbindung → Hinweis; mehrere Dateien auf einmal teilen → eine gemeinsame Vorschau

## 8. API-Übersicht

Alle Endpunkte unter `/api/v1`. Authentifizierung per API-Schlüssel (App-Zugang, Port 8100) bzw. über Ingress (Weboberfläche, Port 8099).

| Bereich | Funktionen |
|---|---|
| Rezepte | Liste mit Suche/Filtern, Einzelabruf (mit `parent_recipe_id` und `family`), Anlegen, Ändern, Löschen, Favorit setzen, Bild hochladen, Variante anlegen (`POST /recipes/{id}/variants`, der Server kopiert) |
| Import | URL einlesen (liefert Entwurf, Bildadresse und Dubletten), Zutatenzeilen zerlegen; Bild der Quellseite beim Speichern übernehmen; Datei-Import: `POST /import/file` (eine oder mehrere Dateien, liefert Vorschau, Kennung und `unknown_categories`), `POST /import/file/{kennung}` (ausgewählte Rezepte speichern; `categories` ordnet unbekannte Kategorien zu oder legt sie an) |
| Essensplan | Einträge eines Zeitraums, Anlegen, Ändern, Löschen, Verschieben/Tauschen mit Warnprüfung (erst prüfen, dann bestätigen) |
| Menüs | `POST /menus`, `GET/PUT/DELETE /menus/{id}` (`?with_entries=true` löscht die Gänge mit, sonst „auflösen“), `POST /menus/{id}/move` (wie beim Eintrag, auch `swap_with_menu`), `POST /menus/{id}/all-available`, `POST /meal-plan/{id}/to-menu`, `POST /meal-plan/{id}/detach`; Gänge über `POST /meal-plan` mit `menu_id` und `course_tag_id`; `GET /meal-plan` und `/sync` liefern `menus` |
| Einkauf | Vorschau der Einkaufsliste (aktueller oder folgender Einkaufszeitraum), An Bring senden, Differenz nachsenden |
| Stammdaten | Kategorien (`GET /tags`), Vorratsliste (inkl. Prüfung, ob Zutatennamen im Vorrat sind), Einstellungen |
| Kategorien verwalten | nur über Ingress: `POST /tags/bulk` (`add`/`remove` je `[{recipe_id, tag_id}]`, eine Transaktion; Antwort nennt nur tatsächlich Geändertes – vertauscht zurückgeschickt = rückgängig; Rezeptliste mit `course=none` und `tag_ids` je Rezept), `GET/POST /tag-groups`, `PUT /tag-groups/order`, `PUT/DELETE /tag-groups/{id}` (`?move_to=` oder `?with_tags=true`), `PUT /tag-groups/{id}/tags` (Inhalt und Reihenfolge), `POST /tags`, `PUT/DELETE /tags/{id}`, `POST /tags/{id}/merge` |
| Statistik | `GET /statistics?period=3m\|12m\|all` (bei Bedarf berechnet); Rezeptliste mit `planned=never` |
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

- **Claude-Skill als Download (7.6c):** Die GitHub Action packt `skills/rezeptplaner-import/` als `rezeptplaner-import-skill.zip` (Ordner `rezeptplaner-import/` mit `SKILL.md` darin) und hängt sie unter diesem festen Namen an jedes Android-Release. Ändert sich die Skill zwischen zwei Releases (Push auf `main` mit Änderungen unter `skills/`), ersetzt eine Action die Datei am neuesten Release. So zeigt `…/releases/latest/download/rezeptplaner-import-skill.zip` immer auf die aktuelle Skill. Ändert sich das Importformat, wird die Skill im selben Release mitgeändert (Regel in CLAUDE.md).

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
| 7.6 | (umgesetzt, App 0.6.0) Teilen einer URL (Browser, Chefkoch-App) bzw. „+“ → „Rezept aus Internetadresse“ öffnet den Entwurf zum Prüfen, Bild der Quellseite wird beim Speichern übernommen; neues Rezept, Rezept bearbeiten (Zutaten mit Überschriften, „Zeilen einfügen“ über den Zutaten-Parser des Servers, Vorratsabgleich für „kaufen“, Reihenfolge per Pfeil statt Ziehen – auf dem Handy zuverlässiger), Bild per Kamera/Galerie/entfernen, Favorit, Variante speichern (Zurück ohne Speichern verwirft sie), Löschen; nur online. Im Emulator geprüft (debug und qa/R8): URL-Entwurf mit Bild, Bearbeiten, Einfügen, Sortieren, Galerie, Favorit, Variante verwerfen, Löschen, geteilte URL vor der Einrichtung; Kamera steht noch aus (A54) |
| 7.6b | Datei-Import in der App, siehe „Datei-Import in der App (7.6b)“ unten: Teilen (`ACTION_SEND`/`_MULTIPLE`), „Öffnen mit“ (`ACTION_VIEW`), Knopf „Datei importieren“ (Dateiauswahl), WhatsApp als Quelle, Vorschau, nur online – App 0.5.0; im Emulator geprüft: Dateiauswahl, „Öffnen mit“ und Teilen aus der Dateien-App, mehrere Dateien, fremde ZIP, Foto (Rezeptplaner nicht im Teilen-Menü), Datei vor der Einrichtung; WhatsApp und Claude-App stehen noch aus |
| 7.6c | (umgesetzt, App 0.6.4) Claude-Skill als Download: ZIP `rezeptplaner-import-skill.zip` am neuesten Release (fester Link), Weboberfläche „App installieren“ → dritter Punkt „Claude-Skill“ mit Anleitung |
| 7.6d | (umgesetzt, Server 0.7.0, Skill 2.0) Skill vergibt fehlende Titel und zeichnet Illustrationen, wenn kein brauchbares Foto da ist; Format Version 2 mit `image_kind`, Filter „Illustrationen“ in Web und App |
| 7.7 | (umgesetzt, App 0.7.0) Foto-Import: „+“ → „Rezept aus Foto“, Seiten per Kamera-App oder aus der Galerie (bis 5 auf einmal, weitere Seiten anhängbar). Texterkennung von ML Kit über die Google-Play-Dienste auf dem Handy (Modell lädt das Handy selbst, APK bleibt klein; Fotos verlassen das Handy nicht). Zeilen werden vorbelegt – Überschriften „Zutaten“/„Zubereitung“ schalten den Abschnitt um, sonst Mengenangabe am Anfang = Zutat, lange Zeile/Satzende = Schritt, erste kurze Zeile = Titel; Seitenzahlen und Schrittnummern werden ignoriert. Korrektur: Art wählen, Zeile antippen; lange drücken markiert den Bereich seit der zuletzt angetippten Zeile. Schritte = Absätze der Texterkennung, Nummern und Silbentrennung am Zeilenende werden aufgelöst. Zutaten zerlegt der Server (gemeinsamer Parser, Vorratsabgleich), ohne Verbindung bleiben die Zeilen als Namen. Optional Gerichtfoto. Weiter im Bearbeiten-Bildschirm als Entwurf. Im Emulator geprüft (debug und qa/R8) mit einer Kochbuchseite aus der Galerie; Kamera steht noch aus (A54). Dazu Filter „Illustrationen“ (7.6d) |
| 7.8 | (umgesetzt, App 0.8.0) Einkauf wie in der Weboberfläche: Umschalter „Eingekaufte Woche“/„Nächster Einkauf“ (öffnet den nächsten Einkauf, wenn für die eingekaufte Woche nichts mehr zu senden ist), Gerichte mit Grund, warum sie nicht eingehen (Antippen öffnet die Kochansicht mit den Plan-Portionen), Artikel mit Häkchen, „Alle“/„Keine“, „An Bring senden“ immer sichtbar unten; danach Abgleich, damit der Plan „an Bring gesendet“ zeigt. „Nach dem Senden geändert“: Differenz nachsenden, zu viel Gesendetes als Hinweis. Ist keine Bring-Liste gewählt, bietet die App die Listen aus Home Assistant zur Auswahl an (speichert die Einstellung `bring_entity`). Nur online, gerechnet wird auf dem Server. Im Emulator geprüft (debug und qa/R8) mit der Testliste des lokalen Servers |
| 7.9 | (umgesetzt, Server 0.7.1, App 0.8.1) Messung nach Phase 7 (siehe Ressourcen auf dem Green), daraus gzip für JSON am App-Zugang; Logzeilen für Abgleich und Texterkennung; Doku: README (App, Skill), ANDROID.md Abschnitt 7 (Funktionen und Daten auf dem Handy) |
| 8. Feinschliff | Erkenntnisse aus dem Alltag: Einträge ohne Rezept in der App, Rezeptlink und Foto beim Datei-Import, Kategorien verwalten, Statistik | Schritte siehe unten |

**Schritte der Phase 8** (festgelegt 2026-10-24):

| Schritt | Inhalt |
|---|---|
| 8.A | (umgesetzt, App 0.9.0) App gleichwertig zum Web beim Planen: Rezeptauswahl mit Mittag/Abend, „Eintrag ohne Rezept“ (Freitext bis 200 Zeichen, Schnellauswahl „Essen gehen“, „Reste“, „Pizza bestellen“ wie im Web, dazu die bis zu fünf zuletzt verwendeten eigenen Texte aus den Daten auf dem Handy), Besteck-Symbol statt Bild, „Rezept wählen“ für Einträge ohne Rezept und umgekehrt Text statt Rezept beim Ersetzen. Dazu die kleinen Lücken: Favorit im Eintrags-Menü, Verschieben auf ein beliebiges Datum (Kalender), „Zutaten an Bring senden“ im Warnhinweis beim Verschieben, Kochansicht aus dem Plan mit „Im Essensplan: Tag · Portionen · Notiz“, Kategorie in der Kochansicht öffnet die gefilterte Rezeptübersicht, Trefferzahl und „Filter zurücksetzen“, Dubletten beim URL-Import als Link. Verschieben bleibt bei der Tagesauswahl (kein Ziehen) |
| 8.B | (umgesetzt, Server 0.8.0, Skill 2.1) Skill 2.1 sucht per Websuche den Originallink (`source.url`, nur bei eindeutigem Treffer); der Server lädt beim Datei-Import das Foto von dieser Seite, wenn das Rezept kein Bild oder nur eine Illustration hat (nie ein echtes Foto ersetzen), Vorschau zeigt es. Format bleibt Version 2. Grenzen: höchstens 20 s für alle Seiten einer Vorschau, zwei gleichzeitig, sonst wie der URL-Import (SSRF-Sperre, 5 MB Seite, 20 MB Bild); das Bild liegt bis zum Bestätigen im Upload-Ordner. Die Skill kann außerdem eine im Chat eingefügte Kategorieliste statt `categories.md` verwenden (`--categories`) |
| 8.C | (umgesetzt, Server 0.9.0, App 0.10.0) Kategorien und Gruppen verwalten (nur Weboberfläche): Tabelle `tag_groups` (Migration 011), Namen über alle Gruppen eindeutig (ohne Rücksicht auf Groß-/Kleinschreibung und Umlaute), Zusammenführen, Abgleich über `updated_at` der betroffenen Rezepte; Kategorieliste für die Claude-Skill kopieren; unbekannte Kategorien in der Import-Vorschau zuordnen, neu anlegen oder weglassen (Web und App) |
| 8.F | (umgesetzt, Server 0.12.0, App 0.13.0) QR-Code-Scanner beim URL-Import, siehe Abschnitt 6 A |
| 8.E | (umgesetzt, Server 0.11.0, App 0.12.0) Kurzvorschau beim Einplanen mit Zutaten-Abwahl je Eintrag, siehe „Einkaufsliste und Bring“. Migration 012 markiert alle Rezepte einmal als geändert, damit die App die neuen `item_key` beim Abgleich erhält. Die App schickt die neuen Felder nur mit, wenn sie gefüllt sind (ältere Server lehnen unbekannte Felder ab) |
| 8.D | (umgesetzt, Server 0.10.0, App 0.11.0) Statistik in Web und App (eigener Bereich „Statistik“ in der Navigation) (App offline aus den lokalen Daten, Zeitraum nur online); gezählt werden Planeinträge bis einschließlich heute – ebenso `plan_count` und die Sortierung „Am häufigsten eingeplant“; Filter „Noch nie eingeplant“ |

**Bewusst nicht in der App:** API-Schlüssel und verbundene Geräte, Adresse für die App, „App installieren“, Kategorien verwalten. **Nie geplant:** Plan weiter als 14 Tage zurück in der App, Ziehen mit dem Finger im Plan der App (Tagesauswahl genügt).

### Phase 9: Gang, Kuchen und Menüs (entschieden 2026-10-24)

Ziel: Rezepte nach Gang finden, Kuchen und Backwaren sauber planen und an Wochenenden oder bei Besuch Menüs aus mehreren Rezepten zusammenstellen, die im Plan als ein Eintrag erscheinen. Entscheidungen: Gang als markierte Kategoriegruppe (1-C), fester Slot „Kaffee & Kuchen“ (2a-A), Ergebnis-Einheit am Rezept (2b-B), Menü als Klammer um normale Planeinträge (3-A).

| Schritt | Inhalt |
|---|---|
| 9.1 | (umgesetzt, Server 0.13.0, App 0.14.0, Skill 2.2) **Gang:** Kategoriegruppe „Gang“ mit Kennzeichen `role = course` an `tag_groups` (Migration; genau eine Gruppe kann Gang-Gruppe sein, umschaltbar in der Kategorienverwaltung). Startliste: Vorspeise, Suppe/Eintopf und Salat (beide aus „Art“ verschoben, Zuordnungen bleiben), Hauptgericht, Beilage, Nachspeise, Kuchen & Torten, Gebäck, Brot, Snack. Schnellfilter-Reiter in Rezeptübersicht und Rezeptauswahl (Web und App): Alle \| Hauptgerichte \| Vorspeisen, Suppen & Salate \| Nachspeisen \| Backen (Kuchen & Torten, Gebäck, Brot). Seite „Gänge zuordnen“ (nur Web): Vorschläge nach Titel, Zutaten und Kategorien für Rezepte ohne Gang, gesammelt übernehmen. Skill: genau einen Gang vergeben (von Hand dürfen es mehrere sein) |
| 9.2 | (umgesetzt, Server 0.14.0, App 0.15.0, Skill 2.3, Format 3) **Ergebnis-Einheit** `yield_unit` am Rezept (leer = Portionen; freier Text mit Vorschlägen Stück, Kuchen, Blech, Springform 26 cm, Laib). Anzeige „1 Springform 26 cm“ statt „1 Portion“; Mengen werden weiter linear umgerechnet. Beim Einplanen gilt bei gesetzter Einheit die Grundmenge des Rezepts statt der Familiengröße. Importformat Version 3, Skill liest „für eine Springform …“, „ergibt 12 Stück“ aus. Umrechnung nach Formgröße später |
| 9.3 | (umgesetzt, Server 0.15.0, App 0.16.0, Integration 0.5.0) **Slot „Kaffee & Kuchen“** (`coffee`) zwischen Mittag und Abend (Migration mit Tabellenumbau wegen CHECK), abschaltbar in den Einstellungen (Standard an). Rezeptauswahl im Kaffee-Slot wählt den Reiter „Backen“ vor. Uhrzeiten für den Home-Assistant-Kalender in den Einstellungen: Mittag 12:00, Kaffee 15:30, Abend 18:30. Karte und Kalender zeigen „Kaffee:“ |
| 9.4 | (umgesetzt, Server 0.16.0, App 0.17.0, Integration 0.6.0) **Menüs:** Tabelle `meal_plan_menus` (Tag, Slot, Titel, Notiz, Portionen); jedes Gericht bleibt ein eigener Planeintrag mit `menu_id` und Gang. Plan: eine Karte „Menü: <Titel> · 4 Gänge“, ausklappbar; Titel standardmäßig das Hauptgericht, frei überschreibbar. Anlegen über Plus → „Menü zusammenstellen“ oder „Zu einem Menü erweitern“ am Eintrag; auch Einträge ohne Rezept erlaubt. Portionen am Menü, beim Ändern auf alle Gänge übertragbar, Abweichung je Gang möglich. Verschieben und Tauschen als ganzer Block (Warnregeln für alle Gänge zusammengefasst); „Aus Menü lösen“ für einen einzelnen Gang. Kurzvorschau mit einem Abschnitt je Gang. Einkauf, Bring, Differenzen und Abwahl unverändert pro Gericht. Kochansicht „Menü“ mit Notiz und allen Gängen. Home Assistant: ein Kalendertermin je Menü, Karte kompakt. Statistik zählt jedes Gericht einzeln plus „x Menüs im Zeitraum“ |
| 9.5 | (später) Menü-Vorlagen („Weihnachtsmenü“): speichern und wieder einplanen |
| 9.6 | (umgesetzt, Server 0.17.0, nur Weboberfläche) **Gänge von Hand zuordnen:** Reiter „Ohne Gang (n)“ in der Rezeptübersicht (`course=none`, kombinierbar mit Suche, Kategorien, Favoriten). Zuweisungsmodus „Gang zuweisen“: Ziel wählen, Kacheln an-/abwählen, „Alle sichtbaren auswählen“, „Übernehmen“ in einem Schritt (`POST /tags/bulk`, eine Transaktion, `updated_at` für den Abgleich), danach Hinweis mit „Rückgängig“; der Modus bleibt bis „Fertig“. Der Gang kommt **hinzu** (Entscheidung 2026-10-24, das Modell erlaubt mehrere Gänge); vorher steht, wie viele schon einen anderen Gang haben. Ziele auch „Gang entfernen“, „Alle Gänge entfernen“ sowie jede andere Kategorie (hinzufügen/entfernen). Kacheln zeigen ihre Gänge als Etikett. Einstellungen → Gänge: „Vorschläge für … prüfen“ öffnet den Modus mit den Vorschlägen ausgewählt. Die App bekommt die Zuordnungen über den Abgleich; dort ändert sich nichts |

---

## 11. Offene Punkte

- ~~**Bring-Mengenangabe**~~ – geklärt 2026-10-21: Die Beschreibung (`description`) eines To-do-Eintrags wird in Bring zur Spezifikation. Senden (18 Artikel in die gewählte Liste) und Nachsenden der Differenz auf dem Green mit App 0.3.0 getestet: ok.

- ~~**IDs nach dem Löschen**~~ – geklärt 2026-10-23: `AUTOINCREMENT` für die abgeglichenen Tabellen (Migration 008, App 0.5.1). Der Zähler startet hinter der größten ID, auch hinter bereits gelöschten aus `deletions`.

- **Googles Entwickler-Verifizierung (vor 2027 erledigen):** Ab 2027 sollen auf zertifizierten Geräten weltweit nur noch Apps registrierter Entwickler installierbar sein, auch außerhalb des Play Store (Details und Quellen in [ANDROID.md](ANDROID.md), Abschnitt 6). Plan: kostenloses Konto für eingeschränkten Vertrieb (bis 20 Geräte), Paketname mit dem Signaturzertifikat registrieren. Termin für Deutschland beobachten.

## 12. Mögliche spätere Erweiterungen

- **Ideen aus dem Alltag (2026-10-23), zurückgestellt:** ~~QR-Code-Scanner beim URL-Import~~ (umgesetzt als 8.F), Vorratsliste auf alle Rezepte anwenden (mit Ausnahmen; dazu in der App „Vorrat auf alle Rezepte übertragen“, das das Web schon hat), Bildersuche beim Bearbeiten, Online-Abgleich beim Foto-Import (Originalrezept suchen)
- **App, später:** Haushalt-Einstellungen (Familiengröße, Einkaufstag, Uhrzeit, Bring-Liste) und Vorratsliste in der App bearbeiten, Systeminfo (Speicher, Datenbankgröße) in den Einstellungen der App
- Optionale KI-Strukturierung beim Foto-Import
- Umrechnung unterschiedlicher Einheiten derselben Zutat
- Auslesen des tatsächlichen Abhak-Status aus Bring
- Cloudflare Access mit Service Token
