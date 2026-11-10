# Rezeptplaner

**English** · [Deutsch](README.de.md)

Recipes, meal plan and shopping list for Home Assistant. Rezeptplaner runs as an app (add-on) on your Home Assistant, keeps your recipes in one place, plans the week's meals, and sends the shopping list to Bring!. It's built to be lean enough that a Home Assistant Green barely notices it. An Android app, a dashboard card and a Claude skill for photographed recipes are optional extras.

Rezeptplaner is a hobby project and free to use.

<!--
  Screenshots: uncomment this block once the files exist in docs/screenshots/
  (list of shots: docs/SCREENSHOTS.md).

<p>
  <img src="docs/screenshots/web-meal-plan.png" alt="Meal plan in the web interface" width="49%">
  <img src="docs/screenshots/web-recipe.png" alt="Recipe with serving calculator" width="49%">
</p>
<p>
  <img src="docs/screenshots/web-shopping.png" alt="Shopping list before sending it to Bring!" width="49%">
  <img src="docs/screenshots/web-import-preview.png" alt="Import preview of a file from the Claude skill" width="49%">
</p>
<p>
  <img src="docs/screenshots/android-meal-plan.png" alt="Android app: meal plan" width="24%">
  <img src="docs/screenshots/android-recipe.png" alt="Android app: cooking view" width="24%">
  <img src="docs/screenshots/android-shopping.png" alt="Android app: shopping list" width="24%">
  <img src="docs/screenshots/android-photo-import.png" alt="Android app: photo import" width="24%">
</p>
-->

**Videos:** an English and a German walkthrough are coming soon.
<!-- TODO videos: replace the line above with
**Videos:** [Walkthrough in English](https://…) · [Erklärvideo auf Deutsch](https://…)
-->

## Features

- **Recipes:** import from a web page (schema.org recipe data, e.g. allrecipes.com, bbcgoodfood.com, marmiton.org, chefkoch.de), from a photo (Android app) or from a file; create, edit and vary them freely; serving calculator and cooking view
- **Meal plan:** freely scrolling days, plan as far ahead as you like, lunch, dinner and an optional afternoon slot, move and swap dishes by drag and drop, multi-course menus
- **Shopping list** for the current shopping week, sent to **Bring!** via Home Assistant; your pantry staples are left off automatically
- **Five languages:** English, German, French, Spanish and Italian
- **Home Assistant integration (optional):** a meal plan calendar, a “Today” sensor and a dashboard card – separate repository [rezeptplaner-ha](https://github.com/Thomas1607Haja/rezeptplaner-ha)
- **Android app (optional):** read recipes and the plan offline, plan, create and edit recipes, photo import with on-device text recognition, share recipes as links or PDF, send the shopping list to Bring!
- **Claude skill (optional):** turns photos of recipe cards, cookbook pages or handwritten notes into an import file; if there's no photo of the dish, it draws an illustration

## Requirements

| What | Needed for | Notes |
|---|---|---|
| **Home Assistant OS** or **Home Assistant Supervised** | everything | Apps (add-ons) need the Supervisor. Home Assistant Container and Core can't run apps. |
| **ARM64 hardware** | the app | e.g. Home Assistant Green, Raspberry Pi 4 or 5. x86-64 (mini PCs, most virtual machines) isn't supported yet. |
| Home Assistant **2026.11** or newer | recommended | Older versions work; the integration is then installed through HACS instead of the Marketplace. |
| **Android 10** or newer | Android app (optional) | Installed and updated with [Obtainium](https://github.com/ImranR98/Obtainium) from the GitHub releases. |
| **Bring!** account | shopping list (optional) | Uses Home Assistant's own Bring! integration. |
| **Claude** account with skills | Claude skill (optional) | To capture recipes from photos. |

## Installation

### 1. Install the app

[![Open your Home Assistant instance and show the Rezeptplaner app (adds the repository if needed).](https://my.home-assistant.io/badges/supervisor_app.svg)](https://my.home-assistant.io/redirect/supervisor_app/?app=45a92d34_rezeptplaner&repository_url=https%3A%2F%2Fgithub.com%2FThomas1607Haja%2FRezepteverwaltung_Essenplan)

1. Click the button. The first time, my.home-assistant.io asks for your Home Assistant address. Confirm adding the repository.
2. **Install** “Rezeptplaner”, then **Start** it and turn on **Show in sidebar**.

If the app doesn't show up, [add the repository](https://my.home-assistant.io/redirect/supervisor_add_addon_repository/?repository_url=https%3A%2F%2Fgithub.com%2FThomas1607Haja%2FRezepteverwaltung_Essenplan) first – or go to **Settings → Apps → App store → ⋮ → Repositories** and add `https://github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan`.

### 2. First steps

Open **Rezeptplaner** in the sidebar. Under **Settings → Household** set the household size, the shopping day and, if you like, a fixed language (by default it follows Home Assistant). Then add your first recipe under **Recipes** – for example by pasting the address of a recipe page.

### 3. Send the shopping list to Bring! (optional)

1. Set up the Bring! integration in Home Assistant:

   [![Open your Home Assistant instance and start setting up the Bring! integration.](https://my.home-assistant.io/badges/config_flow_start.svg)](https://my.home-assistant.io/redirect/config_flow_start/?domain=bring)

2. In Rezeptplaner choose the list under **Settings → Household → Bring list** and save.
3. Plan some dishes, open **Shopping** and choose **Send to Bring**.

### 4. Calendar, sensor and dashboard card (optional)

1. Open the integration in the Marketplace and download **Rezeptplaner**:

   [![Open your Home Assistant instance and open this repository in the Marketplace.](https://my.home-assistant.io/badges/marketplace_repository.svg)](https://my.home-assistant.io/redirect/marketplace_repository/?owner=Thomas1607Haja&repository=rezeptplaner-ha&category=integration)

   Before Home Assistant 2026.11 the button opens HACS instead.
2. Restart Home Assistant.
3. “Rezeptplaner” now appears under **Settings → Devices & services** as **Discovered** – confirm it, or start the setup here:

   [![Open your Home Assistant instance and start setting up the Rezeptplaner integration.](https://my.home-assistant.io/badges/config_flow_start.svg)](https://my.home-assistant.io/redirect/config_flow_start/?domain=rezeptplaner)

4. Reload the browser page and add the **Meal plan** card to a dashboard.

Card options and details: [rezeptplaner-ha](https://github.com/Thomas1607Haja/rezeptplaner-ha). The same links are in the web interface under **Settings → Home Assistant**.

### 5. Android app (optional)

In the web interface open **Settings → Install the app**:

- **Step 1** shows QR codes to install [Obtainium](https://github.com/ImranR98/Obtainium) and to add Rezeptplaner to it. Obtainium installs the app from the GitHub releases and keeps it up to date. Android asks once to allow installing apps from Obtainium.
- **Step 2** connects the phone: enter the address under which the phone reaches the app access, create a key for the device and scan the QR code in the app.

The app talks to its own **app access** (port 8100), not to Home Assistant. To use it away from home, read [Security](#security) first.

### 6. Claude skill (optional)

Download the skill under **Settings → Install the app → Claude skill** and upload it in Claude under **Settings → Capabilities → Skills**. Then drop photos of recipes into a chat. Import the resulting ZIP file by sharing it to the Android app or in the web interface under **Recipes → Import file**. Nothing is saved before you've seen a preview with duplicates, unknown categories and hints. If the file came from an older version of the skill, the preview offers the new one.

- The skill captures recipes **in the language of the original** and never translates or converts units.
- Categories must match your Rezeptplaner. If it doesn't run in German, copy your list first (**Settings → Categories → Copy category list for Claude**) and paste it into the chat.
- The skill's own instructions are in German; that doesn't affect the recipes. Format: [docs/importformat.md](docs/importformat.md).

## Security

- **Web interface:** only reachable through Home Assistant (Ingress). Home Assistant handles the login; nothing else is exposed.
- **App access (port 8100):** used only by the Android app. Every request needs an **API key**. Keys are created per device in the web interface, shown once as a QR code, and only a hash is stored. A lost phone can be locked out at any time with **Revoke** under **Settings → Install the app**.
- **Never forward port 8100 (or any Home Assistant port) directly to the internet.** To use the app away from home, use a **tunnel** (e.g. the Cloudflared app) or a **VPN** (e.g. WireGuard or Tailscale) into your home network.
- **Optional extra lock with Cloudflare Access:** with a tunnel, Cloudflare can require a service token before a request even reaches your home. A stolen API key alone is then useless. The Android app can send the token.
- Step-by-step guide for tunnel, cache rule and Cloudflare Access: [docs/APP-ACCESS.md](docs/APP-ACCESS.md).
- The integration gets its own read-only key through Home Assistant's discovery; it only works inside the local network.
- When importing from a web page, Rezeptplaner loads the page and image itself; addresses in your home network are blocked.

## Languages

The web interface, the Android app, the integration and the card are available in **English, German, French, Spanish and Italian**. English is the fallback whenever a language or a single text is missing.

- **Web interface:** follows the Home Assistant system language (Settings → System → General), then the browser language. **Settings → Household → Language** sets a fixed language for the whole household.
- **Android app:** follows the phone's language. From Android 13 you can pick a language just for this app (Android settings → Apps → Rezeptplaner → Language).
- **Integration and card:** the integration uses the Home Assistant system language, the card the language of the logged-in user.
- **Your content is never translated:** recipes, ingredients, notes and your own categories stay as you entered them. A new installation creates its default categories and pantry list in the Home Assistant language.

French, Spanish and Italian haven't been reviewed by native speakers yet – some phrases may be clumsy. **Corrections and new languages are very welcome:** see [CONTRIBUTING.md](CONTRIBUTING.md).

## Help, ideas and contributing

- **Bugs and feature requests:** [issues](https://github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/issues) – for the integration or the card please use [rezeptplaner-ha](https://github.com/Thomas1607Haja/rezeptplaner-ha/issues).
- **Questions and ideas:** [discussions](https://github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/discussions)
- **Translations and code:** [CONTRIBUTING.md](CONTRIBUTING.md)
- **What's new:** [CHANGELOG.md](CHANGELOG.md)

## Under the hood

| Part | Technology |
|---|---|
| Server | Go with SQLite, runs as a Home Assistant app |
| Web interface | Svelte, embedded in the server, opened from the Home Assistant sidebar |
| Integration and dashboard card | Python and JavaScript, own repository [rezeptplaner-ha](https://github.com/Thomas1607Haja/rezeptplaner-ha) |
| Android app | Kotlin, Jetpack Compose |
| Claude skill | Python scripts and instructions in `skills/rezeptplaner-import` |

```
rezeptplaner/   Home Assistant app (config, changelog, docs shown in Home Assistant)
server/         Go backend
web/            web interface
android/        Android app
skills/         Claude skill "rezeptplaner-import" (with tests)
docs/           planning (German), import format, translations, app access
```

The full planning document – architecture, domain rules, data model, screens and project phases – is [docs/PLANUNG.md](docs/PLANUNG.md) (German).

## Support

Rezeptplaner is free. If you like it, I'd be happy about a coffee.

<a href="https://buymeacoffee.com/rezeptplaner"><img src="https://cdn.buymeacoffee.com/buttons/v2/default-yellow.png" alt="Buy Me a Coffee" height="40"></a>

## License

[MIT](LICENSE)
