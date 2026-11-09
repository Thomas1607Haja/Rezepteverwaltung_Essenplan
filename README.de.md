# Rezeptplaner

[English](README.md) · **Deutsch**

Ein selbst gebautes, schlankes System zur Verwaltung von Kochrezepten und zur Wochenplanung des Essens – als App (Add-on) für Home Assistant.

## Funktionen

- **Rezepte** per URL (z. B. chefkoch.de, auch englische, französische, spanische und italienische Rezeptseiten), per Foto (Android-App) oder aus einer Datei importieren, manuell anlegen und frei bearbeiten
- **Essensplan** mit frei scrollbaren Tagen, Planung beliebig weit im Voraus, Verschieben per Drag and Drop
- **Einkaufsliste** für den aktuellen Einkaufszeitraum, Übertragung an die Bring-App über Home Assistant
- **Dashboard-Karte** mit dem aktuellen Essensplan in Home Assistant
- **Kalender und Sensor** in Home Assistant (Integration über HACS, Repository [rezeptplaner-ha](https://github.com/Thomas1607Haja/rezeptplaner-ha))
- **Android-App:** Rezepte und Plan auch offline lesen, planen, Rezepte anlegen und bearbeiten, Rezept-Links teilen, Foto-Import mit Texterkennung auf dem Handy, Einkauf an Bring senden
- **Claude-Skill** „rezeptplaner-import“: erfasst Rezepte aus Fotos (Rezeptkarten, Kochbuchseiten, Zettel) und liefert eine Importdatei; fehlt ein Foto vom Gericht, zeichnet sie eine Illustration

## Sprachen

Weboberfläche, Android-App, Home-Assistant-Integration und Dashboard-Karte gibt es auf **Deutsch, Englisch, Französisch, Spanisch und Italienisch**. Fehlt eine Sprache oder ein einzelner Text, gilt Englisch.

- **Weboberfläche:** folgt der Systemsprache von Home Assistant (Einstellungen → System → Allgemein), sonst der Sprache des Browsers. Unter **Einstellungen → Haushalt → Sprache** lässt sich eine feste Sprache für den ganzen Haushalt wählen.
- **Android-App:** folgt der Sprache des Handys. Ab Android 13 lässt sich die Sprache nur für diese App umstellen (Android-Einstellungen → Apps → Rezeptplaner → Sprache).
- **Integration und Karte:** Die Integration nimmt die Systemsprache von Home Assistant, die Karte die Sprache des angemeldeten Nutzers.
- **Inhalte werden nie übersetzt:** Rezepte, Zutaten, Notizen und eigene Kategorien bleiben, wie sie eingegeben wurden. Eine neue Installation legt Startkategorien und Vorratsliste in der Sprache von Home Assistant an; bestehende Installationen bleiben unverändert.

Französisch, Spanisch und Italienisch sind ohne Prüfung durch Muttersprachler übersetzt – manches kann noch holpern. **Verbesserungen sind willkommen!** Wie man eine Übersetzung korrigiert oder eine neue Sprache beisteuert (Pull Request), steht in [docs/TRANSLATIONS.md](docs/TRANSLATIONS.md); die verbindlichen Begriffe im [Glossar](docs/glossar.md).

## Installation

1. **App:** In Home Assistant unter Einstellungen → Apps → App-Store → ⋮ → Repositories diese Adresse hinzufügen und „Rezeptplaner“ installieren:
   `https://github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan`
2. **Integration und Dashboard-Karte (optional):** Sie haben ein eigenes Repository: **[rezeptplaner-ha](https://github.com/Thomas1607Haja/rezeptplaner-ha)** – [in HACS öffnen](https://my.home-assistant.io/redirect/hacs_repository/?owner=Thomas1607Haja&repository=rezeptplaner-ha&category=integration), „Rezeptplaner“ herunterladen und Home Assistant neu starten. Die Integration erscheint dann unter Einstellungen → Geräte & Dienste als „Entdeckt“ – nur noch bestätigen. Einstellungen der Karte, der Hinweis zum Cloudflare-Tunnel und der Umstieg von diesem Repository (bis Integration 0.6 lag sie hier) stehen dort.

### Android-App

In der Weboberfläche unter **Einstellungen → App installieren**: Schritt 1 zeigt QR-Codes für Obtainium (installiert und aktualisiert die App aus den GitHub-Releases), Schritt 2 verbindet das Handy per QR-Code. Damit die App von unterwegs erreichbar ist, braucht der App-Zugang (Port 8100) einen eigenen Hostnamen, z. B. im bestehenden Cloudflare-Tunnel – Anleitung in [docs/APP-ZUGANG.md](docs/APP-ZUGANG.md). Details zur App: [docs/ANDROID.md](docs/ANDROID.md).

### Claude-Skill

Unter **Einstellungen → App installieren → Claude-Skill** liegt die aktuelle Skill zum Herunterladen. In Claude unter Einstellungen → Fähigkeiten → Skills hochladen; danach Fotos von Rezepten in einen Chat geben. Die erzeugte ZIP-Datei in der App per Teilen oder in der Weboberfläche unter **Rezepte → Datei importieren** übernehmen. Format: [docs/importformat.md](docs/importformat.md).

Die Skill erfasst Rezepte in der Sprache der Vorlage (sie übersetzt nicht) und schreibt Einheiten so, wie der Rezeptplaner sie beim Webseiten-Import schreibt. Läuft der Rezeptplaner nicht auf Deutsch, vorher unter **Einstellungen → Kategorien → „Kategorieliste für Claude kopieren“** die eigene Liste kopieren und in den Chat einfügen.

## Bausteine

| Baustein | Technik |
|---|---|
| Server | Go mit SQLite, läuft als Home-Assistant-Add-on |
| Weboberfläche | in den Server eingebettet, erreichbar über die HA-Seitenleiste |
| HA-Integration und Dashboard-Karte | Python bzw. JavaScript, installierbar über HACS – eigenes Repository [rezeptplaner-ha](https://github.com/Thomas1607Haja/rezeptplaner-ha) |
| Android-App | Kotlin, Jetpack Compose |

Grundsatz: Das System muss so schlank sein, dass es einen Home Assistant Green neben seiner eigentlichen Aufgabe kaum belastet.

## Projektstand

Fertig: Rezepte, Essensplan, Einkaufsliste mit Bring, Integration mit Dashboard-Karte, Android-App, Datei-Import mit Claude-Skill, fünf Sprachen. Die vollständige Planung – Architektur, fachliche Regeln, Datenmodell, Bildschirme und Projektphasen – steht in [docs/PLANUNG.md](docs/PLANUNG.md).

## Projektstruktur

```
rezeptplaner/                    Home-Assistant-Add-on
server/                          Go-Backend
web/                             Weboberfläche
android/                         Android-App
skills/                          Claude-Skill „rezeptplaner-import“ (mit Tests)
docs/                            Planung und Notizen
```

## Lizenz

[MIT](LICENSE)

## Unterstützen

Der Rezeptplaner ist kostenlos. Wenn er dir gefällt, freue ich mich über einen Kaffee.

<a href="https://buymeacoffee.com/rezeptplaner"><img src="https://cdn.buymeacoffee.com/buttons/v2/default-yellow.png" alt="Buy Me a Coffee" height="40"></a>
