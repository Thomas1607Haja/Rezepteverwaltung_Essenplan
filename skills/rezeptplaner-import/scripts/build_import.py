#!/usr/bin/env python3
"""Baut eine Importdatei für den Rezeptplaner (ZIP mit recipes.json und Bildern).

Teil der Claude-Skill "rezeptplaner-import". Format: docs/importformat.md im
Repository, Schema: ../schema.json.

Befehle:

  build   Arbeitsdatei -> ZIP-Datei
          python build_import.py build --data arbeit.json --out-dir /mnt/user-data/outputs

  crop    Zuschnitt eines Fotos ansehen, bevor die ZIP-Datei entsteht
          python build_import.py crop --file foto.jpg --crop 0.05 0.1 0.95 0.8 --out vorschau.jpg

  check   Vorhandene recipes.json oder ZIP-Datei prüfen
          python build_import.py check datei.zip

Die Arbeitsdatei hat den Aufbau von recipes.json (format/version dürfen
fehlen), nur steht beim Rezept statt "image" entweder ein Foto als
"image_source":

  "image_source": {
    "file": "/mnt/user-data/uploads/karte-vorne.jpg",
    "crop": [0.0, 0.0, 1.0, 0.62],   # optional: links, oben, rechts, unten (Anteile 0..1)
    "rotate": 0                      # optional: 0, 90, 180, 270 (Grad gegen den Uhrzeigersinn)
  }

oder – wenn die Vorlage kein brauchbares Foto des Gerichts hat – eine
gezeichnete Illustration (Beschreibung wie bei illustrate.py, "title" darf
fehlen):

  "illustration": {
    "vessel": "plate",
    "ingredients": [{"name": "Spaghetti", "shape": "noodles", "color": "#E9C46A"}]
  }

Das Skript setzt "image_kind" ("photo" bzw. "illustration") selbst.

Geprüft wird ohne Zusatzpakete (nur Python-Standardbibliothek); für Bilder
wird Pillow gebraucht.
"""

from __future__ import annotations

import argparse
import datetime as dt
import io
import json
import re
import sys
import unicodedata
import zipfile
from pathlib import Path

SKILL_DIR = Path(__file__).resolve().parent.parent
SCHEMA_PATH = SKILL_DIR / "schema.json"
CATEGORIES_PATH = SKILL_DIR / "categories.md"
GENERATOR = "claude-skill rezeptplaner-import 2.0"
FORMAT_VERSION = 2

MAX_SIDE = 1600       # längste Bildkante in Pixeln
JPEG_QUALITY = 85
MAX_ZIP_BYTES = 30 << 20  # Grenze des Servers für einen Upload


# ---------------------------------------------------------------------------
# Prüfung gegen das JSON-Schema (nur die Teile, die schema.json benutzt)
# ---------------------------------------------------------------------------

def _type_ok(value, expected: str) -> bool:
    if expected == "null":
        return value is None
    if expected == "boolean":
        return isinstance(value, bool)
    if expected == "integer":
        return isinstance(value, int) and not isinstance(value, bool)
    if expected == "number":
        return isinstance(value, (int, float)) and not isinstance(value, bool)
    if expected == "string":
        return isinstance(value, str)
    if expected == "array":
        return isinstance(value, list)
    if expected == "object":
        return isinstance(value, dict)
    raise ValueError(f"Typ {expected} im Schema nicht unterstützt")


def validate(value, schema: dict, root: dict, path: str = "$") -> list[str]:
    """Liefert eine Liste von Fehlermeldungen (leer = gültig)."""
    if "$ref" in schema:
        ref = schema["$ref"]
        if not ref.startswith("#/$defs/"):
            raise ValueError(f"$ref {ref} nicht unterstützt")
        return validate(value, root["$defs"][ref[len("#/$defs/"):]], root, path)

    errors: list[str] = []
    if "const" in schema and value != schema["const"]:
        return [f"{path}: muss {json.dumps(schema['const'], ensure_ascii=False)} sein"]
    if "enum" in schema and value not in schema["enum"]:
        allowed = " oder ".join(json.dumps(v, ensure_ascii=False) for v in schema["enum"])
        return [f"{path}: muss {allowed} sein"]

    if "type" in schema:
        types = schema["type"] if isinstance(schema["type"], list) else [schema["type"]]
        if not any(_type_ok(value, t) for t in types):
            return [f"{path}: falscher Typ (erwartet {' oder '.join(types)})"]

    if isinstance(value, str):
        if len(value) < schema.get("minLength", 0):
            errors.append(f"{path}: darf nicht leer sein" if schema.get("minLength") == 1 else f"{path}: zu kurz")
        if "maxLength" in schema and len(value) > schema["maxLength"]:
            errors.append(f"{path}: höchstens {schema['maxLength']} Zeichen")
        if "pattern" in schema and not re.search(schema["pattern"], value):
            errors.append(f"{path}: passt nicht zum Muster {schema['pattern']}")

    if isinstance(value, (int, float)) and not isinstance(value, bool):
        if "minimum" in schema and value < schema["minimum"]:
            errors.append(f"{path}: mindestens {schema['minimum']}")
        if "maximum" in schema and value > schema["maximum"]:
            errors.append(f"{path}: höchstens {schema['maximum']}")

    if isinstance(value, list):
        if len(value) < schema.get("minItems", 0):
            errors.append(f"{path}: mindestens {schema['minItems']} Einträge")
        if "maxItems" in schema and len(value) > schema["maxItems"]:
            errors.append(f"{path}: höchstens {schema['maxItems']} Einträge")
        if schema.get("uniqueItems"):
            seen = [json.dumps(v, sort_keys=True) for v in value]
            if len(seen) != len(set(seen)):
                errors.append(f"{path}: Einträge doppelt")
        if "items" in schema:
            for i, item in enumerate(value):
                errors += validate(item, schema["items"], root, f"{path}[{i}]")

    if isinstance(value, dict):
        props = schema.get("properties", {})
        for key in schema.get("required", []):
            if key not in value:
                errors.append(f"{path}: Feld „{key}“ fehlt")
        for key, item in value.items():
            if key in props:
                errors += validate(item, props[key], root, f"{path}.{key}")
            elif schema.get("additionalProperties") is False:
                errors.append(f"{path}: unbekanntes Feld „{key}“")
    return errors


def load_schema() -> dict:
    return json.loads(SCHEMA_PATH.read_text(encoding="utf-8"))


def load_categories() -> set[str]:
    names = set()
    in_list = False
    for line in CATEGORIES_PATH.read_text(encoding="utf-8").splitlines():
        if line.startswith("## "):
            in_list = not line.startswith("## Hinweise")
        elif in_list and line.startswith("- "):
            names.add(line[2:].strip())
    return names


def check_document(doc: dict) -> tuple[list[str], list[str]]:
    """Prüft recipes.json: (Fehler, Hinweise)."""
    errors = validate(doc, load_schema(), load_schema())
    warnings: list[str] = []
    known = load_categories()
    for i, r in enumerate(doc.get("recipes", []) if isinstance(doc, dict) else []):
        if not isinstance(r, dict):
            continue
        title = r.get("title") or f"Rezept {i + 1}"
        for c in r.get("categories", []) or []:
            if c not in known:
                warnings.append(f"{title}: Kategorie „{c}“ steht nicht in categories.md und wird beim Import weggelassen")
        for j, step in enumerate(r.get("steps", []) or []):
            if isinstance(step, str) and re.match(r"^\s*(\d+[.)]|Schritt \d+)", step):
                warnings.append(f"{title}: Schritt {j + 1} beginnt mit einer Nummer – Nummern weglassen")
        if not r.get("ingredients"):
            warnings.append(f"{title}: keine Zutaten")
    return errors, warnings


# ---------------------------------------------------------------------------
# Bilder
# ---------------------------------------------------------------------------

def _pillow():
    try:
        from PIL import Image, ImageOps  # noqa: F401
    except ImportError:
        sys.exit("Pillow fehlt: pip install pillow (oder Rezepte ohne Bild bauen)")
    from PIL import Image, ImageOps
    return Image, ImageOps


def render_image(spec: dict) -> bytes:
    """Liest ein Foto, dreht, schneidet zu, verkleinert und liefert JPEG-Bytes.

    Neu gespeichert wird ohne EXIF-Daten (also auch ohne GPS-Position).
    """
    Image, ImageOps = _pillow()
    with Image.open(spec["file"]) as img:
        img = ImageOps.exif_transpose(img)  # Ausrichtung des Handys übernehmen
        rotate = int(spec.get("rotate", 0)) % 360
        if rotate:
            img = img.rotate(rotate, expand=True)
        crop = spec.get("crop")
        if crop:
            left, top, right, bottom = (float(v) for v in crop)
            if not (0 <= left < right <= 1 and 0 <= top < bottom <= 1):
                raise ValueError(f"crop {crop}: erwartet 0 ≤ links < rechts ≤ 1 und 0 ≤ oben < unten ≤ 1")
            w, h = img.size
            img = img.crop((round(left * w), round(top * h), round(right * w), round(bottom * h)))
        img = img.convert("RGB")
        img.thumbnail((MAX_SIDE, MAX_SIDE), Image.Resampling.LANCZOS)
        out = io.BytesIO()
        img.save(out, "JPEG", quality=JPEG_QUALITY, optimize=True)
        return out.getvalue()


def render_illustration(spec: dict, title: str) -> bytes:
    """Zeichnet die Illustration (scripts/illustrate.py) und liefert JPEG-Bytes."""
    sys.path.insert(0, str(Path(__file__).resolve().parent))
    import illustrate  # liegt neben diesem Skript

    spec = dict(spec)
    spec.setdefault("title", title)
    return illustrate.render(spec)


# ---------------------------------------------------------------------------
# Dateinamen
# ---------------------------------------------------------------------------

# Füllwörter, mit denen ein gekürzter Name nicht enden soll
_FILLERS = {"und", "mit", "an", "auf", "in", "im", "vom", "von", "der", "die", "das", "dem", "den", "zu", "aus"}


def slug(text: str, max_len: int = 40) -> str:
    """Kurzer Dateiname aus einem Titel, nur ganze Wörter."""
    text = text.lower().replace("ß", "ss").replace("ä", "ae").replace("ö", "oe").replace("ü", "ue")
    text = unicodedata.normalize("NFKD", text).encode("ascii", "ignore").decode()
    words = [w for w in re.split(r"[^a-z0-9]+", text) if w]
    out: list[str] = []
    for w in words:
        if out and len("-".join(out + [w])) > max_len:
            break
        out.append(w[:max_len])
    while len(out) > 1 and out[-1] in _FILLERS:
        out.pop()
    return "-".join(out) or "rezept"


def zip_name(titles: list[str], today: dt.date) -> str:
    base = f"rezepte-{today.isoformat()}-{slug(titles[0], 30)}"
    if len(titles) == 2:
        base += "-und-1-weiteres"
    elif len(titles) > 2:
        base += f"-und-{len(titles) - 1}-weitere"
    return base + ".zip"


# ---------------------------------------------------------------------------
# Befehle
# ---------------------------------------------------------------------------

_STEP_NUMBER = re.compile(r"^\s*(?:Schritt\s+)?\d+\s*[.):]\s*")


def cmd_build(args) -> int:
    work = json.loads(Path(args.data).read_text(encoding="utf-8"))
    recipes = work.get("recipes") if isinstance(work, dict) else None
    if not recipes:
        print("FEHLER: Die Arbeitsdatei enthält keine Rezepte (Feld „recipes“).")
        return 1

    images: dict[str, bytes] = {}
    out_recipes = []
    for i, r in enumerate(recipes):
        r = dict(r)
        # Nummern am Anfang der Schritte zählt der Rezeptplaner selbst.
        if isinstance(r.get("steps"), list):
            r["steps"] = [_STEP_NUMBER.sub("", s, count=1) if isinstance(s, str) else s for s in r["steps"]]
        spec = r.pop("image_source", None)
        drawing = r.pop("illustration", None)
        r.pop("image", None)
        r.pop("image_kind", None)
        name = f"images/{i + 1:02d}-{slug(r.get('title', ''))}.jpg"
        if spec and drawing:
            print(f"HINWEIS: „{r.get('title')}“ hat Foto und Illustration – das Foto wird verwendet.")
        if spec:
            try:
                images[name] = render_image(spec)
                r["image"], r["image_kind"] = name, "photo"
            except Exception as e:  # Rezept trotzdem bauen, nur ohne Bild
                print(f"HINWEIS: Bild für „{r.get('title')}“ nicht verwendbar ({e}) – Rezept ohne Bild.")
        elif drawing:
            try:
                images[name] = render_illustration(drawing, r.get("title", ""))
                r["image"], r["image_kind"] = name, "illustration"
            except Exception as e:
                print(f"HINWEIS: Illustration für „{r.get('title')}“ nicht möglich ({e}) – Rezept ohne Bild.")
        out_recipes.append(r)

    doc = {"format": "rezeptplaner-import", "version": FORMAT_VERSION, "generator": GENERATOR, "recipes": out_recipes}
    errors, warnings = check_document(doc)
    if errors:
        print("FEHLER – keine ZIP-Datei erzeugt:")
        for e in errors:
            print("  -", e)
        return 1

    out_dir = Path(args.out_dir)
    out_dir.mkdir(parents=True, exist_ok=True)
    titles = [r["title"] for r in out_recipes]
    target = out_dir / (args.name or zip_name(titles, dt.date.today()))
    with zipfile.ZipFile(target, "w", zipfile.ZIP_DEFLATED) as zf:
        zf.writestr("recipes.json", json.dumps(doc, ensure_ascii=False, indent=2) + "\n")
        for name, data in images.items():
            zf.writestr(name, data, compress_type=zipfile.ZIP_STORED)  # JPEG ist schon gepackt

    size = target.stat().st_size
    print(f"OK: {target} ({size / 1024:.0f} KB)")
    if size > MAX_ZIP_BYTES:
        print(f"WARNUNG: größer als {MAX_ZIP_BYTES >> 20} MB – der Rezeptplaner lehnt die Datei ab. Auf mehrere Dateien aufteilen.")
    for r in out_recipes:
        picture = {"photo": "mit Foto", "illustration": "mit Illustration"}.get(r.get("image_kind"), "ohne Bild")
        print(f"  - {r['title']} ({r['base_servings']} Portionen, {len(r.get('ingredients', []))} Zutaten, "
              f"{len(r.get('steps', []))} Schritte, {picture})")
    for w in warnings:
        print("HINWEIS:", w)
    return 0


def cmd_crop(args) -> int:
    data = render_image({"file": args.file, "crop": args.crop, "rotate": args.rotate})
    Path(args.out).write_bytes(data)
    print(f"OK: {args.out} ({len(data) / 1024:.0f} KB)")
    return 0


def cmd_check(args) -> int:
    path = Path(args.file)
    image_names: set[str] = set()
    if zipfile.is_zipfile(path):
        with zipfile.ZipFile(path) as zf:
            names = zf.namelist()
            if "recipes.json" not in names:
                print("FEHLER: recipes.json fehlt in der ZIP-Datei (muss oben liegen).")
                return 1
            doc = json.loads(zf.read("recipes.json").decode("utf-8"))
            image_names = {n for n in names if n.startswith("images/")}
    else:
        doc = json.loads(path.read_text(encoding="utf-8-sig"))
    errors, warnings = check_document(doc)
    for r in doc.get("recipes", []) if isinstance(doc, dict) else []:
        img = r.get("image") if isinstance(r, dict) else None
        if img and img not in image_names:
            warnings.append(f"{r.get('title')}: Bild {img} fehlt in der Datei")
    for e in errors:
        print("FEHLER:", e)
    for w in warnings:
        print("HINWEIS:", w)
    if not errors:
        print("OK: gültig")
    return 1 if errors else 0


def main(argv=None) -> int:
    parser = argparse.ArgumentParser(description="Importdatei für den Rezeptplaner bauen")
    sub = parser.add_subparsers(dest="cmd", required=True)

    b = sub.add_parser("build", help="Arbeitsdatei -> ZIP-Datei")
    b.add_argument("--data", required=True, help="Arbeitsdatei (JSON)")
    b.add_argument("--out-dir", required=True, help="Zielordner, z. B. /mnt/user-data/outputs")
    b.add_argument("--name", help="Dateiname der ZIP-Datei (sonst automatisch)")
    b.set_defaults(func=cmd_build)

    c = sub.add_parser("crop", help="Zuschnitt eines Fotos als Vorschau speichern")
    c.add_argument("--file", required=True)
    c.add_argument("--crop", type=float, nargs=4, metavar=("LINKS", "OBEN", "RECHTS", "UNTEN"))
    c.add_argument("--rotate", type=int, default=0)
    c.add_argument("--out", required=True)
    c.set_defaults(func=cmd_crop)

    k = sub.add_parser("check", help="recipes.json oder ZIP-Datei prüfen")
    k.add_argument("file")
    k.set_defaults(func=cmd_check)

    args = parser.parse_args(argv)
    return args.func(args)


if __name__ == "__main__":
    sys.exit(main())
