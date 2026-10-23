# Kategorien des Rezeptplaners

> **Bei Änderungen aktualisieren:** Diese Liste muss zu den Kategorien im
> Rezeptplaner passen (Startkategorien aus
> `server/internal/database/migrations/002_tags.sql`; eigene Änderungen in
> den Einstellungen des Rezeptplaners hier nachtragen). Ein Test im
> Repository vergleicht sie mit den Startkategorien. Unbekannte Kategorien
> legt der Import nicht an, sondern meldet sie und lässt sie weg.

Nur diese Namen (genau so geschrieben) in `categories` verwenden. Pro
Rezept nur zutreffende Kategorien wählen, lieber zu wenige als falsche.

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
- Suppe/Eintopf
- Salat
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
- **Vegetarisch:** kein Fleisch, kein Fisch. **Vegan** nur, wenn auch keine
  Milchprodukte, Eier oder Honig vorkommen (dann zusätzlich Vegetarisch).
- **Hackfleisch** zusätzlich zu Rind/Schwein, wenn erkennbar (z. B.
  „gemischtes Hackfleisch“ → Hackfleisch, Rind, Schwein).
- **Herzhaft** für fast alle Hauptgerichte, **Süß** für Desserts, Kuchen,
  Milchreis usw.
- **Schnell:** Gesamtzeit höchstens 30 Minuten.
- **Kinderliebling, Gut vorzubereiten, Resteverwertung, Für Gäste** nur,
  wenn die Vorlage das ausdrücklich sagt.
