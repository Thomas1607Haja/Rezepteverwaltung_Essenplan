#!/usr/bin/env python3
"""Zeichnet eine einfache Illustration eines Gerichts (Draufsicht, flacher Stil).

Teil der Claude-Skill "rezeptplaner-import". Ersatz für das Rezeptbild, wenn
die Vorlage kein brauchbares Foto des Gerichts hat. Braucht nur Pillow und die
Python-Standardbibliothek – keine Installation, kein Netz, keine Schriften.

    python illustrate.py --spec gericht.json --out bild.jpg

Beschreibung des Gerichts (JSON):

  {
    "title": "Spaghetti mit Tomatensoße",      # nur Startwert für den Zufall
    "vessel": "plate",                          # plate | bowl | soup | casserole | pan
    "vessel_color": "#F7F4EE",                  # optional
    "ingredients": [                            # Reihenfolge = von unten nach oben
      {"name": "Spaghetti", "shape": "noodles", "color": "#E9C46A"},
      {"name": "Tomatensoße", "shape": "sauce", "color": "#C0392B", "amount": "some"},
      {"name": "Basilikum", "shape": "leaves", "color": "#3A7D44", "amount": "little"}
    ]
  }

Formen (shape): liquid, sauce, grains, noodles, pasta, chunks, cubes, strips,
slices, rounds, leaves, herbs, crumbs, crust, drizzle.
Menge (amount): little | some | much (Standard some).
Größe (size): small | medium | large (Standard medium).

Gleiche Beschreibung ergibt immer dasselbe Bild. Ausgabe: JPEG 1600 × 1200,
das Gericht liegt mittig, damit auch ein Zuschnitt auf 16:9 passt.
"""

from __future__ import annotations

import argparse
import hashlib
import io
import json
import math
import random
import sys
from pathlib import Path

WIDTH, HEIGHT = 1600, 1200
SUPER = 2  # intern doppelt so groß zeichnen, dann verkleinern (glatte Kanten)
JPEG_QUALITY = 88

BACKGROUND = (242, 236, 227)
NAPKIN = (214, 224, 210)
VESSELS = {
    "plate": (251, 250, 247),
    "bowl": (250, 248, 244),
    "soup": (251, 250, 247),
    "casserole": (247, 244, 238),
    "pan": (62, 60, 58),
}
SHAPES = {"liquid", "sauce", "grains", "noodles", "pasta", "chunks", "cubes", "strips",
          "slices", "rounds", "leaves", "herbs", "crumbs", "crust", "drizzle"}
AMOUNTS = {"little": 0.55, "some": 1.0, "much": 1.6}
SIZES = {"small": 0.6, "medium": 1.0, "large": 1.6}


def _pillow():
    try:
        from PIL import Image, ImageChops, ImageDraw, ImageFilter
    except ImportError:
        sys.exit("Pillow fehlt – Illustration nicht möglich; Rezept ohne Bild bauen und im Chat erwähnen.")
    return Image, ImageChops, ImageDraw, ImageFilter


# ---------------------------------------------------------------------------
# Hilfen
# ---------------------------------------------------------------------------

def hex_rgb(value: str) -> tuple[int, int, int]:
    v = value.strip().lstrip("#")
    if len(v) == 3:
        v = "".join(c * 2 for c in v)
    if len(v) != 6:
        raise ValueError(f"Farbe {value!r}: erwartet #RRGGBB")
    return tuple(int(v[i:i + 2], 16) for i in (0, 2, 4))


def shade(rgb, factor: float) -> tuple[int, int, int]:
    """factor < 1 dunkler, > 1 heller (Richtung Weiß)."""
    if factor <= 1:
        return tuple(max(0, round(c * factor)) for c in rgb)
    t = min(1.0, factor - 1)
    return tuple(round(c + (255 - c) * t) for c in rgb)


def jitter(rgb, rng: random.Random, amount: float = 0.08):
    return shade(rgb, 1 + rng.uniform(-amount, amount))


def blob(cx, cy, r, rng: random.Random, wobble=0.12, points=36):
    """Unregelmäßige, weiche Fläche als Polygon."""
    phases = [rng.uniform(0, 2 * math.pi) for _ in range(3)]
    amps = [wobble * rng.uniform(0.5, 1.0) / (k + 1) for k in range(3)]
    out = []
    for i in range(points):
        a = 2 * math.pi * i / points
        f = 1 + sum(amps[k] * math.sin((k + 2) * a + phases[k]) for k in range(3))
        out.append((cx + r * f * math.cos(a), cy + r * f * math.sin(a)))
    return out


def rotated(points, cx, cy, angle):
    c, s = math.cos(angle), math.sin(angle)
    return [(cx + x * c - y * s, cy + x * s + y * c) for x, y in points]


def rounded_rect_poly(w, h, r, steps=6):
    """Abgerundetes Rechteck um (0, 0) als Polygon (zum Drehen)."""
    r = min(r, w / 2, h / 2)
    pts = []
    for (cx, cy, a0) in ((w / 2 - r, h / 2 - r, 0), (-w / 2 + r, h / 2 - r, 90),
                         (-w / 2 + r, -h / 2 + r, 180), (w / 2 - r, -h / 2 + r, 270)):
        for k in range(steps + 1):
            a = math.radians(a0 + 90 * k / steps)
            pts.append((cx + r * math.cos(a), cy + r * math.sin(a)))
    return pts


class Area:
    """Fläche, die das Essen füllen darf (Kreis oder abgerundetes Rechteck)."""

    def __init__(self, kind, cx, cy, rx, ry, corner=0):
        self.kind, self.cx, self.cy, self.rx, self.ry, self.corner = kind, cx, cy, rx, ry, corner

    @property
    def r(self):
        return min(self.rx, self.ry)

    def point(self, rng: random.Random, spread=0.85):
        """Zufälliger Punkt, eher zur Mitte hin."""
        if self.kind == "circle":
            a = rng.uniform(0, 2 * math.pi)
            d = math.sqrt(rng.random()) * spread
            return self.cx + self.rx * d * math.cos(a), self.cy + self.ry * d * math.sin(a)
        return (self.cx + rng.uniform(-1, 1) * self.rx * spread,
                self.cy + rng.uniform(-1, 1) * self.ry * spread)

    def mask(self, Image, ImageDraw, size):
        m = Image.new("L", size, 0)
        d = ImageDraw.Draw(m)
        box = (self.cx - self.rx, self.cy - self.ry, self.cx + self.rx, self.cy + self.ry)
        if self.kind == "circle":
            d.ellipse(box, fill=255)
        else:
            d.rounded_rectangle(box, radius=self.corner, fill=255)
        return m


# ---------------------------------------------------------------------------
# Zutaten
# ---------------------------------------------------------------------------

def _coverage_blob(area: Area, rng, amount):
    """Grundfläche für Soße, Reis, Nudeln: je nach Menge kleiner oder größer."""
    if area.kind == "rect":
        # Auflaufform: fast ganz gefüllt
        f = min(1.0, 0.75 + 0.2 * amount)
        rx, ry = area.rx * f, area.ry * f
        pts = blob(0, 0, 1, rng, wobble=0.05)
        return [(area.cx + x * rx, area.cy + y * ry) for x, y in pts]
    r = area.r * min(0.95, 0.5 + 0.3 * amount)
    ox, oy = rng.uniform(-0.06, 0.06) * area.r, rng.uniform(-0.06, 0.06) * area.r
    return blob(area.cx + ox, area.cy + oy, r, rng)


def clip_to(layer, mask, ImageChops):
    """Kopie von layer, sichtbar nur innerhalb der Maske (Farben bleiben unverändert)."""
    out = layer.copy()
    out.putalpha(ImageChops.multiply(layer.getchannel("A"), mask))
    return out


def draw_ingredient(layer, ing, area: Area, rng: random.Random, Image, ImageChops, ImageDraw, ImageFilter):
    """Zeichnet eine Zutat als eigene Ebene und legt sie über die bisherigen.

    Die Ebene ist in der Zutatenfarbe durchsichtig angelegt; so mischen sich
    halbtransparente Punkte und weichgezeichnete Flecken nicht mit Schwarz.
    """
    top = Image.new("RGBA", layer.size, hex_rgb(ing["color"]) + (0,))
    _draw_ingredient(top, ing, area, rng, Image, ImageChops, ImageDraw, ImageFilter)
    layer.alpha_composite(top)


def _draw_ingredient(layer, ing, area: Area, rng: random.Random, Image, ImageChops, ImageDraw, ImageFilter):
    shape = ing["shape"]
    color = hex_rgb(ing["color"])
    amount = AMOUNTS[ing.get("amount", "some")]
    size = SIZES[ing.get("size", "medium")]
    R = area.r
    d = ImageDraw.Draw(layer)

    if shape == "liquid":
        full = area.mask(Image, ImageDraw, layer.size)
        layer.paste(color + (255,), (0, 0), full)
        # sanfte hellere Schlieren und ein paar Fettaugen
        tint = Image.new("RGBA", layer.size, shade(color, 1.12) + (0,))
        td = ImageDraw.Draw(tint)
        for _ in range(5):
            x, y = area.point(rng, 0.6)
            r = R * rng.uniform(0.15, 0.35)
            td.ellipse((x - r, y - r * 0.7, x + r, y + r * 0.7), fill=shade(color, 1.12) + (90,))
        tint = tint.filter(ImageFilter.GaussianBlur(R * 0.08))
        layer.alpha_composite(tint)
        for _ in range(int(14 * amount)):
            x, y = area.point(rng, 0.8)
            r = R * rng.uniform(0.008, 0.02)
            d.ellipse((x - r, y - r, x + r, y + r), fill=shade(color, 1.25) + (200,))
        return

    if shape == "crust":
        full = area.mask(Image, ImageDraw, layer.size)
        layer.paste(color + (255,), (0, 0), full)
        # gebräunte Stellen: transparent in der Bräunungsfarbe anlegen, sonst
        # zieht der Weichzeichner Schwarz in die Ränder
        brown = tuple(round(c * 0.4 + w * 0.6) for c, w in zip(color, (170, 90, 28)))
        spots = Image.new("RGBA", layer.size, brown + (0,))
        sd = ImageDraw.Draw(spots)
        for _ in range(int(45 * amount)):
            x, y = area.point(rng, 0.95)
            r = R * rng.uniform(0.03, 0.09)
            sd.polygon(blob(x, y, r, rng, 0.35, 14), fill=jitter(brown, rng, 0.06) + (rng.randint(150, 215),))
        spots = spots.filter(ImageFilter.GaussianBlur(R * 0.014))
        layer.alpha_composite(clip_to(spots, full, ImageChops))
        for _ in range(int(45 * amount)):  # Bläschen im Käse
            x, y = area.point(rng, 0.95)
            r = R * rng.uniform(0.01, 0.025)
            d.ellipse((x - r, y - r, x + r, y + r), fill=shade(color, 1.2) + (210,))
        return

    if shape == "sauce":
        if area.kind == "rect":
            pts = _coverage_blob(area, rng, amount)
            cx, cy, r0 = area.cx, area.cy, R
        else:
            cx, cy = area.cx + rng.uniform(-0.05, 0.05) * R, area.cy + rng.uniform(-0.05, 0.05) * R
            r0 = R * (0.25 + 0.3 * amount)
            pts = blob(cx, cy, r0, rng, 0.22, 48)
        d.polygon([(x + R * 0.012, y + R * 0.018) for x, y in pts], fill=shade(color, 0.78) + (255,))
        d.polygon(pts, fill=color + (255,))
        # Stückchen und Glanz, nur innerhalb der Soße
        if area.kind != "rect":
            for _ in range(int(14 * amount) + 6):  # Ränder auflockern
                a = rng.uniform(0, 2 * math.pi)
                rr = r0 * rng.uniform(0.85, 1.02)
                x, y = cx + rr * math.cos(a), cy + rr * math.sin(a)
                d.polygon(blob(x, y, R * rng.uniform(0.04, 0.08), rng, 0.3, 12), fill=jitter(color, rng, 0.05) + (255,))
        for _ in range(int(45 * amount) + 10):
            a, rr = rng.uniform(0, 2 * math.pi), r0 * math.sqrt(rng.random()) * 0.85
            x, y = cx + rr * math.cos(a), cy + rr * math.sin(a)
            r = R * rng.uniform(0.012, 0.03)
            d.polygon(blob(x, y, r, rng, 0.3, 8), fill=jitter(shade(color, 0.8), rng) + (255,))
        for _ in range(int(10 * amount) + 4):
            a, rr = rng.uniform(0, 2 * math.pi), r0 * math.sqrt(rng.random()) * 0.7
            x, y = cx + rr * math.cos(a), cy + rr * math.sin(a)
            r = R * rng.uniform(0.008, 0.016)
            d.ellipse((x - r * 1.6, y - r, x + r * 1.6, y + r), fill=shade(color, 1.35) + (200,))
        return

    if shape == "grains":
        region = Image.new("L", layer.size, 0)
        ImageDraw.Draw(region).polygon(_coverage_blob(area, rng, amount), fill=255)
        tmp = Image.new("RGBA", layer.size, color + (0,))
        td = ImageDraw.Draw(tmp)
        x0, y0, x1, y1 = region.getbbox()
        tmp.paste(shade(color, 0.86) + (255,), (0, 0), region)
        g = R * 0.022 * size
        count = int((x1 - x0) * (y1 - y0) / (g * g) * 0.9)
        for _ in range(count):
            x, y = rng.uniform(x0, x1), rng.uniform(y0, y1)
            pts = rotated(rounded_rect_poly(g * 2.2, g, g / 2, 3), x, y, rng.uniform(0, math.pi))
            td.polygon(pts, fill=jitter(color, rng, 0.07) + (255,))
        layer.alpha_composite(clip_to(tmp, region, ImageChops))
        return

    if shape == "noodles":
        # Nudelnest: jede Nudel wandert in Schlingen und kehrt am Rand um,
        # dadurch ein natürlicher Rand ohne Ausschneidekante.
        w = R * 0.03 * size
        if area.kind == "rect":
            limit_x, limit_y = area.rx * 0.95, area.ry * 0.95
        else:
            limit_x = limit_y = R * min(0.92, 0.55 + 0.25 * amount)
        for _ in range(int(110 * amount)):
            a2 = rng.uniform(0, 2 * math.pi)
            rr = math.sqrt(rng.random()) * 0.8
            x, y = area.cx + limit_x * rr * math.cos(a2), area.cy + limit_y * rr * math.sin(a2)
            a = rng.uniform(0, 2 * math.pi)
            turn = rng.uniform(-0.25, 0.25)
            step = R * 0.02
            pts = [(x, y)]
            for _ in range(rng.randint(35, 70)):
                turn += rng.uniform(-0.08, 0.08)
                turn = max(-0.3, min(0.3, turn))
                a += turn
                nx, ny = (x - area.cx) / limit_x, (y - area.cy) / limit_y
                if nx * nx + ny * ny > 0.85:  # am Rand zur Mitte lenken
                    target = math.atan2(area.cy - y, area.cx - x)
                    a += 0.35 * math.sin(target - a)
                x += math.cos(a) * step
                y += math.sin(a) * step
                pts.append((x, y))
            d.line(pts, fill=shade(color, 0.72) + (255,), width=round(w * 1.35), joint="curve")
            d.line(pts, fill=jitter(color, rng, 0.05) + (255,), width=round(w), joint="curve")
            d.line([(px - w * 0.18, py - w * 0.18) for px, py in pts], fill=shade(color, 1.18) + (255,),
                   width=max(1, round(w * 0.28)))
        return

    if shape == "drizzle":
        cx, cy = area.point(rng, 0.2)
        pts = []
        turns = 2.2 * amount
        for k in range(160):
            t = k / 159
            a = t * turns * 2 * math.pi
            r = R * (0.08 + 0.45 * t)
            pts.append((cx + r * math.cos(a), cy + r * math.sin(a) * 0.9))
        d.line(pts, fill=color + (240,), width=round(R * 0.045 * size), joint="curve")
        return

    # Einzelne Stücke, verteilt über die Fläche
    base_counts = {"pasta": 26, "chunks": 14, "cubes": 18, "strips": 14, "slices": 8,
                   "rounds": 16, "leaves": 10, "herbs": 60, "crumbs": 140}
    count = max(1, int(base_counts[shape] * amount))
    spread = 0.62 if area.kind == "circle" else 0.85
    for _ in range(count):
        x, y = area.point(rng, spread)
        a = rng.uniform(0, 2 * math.pi)
        col = jitter(color, rng)
        if shape == "pasta":
            s = R * 0.07 * size
            poly = rotated(rounded_rect_poly(s * 2.2, s * 0.8, s * 0.3), x, y, a)
            d.polygon([(px + s * 0.08, py + s * 0.1) for px, py in poly], fill=shade(color, 0.75) + (255,))
            d.polygon(poly, fill=col + (255,))
            for k in (-0.5, 0, 0.5):  # Rillen
                p = rotated([(k * s, -s * 0.38), (k * s + s * 0.2, s * 0.38)], x, y, a)
                d.line(p, fill=shade(color, 0.85) + (255,), width=max(1, round(s * 0.08)))
        elif shape in ("chunks", "cubes", "strips"):
            s = R * (0.075 if shape == "chunks" else 0.06) * size
            if shape == "chunks":
                poly = blob(x, y, s, rng, 0.3, 9)
            elif shape == "cubes":
                poly = rotated(rounded_rect_poly(s * 1.6, s * 1.6, s * 0.25), x, y, a)
            else:
                poly = rotated(rounded_rect_poly(s * 3.2, s * 0.8, s * 0.35), x, y, a)
            d.polygon([(px + s * 0.12, py + s * 0.16) for px, py in poly], fill=shade(color, 0.7) + (255,))
            d.polygon(poly, fill=col + (255,))
            hl = [(px * 0.55 + x * 0.45 - s * 0.12, py * 0.55 + y * 0.45 - s * 0.12) for px, py in poly]
            d.polygon(hl, fill=shade(col, 1.12) + (255,))
        elif shape == "slices":
            r = R * 0.1 * size
            d.ellipse((x - r + r * 0.08, y - r + r * 0.1, x + r + r * 0.08, y + r + r * 0.1), fill=shade(color, 0.7) + (255,))
            d.ellipse((x - r, y - r, x + r, y + r), fill=shade(col, 0.88) + (255,))
            ri = r * 0.8
            d.ellipse((x - ri, y - ri, x + ri, y + ri), fill=col + (255,))
            rc = r * 0.3
            d.ellipse((x - rc, y - rc, x + rc, y + rc), fill=shade(col, 1.15) + (255,))
        elif shape == "rounds":
            r = R * 0.045 * size
            d.ellipse((x - r + r * 0.15, y - r + r * 0.2, x + r + r * 0.15, y + r + r * 0.2), fill=shade(color, 0.65) + (255,))
            d.ellipse((x - r, y - r, x + r, y + r), fill=col + (255,))
            h = r * 0.35
            d.ellipse((x - r * 0.45 - h, y - r * 0.45 - h, x - r * 0.45 + h, y - r * 0.45 + h), fill=shade(col, 1.3) + (255,))
        elif shape in ("leaves", "herbs"):
            s = R * (0.075 if shape == "leaves" else 0.022) * size
            leaf = [(s * math.cos(t), s * 0.42 * math.sin(t) * (1 - 0.3 * math.cos(t)))
                    for t in (2 * math.pi * k / 24 for k in range(24))]
            poly = rotated(leaf, x, y, a)
            d.polygon(poly, fill=col + (255,))
            if shape == "leaves":
                d.line(rotated([(-s * 0.9, 0), (s * 0.85, 0)], x, y, a), fill=shade(col, 0.75) + (255,),
                       width=max(1, round(s * 0.07)))
        elif shape == "crumbs":
            r = R * rng.uniform(0.006, 0.013) * size
            d.ellipse((x - r, y - r, x + r, y + r), fill=col + (255,))


# ---------------------------------------------------------------------------
# Gefäße
# ---------------------------------------------------------------------------

def draw_scene(spec: dict):
    Image, ImageChops, ImageDraw, ImageFilter = _pillow()
    rng = random.Random(int(hashlib.sha256(spec.get("title", "").encode("utf-8")).hexdigest()[:12], 16))
    W, H = WIDTH * SUPER, HEIGHT * SUPER
    img = Image.new("RGBA", (W, H), BACKGROUND + (255,))
    vessel = spec.get("vessel", "plate")
    vcol = hex_rgb(spec["vessel_color"]) if spec.get("vessel_color") else VESSELS[vessel]
    cx, cy = W / 2, H / 2
    R = H * 0.36  # Kreisgefäße: Durchmesser 72 % der Höhe, passt auch in 16:9

    if vessel == "pan":
        cx -= W * 0.05

    # Serviette unten links, halb unter dem Gefäß
    napkin = Image.new("RGBA", (W, H), (0, 0, 0, 0))
    nd = ImageDraw.Draw(napkin)
    npoly = rotated(rounded_rect_poly(R * 1.5, R * 1.1, R * 0.04), cx - R * 0.95, cy + R * 0.55, math.radians(-8))
    nd.polygon(npoly, fill=NAPKIN + (255,))
    nd.line(rotated([(-R * 0.6, R * 0.42), (R * 0.6, R * 0.42)], cx - R * 0.95, cy + R * 0.55, math.radians(-8)),
            fill=shade(NAPKIN, 0.9) + (255,), width=round(R * 0.012))
    img.alpha_composite(napkin)

    # Schatten
    shadow = Image.new("RGBA", (W, H), (70, 55, 40, 0))
    sd = ImageDraw.Draw(shadow)
    if vessel == "casserole":
        rx, ry = R * 1.45, R * 0.98
        sd.rounded_rectangle((cx - rx + R * 0.05, cy - ry + R * 0.08, cx + rx + R * 0.05, cy + ry + R * 0.08),
                             radius=R * 0.25, fill=(70, 55, 40, 80))
    else:
        rs = R * (0.86 if vessel == "bowl" else 1.0)
        sd.ellipse((cx - rs + R * 0.04, cy - rs + R * 0.07, cx + rs + R * 0.04, cy + rs + R * 0.07), fill=(70, 55, 40, 80))
        if vessel == "pan":
            sd.rounded_rectangle((cx + R * 0.95, cy - R * 0.08 + R * 0.07, cx + R * 1.85, cy + R * 0.1 + R * 0.07),
                                 radius=R * 0.08, fill=(70, 55, 40, 80))
    img.alpha_composite(shadow.filter(ImageFilter.GaussianBlur(R * 0.05)))

    d = ImageDraw.Draw(img)
    rim = shade(vcol, 0.93) if vessel != "pan" else shade(vcol, 1.25)
    if vessel == "plate":
        d.ellipse((cx - R, cy - R, cx + R, cy + R), fill=vcol + (255,))
        ri = R * 0.76
        d.ellipse((cx - ri, cy - ri, cx + ri, cy + ri), fill=shade(vcol, 0.975) + (255,), outline=rim + (255,),
                  width=round(R * 0.012))
        area = Area("circle", cx, cy, R * 0.74, R * 0.74)
    elif vessel == "soup":
        d.ellipse((cx - R, cy - R, cx + R, cy + R), fill=vcol + (255,))
        ri = R * 0.66
        d.ellipse((cx - ri, cy - ri, cx + ri, cy + ri), fill=rim + (255,))
        area = Area("circle", cx, cy, R * 0.63, R * 0.63)
    elif vessel == "bowl":
        ro = R * 0.86
        d.ellipse((cx - ro, cy - ro, cx + ro, cy + ro), fill=vcol + (255,), outline=rim + (255,), width=round(R * 0.01))
        ri = ro * 0.86
        d.ellipse((cx - ri, cy - ri, cx + ri, cy + ri), fill=shade(vcol, 0.9) + (255,))
        area = Area("circle", cx, cy, ri * 0.97, ri * 0.97)
    elif vessel == "pan":
        d.rounded_rectangle((cx + R * 0.9, cy - R * 0.08, cx + R * 1.85, cy + R * 0.1), radius=R * 0.08, fill=shade(vcol, 0.8) + (255,))
        d.ellipse((cx - R, cy - R, cx + R, cy + R), fill=vcol + (255,))
        ri = R * 0.9
        d.ellipse((cx - ri, cy - ri, cx + ri, cy + ri), fill=shade(vcol, 0.85) + (255,), outline=rim + (255,), width=round(R * 0.015))
        area = Area("circle", cx, cy, ri * 0.97, ri * 0.97)
    elif vessel == "casserole":
        rx, ry = R * 1.45, R * 0.98
        for side in (-1, 1):  # Griffe
            hx = cx + side * (rx + R * 0.06)
            d.rounded_rectangle((hx - R * 0.14, cy - R * 0.28, hx + R * 0.14, cy + R * 0.28), radius=R * 0.1,
                                fill=shade(vcol, 0.94) + (255,), outline=rim + (255,), width=round(R * 0.01))
        d.rounded_rectangle((cx - rx, cy - ry, cx + rx, cy + ry), radius=R * 0.25, fill=vcol + (255,),
                            outline=rim + (255,), width=round(R * 0.012))
        inset = R * 0.13
        d.rounded_rectangle((cx - rx + inset, cy - ry + inset, cx + rx - inset, cy + ry - inset), radius=R * 0.15,
                            fill=shade(vcol, 0.9) + (255,))
        area = Area("rect", cx, cy, rx - inset * 1.05, ry - inset * 1.05, R * 0.14)
    else:
        raise ValueError(f"Gefäß {vessel!r} unbekannt")

    food = Image.new("RGBA", (W, H), (0, 0, 0, 0))
    for ing in spec.get("ingredients", []):
        draw_ingredient(food, ing, area, rng, Image, ImageChops, ImageDraw, ImageFilter)
    amask = area.mask(Image, ImageDraw, (W, H))
    img.alpha_composite(clip_to(food, amask, ImageChops))

    # leichter Innenschatten am Rand gibt dem Gefäß Tiefe
    inner = Image.new("RGBA", (W, H), (60, 45, 30, 0))
    box = (area.cx - area.rx, area.cy - area.ry, area.cx + area.rx, area.cy + area.ry)
    ring = ImageDraw.Draw(inner)
    if area.kind == "circle":
        ring.ellipse(box, outline=(60, 45, 30, 90), width=round(R * 0.05))
    else:
        ring.rounded_rectangle(box, radius=area.corner, outline=(60, 45, 30, 90), width=round(R * 0.05))
    inner = inner.filter(ImageFilter.GaussianBlur(R * 0.03))
    img.alpha_composite(clip_to(inner, amask, ImageChops))

    return img.convert("RGB").resize((WIDTH, HEIGHT), Image.Resampling.LANCZOS)


def check_spec(spec) -> list[str]:
    errors = []
    if not isinstance(spec, dict):
        return ["Beschreibung muss ein JSON-Objekt sein"]
    if spec.get("vessel", "plate") not in VESSELS:
        errors.append(f"vessel: eines von {', '.join(VESSELS)}")
    ings = spec.get("ingredients")
    if not isinstance(ings, list) or not ings:
        errors.append("ingredients: mindestens eine Zutat")
        return errors
    for i, ing in enumerate(ings):
        where = f"ingredients[{i}]"
        if not isinstance(ing, dict):
            errors.append(f"{where}: muss ein Objekt sein")
            continue
        if ing.get("shape") not in SHAPES:
            errors.append(f"{where}.shape: eines von {', '.join(sorted(SHAPES))}")
        try:
            hex_rgb(str(ing.get("color", "")))
        except ValueError:
            errors.append(f"{where}.color: Farbe als #RRGGBB")
        if ing.get("amount", "some") not in AMOUNTS:
            errors.append(f"{where}.amount: little, some oder much")
        if ing.get("size", "medium") not in SIZES:
            errors.append(f"{where}.size: small, medium oder large")
    if spec.get("vessel_color"):
        try:
            hex_rgb(spec["vessel_color"])
        except ValueError:
            errors.append("vessel_color: Farbe als #RRGGBB")
    return errors


def render(spec: dict) -> bytes:
    """Illustration als JPEG-Bytes (ohne Metadaten)."""
    errors = check_spec(spec)
    if errors:
        raise ValueError("; ".join(errors))
    out = io.BytesIO()
    draw_scene(spec).save(out, "JPEG", quality=JPEG_QUALITY, optimize=True)
    return out.getvalue()


def main(argv=None) -> int:
    parser = argparse.ArgumentParser(description="Illustration eines Gerichts zeichnen")
    parser.add_argument("--spec", required=True, help="Beschreibung des Gerichts (JSON-Datei)")
    parser.add_argument("--out", required=True, help="Zieldatei (.jpg)")
    args = parser.parse_args(argv)
    spec = json.loads(Path(args.spec).read_text(encoding="utf-8"))
    try:
        data = render(spec)
    except ValueError as e:
        print(f"FEHLER: {e}")
        return 1
    Path(args.out).write_bytes(data)
    print(f"OK: {args.out} ({len(data) / 1024:.0f} KB)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
