# Rezeptplaner

**English** · [Deutsch](README.de.md)

A lean, self-built system for managing recipes and planning the week's meals – running as an app (add-on) in Home Assistant.

## Features

- **Recipes:** import from a web page (schema.org recipe data, e.g. allrecipes.com, marmiton.org, chefkoch.de), from a photo (Android app) or from a file; create and edit them freely
- **Meal plan** with freely scrolling days, plan as far ahead as you like, move and swap dishes by drag and drop, multi-course menus
- **Shopping list** for the current shopping period, sent to the Bring! app via Home Assistant
- **Dashboard card** with the current meal plan in Home Assistant
- **Calendar and sensor** in Home Assistant (integration via HACS)
- **Android app:** read recipes and the plan offline, plan, create and edit recipes, share recipe links and PDFs, photo import with on-device text recognition, send the shopping list to Bring!
- **Claude skill** “rezeptplaner-import”: captures recipes from photos (recipe cards, cookbook pages, handwritten notes) and produces an import file; if there is no photo of the dish, it draws an illustration

## Languages

The web interface, the Android app, the Home Assistant integration and the dashboard card are available in **English, German, French, Spanish and Italian**. English is the fallback whenever a language or a single text is missing.

- **Web interface:** follows the system language of Home Assistant (Settings → System → General), then the browser language. Under **Settings → Household → Language** you can set a fixed language for the whole household.
- **Android app:** follows the Android system language. From Android 13 you can choose a language just for this app (Android settings → Apps → Rezeptplaner → Language).
- **Integration and card:** the integration uses the Home Assistant system language, the card the language of the logged-in user.
- **Your content is never translated:** recipes, ingredients, notes and your own categories stay exactly as you entered them. A new installation creates its default categories and pantry list in the Home Assistant language; existing installations are not touched.

French, Spanish and Italian were translated without a native speaker review – some phrases may still be a bit clumsy. **Improvements are very welcome!** See [docs/TRANSLATIONS.md](docs/TRANSLATIONS.md) for how to correct a translation or contribute a new language via pull request. The terms used everywhere are listed in the [glossary](docs/glossar.md).

## Installation

1. **App:** In Home Assistant go to Settings → Apps → App store → ⋮ → Repositories, add this address and install “Rezeptplaner”:
   `https://github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan`
2. **Integration and dashboard card (optional):** In HACS → ⋮ → Custom repositories add the same address with type “Integration”, download “Rezeptplaner” and restart Home Assistant. The integration then appears under Settings → Devices & services as “Discovered” – just confirm it.
3. After setting it up, reload the page once (F5), then add a “Meal plan” card to a dashboard (from the card picker or as `type: custom:rezeptplaner-card`). The integration registers the card as a dashboard resource itself (Settings → Dashboards → ⋮ → Resources) and keeps the entry up to date. If your resources are maintained in YAML (`lovelace: resource_mode: yaml`), add an entry with `url: /rezeptplaner_static/rezeptplaner-card.js` and `type: module`; until then the integration loads the card via the start page as a fallback.

   Card options (visual editor or YAML):

   | Field | Meaning | Default |
   |---|---|---|
   | `title` | Heading | “Meal plan” in the user's language |
   | `days` | Number of days from today (1–31) | current shopping period |
   | `height` | Height in pixels; if not everything fits, the card scrolls | as tall as the content |

### Access through a Cloudflare tunnel

Without a rule of its own, Cloudflare also caches error responses and tells browsers to keep them for 4 hours. If a browser or the companion app requests the card while Home Assistant is restarting, it stays missing for hours afterwards (“configuration error”). Fix it in the Cloudflare dashboard of your domain:

1. **Caching → Cache Rules → Create rule:** name “Do not cache Home Assistant”, condition *Hostname equals* `<HA host name>`, action **Bypass cache**, save.
2. **Caching → Configuration → Browser Cache TTL:** **Respect Existing Headers**.
3. Once: **Caching → Configuration → Purge Everything**; in the companion app reset the frontend cache under Settings → Companion app → Troubleshooting.

### Android app

In the web interface under **Settings → Install the app**: step 1 shows QR codes for Obtainium (installs and updates the app from the GitHub releases), step 2 connects the phone via QR code. To reach the app from outside your home, the app access (port 8100) needs its own host name, e.g. in your existing Cloudflare tunnel – see [docs/APP-ZUGANG.md](docs/APP-ZUGANG.md) (German). More about the app: [docs/ANDROID.md](docs/ANDROID.md) (German).

### Claude skill

The current skill can be downloaded under **Settings → Install the app → Claude skill**. In Claude, upload it under Settings → Capabilities → Skills, then drop photos of recipes into a chat. Import the resulting ZIP file by sharing it to the app or in the web interface under **Recipes → Import file**. Format: [docs/importformat.md](docs/importformat.md).

In short (the skill's own instructions are in German):

- It captures recipes **in the language of the original** – a French recipe stays French; nothing is translated.
- Units are written the way Rezeptplaner's parser writes them (`tbsp`, `tsp`, `oz`, `c. à s.`, `cda`, `cucchiaio`, `EL` …) and are never converted.
- Categories must match your Rezeptplaner exactly. If Rezeptplaner runs in another language than German, copy your category list (**Settings → Categories → Copy category list for Claude**) and paste it into the chat first.
- Before saving anything, Rezeptplaner shows a preview with duplicates, unknown categories and hints.

## Building blocks

| Part | Technology |
|---|---|
| Server | Go with SQLite, runs as a Home Assistant add-on |
| Web interface | embedded in the server, reachable from the HA sidebar |
| HA integration and dashboard card | Python and JavaScript, installable via HACS |
| Android app | Kotlin, Jetpack Compose |

Principle: the system has to be so lean that it barely affects a Home Assistant Green next to its actual job.

## Project status

Done: recipes, meal plan, shopping list with Bring!, integration with dashboard card, Android app, file import with Claude skill, five languages. The complete plan – architecture, domain rules, data model, screens and project phases – is in [docs/PLANUNG.md](docs/PLANUNG.md) (German).

## Project structure

```
rezeptplaner/                    Home Assistant add-on
custom_components/rezeptplaner/  HA integration (HACS) with dashboard card (frontend/)
server/                          Go backend
web/                             Web interface
android/                         Android app
skills/                          Claude skill "rezeptplaner-import" (with tests)
docs/                            Planning and notes
```

## License

[MIT](LICENSE)

## Support

Rezeptplaner is free. If you like it, I'd be happy about a coffee.

<a href="https://buymeacoffee.com/rezeptplaner"><img src="https://cdn.buymeacoffee.com/buttons/v2/default-yellow.png" alt="Buy Me a Coffee" height="40"></a>
