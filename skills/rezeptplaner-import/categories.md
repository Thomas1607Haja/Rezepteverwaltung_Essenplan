# Kategorien des Rezeptplaners

> **Bei Änderungen aktualisieren:** Diese Liste muss zu den Kategorien im
> Rezeptplaner passen (Startkategorien aus
> `server/internal/database/migrations/002_tags.sql`; eigene Änderungen in
> den Einstellungen des Rezeptplaners hier nachtragen). Ein Test im
> Repository vergleicht sie mit den Startkategorien. Unbekannte Kategorien
> meldet der Import; der Nutzer ordnet sie in der Vorschau zu, legt sie an
> oder lässt sie weg. Eine im Chat eingefügte, aktuelle Liste aus dem
> Rezeptplaner (Einstellungen → Kategorien) hat Vorrang vor dieser Datei.

Nur diese Namen (genau so geschrieben) in `categories` verwenden. Pro
Rezept nur zutreffende Kategorien wählen, lieber zu wenige als falsche.

## Gang
- Vorspeise
- Suppe/Eintopf
- Salat
- Hauptgericht
- Beilage
- Nachspeise
- Kuchen & Torten
- Gebäck
- Brot
- Snack

## Hauptzutat
- Vegetarisch
- Vegan
- Rind
- Schwein
- Geflügel
- Hackfleisch
- Fisch
- Meeresfrüchte

## Geschmack
- Herzhaft
- Süß

## Art
- Pasta
- Reis
- Kartoffeln
- Auflauf/Ofen
- Pfanne
- Grill

## Eigenschaften
- Schnell
- Kinderliebling
- Gut vorzubereiten
- Resteverwertung
- Für Gäste

## Hinweise zur Auswahl
- **Gang:** immer **genau einen** vergeben – der Gang, als der das Gericht
  meistens gegessen wird (Rezeptplaner filtert danach: Hauptgerichte,
  Nachspeisen, Backen …). Kuchen, Torten, Tartes → „Kuchen & Torten“;
  Kekse, Muffins, Hefegebäck → „Gebäck“; Brot und Brötchen → „Brot“;
  Desserts, Pudding, Mousse → „Nachspeise“; Eintöpfe zählen als
  „Suppe/Eintopf“; Bowls und Hauptgericht-Salate als „Salat“.
- **Vegetarisch:** kein Fleisch, kein Fisch. **Vegan** nur, wenn auch keine
  Milchprodukte, Eier oder Honig vorkommen (dann zusätzlich Vegetarisch).
- **Hackfleisch** zusätzlich zu Rind/Schwein, wenn erkennbar (z. B.
  „gemischtes Hackfleisch“ → Hackfleisch, Rind, Schwein).
- **Herzhaft** für fast alle Hauptgerichte, **Süß** für Desserts, Kuchen,
  Milchreis usw.
- **Schnell:** Gesamtzeit höchstens 30 Minuten.
- **Kinderliebling, Gut vorzubereiten, Resteverwertung, Für Gäste** nur,
  wenn die Vorlage das ausdrücklich sagt.
