# Änderungen

## 0.17.0

- **Gänge schnell von Hand zuordnen (Weboberfläche):** In der Rezeptübersicht gibt es den Reiter „Ohne Gang“ mit Anzahl – kombinierbar mit Suche, Kategorien und Favoriten. „Gang zuweisen“ schaltet in einen Auswahlmodus: Gang wählen, Rezepte anklicken (oder „Alle sichtbaren auswählen“), „Übernehmen“ – mit „Rückgängig“ für einige Sekunden. Der Modus bleibt offen für den nächsten Gang bis „Fertig“. Auch „Gang entfernen“ und jede andere Kategorie lassen sich so zuweisen oder entfernen. Die Kacheln zeigen ihren Gang.
- Unter Einstellungen → Gänge öffnet „Vorschläge für … prüfen“ diesen Modus mit den Vorschlägen bereits ausgewählt.
- Die App erhält die neuen Zuordnungen beim nächsten Abgleich; eine neue App-Version ist nicht nötig.

## 0.16.0

- **Menüs:** Mehrere Gänge an einem Abend, z. B. Suppe, Hauptgericht und Nachtisch. Anlegen über Plus → „Menü zusammenstellen …“ oder am Eintrag „Zu einem Menü erweitern“. Jeder Gang bleibt ein eigenes Gericht im Plan (Einkauf, Bring und abgewählte Zutaten wie bisher); das Menü hat einen Titel (automatisch das Hauptgericht), eine Notiz und die Personenzahl, die sich auf alle Gänge übertragen lässt. Verschieben und Tauschen nur als ganzes Menü, ein einzelner Gang lässt sich lösen. „Zutaten vorhanden …“ für alle Gänge auf einmal, Kochansicht mit allen Gängen. Die Statistik zählt jedes Gericht und zusätzlich die Menüs.
- Die Home-Assistant-Integration ab 0.6.0 zeigt ein Menü als einen Kalendertermin mit allen Gängen; Karte und Sensor nennen das Menü einmal.
- Die Datenbank wird beim ersten Start erweitert; alle Planeinträge bleiben erhalten.

## 0.15.0

- **Kaffee & Kuchen:** Neuer Slot zwischen Mittag- und Abendessen, z. B. für Kuchen am Wochenende. Beim Einplanen für den Kaffee zeigt die Rezeptauswahl gleich die Backrezepte. Abschaltbar unter Einstellungen → Haushalt.
- **Uhrzeiten für den Kalender:** Mittag, Kaffee und Abend (Standard 12:00, 15:30, 18:30) unter Einstellungen → Haushalt. Die Home-Assistant-Integration ab 0.5.0 zeigt die Gerichte damit als Termine zu diesen Uhrzeiten statt ganztägig; Karte und Sensor kennen den Kaffee-Slot und zeigen Kuchen als „1 Springform 26 cm“.
- Die Datenbank wird beim ersten Start umgestellt; alle Planeinträge bleiben erhalten.

## 0.14.0

- **Kuchen, Brot und Gebäck:** Rezepte haben jetzt eine Einheit für die Menge (Editor: „Einheit“, z. B. „Springform 26 cm“, „Blech“, „Stück“, „Kuchen“; leer = Portionen). Angezeigt wird „1 Springform 26 cm“ statt „1 Portion“; Mengen werden wie gewohnt umgerechnet. Beim Einplanen gilt dann die Menge des Rezepts statt der Familiengröße.
- Importformat **Version 3** (Feld `yield_unit`); ältere Dateien werden weiter angenommen. Die Claude-Skill 2.3 liest Form und Stückzahl aus – sie braucht diese Version des Rezeptplaners.

## 0.13.0

- **Gang:** Neue Kategoriegruppe „Gang“ (Vorspeise, Suppe/Eintopf, Salat, Hauptgericht, Beilage, Nachspeise, Kuchen & Torten, Gebäck, Brot, Snack). „Suppe/Eintopf“ und „Salat“ wandern dafür aus „Art“; ihre Zuordnungen bleiben. In der Kategorienverwaltung lässt sich festlegen, welche Gruppe die Gänge enthält.
- **Reiter in Rezeptübersicht und Rezeptauswahl:** Alle | Hauptgerichte | Vorspeisen, Suppen & Salate | Nachspeisen | Backen (auch in der App ab 0.14.0).
- **Gänge zuordnen** (Einstellungen → Kategorien → „Gänge zuordnen“): Vorschläge für alle Rezepte ohne Gang, prüfen und gesammelt übernehmen.
- Die Claude-Skill (Version 2.2) vergibt beim Import immer einen Gang.

## 0.12.1

- **Suche auch in der Zubereitung** (Weboberfläche): Die Rezeptsuche findet Wörter jetzt auch in den Zubereitungsschritten, z. B. „Ofen“ oder „über Nacht“.

## 0.12.0

- **Rezept per QR-Code:** Beim neuen Rezept gibt es neben dem Adressfeld den Knopf „QR-Code“, wenn der Browser QR-Codes mit der Kamera lesen kann (z. B. Chrome auf Android); die gescannte Adresse wird sofort eingelesen. In der Android-App (ab 0.13.0): Rezepte → Plus → „Rezept aus QR-Code“ und ein Scan-Knopf bei „Rezept aus Internetadresse“.

## 0.11.0

- **Kurzvorschau beim Einplanen** (Weboberfläche und App ab 0.12.0): Beim Hinzufügen eines Rezepts zum Essensplan erscheinen Portionen und Zutaten. Häkchen entfernen, was noch im Haus ist (z. B. im Tiefkühler) – diese Zutaten kommen nicht auf die Einkaufsliste und nicht an Bring. „Alles vorhanden“ markiert gleich das ganze Gericht. Die Abwahl gilt nur für diesen Eintrag, wandert beim Verschieben und Tauschen mit und verfällt beim Ersetzen.
- Nachträglich über „Zutaten vorhanden …“ im Menü des Eintrags; ist schon an Bring gesendet, erscheint die Änderung im Einkauf als Differenz.
- Im Essensplan zeigt der Eintrag z. B. „2 Zutaten vorhanden“.
- Neue Einstellung **Kurzvorschau beim Einplanen** (Einstellungen → Haushalt, Standard an).
- Beim ersten Start gelten alle Rezepte einmal als geändert, damit die App sie mit den neuen Angaben abgleicht.

## 0.10.0

- **Statistik** (neuer Bereich in der Navigation, auch in der App ab 0.11.0): Anzahl der Rezepte, wie viele schon bzw. noch nie eingeplant waren (mit Link auf die Rezeptliste), Rezepte mit Illustration, die 20 am häufigsten gegessenen Rezepte der letzten 3 Monate, des letzten Jahres oder insgesamt, Einträge ohne Rezept („2× Essen gehen“), Lieblingsgerichte, die lange nicht dran waren, zuletzt hinzugefügte Rezepte und die Verteilung auf die Kategorien.
- **Gezählt wird bis heute:** Geplante Gerichte zählen erst, wenn ihr Tag da ist – in der Statistik und bei der Sortierung „Am häufigsten eingeplant“.
- Neuer Filter **Nie eingeplant** in der Rezeptübersicht.

## 0.9.0

- **Kategorien verwalten** (Einstellungen → Kategorien): Gruppen und Kategorien anlegen, umbenennen, per Ziehen sortieren – Kategorien auch in eine andere Gruppe –, zusammenführen (z. B. „Hähnchen“ in „Geflügel“) und löschen. Beim Löschen zeigt der Rezeptplaner, wie vielen Rezepten die Kategorie zugeordnet ist; die Rezepte bleiben, nur die Zuordnung fällt weg. Eine Gruppe mit Kategorien lässt sich löschen, indem man die Kategorien verschiebt oder mitlöscht. Namen gibt es nur einmal (Groß-/Kleinschreibung und Umlaute zählen nicht). Änderungen kommen mit dem nächsten Abgleich in die App.
- **Kategorieliste für Claude kopieren:** Text für den Chat mit der Claude-Skill, damit sie die eigenen Kategorien kennt.
- **Unbekannte Kategorien beim Datei-Import** (Weboberfläche und App ab 0.10.0): einer vorhandenen Kategorie zuordnen, neu anlegen oder weglassen.
- Die Datenbank bekommt beim ersten Start eine eigene Tabelle für die Gruppen; vorhandene Kategorien und Zuordnungen bleiben erhalten.

## 0.8.0

- **Foto aus dem Rezeptlink beim Datei-Import:** Hat ein importiertes Rezept eine Webadresse, aber kein Foto (oder nur eine Illustration), holt der Rezeptplaner das Rezeptbild von dieser Seite. Die Vorschau zeigt es mit „Foto von … wird übernommen“; ein Foto aus der Datei wird nie ersetzt, und von der Seite kommt nur das Bild. Die Claude-Skill (Version 2.1) sucht dafür das Originalrezept per Websuche und trägt den Link nur bei eindeutigem Treffer ein.
- Die Skill kann eine im Chat eingefügte, aktuelle Kategorieliste statt ihrer eingebauten Liste verwenden.

## 0.7.2

- **Glatte Mengen auf der Einkaufsliste:** Umgerechnete Mengen in Gramm, Milliliter, Kilogramm, Liter und Löffeln werden für Einkaufsliste und Bring auf glatte Werte aufgerundet – z. B. 540 g statt 533 g, und beim Nachsenden 260 g statt 267 g. Die Kochansicht zeigt weiter die genaue Menge.

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
