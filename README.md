# Rezeptplaner

Ein selbst gebautes, schlankes System zur Verwaltung von Kochrezepten und zur Wochenplanung des Essens – als App (Add-on) für Home Assistant.

## Funktionen

- **Rezepte** per URL (z. B. chefkoch.de), per Foto (Android-App) oder aus einer Datei importieren, manuell anlegen und frei bearbeiten
- **Essensplan** mit frei scrollbaren Tagen, Planung beliebig weit im Voraus, Verschieben per Drag and Drop
- **Einkaufsliste** für den aktuellen Einkaufszeitraum, Übertragung an die Bring-App über Home Assistant
- **Dashboard-Karte** mit dem aktuellen Essensplan in Home Assistant
- **Kalender und Sensor** in Home Assistant (Integration über HACS)
- **Android-App:** Rezepte und Plan auch offline lesen, planen, Rezepte anlegen und bearbeiten, Rezept-Links teilen, Foto-Import mit Texterkennung auf dem Handy, Einkauf an Bring senden
- **Claude-Skill** „rezeptplaner-import“: erfasst Rezepte aus Fotos (Rezeptkarten, Kochbuchseiten, Zettel) und liefert eine Importdatei; fehlt ein Foto vom Gericht, zeichnet sie eine Illustration

## Installation

1. **App:** In Home Assistant unter Einstellungen → Apps → App-Store → ⋮ → Repositories diese Adresse hinzufügen und „Rezeptplaner“ installieren:
   `https://github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan`
2. **Integration und Dashboard-Karte (optional):** In HACS → ⋮ → Benutzerdefinierte Repositories dieselbe Adresse mit Typ „Integration“ hinzufügen, „Rezeptplaner“ herunterladen und Home Assistant neu starten. Die Integration erscheint dann unter Einstellungen → Geräte & Dienste als „Entdeckt“ – nur noch bestätigen.
3. Nach der Einrichtung die Seite einmal neu laden (F5), dann im Dashboard eine Karte „Essensplan“ hinzufügen (in der Kartenauswahl oder als `type: custom:rezeptplaner-card`). Die Integration trägt die Karte selbst als Dashboard-Ressource ein (Einstellungen → Dashboards → ⋮ → Ressourcen) und hält den Eintrag bei Updates aktuell. Werden die Ressourcen in YAML gepflegt (`lovelace: resource_mode: yaml`), dort einen Eintrag mit `url: /rezeptplaner_static/rezeptplaner-card.js` und `type: module` ergänzen; bis dahin lädt die Integration die Karte ersatzweise über die Startseite.

   Einstellungen der Karte (im visuellen Editor oder in YAML):

   | Feld | Bedeutung | Standard |
   |---|---|---|
   | `title` | Überschrift | „Essensplan“ |
   | `days` | Anzahl Tage ab heute (1–31) | aktueller Einkaufszeitraum |
   | `height` | Höhe in Pixel; passt nicht alles hinein, scrollt der Inhalt innerhalb der Karte | so hoch wie der Inhalt |

### Zugriff über einen Cloudflare-Tunnel

Cloudflare speichert ohne eigene Regel auch Fehlerantworten zwischen und weist Browser an, sie 4 Stunden zu behalten. Fragt ein Browser oder die Companion-App die Karte während eines Neustarts von Home Assistant ab, bleibt sie danach stundenlang verschwunden („Konfigurationsfehler“). Abhilfe im Cloudflare-Dashboard der Domain:

1. **Caching → Cache Rules → Create rule:** Name „Home Assistant nicht cachen“, Bedingung *Hostname equals* `<HA-Hostname>`, Aktion **Bypass cache**, speichern.
2. **Caching → Configuration → Browser Cache TTL:** **Respect Existing Headers**.
3. Einmalig **Caching → Configuration → Purge Everything**; in der Companion-App unter Einstellungen → Companion-App → Fehlerbehebung den Frontend-Cache zurücksetzen.

### Android-App

In der Weboberfläche unter **Einstellungen → App installieren**: Schritt 1 zeigt QR-Codes für Obtainium (installiert und aktualisiert die App aus den GitHub-Releases), Schritt 2 verbindet das Handy per QR-Code. Damit die App von unterwegs erreichbar ist, braucht der App-Zugang (Port 8100) einen eigenen Hostnamen, z. B. im bestehenden Cloudflare-Tunnel – Anleitung in [docs/APP-ZUGANG.md](docs/APP-ZUGANG.md). Details zur App: [docs/ANDROID.md](docs/ANDROID.md).

### Claude-Skill

Unter **Einstellungen → App installieren → Claude-Skill** liegt die aktuelle Skill zum Herunterladen. In Claude unter Einstellungen → Fähigkeiten → Skills hochladen; danach Fotos von Rezepten in einen Chat geben. Die erzeugte ZIP-Datei in der App per Teilen oder in der Weboberfläche unter **Rezepte → Datei importieren** übernehmen. Format: [docs/importformat.md](docs/importformat.md).

## Bausteine

| Baustein | Technik |
|---|---|
| Server | Go mit SQLite, läuft als Home-Assistant-Add-on |
| Weboberfläche | in den Server eingebettet, erreichbar über die HA-Seitenleiste |
| HA-Integration und Dashboard-Karte | Python bzw. JavaScript, installierbar über HACS |
| Android-App | Kotlin, Jetpack Compose |

Grundsatz: Das System muss so schlank sein, dass es einen Home Assistant Green neben seiner eigentlichen Aufgabe kaum belastet.

## Projektstand

Fertig: Rezepte, Essensplan, Einkaufsliste mit Bring, Integration mit Dashboard-Karte, Android-App, Datei-Import mit Claude-Skill. Als Nächstes: Feinschliff (Statistik, Kategorien verwalten). Die vollständige Planung – Architektur, fachliche Regeln, Datenmodell, Bildschirme und Projektphasen – steht in [docs/PLANUNG.md](docs/PLANUNG.md).

## Projektstruktur

```
rezeptplaner/                    Home-Assistant-Add-on
custom_components/rezeptplaner/  HA-Integration (HACS) mit Dashboard-Karte (frontend/)
server/                          Go-Backend
web/                             Weboberfläche
android/                         Android-App
skills/                          Claude-Skill „rezeptplaner-import“ (mit Tests)
docs/                            Planung und Notizen
```

## Lizenz

[MIT](LICENSE)
