"""HTTP-Endpunkte für die Dashboard-Karte.

Die Karte läuft im Browser und erreicht den Rezeptplaner-Server nicht
direkt. Diese Endpunkte reichen Plan, Rezepte und Bilder durch und
verlangen dabei eine Anmeldung bei Home Assistant (Bilder über signierte
Adressen, siehe auth/sign_path).
"""

from __future__ import annotations

import re
from typing import TYPE_CHECKING

from aiohttp import web

from homeassistant.components.http import HomeAssistantView
from homeassistant.helpers.http import KEY_HASS

from .api import RezeptplanerError
from .const import DOMAIN

if TYPE_CHECKING:
    from . import RuntimeData

# Bildnamen vergibt der Server (z. B. "12-ab34cd.jpg"); alles andere wird
# abgelehnt, damit niemand über den Namen andere Pfade ansprechen kann.
IMAGE_NAME = re.compile(r"^[A-Za-z0-9][A-Za-z0-9._-]*$")


def _runtime(request: web.Request) -> RuntimeData | None:
    return request.app[KEY_HASS].data.get(DOMAIN)


def _unavailable() -> web.Response:
    return web.json_response({"error": "rezeptplaner nicht verbunden"}, status=503)


class PlanView(HomeAssistantView):
    """Aktueller Einkaufszeitraum, immer frisch vom Server."""

    url = "/api/rezeptplaner/plan"
    name = "api:rezeptplaner:plan"

    async def get(self, request: web.Request) -> web.Response:
        rt = _runtime(request)
        if rt is None:
            return _unavailable()
        try:
            data = await rt.client.meal_plan()
        except RezeptplanerError as err:
            return web.json_response({"error": str(err)}, status=502)
        # Link zur Weboberfläche der App (Seitenleiste über Ingress)
        data["app_url"] = f"/hassio/ingress/{rt.slug}"
        return self.json(data)


class RecipeView(HomeAssistantView):
    """Ein Rezept für die Kochansicht der Karte."""

    url = "/api/rezeptplaner/recipes/{recipe_id}"
    name = "api:rezeptplaner:recipe"

    async def get(self, request: web.Request, recipe_id: str) -> web.Response:
        rt = _runtime(request)
        if rt is None:
            return _unavailable()
        if not recipe_id.isdigit():
            return web.json_response({"error": "ungültige rezept-id"}, status=400)
        try:
            return self.json(await rt.client.recipe(int(recipe_id)))
        except RezeptplanerError as err:
            return web.json_response({"error": str(err)}, status=502)


class ImageView(HomeAssistantView):
    """Bilder und Vorschaubilder der Rezepte."""

    url = "/api/rezeptplaner/images/{name}"
    name = "api:rezeptplaner:image"

    async def get(self, request: web.Request, name: str) -> web.Response:
        rt = _runtime(request)
        if rt is None:
            return _unavailable()
        if not IMAGE_NAME.match(name):
            return web.Response(status=400)
        try:
            body, content_type = await rt.client.image(name)
        except RezeptplanerError:
            return web.Response(status=404)
        # Bilddateien ändern sich nie (neues Bild = neuer Name).
        return web.Response(
            body=body,
            content_type=content_type,
            headers={"Cache-Control": "private, max-age=604800, immutable"},
        )
