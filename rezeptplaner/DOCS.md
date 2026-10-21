# Rezeptplaner

## Installation

1. In Home Assistant **Einstellungen → Apps → App-Store** öffnen.
2. Oben rechts im Menü (⋮) **Repositories** wählen und diese Adresse hinzufügen:
   `https://github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan`
3. Im App-Store **Rezeptplaner** auswählen und **Installieren**.
4. **Starten** und **In Seitenleiste anzeigen** einschalten.

Die Weboberfläche erscheint dann als **Essensplan** in der Seitenleiste.

## Einstellungen

Die App selbst hat keine Optionen in Home Assistant. Familiengröße (Standard-Portionen im Essensplan) und Einkaufstag (Beginn des einwöchigen Einkaufszeitraums) stellst du in der Weboberfläche unter **Einstellungen** ein.

## Zugriff und Sicherheit

- Die App ist nur über Home Assistant erreichbar (Ingress). Die Anmeldung übernimmt Home Assistant; die App öffnet keinen eigenen Port.
- Beim Übernehmen eines Rezepts aus dem Internet lädt die App die Seite und das Bild selbst. Adressen im Heimnetz sind dabei gesperrt.

## Daten und Backups

Datenbank und Bilder liegen im Datenordner der App und sind Teil der Home-Assistant-Backups. Während eines Backups wird die App für einige Sekunden gestoppt, damit die Datenbank in einem sauberen Zustand gesichert wird.

## Bekannte Einschränkungen

- Nur für ARM64-Geräte (z. B. Home Assistant Green).
- Einkaufsliste mit Bring und Android-App sind noch in Arbeit.

## Hilfe und Fehler melden

Fragen und Fehler bitte als Issue im Repository melden:
https://github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/issues

## Lizenz

Siehe [LICENSE](https://github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/blob/main/LICENSE) im Repository.
