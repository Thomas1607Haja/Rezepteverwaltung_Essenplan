# Projektregeln für Claude

## Verbindliche Planung

[docs/PLANUNG.md](docs/PLANUNG.md) ist die verbindliche Planung (Architektur, fachliche Regeln, Datenmodell, API, Projektphasen). Vor Änderungen dort nachlesen. Weicht eine Umsetzung von der Planung ab, das ansprechen und die Planung nach Absprache fortschreiben.

## Kommunikation

- Mit dem Nutzer auf **Deutsch** kommunizieren.
- Der Nutzer lernt Go gerade: neue Go-Konzepte beim ersten Auftreten kurz erklären.
- In kleinen, nachvollziehbaren Schritten arbeiten und nach jedem größeren Schritt zur Bestätigung anhalten.

## Code und Dokumentation

- **Bezeichner im Code auf Englisch** (Variablen, Funktionen, Typen, Tabellen, Spalten, API-Pfade).
- **Kommentare und Dokumentation auf Deutsch.**
- Fachbegriffe im Code einheitlich mit dem Datenmodell der Planung benennen (z. B. `meal_plan_entries`, `shopping_relevant`).

## Öffentliches Repository

Das Repository ist öffentlich. Niemals einchecken:

- Zugangsdaten, Tokens, API-Schlüssel, Passwörter
- Domains, Tunnel-Konfiguration, IP-Adressen des Heimnetzes
- persönliche Daten (Datenbanken, hochgeladene Bilder, echte Rezeptsammlung)

Alles Persönliche gehört in die Add-on-Konfiguration oder die Einstellungen der Anwendung. Neue Arten lokaler Dateien sofort in die `.gitignore` aufnehmen.

## Schlankheit hat Vorrang

Das System läuft auf einem Home Assistant Green (ARM64, wenig RAM). Bei jeder Entscheidung gilt:

- Wenige, schlanke Abhängigkeiten; Go-Standardbibliothek bevorzugen.
- SQLite in reiner Go-Variante (ohne cgo).
- Keine Hintergrundlast im Leerlauf; Statistik und Statushinweise per Abfrage berechnen statt speichern.
- Bilder nur einmal beim Hochladen verkleinern.
- Nichts auf dem Green bauen: Images entstehen per GitHub Action.

## Git

- Identität (`user.name`, `user.email`) ist nur lokal in diesem Repository gesetzt. Globale Git-Einstellungen nicht verändern (auf dem Rechner existiert ggf. ein Firmenkonto).
- Remote läuft über den SSH-Alias `github-privat`.
- Committen und pushen nur nach Freigabe durch den Nutzer.
