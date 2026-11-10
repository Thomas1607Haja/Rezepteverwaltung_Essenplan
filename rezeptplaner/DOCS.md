# Rezeptplaner

*Deutsch: weiter unten.*

## Installation

1. In Home Assistant open **Settings → Apps → App store**.
2. In the menu (⋮) at the top right choose **Repositories** and add this address:
   `https://github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan`
3. Select **Rezeptplaner** in the app store and **Install** it.
4. **Start** it and turn on **Show in sidebar**.

The web interface then appears as **Rezeptplaner** in the sidebar.

## Settings

The app itself has no options in Home Assistant. Household size (default servings in the meal plan), shopping day (start of the one-week shopping period) and language are set in the web interface under **Settings**. Without a fixed language, the Home Assistant system language is used.

## Languages

English, German, French, Spanish and Italian. Your recipes, ingredients and categories are never translated. A new installation creates its default categories and pantry list in the Home Assistant language.

## Importing recipes from files

**Recipes → Import file** reads a ZIP or JSON file in the “rezeptplaner-import” format (e.g. created by the Claude skill `skills/rezeptplaner-import` from photos of recipe cards). First a preview shows duplicates, unknown categories and uncertain spots; nothing is saved until you choose **Import**. Format: `docs/importformat.md` in the repository.

## Sending the shopping list to Bring!

1. In Home Assistant set up the **Bring!** integration (**Settings → Devices & services → Add integration**). Every Bring! list then appears as a to-do list.
2. In the web interface select the list under **Settings → Household → Bring! list** and save.

The app writes the items into this list via the Home Assistant API; the quantity appears in Bring! as the description below the item.

## Dashboard card, calendar and sensor

There is an optional integration for Home Assistant:

Integration and card have their own repository: **https://github.com/Thomas1607Haja/rezeptplaner-ha**

1. [Open it in the Marketplace](https://my.home-assistant.io/redirect/marketplace_repository/?owner=Thomas1607Haja&repository=rezeptplaner-ha&category=integration) – or under **Settings → Marketplace → Add from link** enter `Thomas1607Haja/rezeptplaner-ha`. (Before Home Assistant 2026.11 the link opens HACS; there: **HACS** → ⋮ → **Custom repositories**, type **Integration**.)
2. Download **Rezeptplaner** and restart Home Assistant.
3. Under **Settings → Devices & services**, “Rezeptplaner” appears as **Discovered** – tap **Add** and confirm, or [start the setup](https://my.home-assistant.io/redirect/config_flow_start/?domain=rezeptplaner). The app provides address and key itself.
4. Reload the page once (F5) so the browser knows the card. The integration registers the card as a dashboard resource itself. If you access Home Assistant through a Cloudflare tunnel, set up the cache rule from the README of rezeptplaner-ha first.
5. Add a **Meal plan** card to a dashboard.

Installed the integration from this repository before (up to version 0.6)? HACS showed the Android app's version there and no updates. Switch as described under “Moving from the old repository” in the README of rezeptplaner-ha – your setup is kept.

“Open” on the card leads to the app's web interface – even if the app is not in the sidebar.

You then have a calendar “Meal plan” and a sensor “Today” (e.g. for an announcement “Today we're having …”). Their entity IDs follow the language at setup, e.g. `calendar.rezeptplaner_meal_plan` in English or `calendar.rezeptplaner_essensplan` in German.

## Access and security

- The web interface is only reachable through Home Assistant (Ingress). Home Assistant handles the login.
- The Android app has a separate **app access** on port 8100. Every request there needs an API key (created under **Settings**, only a hash is stored). The port is not exposed; a Cloudflare tunnel reaches it via the app's host name. Guide: `docs/APP-ZUGANG.md` in the repository.
- The app may use the Home Assistant API (`homeassistant_api`). It only reads the existing to-do lists and the system language, and writes to the selected list.
- For the integration the app creates its own key and passes it to Home Assistant via discovery. This key can only read the meal plan, recipes and images inside the local network.
- When importing a recipe from the internet, the app loads the page and the image itself. Addresses in the home network are blocked.

## Data and backups

Database and images are stored in the app's data folder and are part of Home Assistant backups. During a backup the app is stopped for a few seconds so the database is saved in a clean state.

## Known limitations

- Only for ARM64 devices (e.g. Home Assistant Green).

## Help and bug reports

Please report questions and bugs as an issue in the repository:
https://github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/issues

## License

See [LICENSE](https://github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/blob/main/LICENSE) in the repository.

---

# Deutsch

## Installation

1. In Home Assistant **Einstellungen → Apps → App-Store** öffnen.
2. Oben rechts im Menü (⋮) **Repositories** wählen und diese Adresse hinzufügen:
   `https://github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan`
3. Im App-Store **Rezeptplaner** auswählen und **Installieren**.
4. **Starten** und **In Seitenleiste anzeigen** einschalten.

Die Weboberfläche erscheint dann als **Rezeptplaner** in der Seitenleiste.

## Einstellungen

Die App selbst hat keine Optionen in Home Assistant. Familiengröße (Standard-Portionen im Essensplan), Einkaufstag (Beginn des einwöchigen Einkaufszeitraums) und Sprache stellst du in der Weboberfläche unter **Einstellungen** ein. Ohne feste Sprache gilt die Systemsprache von Home Assistant.

## Rezepte aus Dateien importieren

**Rezepte → Datei importieren** liest eine ZIP- oder JSON-Datei im Format „rezeptplaner-import“ (z. B. erzeugt von der Claude-Skill `skills/rezeptplaner-import` aus Fotos von Rezeptkarten). Zuerst erscheint eine Vorschau mit Hinweisen auf Dubletten, unbekannte Kategorien und unsichere Stellen; gespeichert wird erst nach **Importieren**. Format: `docs/importformat.md` im Repository.

## Einkaufsliste an Bring senden

1. In Home Assistant die Integration **Bring!** einrichten (**Einstellungen → Geräte & Dienste → Integration hinzufügen**). Jede Bring-Liste erscheint dann als To-do-Liste.
2. In der Weboberfläche unter **Einstellungen → Haushalt → Bring-Liste** die gewünschte Liste auswählen und speichern.

Die App schreibt die Artikel über die Home-Assistant-API in diese Liste; die Mengenangabe steht in Bring als Beschreibung unter dem Artikel.

## Dashboard-Karte, Kalender und Sensor

Optional gibt es eine Integration für Home Assistant:

Integration und Karte haben ein eigenes Repository: **https://github.com/Thomas1607Haja/rezeptplaner-ha**

1. [Im Marketplace öffnen](https://my.home-assistant.io/redirect/marketplace_repository/?owner=Thomas1607Haja&repository=rezeptplaner-ha&category=integration) – oder unter **Einstellungen → Marketplace → Add from link** `Thomas1607Haja/rezeptplaner-ha` eintragen. (Vor Home Assistant 2026.11 öffnet der Link HACS; dort **HACS** → ⋮ → **Benutzerdefinierte Repositories**, Typ **Integration**.)
2. **Rezeptplaner** herunterladen und Home Assistant neu starten.
3. Unter **Einstellungen → Geräte & Dienste** erscheint „Rezeptplaner“ als **Entdeckt** – auf **Hinzufügen** tippen und bestätigen oder [die Einrichtung starten](https://my.home-assistant.io/redirect/config_flow_start/?domain=rezeptplaner). Adresse und Schlüssel liefert die App selbst.
4. Die Seite einmal neu laden (F5), damit der Browser die Karte kennt. Die Integration trägt die Karte dafür selbst als Dashboard-Ressource ein. Bei Zugriff über einen Cloudflare-Tunnel vorher die Cache-Regel aus dem README von rezeptplaner-ha einrichten.
5. Im Dashboard eine Karte **Essensplan** hinzufügen.

Die Integration früher aus diesem Repository installiert (bis Version 0.6)? HACS zeigte dort die Version der Android-App und keine Updates. Umstieg wie im README von rezeptplaner-ha unter „Umstieg vom bisherigen Repository“ beschrieben – die Einrichtung bleibt erhalten.

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

## Hilfe und Fehler melden

Fragen und Fehler bitte als Issue im Repository melden:
https://github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/issues

## Lizenz

Siehe [LICENSE](https://github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/blob/main/LICENSE) im Repository.
