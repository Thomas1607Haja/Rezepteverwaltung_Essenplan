# Importformat „rezeptplaner-import“, Version 1

Mit diesem Format kommen Rezepte aus Dateien in den Rezeptplaner – vor allem
aus der Claude-Skill [`skills/rezeptplaner-import/`](../skills/rezeptplaner-import/SKILL.md),
die Fotos von Rezeptkarten und Kochbuchseiten erfasst. Es lässt sich aber
auch von Hand oder mit anderen Programmen erzeugen.

Verbindlich ist das JSON-Schema [`importformat.schema.json`](importformat.schema.json)
(JSON Schema 2020-12). Die Skill bringt eine Kopie mit; ein Test im
Repository prüft, dass beide gleich sind. Server und Skill prüfen dieselben
Regeln.

## Dateien

| Form | Inhalt |
|---|---|
| **ZIP-Datei** (empfohlen) | `recipes.json` und optional Bilder im Ordner `images/`. `recipes.json` liegt oben oder in genau einem Ordner (z. B. ZIP eines Ordners `rezepte/`); Bildpfade gelten dann relativ zu diesem Ordner. |
| **einzelne `.json`-Datei** | Inhalt wie `recipes.json`, ohne Bilder. |

Mehrere Dateien lassen sich auf einmal hochladen (höchstens 10).

Das Format wird **am Inhalt erkannt**, nicht an Endung oder Dateityp (beim
Teilen auf dem Handy oft `application/octet-stream` oder ganz ohne Endung):
ZIP-Dateien an der Kennung `PK`, JSON am Feld `"format": "rezeptplaner-import"`.

Zeichensatz UTF-8 (ein BOM am Anfang wird ignoriert).

## Aufbau von `recipes.json`

```json
{
  "format": "rezeptplaner-import",
  "version": 1,
  "generator": "claude-skill rezeptplaner-import 1.0",
  "recipes": [ { … }, { … } ]
}
```

| Feld | Pflicht | Bedeutung |
|---|---|---|
| `format` | ja | immer `"rezeptplaner-import"` |
| `version` | ja | `1`; andere Versionen lehnt der Server mit Meldung ab |
| `generator` | nein | wer die Datei erzeugt hat (nur zur Information) |
| `recipes` | ja | 1 bis 100 Rezepte |

Unbekannte Felder sind **nicht erlaubt** – ein Tippfehler wie `"ingredient"`
statt `"ingredients"` fällt so auf, statt dass die Zutaten fehlen.

### Rezept

| Feld | Pflicht | Typ | Bedeutung |
|---|---|---|---|
| `title` | ja | Text, 1–200 Zeichen | |
| `base_servings` | ja | Ganzzahl 1–100 | Grundportionen, auf die sich die Mengen beziehen |
| `description` | nein | Text ≤ 2000 | kurze Beschreibung |
| `prep_minutes` | nein | Ganzzahl oder `null` | Vorbereitungszeit |
| `total_minutes` | nein | Ganzzahl oder `null` | Gesamtzeit |
| `source.name` | nein | Text ≤ 200 | Quelle ohne Adresse, z. B. „HelloFresh-Karte“, „Kochbuch: Einfach kochen“ |
| `source.url` | nein | `http(s)://…` | Webadresse; andere Schemata werden mit Hinweis weggelassen |
| `notes` | nein | Text ≤ 5000 | Notizen, auch unsichere oder unleserliche Stellen |
| `categories` | nein | Liste von Namen | Kategorien des Rezeptplaners (siehe unten) |
| `image` | nein | `images/<name>.jpg\|jpeg\|png\|webp` oder `null` | Rezeptbild in der ZIP-Datei |
| `ingredients` | nein | Liste (≤ 150) | Zutaten in Reihenfolge |
| `steps` | nein | Liste von Texten (≤ 100) | Zubereitungsschritte in Reihenfolge, ohne Nummern |

### Zutat

| Feld | Pflicht | Typ | Bedeutung |
|---|---|---|---|
| `name` | ja | Text 1–200 | nur die Zutat, z. B. „Zwiebel“ |
| `quantity` | nein | Zahl ≥ 0 oder `null` | Menge für `base_servings`; `null` = ohne Menge (wird beim Umrechnen nicht verändert) |
| `unit` | nein | Text ≤ 30 | `g`, `kg`, `ml`, `l`, `EL`, `TL`, `Msp.`, `Pck.`, `Stück`; Wörter wie `Prise`, `Zehe`, `Dose`, `Bund` bleiben als Wort |
| `note` | nein | Text ≤ 200 | Zusatz, z. B. „fein gewürfelt“, „nach Geschmack“, „1–2“ |
| `group` | nein | Text ≤ 100 | Gruppenüberschrift, z. B. „Für die Soße“ |
| `shopping_relevant` | nein | `true`/`false` | `false` = nicht einkaufen (Vorrat). Standard `true` |

## Was der Import macht

1. **Vorschau** (`POST /api/v1/import/file`, multipart, Feld `file`): Der
   Server prüft alles und speichert noch nichts. Je Rezept kommen das
   umgewandelte Rezept, ein Vorschaubild, **Fehler** (Rezept nicht
   importierbar) und **Hinweise**:
   - **Dublette:** gleicher Titel (ohne Rücksicht auf Groß-/Kleinschreibung
     und Umlaute) oder gleiche `source.url` wie ein vorhandenes Rezept.
     Die Weboberfläche wählt Dubletten nicht vor.
   - **Unbekannte Kategorie:** wird **nicht** angelegt, sondern weggelassen.
     Groß-/Kleinschreibung spielt keine Rolle.
   - **Vorrat:** Zutaten, die auf der Vorratsliste stehen (auch über
     Synonyme), werden wie beim URL-Import auf `shopping_relevant: false`
     gesetzt. Umgekehrt wird nie etwas auf `true` gesetzt.
   - **Bild:** fehlt, unlesbar oder kein echtes Bild → Rezept ohne Bild.
2. **Bestätigen** (`POST /api/v1/import/file/{token}` mit
   `{"selected": [0, 2]}`): Die ausgewählten Rezepte werden angelegt. Die
   hochgeladenen Dateien liegen bis dahin höchstens **1 Stunde** auf dem
   Server und werden danach (bzw. nach dem Bestätigen) gelöscht.

Vorschaubild eines Rezepts der Vorschau: `GET /api/v1/import/file/{token}/images/{index}`.

## Grenzen und Sicherheit

| Grenze | Wert |
|---|---|
| Upload (alle Dateien zusammen) | 30 MB |
| Dateien je Upload | 10 |
| Einträge je ZIP-Datei | 200 |
| `recipes.json` bzw. `.json`-Datei | 2 MB |
| ein Bild (entpackt) | 20 MB, höchstens 40 Megapixel |
| alles Entpackte eines Uploads | 150 MB |
| Rezepte je Upload | 100 |

- ZIP-Einträge mit `..`, absoluten Pfaden, `\`, Laufwerksbuchstaben oder
  Symlinks führen zur Ablehnung der Datei (Zip Slip). Entpackt wird nur im
  Speicher; die Größe wird beim Entpacken gezählt, nicht aus den Angaben der
  ZIP-Datei übernommen (ZIP-Bombe).
- Bilder werden am Inhalt erkannt (JPEG, PNG, WebP, GIF) und wie beim Upload
  neu gespeichert (verkleinert, JPEG) – EXIF-Daten einschließlich GPS fallen
  dabei weg.

## Kategorien

Erlaubt sind die Namen der Kategorien im Rezeptplaner. Die Startkategorien
stehen in `server/internal/database/migrations/002_tags.sql`; die Skill
führt sie in `skills/rezeptplaner-import/categories.md` (ein Test vergleicht
beide). Werden Kategorien im Rezeptplaner geändert, die Liste der Skill
nachziehen.

## Änderungen am Format

Nur abwärtskompatible Ergänzungen bleiben bei Version 1 (dann Schema, Kopie
in der Skill und Server gemeinsam ändern). Alles andere bekommt eine neue
`version`; der Server lehnt unbekannte Versionen mit klarer Meldung ab.

## Beispiel

Siehe [`skills/rezeptplaner-import/example/recipes.json`](../skills/rezeptplaner-import/example/recipes.json).
