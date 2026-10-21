# Änderungen

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
