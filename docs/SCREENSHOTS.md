# Screenshots für die READMEs

Alle Bilder **auf Englisch** (die deutschen READMEs nutzen dieselben Bilder),
helles Design, mit echten, appetitlichen Beispielrezepten mit Foto – keine
privaten Namen, Adressen oder Hostnamen im Bild. Format PNG.

Nach dem Ablegen in den READMEs den auskommentierten Block „Screenshots“
einkommentieren (README.md und README.de.md bzw. im Integrations-Repository).

## Vorbereitung

- **Browser:** Weboberfläche auf Englisch stellen (**Settings → Household →
  Language → English**, danach wieder zurück). Browserfenster ca. 1440 × 900,
  Zoom 100 %. Nur den Inhaltsbereich des Rezeptplaners aufnehmen (ohne die
  Seitenleiste von Home Assistant), z. B. mit Win + Umschalt + S.
- **Galaxy A54:** Sprache nur für die App auf Englisch: **Einstellungen → Apps →
  Rezeptplaner → Sprache → English**. Dunkelmodus aus. Screenshot mit
  Ein/Aus + Leiser. Statusleiste möglichst aufgeräumt (keine Benachrichtigungen).
- Im Plan sollten mehrere Tage mit Gerichten stehen, davon ein Menü mit Gängen.

## Hauptrepository `Rezepteverwaltung_Essenplan` → `docs/screenshots/`

| Datei | Gerät | Was zu sehen sein soll |
|---|---|---|
| `web-meal-plan.png` | Browser | **Meal plan**: eine Woche mit Mittag und Abend, ein Menü mit Gängen, Bilder sichtbar |
| `web-recipe.png` | Browser | Rezeptdetail mit Foto, Zutaten (Portionsrechner auf z. B. 4) und Schritten |
| `web-shopping.png` | Browser | **Shopping**: Liste der Woche mit dem Knopf **Send to Bring** |
| `web-import-preview.png` | Browser | **Recipes → Import file** mit der Vorschau einer Datei der Claude-Skill (2–3 Rezepte mit Bild, ein Hinweis) |
| `android-meal-plan.png` | A54 | Startseite Essensplan |
| `android-recipe.png` | A54 | Kochansicht eines Rezepts |
| `android-shopping.png` | A54 | Einkauf mit **Send to Bring** |
| `android-photo-import.png` | A54 | Foto-Import: erkannte Zutaten nach dem Foto einer Kochbuchseite |

## Integrations-Repository `rezeptplaner-ha` → `docs/images/`

| Datei | Gerät | Was zu sehen sein soll |
|---|---|---|
| `card.png` | Browser | Dashboard mit der Karte **Meal plan** (z. B. 5 Tage), Home Assistant auf Englisch (Profil → Language), helles Design; nur die Karte und etwas Rand |
| `card-editor.png` | Browser | (optional) Karteneditor mit den Feldern Title, Days, Height |
