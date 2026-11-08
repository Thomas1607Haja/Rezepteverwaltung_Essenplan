"""Texte der Integration, die Home Assistant nicht selbst übersetzt.

Namen von Entitäten, Einrichtung und Fehler stehen in translations/*.json
(übersetzt Home Assistant). Die Titel der Kalendertermine und den Zustand
des Sensors baut die Integration aber selbst zusammen; deren Texte stehen
in texts/<sprache>.json. Es gilt die Systemsprache von Home Assistant,
Rückfall Englisch.
"""

from __future__ import annotations

import json
from pathlib import Path
from typing import Any

from homeassistant.core import HomeAssistant

LANGUAGES = ("en", "de", "fr", "es", "it")

# Einmal beim Laden des Moduls lesen (das geschieht nicht in der
# Ereignisschleife von Home Assistant).
_TEXTS: dict[str, dict[str, Any]] = {
    lang: json.loads((Path(__file__).parent / "texts" / f"{lang}.json").read_text(encoding="utf-8"))
    for lang in LANGUAGES
}


def language(hass: HomeAssistant | None) -> str:
    """Systemsprache von Home Assistant als Code ("de"), sonst "en"."""
    tag = (getattr(getattr(hass, "config", None), "language", None) or "en").lower()
    base = tag.replace("_", "-").split("-")[0]
    return base if base in LANGUAGES else "en"


def text(lang: str, key: str, **params: Any) -> str:
    """Text zu key in lang; count wählt die Pluralform (one bei 1)."""
    value = _TEXTS.get(lang, {}).get(key, _TEXTS["en"][key])
    if isinstance(value, dict):
        # Für die fünf Sprachen genügt: Einzahl bei 1 (Französisch auch bei 0).
        count = params.get("count", 0)
        one = count == 1 or (lang == "fr" and count == 0)
        value = value["one"] if one else value["other"]
    for name, val in params.items():
        value = value.replace("{" + name + "}", str(val))
    return value
