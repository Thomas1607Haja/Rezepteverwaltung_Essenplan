"""Zugriff auf die REST-API des Rezeptplaner-Servers (App im Supervisor)."""

from __future__ import annotations

from typing import Any

import aiohttp


class RezeptplanerError(Exception):
    """Der Server ist nicht erreichbar oder antwortet mit einem Fehler."""


class RezeptplanerClient:
    """Liest Essensplan, Rezepte und Bilder; die Integration schreibt nie."""

    def __init__(self, session: aiohttp.ClientSession, host: str, port: int, token: str) -> None:
        self._session = session
        self._base = f"http://{host}:{port}/api/v1"
        self._headers = {"Authorization": f"Bearer {token}"}

    async def _get(self, path: str, params: dict[str, str] | None = None) -> aiohttp.ClientResponse:
        try:
            resp = await self._session.get(
                self._base + path,
                params=params,
                headers=self._headers,
                timeout=aiohttp.ClientTimeout(total=15),
            )
        except (aiohttp.ClientError, TimeoutError) as err:
            raise RezeptplanerError(f"Server nicht erreichbar: {err}") from err
        if resp.status != 200:
            text = await resp.text()
            resp.release()
            raise RezeptplanerError(f"{path}: Status {resp.status}: {text[:200]}")
        return resp

    async def _get_json(self, path: str, params: dict[str, str] | None = None) -> Any:
        resp = await self._get(path, params)
        async with resp:
            return await resp.json()

    async def meal_plan(self, start: str | None = None, end: str | None = None) -> dict[str, Any]:
        """Einträge von start bis end ("2026-10-24"); ohne Angabe der aktuelle Einkaufszeitraum.

        Ergebnis wie GET /api/v1/meal-plan: {"entries", "today", "shopping_period"}.
        """
        params = {"from": start, "to": end} if start and end else None
        return await self._get_json("/meal-plan", params)

    async def recipe(self, recipe_id: int) -> dict[str, Any]:
        """Ein vollständiges Rezept mit Zutaten und Schritten."""
        return await self._get_json(f"/recipes/{recipe_id}")

    async def image(self, name: str) -> tuple[bytes, str]:
        """Bilddatei und ihr Content-Type."""
        resp = await self._get(f"/images/{name}")
        async with resp:
            return await resp.read(), resp.content_type
