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

## Einkaufsliste an Bring senden

1. In Home Assistant die Integration **Bring!** einrichten (**Einstellungen → Geräte & Dienste → Integration hinzufügen**). Jede Bring-Liste erscheint dann als To-do-Liste.
2. In der Weboberfläche unter **Einstellungen → Haushalt → Bring-Liste** die gewünschte Liste auswählen und speichern.

Die App schreibt die Artikel über die Home-Assistant-API in diese Liste; die Mengenangabe steht in Bring als Beschreibung unter dem Artikel.

## Dashboard-Karte, Kalender und Sensor

Optional gibt es eine Integration für Home Assistant:

1. In **HACS** → ⋮ → **Benutzerdefinierte Repositories** die Adresse dieses Repositorys mit Typ **Integration** hinzufügen.
2. **Rezeptplaner** herunterladen und Home Assistant neu starten.
3. Unter **Einstellungen → Geräte & Dienste** erscheint „Rezeptplaner“ als **Entdeckt** – auf **Hinzufügen** tippen und bestätigen. Adresse und Schlüssel liefert die App selbst.
4. Die Seite einmal neu laden (F5), damit der Browser die Karte kennt. Die Integration trägt die Karte dafür selbst als Dashboard-Ressource ein. Bei Zugriff über einen Cloudflare-Tunnel vorher die Cache-Regel aus dem README des Repositorys einrichten.
5. Im Dashboard eine Karte **Essensplan** hinzufügen.

„Öffnen“ auf der Karte führt zur Weboberfläche der App – auch wenn die App nicht in der Seitenleiste steht.

Danach gibt es den Kalender `calendar.rezeptplaner_essensplan` und den Sensor `sensor.rezeptplaner_heute` (z. B. für eine Ansage „Heute gibt es …“).

## Zugriff und Sicherheit

- Die Weboberfläche ist nur über Home Assistant erreichbar (Ingress). Die Anmeldung übernimmt Home Assistant.
- Für die Android-App gibt es einen getrennten **App-Zugang** auf Port 8100. Dort braucht jede Anfrage einen API-Schlüssel (anlegen unter **Einstellungen → App-Zugang**, gespeichert wird nur ein Hash). Der Port ist nach außen nicht freigegeben; ein Cloudflare-Tunnel erreicht ihn über den Hostnamen der App. Anleitung: `docs/APP-ZUGANG.md` im Repository.
- Die App darf die Home-Assistant-API nutzen (`homeassistant_api`). Sie liest damit nur die vorhandenen To-do-Listen und schreibt in die gewählte Liste.
- Für die Integration erzeugt die App einen eigenen Schlüssel und teilt ihn Home Assistant über die Discovery mit. Mit diesem Schlüssel lassen sich im internen Netz nur Essensplan, Rezepte und Bilder lesen.
- Beim Übernehmen eines Rezepts aus dem Internet lädt die App die Seite und das Bild selbst. Adressen im Heimnetz sind dabei gesperrt.

## Daten und Backups

Datenbank und Bilder liegen im Datenordner der App und sind Teil der Home-Assistant-Backups. Während eines Backups wird die App für einige Sekunden gestoppt, damit die Datenbank in einem sauberen Zustand gesichert wird.

## Bekannte Einschränkungen

- Nur für ARM64-Geräte (z. B. Home Assistant Green).
- Die Android-App ist noch in Arbeit.

## Hilfe und Fehler melden

Fragen und Fehler bitte als Issue im Repository melden:
https://github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/issues

## Lizenz

Siehe [LICENSE](https://github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/blob/main/LICENSE) im Repository.
