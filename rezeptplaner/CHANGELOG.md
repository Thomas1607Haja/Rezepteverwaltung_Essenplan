# Änderungen

## 0.7.1

- **Weniger Daten für die App:** Der App-Zugang komprimiert JSON-Antworten (gzip). Der volle Abgleich mit 500 Rezepten schrumpft von 1,4 MB auf rund 130 KB; das spart vor allem den Upload aus dem Heimnetz.

## 0.7.0

- **Illustrierte Rezepte:** Die Claude-Skill (Version 2.0) zeichnet ein Bild, wenn die Vorlage kein brauchbares Foto vom Gericht hat, und vergibt einen Titel, wenn keiner da ist (vermerkt in den Notizen). Solche Rezepte tragen in der Übersicht das Kennzeichen „Illustration“; der neue Filter **Illustrationen** findet sie, um sie durch ein Foto zu ersetzen. Ein neu hochgeladenes Bild gilt wieder als Foto.
- Importformat **Version 2** (Feld `image_kind`); Dateien der Version 1 werden weiter angenommen. Die neue Skill braucht diese Version des Rezeptplaners.

## 0.6.4

- **Claude-Skill zum Herunterladen:** Einstellungen → App installieren → „Claude-Skill“ mit Download-Link (immer die aktuelle Version am neuesten Release) und Anleitung zum Hochladen in Claude.

## 0.6.3

- **Eingekaufte Woche und nächster Einkauf:** Der Essensplan zeigt die eingekaufte Woche (warm) und die Woche des nächsten Einkaufs (hellblau) mit Legende.
- Neue Einstellung **Uhrzeit des Einkaufs** (Einstellungen → Haushalt): Erst ab dieser Uhrzeit gilt am Einkaufstag die neue Woche als eingekauft; vorher ist sie der nächste Einkauf.
- Einkaufsseite: Schalter „Eingekaufte Woche / Nächster Einkauf“; sie öffnet direkt den nächsten Einkauf, wenn für die eingekaufte Woche nichts mehr zu senden ist.

## 0.6.2

- **Adresse für die App prüfen:** Beim Speichern (und per Knopf „Adresse prüfen“) ruft der Rezeptplaner die eingetragene Adresse selbst ab und meldet, ob dort der App-Zugang antwortet – oder ob sie zu Home Assistant, zur Weboberfläche oder ins Leere führt.

## 0.6.1

- **Einstellungen → App installieren:** Ein neues Handy in zwei Schritten einrichten. Schritt 1 zeigt QR-Codes für Obtainium und den Eintrag des Rezeptplaners darin (Updates automatisch) bzw. die neueste APK direkt, dazu eine Anleitung für das Galaxy A54. Schritt 2 erzeugt erst nach „Neues Gerät verbinden“ einen Schlüssel; der QR-Code zum Verbinden ist nur einmal sichtbar.
- Neue Option **app_repository** (Reiter „Konfiguration“): GitHub-Repository mit den Releases der Android-App, nur für einen eigenen Fork ändern.

## 0.6.0

- **Datei-Import:** Unter **Rezepte → Datei importieren** lassen sich Rezepte aus einer ZIP-Datei (mit Bildern) oder JSON-Datei übernehmen – z. B. aus der neuen Claude-Skill „rezeptplaner-import“, die Fotos von Rezeptkarten und Kochbuchseiten erfasst. Vor dem Speichern zeigt eine Vorschau jedes Rezept mit Bild, Hinweisen auf Dubletten, unbekannte Kategorien und unsichere Stellen; Rezepte lassen sich einzeln abwählen. Vorratszutaten werden automatisch nicht zum Einkaufen vorgemerkt.
- **Quelle als Name:** Rezepte können eine Quelle ohne Webadresse haben, z. B. „HelloFresh-Karte“ oder „Kochbuch“ (Rezeptansicht und Bearbeiten).
- **Vorbereitung Android-App:** Abgleich-Schnittstelle für die App, Einstellung **Adresse für die App** und QR-Code zur Einrichtung beim Anlegen eines API-Schlüssels (Einstellungen → App-Zugang).

## 0.5.1

- Vorbereitung für die Android-App: Rezepte, Planeinträge, Vorrat und Kategorien bekommen nach dem Löschen nie wieder dieselbe Nummer, damit der spätere Abgleich mit der App eindeutig bleibt. Die Datenbank wird beim ersten Start einmalig umgestellt; vorhandene Daten bleiben erhalten.

## 0.5.0

- **Rezeptvarianten:** „Variante speichern“ in der Rezeptansicht legt eine Kopie als eigenes Rezept an (mit Zutaten, Schritten, Kategorien und eigenem Bild) und öffnet sie gleich zum Bearbeiten. Varianten bleiben mit dem Ursprungsrezept verbunden: Mini-Kacheln aller Varianten direkt unter dem Titel (seitwärts scrollbar), Hinweis „Variante von …“, Kennzeichen in der Übersicht. Wird das Ursprungsrezept gelöscht, bleiben die Varianten erhalten.

## 0.4.1

- **App-Zugang für die Android-App** auf Port 8100: Jede Anfrage braucht einen API-Schlüssel. Schlüssel pro Gerät unter **Einstellungen → App-Zugang** anlegen und sperren; gespeichert wird nur ein Hash. Der Port ist nach außen nicht freigegeben (Anleitung für den Cloudflare-Tunnel in `docs/APP-ZUGANG.md`).
- Sicherheit: aktuelle Go-Version (behebt Lücken in der Standardbibliothek), Content-Security-Policy und weitere Schutz-Kopfzeilen für die Weboberfläche, Quell-URLs nur noch mit http/https, Schlüssel der Integration nur noch für Plan, Rezepte und Bilder, weitere gesperrte Adressbereiche beim URL-Import, kürzere Fehlermeldungen (Einzelheiten im Protokoll).

## 0.4.0

- **Integration für Home Assistant** (über HACS, siehe DOCS): Die App meldet sich selbst an, die Integration muss nur bestätigt werden. Sie bringt den Kalender **Essensplan**, den Sensor **Heute** und die Dashboard-Karte **Essensplan** mit (Tage mit Bild und Gericht, Antippen öffnet die Kochansicht).
- Die App nutzt dafür die Discovery des Supervisors; die Integration darf nur lesen.

## 0.3.0

- **Einkauf**: Einkaufsliste für diese oder nächste Woche aus dem Essensplan – Mengen auf die Portionen umgerechnet, gleiche Zutaten zusammengefasst (auch „Zwiebel“ und „Zwiebeln“), Eier, Dosen usw. ganz aufgerundet. Einzelne Artikel lassen sich abwählen.
- **An Bring senden** über die Bring-Integration von Home Assistant; die Menge steht in Bring unter dem Artikel. Steht ein Artikel schon auf der Liste, wird nur die Menge ergänzt.
- Nachträgliche Änderungen: Was nach dem Senden noch fehlt, lässt sich nachsenden; zu viel Gesendetes wird angezeigt.
- Beim Verschieben eines Gerichts in den Einkaufszeitraum: „Zutaten an Bring senden“.
- **Vorratsliste** in den Einstellungen (Salz, Öl, … mit Synonymen): Solche Zutaten werden in neuen Rezepten nicht zum Einkaufen vorgemerkt; per Knopf auf alle Rezepte übertragbar.
- Neue Einstellung **Bring-Liste**.
- Der Eintrag in der Seitenleiste heißt jetzt **Essensplan**.
- Die App nutzt jetzt die Home-Assistant-API (`homeassistant_api`), um in die Bring-Liste zu schreiben.

## 0.2.0

- **Essensplan** als Startseite: Tage nebeneinander (Handy 2, Tablet 4, Rechner 7), endlos scrollbar, Einkaufszeitraum farbig.
- Gerichte planen über das Plus am Tag oder „Zum Essensplan hinzufügen“ im Rezept; mehrere Gerichte pro Tag, Mittagessen und Einträge ohne Rezept (z. B. „Essen gehen“).
- Menü am Eintrag: Portionen, Notiz, „Zutaten sind vorhanden“, Rezept ersetzen, verschieben, entfernen.
- **Drag and Drop**: auf ein Gericht ziehen tauscht, auf freie Fläche hängt an; auf dem Handy lange drücken. Warnung, wenn ein Gericht ohne eingekaufte Zutaten in den Einkaufszeitraum rutscht.
- Statushinweis an Rezepten („eingeplant in 5 Tagen“, „zuletzt vor 2 Wochen“) und Sortierung nach „zuletzt“ bzw. „am häufigsten eingeplant“.
- Neue Einstellungen: Familiengröße und Einkaufstag.

## 0.1.1

- Neue Systeminfo unter **Einstellungen**: Version, Laufzeit, Arbeitsspeicher (aktuell und Spitze), Größe von Datenbank und Bildern.
- Die Version steht jetzt auch im Protokoll beim Start.

## 0.1.0

- Erste Version als Home-Assistant-App: Rezepte anlegen, bearbeiten, per Adresse übernehmen, Kochansicht.
