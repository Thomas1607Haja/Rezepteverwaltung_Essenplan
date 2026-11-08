"""Erzeugt die Fehlertexte der Android-App aus denen der Weboberfläche.

Quelle: web/src/locales/errors/<sprache>.json (ein Text je Fehlercode des
Servers, siehe server/internal/apierr). Ziel:
  android/app/src/main/res/values[-<sprache>]/strings_errors.xml
  android/app/src/main/java/.../api/ErrorTexts.kt (Code → Ressource)

Aufruf nach jeder Änderung der Fehlertexte (vom Hauptordner aus):
  python android/tools/gen_errors.py
Der Go-Test server/internal/locales prüft, dass beide Seiten übereinstimmen.
Platzhalter bleiben {name}; ErrorTexts.kt setzt sie zur Laufzeit ein.
"""

import json
import os
import re

ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
SRC = os.path.join(ROOT, "web", "src", "locales", "errors")
RES = os.path.join(ROOT, "android", "app", "src", "main", "res")
KT = os.path.join(ROOT, "android", "app", "src", "main", "java", "io", "github", "thomas1607haja", "rezeptplaner", "api", "ErrorTexts.kt")
LANGS = ["en", "de", "fr", "es", "it"]


def android_escape(text):
    """Text für strings.xml: XML-Zeichen und die Sonderzeichen von Android."""
    text = text.replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;")
    text = text.replace("\\", "\\\\").replace("'", "\\'").replace('"', '\\"')
    if text.startswith("@") or text.startswith("?"):
        text = "\\" + text
    return text


def resource_name(code):
    return "error_" + re.sub(r"[^a-z0-9_]", "_", code)


# Gemeinsame Bereiche der Weboberfläche, die die App ebenso braucht. Sie
# landen in strings_shared.xml als <bereich>_<schlüssel>, z. B. common_save,
# slots_lunch, format_servings (Plural).
SHARED = ["common", "slots", "courses", "quick", "format"]


def write_shared():
    for lang in LANGS:
        folder = "values" if lang == "en" else "values-" + lang
        lines = [
            '<?xml version="1.0" encoding="utf-8"?>',
            "<!-- Erzeugt von android/tools/gen_errors.py aus web/src/locales/{" + ",".join(SHARED) + "} – nicht von Hand ändern. -->",
            "<resources>",
        ]
        for area in SHARED:
            with open(os.path.join(ROOT, "web", "src", "locales", area, lang + ".json"), encoding="utf-8") as f:
                data = json.load(f)
            for key in sorted(data):
                name = f"{area}_{key}"
                value = data[key]
                if isinstance(value, dict):
                    # Französisch, Spanisch und Italienisch kennen die Form „many“
                    # (Millionen); Android-Lint verlangt sie. Sie gleicht „other“.
                    if lang in ("fr", "es", "it") and "many" not in value:
                        value = {**value, "many": value["other"]}
                    lines.append(f'    <plurals name="{name}">')
                    for form in ("zero", "one", "two", "few", "many", "other"):
                        if form in value:
                            lines.append(f'        <item quantity="{form}">{android_escape(value[form])}</item>')
                    lines.append("    </plurals>")
                else:
                    lines.append(f'    <string name="{name}" formatted="false">{android_escape(value)}</string>')
        lines.append("</resources>")
        with open(os.path.join(RES, folder, "strings_shared.xml"), "w", encoding="utf-8", newline="\n") as f:
            f.write("\n".join(lines) + "\n")


def main():
    texts = {}
    for lang in LANGS:
        with open(os.path.join(SRC, lang + ".json"), encoding="utf-8") as f:
            texts[lang] = json.load(f)
    codes = sorted(texts["en"])
    for lang in LANGS:
        folder = "values" if lang == "en" else "values-" + lang
        os.makedirs(os.path.join(RES, folder), exist_ok=True)
        lines = [
            '<?xml version="1.0" encoding="utf-8"?>',
            "<!-- Erzeugt von android/tools/gen_errors.py aus web/src/locales/errors – nicht von Hand ändern. -->",
            "<resources>",
        ]
        for code in codes:
            value = texts[lang].get(code, texts["en"][code])
            # formatted="false": Prozentzeichen sind kein Format, Platzhalter heißen {name}.
            lines.append(f'    <string name="{resource_name(code)}" formatted="false">{android_escape(value)}</string>')
        lines.append("</resources>")
        with open(os.path.join(RES, folder, "strings_errors.xml"), "w", encoding="utf-8", newline="\n") as f:
            f.write("\n".join(lines) + "\n")

    write_shared()

    entries = "\n".join(f'        "{code}" to R.string.{resource_name(code)},' for code in codes)
    kotlin = f'''// Erzeugt von android/tools/gen_errors.py – nicht von Hand ändern.
package io.github.thomas1607haja.rezeptplaner.api

import io.github.thomas1607haja.rezeptplaner.R

/** Fehlercodes des Servers (server/internal/apierr) → Text in res/values…/strings_errors.xml. */
object ErrorTexts {{
    val ids: Map<String, Int> = mapOf(
{entries}
    )
}}
'''
    with open(KT, "w", encoding="utf-8", newline="\n") as f:
        f.write(kotlin)
    print(f"{len(codes)} Fehlertexte in {len(LANGS)} Sprachen")


if __name__ == "__main__":
    main()
