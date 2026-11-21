# Contributing

**English** · [Deutsch](#deutsch)

Thanks for helping with Rezeptplaner! It's a hobby project, so answers may take a few days. English and German are both fine everywhere – in issues, discussions and pull requests.

## Ways to help

| You want to … | Go to |
|---|---|
| report a bug | [new issue → Bug report](https://github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/issues/new/choose) |
| suggest a feature | [new issue → Feature request](https://github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/issues/new/choose) – or first discuss it under [Ideas](https://github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/discussions/categories/ideas) |
| ask a question | [Q&A in discussions](https://github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/discussions/categories/q-a) |
| improve a translation or add a language | this page, section [Translations](#translations) |
| fix something in the integration or the dashboard card | repository [rezeptplaner-ha](https://github.com/Thomas1607Haja/rezeptplaner-ha) |

## Translations

Rezeptplaner speaks English, German, French, Spanish and Italian. English is the fallback. **French, Spanish and Italian haven't been reviewed by native speakers yet**, so small corrections are just as welcome as a whole new language.

### Correct a translation

The quickest way, without any setup:

1. Find the text: search the repository on GitHub for the wrong phrase (press `/` on the repository page).
2. Open the file, click the pencil icon, fix the text and choose **Propose changes**. GitHub creates the fork and the pull request for you.
3. If the same phrase appears in the web interface, the Android app and the integration, please fix it in all of them.

If you'd rather not touch files at all, open an issue with the **Translation** template and just tell us what's wrong.

Rules of thumb:

- Short and natural, like a well-made app – not word for word. Buttons and tabs must fit on a phone (390 px wide).
- Keep placeholders like `{name}` or `{count}` exactly as they are.
- Use the terms from the [glossary](docs/glossar.md), so a “meal plan” is called the same everywhere.
- Plural texts have the forms `one` and `other`.

### Add a language

1. Copy every English file to the new language code (e.g. `nl.json`, `values-nl/`, `nl.yaml`) and translate it.
2. Add the code to the language lists in the code and the integration.
3. Add a column to the glossary and translate the default categories and pantry list of a new installation.
4. Run the key test (below) and open a pull request.

All files and lists, step by step: **[docs/TRANSLATIONS.md](docs/TRANSLATIONS.md)**. Ingredient units and words for the recipe import (e.g. `tbsp`, `c. à s.`) are in `server/internal/importer/ingredients.go`.

Your recipes and categories are never translated – only the interface, so please don't translate example data in tests.

## Development

You need Go, Node.js and – for the Android app – Android Studio. Commands for Git Bash, Linux or macOS:

```bash
# web interface (builds into the server)
cd web && npm ci && npm run build

# server: tests and a local instance on http://127.0.0.1:8080
cd ../server && go test ./...
REZEPTPLANER_DATA_DIR=./data go run ./cmd/rezeptplaner

# translation keys of web, app and add-on options
go test ./internal/locales

# Android app (flavor "github")
cd ../android && ./gradlew assembleGithubDebug testGithubDebugUnitTest

# Claude skill scripts (needs Pillow)
cd .. && python -m unittest discover -s skills/tests
```

Without Home Assistant the server runs in test mode; Bring! and the system language then aren't available. Architecture, domain rules and data model are described in [docs/PLANUNG.md](docs/PLANUNG.md) (German).

### Pull requests

- One topic per pull request; describe what changes for users.
- Tests must pass – GitHub Actions run them for every pull request.
- If users notice the change, add a line under **Unreleased** in [rezeptplaner/CHANGELOG.md](rezeptplaner/CHANGELOG.md) – in English.
- The import format and the Claude skill belong together: a format change updates both (see [docs/importformat.md](docs/importformat.md)).
- By contributing you agree that your contribution is published under the [MIT license](LICENSE).

---

## Deutsch

Danke fürs Mitmachen! Issues, Discussions und Pull Requests dürfen gern auf Deutsch sein.

- **Fehler und Wünsche:** [neues Issue](https://github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/issues/new/choose) mit der passenden Vorlage. **Fragen und Ideen:** [Discussions](https://github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/discussions).
- **Übersetzung korrigieren:** Text auf GitHub suchen (Taste `/`), Datei öffnen, Stift-Symbol, Text ändern, **Propose changes** – GitHub legt Fork und Pull Request selbst an. Platzhalter wie `{name}` unverändert lassen, Begriffe aus dem [Glossar](docs/glossar.md) verwenden, kurz und natürlich formulieren. Ohne Dateien: Issue mit der Vorlage **Translation**.
- **Neue Sprache:** alle englischen Dateien kopieren und übersetzen, den Code in die Sprachlisten eintragen, Glossar und Startdaten ergänzen – alle Dateien Schritt für Schritt in [docs/TRANSLATIONS.md](docs/TRANSLATIONS.md).
- **Entwicklung:** Befehle oben; Planung und Fachregeln in [docs/PLANUNG.md](docs/PLANUNG.md). Für Nutzer sichtbare Änderungen bekommen eine Zeile unter **Unreleased** in [rezeptplaner/CHANGELOG.md](rezeptplaner/CHANGELOG.md) – auf Englisch.
