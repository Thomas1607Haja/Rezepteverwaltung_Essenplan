# Rezeptplaner

Ein selbst gebautes, schlankes System zur Verwaltung von Kochrezepten und zur Wochenplanung des Essens – als Add-on für Home Assistant.

## Funktionen (geplant)

- **Rezepte** per URL (z. B. chefkoch.de) oder per Foto importieren, manuell anlegen und frei bearbeiten
- **Essensplan** mit frei scrollbaren Tagen, Planung beliebig weit im Voraus, Verschieben per Drag and Drop
- **Einkaufsliste** für den aktuellen Einkaufszeitraum, Übertragung an die Bring-App über Home Assistant
- **Dashboard-Karte** mit dem aktuellen Essensplan in Home Assistant
- **Android-App** mit Offline-Lesezugriff und Foto-Import

## Bausteine

| Baustein | Technik |
|---|---|
| Server | Go mit SQLite, läuft als Home-Assistant-Add-on |
| Weboberfläche | in den Server eingebettet, erreichbar über die HA-Seitenleiste |
| HA-Integration und Dashboard-Karte | Python bzw. JavaScript, installierbar über HACS |
| Android-App | Kotlin, Jetpack Compose |

Grundsatz: Das System muss so schlank sein, dass es einen Home Assistant Green neben seiner eigentlichen Aufgabe kaum belastet.

## Projektstand

Das Projekt befindet sich am Anfang der Entwicklung. Die vollständige Planung – Architektur, fachliche Regeln, Datenmodell, Bildschirme und Projektphasen – steht in [docs/PLANUNG.md](docs/PLANUNG.md).

## Projektstruktur

```
rezeptplaner/                    Home-Assistant-Add-on
custom_components/rezeptplaner/  HA-Integration (HACS)
server/                          Go-Backend
web/                             Weboberfläche
ha-card/                         Dashboard-Karte
android/                         Android-App
docs/                            Planung und Notizen
```

## Lizenz

[MIT](LICENSE)
