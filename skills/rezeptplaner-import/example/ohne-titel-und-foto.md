# Beispiel: Rezept ohne Titel und ohne Foto

## Vorlage

Ein Foto eines handgeschriebenen Zettels, kein Foto vom Gericht. Lesbar ist:

```
500 g Hähnchenbrust
1 Dose Kokosmilch
2 EL rote Currypaste
1 rote Paprika
1 Zucchini
Reis 250 g

Fleisch würfeln, in Öl anbraten. Currypaste kurz mitbraten.
Gemüse dazu, mit Kokosmilch ablöschen, 15 Min köcheln.
Mit Reis servieren.
```

## Erwartetes Ergebnis

- **Titel vergeben** nach den Hauptbestandteilen:
  „Hähnchen-Curry mit Kokosmilch, Paprika und Reis“ – nicht „Curry“,
  „Leckeres Hähnchengericht“ oder „Omas bestes Curry“.
- **Notiz:** „Titel automatisch vergeben (Vorlage ohne Titel). Portionen
  nicht angegeben, 4 angenommen. Bild ist eine Illustration (kein Foto vom
  Gericht).“
- **Bild:** Der Zettel ist ein Textfoto und wird nie Rezeptbild. Stattdessen
  eine Illustration.
- **Quelle:** „Handschriftlich“. Keine Zeit angegeben → `total_minutes`
  weglassen; die 15 Minuten Köcheln sind keine Gesamtzeit.

Eintrag in der Arbeitsdatei (vor `build`):

```json
{
  "title": "Hähnchen-Curry mit Kokosmilch, Paprika und Reis",
  "base_servings": 4,
  "source": { "name": "Handschriftlich" },
  "notes": "Titel automatisch vergeben (Vorlage ohne Titel). Portionen nicht angegeben, 4 angenommen. Bild ist eine Illustration (kein Foto vom Gericht).",
  "categories": ["Geflügel", "Herzhaft", "Reis", "Pfanne"],
  "illustration": {
    "vessel": "plate",
    "ingredients": [
      { "name": "Reis", "shape": "grains", "color": "#F4EFE1", "amount": "some" },
      { "name": "Currysoße", "shape": "sauce", "color": "#E3A33B", "amount": "some" },
      { "name": "Hähnchen", "shape": "chunks", "color": "#E9C99A" },
      { "name": "Paprika", "shape": "strips", "color": "#C8372D", "amount": "little" },
      { "name": "Zucchini", "shape": "slices", "color": "#6E9B3F", "amount": "little" },
      { "name": "Koriander", "shape": "herbs", "color": "#3F8A3C", "amount": "little" }
    ]
  },
  "ingredients": ["… wie in recipes.json …"],
  "steps": ["… wie in recipes.json …"]
}
```

Fertiger Eintrag: zweites Rezept in [`recipes.json`](recipes.json)
(`image_kind: "illustration"`).

Antwort im Chat (Ausschnitt):

> - Hähnchen-Curry mit Kokosmilch, Paprika und Reis – 4 Portionen – 7 Zutaten,
>   4 Schritte – mit Illustration
>   - **Titel vergeben:** „Hähnchen-Curry mit Kokosmilch, Paprika und Reis“
>     (Zettel ohne Überschrift) – bitte prüfen.
>   - **Illustration statt Foto:** kein Foto vom Gericht vorhanden.
>   - Unsicher: Portionen nicht angegeben, 4 angenommen.
