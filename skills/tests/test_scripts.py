"""Tests der Skill-Skripte (build_import.py, illustrate.py).

Liegen außerhalb von skills/rezeptplaner-import/, damit sie nicht mit in die
Skill-ZIP kommen. Aufruf aus dem Hauptordner (braucht Pillow):

    python -m unittest discover -s skills/tests
"""

import io
import json
import socket
import sys
import tempfile
import unittest
import zipfile
from pathlib import Path

SCRIPTS = Path(__file__).resolve().parent.parent / "rezeptplaner-import" / "scripts"
sys.path.insert(0, str(SCRIPTS))

import build_import  # noqa: E402
import illustrate  # noqa: E402
from PIL import Image  # noqa: E402

PASTA = {
    "title": "Spaghetti mit Tomatensoße",
    "vessel": "plate",
    "ingredients": [
        {"name": "Spaghetti", "shape": "noodles", "color": "#E9C46A"},
        {"name": "Tomatensoße", "shape": "sauce", "color": "#B8322A", "amount": "little"},
        {"name": "Basilikum", "shape": "leaves", "color": "#3A7D44", "amount": "little"},
    ],
}


class NoNetwork:
    """Lässt jeden Netzzugriff scheitern (wie in der Code-Ausführung von claude.ai)."""

    def __enter__(self):
        self._socket = socket.socket

        def blocked(*args, **kwargs):
            raise AssertionError("Netzzugriff ist nicht erlaubt")

        socket.socket = blocked

    def __exit__(self, *exc):
        socket.socket = self._socket


class IllustrateTest(unittest.TestCase):
    def test_every_vessel_and_shape_renders_without_network(self):
        shapes = sorted(illustrate.SHAPES)
        for vessel in illustrate.VESSELS:
            spec = {"title": vessel, "vessel": vessel,
                    "ingredients": [{"name": s, "shape": s, "color": "#AA6633"} for s in shapes]}
            with NoNetwork():
                data = illustrate.render(spec)
            with Image.open(io.BytesIO(data)) as img:
                self.assertEqual(img.format, "JPEG")
                self.assertEqual(img.size, (1600, 1200))
                self.assertLessEqual(max(img.size), build_import.MAX_SIDE)

    def test_same_spec_same_image(self):
        self.assertEqual(illustrate.render(PASTA), illustrate.render(dict(PASTA)))
        other = dict(PASTA, title="Spaghetti mit Tomatensoße und Käse")
        self.assertNotEqual(illustrate.render(PASTA), illustrate.render(other))

    def test_invalid_spec_is_reported(self):
        errors = illustrate.check_spec({"vessel": "teller", "ingredients": [
            {"shape": "nudeln", "color": "rot", "amount": "viel"}]})
        joined = " | ".join(errors)
        for part in ("vessel", "shape", "color", "amount"):
            self.assertIn(part, joined)
        with self.assertRaises(ValueError):
            illustrate.render({"ingredients": []})


class BuildImportTest(unittest.TestCase):
    def build(self, recipe):
        with tempfile.TemporaryDirectory() as tmp:
            work = Path(tmp) / "arbeit.json"
            work.write_text(json.dumps({"recipes": [recipe]}), encoding="utf-8")
            out = Path(tmp) / "out"
            code = build_import.main(["build", "--data", str(work), "--out-dir", str(out), "--name", "t.zip"])
            self.assertEqual(code, 0)
            with zipfile.ZipFile(out / "t.zip") as zf:
                doc = json.loads(zf.read("recipes.json"))
                names = zf.namelist()
        return doc, names

    def test_illustration_sets_kind_and_version(self):
        doc, names = self.build({"title": "Spaghetti mit Tomatensoße", "base_servings": 2,
                                 "illustration": {k: v for k, v in PASTA.items() if k != "title"}})
        self.assertEqual(doc["version"], 2)
        r = doc["recipes"][0]
        self.assertEqual(r["image_kind"], "illustration")
        self.assertIn(r["image"], names)
        self.assertNotIn("illustration", r)

    def test_without_image_has_no_kind(self):
        doc, _ = self.build({"title": "Brot", "base_servings": 1})
        self.assertNotIn("image_kind", doc["recipes"][0])

    def test_own_category_list_replaces_categories_md(self):
        doc = {"format": "rezeptplaner-import", "version": 2, "recipes": [
            {"title": "A", "base_servings": 1, "categories": ["Pasta", "Ofengericht"],
             "ingredients": [{"name": "Nudeln"}]}]}
        _, warnings = build_import.check_document(doc)
        self.assertEqual(len(warnings), 1)
        self.assertIn("Ofengericht", warnings[0])
        with tempfile.TemporaryDirectory() as tmp:
            own = Path(tmp) / "kategorien.md"
            own.write_text("# Kategorien\n\n## Art\n- Ofengericht\n- Nudeln\n", encoding="utf-8")
            _, warnings = build_import.check_document(doc, own)
        self.assertEqual(len(warnings), 1)
        self.assertIn("„Pasta“ steht nicht in kategorien.md", warnings[0])

    def test_generator_is_skill_version(self):
        doc, _ = self.build({"title": "Brot", "base_servings": 1})
        self.assertEqual(doc["generator"], "claude-skill rezeptplaner-import 2.2")

    def test_schema_checks_version_and_image_kind(self):
        doc = {"format": "rezeptplaner-import", "version": 3, "recipes": [
            {"title": "A", "base_servings": 1, "image": "images/a.jpg", "image_kind": "gemalt"}]}
        errors, _ = build_import.check_document(doc)
        joined = " | ".join(errors)
        self.assertIn("$.version", joined)
        self.assertIn("image_kind", joined)
        doc["version"] = 1
        doc["recipes"][0].pop("image_kind")
        self.assertEqual(build_import.check_document(doc)[0], [])


if __name__ == "__main__":
    unittest.main()
