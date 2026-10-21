"""Regelmäßiger Abruf des aktuellen Einkaufszeitraums."""

from __future__ import annotations

import logging
from typing import Any

from homeassistant.config_entries import ConfigEntry
from homeassistant.core import HomeAssistant
from homeassistant.helpers.update_coordinator import DataUpdateCoordinator, UpdateFailed

from .api import RezeptplanerClient, RezeptplanerError
from .const import DOMAIN, UPDATE_INTERVAL

_LOGGER = logging.getLogger(__name__)


class RezeptplanerCoordinator(DataUpdateCoordinator[dict[str, Any]]):
    """Hält den Essensplan des aktuellen Einkaufszeitraums für alle Entitäten."""

    def __init__(self, hass: HomeAssistant, entry: ConfigEntry, client: RezeptplanerClient) -> None:
        super().__init__(
            hass,
            _LOGGER,
            config_entry=entry,
            name=DOMAIN,
            update_interval=UPDATE_INTERVAL,
        )
        self.client = client

    async def _async_update_data(self) -> dict[str, Any]:
        try:
            return await self.client.meal_plan()
        except RezeptplanerError as err:
            raise UpdateFailed(str(err)) from err

    def entries_on(self, date: str) -> list[dict[str, Any]]:
        """Einträge eines Tages, Mittagessen vor Abendessen (so liefert sie der Server)."""
        return [e for e in (self.data or {}).get("entries", []) if e["date"] == date]


def entry_title(entry: dict[str, Any]) -> str:
    """Rezepttitel oder Notiz eines Planeintrags."""
    recipe = entry.get("recipe")
    return recipe["title"] if recipe else entry.get("note", "")
