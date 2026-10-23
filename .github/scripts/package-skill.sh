#!/bin/sh
# Packt die Claude-Skill als rezeptplaner-import-skill.zip (Ordner
# rezeptplaner-import/ mit SKILL.md darin – so erwartet Claude das beim
# Hochladen). Aufruf aus dem Hauptordner: .github/scripts/package-skill.sh ZIELDATEI
set -eu
target="$1"
rm -f "$target"
cd skills
# -X: ohne zusätzliche Dateiattribute; Python-Zwischendateien weglassen.
zip -r -X "$target" rezeptplaner-import -x '*/__pycache__/*' '*.pyc'
unzip -l "$target"
