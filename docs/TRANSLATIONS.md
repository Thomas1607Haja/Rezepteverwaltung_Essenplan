# Translations / Übersetzungen

Rezeptplaner is available in English, German, French, Spanish and Italian.
English is the fallback language. French, Spanish and Italian have not been
reviewed by native speakers yet – corrections are very welcome.

*Deutsch: unten.*

## Where the texts live

| Part | Files |
|---|---|
| Web interface | `web/src/locales/<area>/<lang>.json` – one folder per area (`plan`, `recipes`, `settings` …), keys are used as `t('<area>.<key>')` |
| Error messages of the server | `web/src/locales/errors/<lang>.json` – one text per error code (`server/internal/apierr/catalog.go` lists the codes) |
| Android app | `android/app/src/main/res/values-<lang>/strings*.xml` (`values/` is English) |
| Android error and shared texts | generated: `strings_errors.xml` and `strings_shared.xml` come from the web files – edit the web JSON, then run `python android/tools/gen_errors.py` |
| Home Assistant integration | repository [rezeptplaner-ha](https://github.com/Thomas1607Haja/rezeptplaner-ha): `custom_components/rezeptplaner/translations/<lang>.json` (setup, entities) and `custom_components/rezeptplaner/texts/<lang>.json` (calendar titles) |
| Dashboard card | rezeptplaner-ha: `custom_components/rezeptplaner/frontend/rezeptplaner-card.js`, block between `// i18n:begin` and `// i18n:end` (plain JSON) |
| Add-on options | `rezeptplaner/translations/<lang>.yaml` |
| Default categories and pantry of a new installation | `server/internal/seed/seed.go` |
| Terms | `docs/glossar.md` – please use these terms everywhere |

Placeholders look like `{name}` in all files and must stay unchanged. Plural
texts are objects with the forms `one` and `other` (JSON) or `<plurals>` (Android).

## Correcting a translation

1. Fork the repository and change the text in the files above (for the web, the
   app and the integration if the same phrase appears there).
2. Keep it short and natural – like a well-made app, not word for word. Buttons
   and tabs must fit on a phone (390 px wide).
3. Run the key test: `cd server && go test ./internal/locales`
   (it checks that all languages have the same keys and placeholders).
4. Open a pull request and describe what you changed.

## Adding a new language

1. Copy every English file listed above to the new language code (e.g. `nl.json`,
   `values-nl/`, `nl.yaml`) and translate it.
2. Add the code to the language lists: `LANGUAGES` in `web/src/lib/i18n.svelte.js`,
   `Supported` in `server/internal/language/language.go`, `uiLanguages` in
   `server/internal/settings/settings.go`, `languages` in
   `server/internal/locales/locales_test.go`, `LANGS` in `android/tools/gen_errors.py`,
   `LANGUAGES` in `custom_components/rezeptplaner/texts.py` and `tests/test_translations.py`
   (both in rezeptplaner-ha), the language names in the
   settings page, and the default data in `server/internal/seed/seed.go`.
3. Extend `docs/glossar.md` with a column for the new language.
4. Run the tests and open a pull request.

---

## Deutsch

**Eine Übersetzung korrigieren:** Repository forken, den Text in den oben
genannten Dateien ändern (bei gleichen Formulierungen in Web, App und
Integration), mit `cd server && go test ./internal/locales` prüfen und einen
Pull Request stellen. Platzhalter wie `{name}` nicht verändern; die Begriffe
aus `docs/glossar.md` verwenden. Natürlich und kurz formulieren, nicht
wörtlich – Knöpfe und Reiter müssen auf ein Handy passen.

**Eine neue Sprache beisteuern:** alle englischen Dateien aus der Tabelle unter
dem neuen Sprachcode kopieren und übersetzen, den Code in die oben genannten
Sprachlisten eintragen, das Glossar um eine Spalte erweitern, Tests laufen
lassen und einen Pull Request stellen.

Die Fehlertexte und gemeinsamen Texte der Android-App entstehen aus denen
des Webs: nach einer Änderung `python android/tools/gen_errors.py` ausführen.
