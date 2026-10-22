# Rezeptplaner

Ein selbst gebautes, schlankes System zur Verwaltung von Kochrezepten und zur Wochenplanung des Essens – als App (Add-on) für Home Assistant.

## Funktionen

- **Rezepte** per URL (z. B. chefkoch.de) oder per Foto importieren, manuell anlegen und frei bearbeiten
- **Essensplan** mit frei scrollbaren Tagen, Planung beliebig weit im Voraus, Verschieben per Drag and Drop
- **Einkaufsliste** für den aktuellen Einkaufszeitraum, Übertragung an die Bring-App über Home Assistant
- **Dashboard-Karte** mit dem aktuellen Essensplan in Home Assistant
- **Kalender und Sensor** in Home Assistant (Integration über HACS)
- **Android-App** mit Offline-Lesezugriff und Foto-Import (geplant)

## Installation

1. **App:** In Home Assistant unter Einstellungen → Apps → App-Store → ⋮ → Repositories diese Adresse hinzufügen und „Rezeptplaner“ installieren:
   `https://github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan`
2. **Integration und Dashboard-Karte (optional):** In HACS → ⋮ → Benutzerdefinierte Repositories dieselbe Adresse mit Typ „Integration“ hinzufügen, „Rezeptplaner“ herunterladen und Home Assistant neu starten. Die Integration erscheint dann unter Einstellungen → Geräte & Dienste als „Entdeckt“ – nur noch bestätigen.
3. Nach der Einrichtung die Seite einmal neu laden (F5), dann im Dashboard eine Karte „Essensplan“ hinzufügen (in der Kartenauswahl oder als `type: custom:rezeptplaner-card`).

### Zugriff über einen Cloudflare-Tunnel

Cloudflare speichert ohne eigene Regel auch Fehlerantworten zwischen und weist Browser an, sie 4 Stunden zu behalten. Fragt ein Browser oder die Companion-App die Karte während eines Neustarts von Home Assistant ab, bleibt sie danach stundenlang verschwunden („Konfigurationsfehler“). Abhilfe im Cloudflare-Dashboard der Domain:

1. **Caching → Cache Rules → Create rule:** Name „Home Assistant nicht cachen“, Bedingung *Hostname equals* `<HA-Hostname>`, Aktion **Bypass cache**, speichern.
2. **Caching → Configuration → Browser Cache TTL:** **Respect Existing Headers**.
3. Einmalig **Caching → Configuration → Purge Everything**; in der Companion-App unter Einstellungen → Companion-App → Fehlerbehebung den Frontend-Cache zurücksetzen.

## Bausteine

| Baustein | Technik |
|---|---|
| Server | Go mit SQLite, läuft als Home-Assistant-Add-on |
| Weboberfläche | in den Server eingebettet, erreichbar über die HA-Seitenleiste |
| HA-Integration und Dashboard-Karte | Python bzw. JavaScript, installierbar über HACS |
| Android-App | Kotlin, Jetpack Compose |

Grundsatz: Das System muss so schlank sein, dass es einen Home Assistant Green neben seiner eigentlichen Aufgabe kaum belastet.

## Projektstand

Fertig: Rezepte, Essensplan, Einkaufsliste mit Bring, Integration mit Dashboard-Karte. In Arbeit: Android-App. Die vollständige Planung – Architektur, fachliche Regeln, Datenmodell, Bildschirme und Projektphasen – steht in [docs/PLANUNG.md](docs/PLANUNG.md).

## Projektstruktur

```
rezeptplaner/                    Home-Assistant-Add-on
custom_components/rezeptplaner/  HA-Integration (HACS) mit Dashboard-Karte (frontend/)
server/                          Go-Backend
web/                             Weboberfläche
android/                         Android-App
docs/                            Planung und Notizen
```

## Lizenz

[MIT](LICENSE)
