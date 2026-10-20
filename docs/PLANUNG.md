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
| Weboberfläche | Vue oder Svelte, Drag-and-Drop-Bibliothek | Wird statisch gebaut und in die Go-Datei eingebettet; kein Node.js auf dem Green |
| Add-on | Docker-Image für Home Assistant | Ingress für die Seitenleiste, eigener Port für die App, Zugriff auf HA-API über Supervisor |
| Android-App | Kotlin, Jetpack Compose, Room, ML Kit | Offline-Lesen, Teilen-Funktion, Foto-Import |
| HA-Integration | Python (von Home Assistant vorgegeben) | Installierbar über HACS |
| Dashboard-Karte | JavaScript (Lovelace Custom Card) | Installierbar über HACS |

### Zugriff und Sicherheit

- **Weboberfläche:** über Ingress in der HA-Seitenleiste; Anmeldung erfolgt über Home Assistant.
- **App:** über einen zusätzlichen Hostnamen im bestehenden Cloudflare-Tunnel (z. B. `rezepte.eure-domain.de`), der direkt auf den Port des Add-ons zeigt.
- **API-Schlüssel:** einer pro Gerät, nur als Hash gespeichert, einzeln sperrbar. Einrichtung der App per QR-Code (enthält Serveradresse und Schlüssel).
- Optional später: Cloudflare Access mit Service Token.

### Ressourcen auf dem Green

- Server läuft im Leerlauf praktisch ohne CPU-Last.
- Bilder werden einmalig beim Hochladen verkleinert (großes Bild + Vorschaubild).
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

Build-Befehl (später in der GitHub Action):

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

### Zutaten

- Jede Zutat ist ein eigener Baustein: Reihenfolge per Drag and Drop, Menge/Einheit/Name/Notiz bearbeitbar, Häkchen „einkaufsrelevant“.
- Optionale Gruppenüberschriften (z. B. „Für die Soße“).
- Eine leere Menge bedeutet „nicht umrechnen“ (z. B. „1 Prise“, „nach Geschmack“).
- **Globale Vorratsliste** (mit Synonymen): Zutaten darauf werden beim Import/Anlegen automatisch als nicht einkaufsrelevant markiert. Pro Rezept überschreibbar.

### Portionen

- Jedes Rezept hat Grundportionen; jeder Planeintrag hat eigene Portionen (Standard: Familiengröße aus den Einstellungen).
- Die Kochansicht rechnet Mengen per Plus/Minus um, ohne das Rezept zu ändern.
- Umgerechnete Mengen werden sinnvoll gerundet (z. B. keine 1,33 Eier).

### Essensplan

- Einträge mit Datum und Slot statt fester Wochen.
- Slots: *Abendessen* (Standard) und optional *Mittagessen* (z. B. für Ferien). Pro Datum und Slot höchstens ein Eintrag.
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

- Ziehen auf einen belegten Tag/Slot tauscht die Gerichte.
- Die Prüfung der Warnregeln erfolgt auf dem Server; erst nach Bestätigung wird gespeichert.

| Situation | Reaktion |
|---|---|
| Tausch zweier übertragener Gerichte innerhalb des Einkaufszeitraums | Keine Meldung |
| Gericht mit Status `planned` rutscht in den Einkaufszeitraum | Dialog: „Trotzdem verschieben“ / „Zutaten sind vorhanden“ / „Fehlende Zutaten jetzt an Bring senden“ |
| Übertragenes Gericht wandert aus dem Einkaufszeitraum | Hinweis: Zutaten bereits eingekauft, evtl. verderblich |
| Portionen oder Rezept eines übertragenen Eintrags ändern sich | Hinweis mit Angebot, nur die Differenz nachzusenden |

### Einkaufsliste und Bring

- Grundlage: alle Einträge im Einkaufszeitraum mit Status `planned`.
- Mengen werden nach Portionen skaliert, Vorratsartikel und nicht einkaufsrelevante Zutaten entfallen.
- Gleiche Zutaten werden zusammengefasst. Unterschiedliche Einheiten derselben Zutat werden zunächst nicht umgerechnet, sondern gemeinsam angegeben.
- Vor dem Senden lassen sich einzelne Artikel abwählen.
- Übertragung über die Bring-Integration von Home Assistant (To-do-Entität) per Supervisor-API; Artikelname und Mengenangabe getrennt, soweit möglich.
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

### `recipes`

| Feld | Typ | Bemerkung |
|---|---|---|
| id | Integer | Primärschlüssel |
| title | Text | |
| description | Text | optional |
| base_servings | Integer | Grundportionen |
| prep_minutes | Integer | optional |
| total_minutes | Integer | optional |
| source_url | Text | optional, für Dublettenprüfung |
| image | Text | Dateiname großes Bild |
| thumbnail | Text | Dateiname Vorschaubild |
| is_favorite | Boolean | |
| notes | Text | eigene Notizen |
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
| synonyms | Text/Liste | z. B. „Olivenöl, Rapsöl, Sonnenblumenöl“ |

### `meal_plan_entries`

| Feld | Typ | Bemerkung |
|---|---|---|
| id | Integer | |
| date | Datum | |
| slot | Text | `dinner` (Standard) oder `lunch` |
| recipe_id | Integer | optional (Einträge ohne Rezept möglich) |
| servings | Integer | |
| status | Text | `planned`, `transferred`, `have_ingredients` |
| transferred_at | Zeitstempel | optional |
| note | Text | optional, z. B. „Essen gehen“ |

Eindeutigkeit: (`date`, `slot`).

### `transferred_items`

| Feld | Typ |
|---|---|
| id | Integer |
| entry_id | Integer |
| ingredient_name | Text |
| quantity | Dezimalzahl |
| unit | Text |

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
- `shopping_day`
- `bring_entity` (To-do-Entität der Bring-Liste in Home Assistant)

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
- Kacheln mit Bild, Titel, Zeit, Statushinweis.
- Sortierung: Name, zuletzt eingeplant, am häufigsten eingeplant, neu hinzugefügt.
- Plus-Button für Import/Anlegen; Zugang zur Statistik.

### 5.4 Rezeptansicht (Kochansicht)

- Großes Bild, Titel, Herzchen, Kategorien, Zeiten, Statushinweis.
- Portionen-Zähler mit sofortiger Umrechnung.
- Zutaten mit Gruppenüberschriften, nummerierte Schritte.
- Buttons „Zum Essensplan hinzufügen“ (Dialog mit den nächsten Tagen und deren Belegung; bei belegtem Tag: Ersetzen oder als zweites Gericht) und „Bearbeiten“.
- Bildschirm bleibt während der Kochansicht eingeschaltet.

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
- Kein Bearbeiten auf der Karte; Link zur Weboberfläche.

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
3. Übertragung an die Bring-Liste in Home Assistant.
4. Einträge erhalten Status `transferred`, Artikel werden in `transferred_items` protokolliert.

### F. Nachträgliche Änderungen

Vergleich mit den übertragenen Artikeln; Angebot, nur die Differenz nachzusenden.

---

## 7. Android-App: Offline-Verhalten

- Lokale Datenbank (Room) mit allen Rezepten, dem Essensplan (einige Tage zurück bis Ende der Planung) und den Einstellungen.
- Abgleich nur der Änderungen seit dem letzten Abgleich (über `updated_at` und `deletions`).
- Vorschaubilder immer lokal; große Bilder für den aktuellen Einkaufszeitraum vorab, sonst beim ersten Ansehen.
- Abgleich bei App-Start, beim Herunterziehen und regelmäßig im Hintergrund (alle paar Stunden).
- Offline: Banner „Offline – Stand von …“, Lesen und Kochansicht funktionieren, Bearbeiten und Foto-Import sind deaktiviert.

---

## 8. API-Übersicht

Alle Endpunkte unter `/api/v1`. Authentifizierung per API-Schlüssel (App) bzw. über Ingress (Weboberfläche).

| Bereich | Funktionen |
|---|---|
| Rezepte | Liste mit Suche/Filtern, Einzelabruf, Anlegen, Ändern, Löschen, Favorit setzen, Bild hochladen |
| Import | URL einlesen (liefert Entwurf), Zutatenzeilen zerlegen |
| Essensplan | Einträge eines Zeitraums, Anlegen, Ändern, Löschen, Verschieben/Tauschen mit Warnprüfung (erst prüfen, dann bestätigen) |
| Einkauf | Vorschau der Einkaufsliste, An Bring senden, Differenz nachsenden |
| Stammdaten | Kategorien, Vorratsliste, Einstellungen |
| Statistik | Häufigkeiten nach Zeitraum |
| Abgleich | Änderungen seit Zeitpunkt X |
| Verwaltung | API-Schlüssel anlegen (mit QR-Code), sperren |
| Bilder | Bild und Vorschaubild abrufen |

---

## 9. Projektstruktur

Das Repository ist gleichzeitig Add-on-Repository für Home Assistant und HACS-Repository für die Integration. Beides stellt Anforderungen an die Struktur:

```
Rezepteverwaltung_Essenplan/
├── repository.yaml                  Kennzeichnet das Repo als Add-on-Repository
├── hacs.json                        HACS-Metadaten für die Integration
├── rezeptplaner/                    Add-on: config.yaml, Doku, Icon
├── custom_components/rezeptplaner/  HA-Integration (Pfad von HACS vorgegeben)
├── server/                          Go-Backend
├── web/                             Weboberfläche (wird in den Server eingebettet)
├── ha-card/                         Quellcode der Dashboard-Karte
├── android/                         Android-App
├── .github/workflows/               Automatische Builds
└── docs/                            Planung und Notizen
```

### Builds und Installation

- **Add-on-Image wird nicht auf dem Green gebaut.** Ohne vorgefertigtes Image würde der Green bei jeder Installation und jedem Update Go und die Weboberfläche selbst kompilieren. Stattdessen baut eine GitHub Action das ARM64-Image und legt es in der GitHub Container Registry (ghcr.io) ab; die `config.yaml` des Add-ons verweist darauf. Der Green lädt nur noch das fertige Image herunter.
- **Dashboard-Karte wird von der Integration mitgeliefert.** Die gebaute JavaScript-Datei der Karte liegt in der Integration, die sie beim Start in Home Assistant registriert. So genügt eine einzige HACS-Installation für Integration und Karte. (Bei der Umsetzung prüfen, ob das automatische Registrieren wie geplant funktioniert; sonst Fallback auf ein eigenes Repository für die Karte.)
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
| 7. Android-App | QR-Einrichtung, Offline-Speicher, Teilen-Funktion, Foto-Import mit ML Kit | |
| 8. Feinschliff | Statistik, Kategorien-Verwaltung, Erkenntnisse aus dem Alltag | |

---

## 11. Offene Punkte

- **Frontend-Framework:** Vue oder Svelte – Entscheidung zu Beginn von Phase 2.
- **Bring-Mengenangabe:** Prüfen, wie die HA-To-do-Schnittstelle Artikelname und Spezifikation an Bring übergibt.
- **Git-Identität:** Name/E-Mail pro Repository und eigener SSH-Schlüssel für das private GitHub-Konto einrichten.

## 12. Mögliche spätere Erweiterungen

- Optionale KI-Strukturierung beim Foto-Import
- Umrechnung unterschiedlicher Einheiten derselben Zutat
- Auslesen des tatsächlichen Abhak-Status aus Bring
- Cloudflare Access mit Service Token
