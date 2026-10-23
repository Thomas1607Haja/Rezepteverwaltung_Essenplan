---
name: rezeptplaner-import
description: Erfasst Rezepte aus Fotos (Rezeptkarten wie HelloFresh mit Vorder- und Rückseite, Kochbuchseiten, Zeitschriften, handgeschriebene Rezepte) und erzeugt daraus eine Importdatei (ZIP) für den selbst gebauten Rezeptplaner. Verwenden, sobald jemand Fotos von Rezepten schickt und sie erfassen, importieren, digitalisieren oder in den Rezeptplaner bzw. die Rezepte-App übernehmen möchte – auch bei mehreren Rezepten auf einmal.
---

# Rezeptplaner-Import

Du erfasst Rezepte aus Fotos und lieferst **eine ZIP-Datei** im Format
„rezeptplaner-import“, Version 1. Der Nutzer schickt sie per Teilen an die
Rezeptplaner-App (oder lädt sie in der Weboberfläche hoch); dort erscheint
eine Vorschau, erst danach wird gespeichert.

Sprache: **Deutsch** – im Chat und in allen Rezepttexten.

Dateien dieser Skill (Pfade relativ zum Skill-Ordner):

- `schema.json` – verbindliches JSON-Schema von `recipes.json`
- `categories.md` – die **einzigen** erlaubten Kategorien
- `example/recipes.json` – Beispiel einer fertigen Datei
- `scripts/build_import.py` – baut, prüft und packt die ZIP-Datei

## Ablauf

1. **Fotos sichten und zuordnen.** Ein Rezept kann aus mehreren Fotos
   bestehen (Vorderseite mit Gericht, Rückseite mit Zutaten und Schritten;
   Doppelseiten im Kochbuch). Ordne die Fotos am **Inhalt** zu – gleicher
   Titel, gleiches Gericht, fortlaufende Schritte –, nicht an der
   Reihenfolge. Ist eine Zuordnung unklar, frage kurz nach, bevor du
   weitermachst.
2. **Jedes Rezept vollständig ablesen** nach den Regeln unten.
3. **Rezeptbild festlegen**: nur ein Foto, das das fertige Gericht zeigt.
   Bestimme den Zuschnitt (Anteile 0–1 von links, oben, rechts, unten), so
   dass nur das Gerichtfoto übrig bleibt – ohne Kartenrand, Text, Logo oder
   Hand. Prüfe den Zuschnitt mit `crop` und sieh dir das Ergebnis an;
   korrigiere, bis es passt.
4. **Arbeitsdatei schreiben** (Aufbau wie `example/recipes.json`, aber mit
   `image_source` statt `image`, siehe unten) und mit `build` die ZIP-Datei
   erzeugen. Meldet das Skript Fehler, korrigieren und erneut bauen. Gib nie
   eine Datei aus, die das Skript nicht als `OK` gebaut hat.
5. **Ausgabe** im Chat (siehe „Antwort“).

## Regeln für den Inhalt

**Titel:** wie auf der Vorlage, ohne Werbezusätze („Neu!“, „Bestseller“),
normale Groß-/Kleinschreibung statt GROSSBUCHSTABEN.

**Beschreibung:** der kurze Einleitungssatz der Vorlage, falls vorhanden;
sonst leer lassen. Nichts dazudichten.

**Grundportionen (`base_servings`):** Hat die Karte mehrere Mengenspalten
(z. B. für 2, 3 und 4 Personen), übernimm **die Spalte für 2 Personen** und
setze `base_servings` auf 2. Sonst die angegebene Personenzahl. Fehlt jede
Angabe, nimm 4 und vermerke das in `notes`.

**Zeiten:** `total_minutes` = Gesamt- bzw. Zubereitungszeit der Vorlage,
`prep_minutes` = Vorbereitungs-/Arbeitszeit, nur wenn ausdrücklich genannt.
Nicht angegeben → weglassen (nicht schätzen).

**Zutaten** – je Zutat ein Eintrag:

- `quantity`: Zahl für die gewählten Grundportionen (`½` → 0.5,
  `1 ½` → 1.5). Bereiche wie „1–2“: kleinere Zahl als Menge, den Bereich in
  `note` („1–2“). Ohne Zahl („nach Geschmack“, „etwas“) → `null`, den Text
  in `note`.
- `unit` in dieser Schreibweise: `g`, `kg`, `ml`, `l`, `EL`, `TL`,
  `Msp.`, `Pck.` (auch für „Pckg.“, „Päckchen“), `Stück`; Wörter wie
  `Prise`, `Zehe`/`Zehen`, `Dose`, `Becher`, `Bund`, `Scheibe`/`Scheiben`,
  `Glas`, `Zweig` bleiben als Wort stehen. Ohne Einheit → `""`.
- `name`: nur die Zutat („Zwiebel“, „Cherrytomaten“). Zusätze wie
  „fein gehackt“ oder „TK“ gehören in `note`.
- `group`: Gruppenüberschrift der Vorlage („Für die Soße“). Zutaten, die
  die Karte als **„Aus deiner Vorratskammer“ / „Das brauchst du noch“ /
  „Haben wir zu Hause“** ausweist, in die Gruppe
  `"Aus deiner Vorratskammer"`.
- `shopping_relevant: false` für alles, was die Vorlage als
  Vorrat/„selbst im Haus“ ausweist, und für Grundzutaten wie Salz, Pfeffer,
  Öl, Zucker, Wasser. Sonst weglassen (Standard: einkaufen).
- Mitgelieferte Zutaten (HelloFresh-Box) und „brauchst du noch“-Zutaten
  gehören beide in die Liste – nur die Vorratszutaten werden markiert.

**Schritte (`steps`):** je Arbeitsschritt ein Eintrag, in Reihenfolge,
**ohne Nummer** am Anfang. Lange Kartenschritte mit mehreren Absätzen
dürfen als ein Schritt bleiben. Weglassen: Kopfzeilen, Werbung,
Nährwerte, Allergene, Bestellhinweise, QR-Code-Texte, Weinempfehlungen.
Ein „Tipp“ der Vorlage kommt in `notes`.

**Kategorien:** nur Namen aus `categories.md`, genau so geschrieben.
Lieber zu wenige als falsche. Nicht passende Begriffe der Vorlage
(„Familienliebling“, „Low Carb“) nicht übernehmen.

**Quelle (`source.name`):** „HelloFresh-Karte“ bei HelloFresh-Karten,
sonst z. B. „Kochbuch: <Titel>“ (falls der Buchtitel erkennbar ist),
„Kochbuch“, „Zeitschrift: <Name>“, „Handschriftlich“. Steht eine
Webadresse zum Rezept auf der Vorlage, zusätzlich `source.url` (nur
`http`/`https`). Nichts erfinden.

**Nichts erfinden:** Unleserliche oder abgeschnittene Mengen, Zeiten oder
Wörter **nicht raten**. Menge `null`, Zeit weglassen, und in `notes` des
Rezepts festhalten, was unsicher ist („Unsicher: Menge Sahne unleserlich“).
Fehlt ein ganzer Teil (z. B. Rückseite nicht fotografiert), das Rezept
trotzdem erfassen und den Hinweis in `notes` schreiben.

## Rezeptbild

- Nur ein Foto des **fertigen Gerichts** (typisch: Vorderseite der Karte).
  **Fotos von Text nie als Rezeptbild verwenden.** Kein Gerichtfoto →
  kein Bild (`image_source` weglassen).
- Zuschnitt: großzügig das Gericht, keine Ränder, Schrift, Logos, Finger.
  Querformat ist gut, quadratisch auch.
- Das Skript verkleinert auf höchstens 1600 px Kantenlänge, speichert als
  JPEG und entfernt dabei alle Metadaten (auch GPS).
- Kannst du die Fotos nicht als Datei öffnen (kein Zugriff in der
  Ausführungsumgebung), baue die Rezepte ohne Bild und sage das im Chat.

## Arbeitsdatei und Skript

Die Arbeitsdatei ist eine JSON-Datei mit `"recipes": [...]` im Format von
`schema.json`. Beim Rezept steht statt `"image"`:

```json
"image_source": {
  "file": "/mnt/user-data/uploads/karte-1-vorne.jpg",
  "crop": [0.0, 0.0, 1.0, 0.62],
  "rotate": 0
}
```

`rotate` dreht gegen den Uhrzeigersinn (90 = Hochformat-Foto einer
Querformat-Karte nach links drehen); die EXIF-Ausrichtung des Handys wird
vorher schon berücksichtigt.

```bash
# Zuschnitt ansehen
python scripts/build_import.py crop --file /mnt/user-data/uploads/karte-1-vorne.jpg \
    --crop 0 0 1 0.62 --out /tmp/vorschau-1.jpg

# ZIP-Datei bauen (prüft gegen schema.json und categories.md)
python scripts/build_import.py build --data /tmp/arbeit.json --out-dir /mnt/user-data/outputs

# vorhandene Datei prüfen
python scripts/build_import.py check /mnt/user-data/outputs/rezepte-….zip
```

Pfade: Hochgeladene Dateien liegen meist unter `/mnt/user-data/uploads/`,
Ausgaben gehören nach `/mnt/user-data/outputs/`. Weicht die Umgebung ab,
passende Ordner verwenden. Fehlt Pillow: `pip install pillow`.

Der Dateiname entsteht automatisch, z. B.
`rezepte-2026-10-23-gnocchi-pfanne-und-2-weitere.zip`. **Alle Rezepte eines
Durchgangs kommen in eine ZIP-Datei.**

## Antwort

Nach dem Bauen im Chat, kurz:

1. Die ZIP-Datei als Download.
2. Liste der erkannten Rezepte: Titel – Portionen – Anzahl Zutaten/Schritte
   – mit/ohne Bild.
3. **Alle unsicheren Stellen** je Rezept (dieselben wie in `notes`) und
   Hinweise des Skripts (z. B. weggelassene Kategorien).
4. Ein Satz zum Weg in die App: „Datei antippen → Teilen → Rezeptplaner“
   bzw. in der Weboberfläche „Rezepte → Datei importieren“.

Keine langen Erklärungen, keine Wiederholung aller Zutaten im Chat.
