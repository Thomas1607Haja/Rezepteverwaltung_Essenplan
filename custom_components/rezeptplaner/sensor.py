"""Sensor „Heute“: das heutige Abendessen, z. B. für Sprachausgaben."""

from __future__ import annotations

from typing import Any

from homeassistant.components.sensor import SensorEntity
from homeassistant.core import HomeAssistant
from homeassistant.helpers.entity_platform import AddConfigEntryEntitiesCallback

from . import RezeptplanerConfigEntry
from .coordinator import RezeptplanerCoordinator, entry_title
from .entity import RezeptplanerEntity


async def async_setup_entry(
    hass: HomeAssistant,
    entry: RezeptplanerConfigEntry,
    async_add_entities: AddConfigEntryEntitiesCallback,
) -> None:
    async_add_entities([TodaySensor(entry.runtime_data.coordinator)])


class TodaySensor(RezeptplanerEntity, SensorEntity):
    """Zustand: heutige Abendgerichte, durch Komma getrennt; leer = nichts geplant."""

    def __init__(self, coordinator: RezeptplanerCoordinator) -> None:
        super().__init__(coordinator, "today")

    def _today(self) -> list[dict[str, Any]]:
        return self.coordinator.entries_on((self.coordinator.data or {}).get("today", ""))

    @property
    def native_value(self) -> str | None:
        # Ein Menü zählt einmal, mit seinem Titel (Rezeptplaner 0.16).
        menus = {m["id"]: m for m in (self.coordinator.data or {}).get("menus", [])}
        dinner = []
        for e in self._today():
            if e.get("slot", "dinner") != "dinner":
                continue
            menu = menus.get(e.get("menu_id"))
            title = f"Menü: {menu.get('display_title')}" if menu else entry_title(e)
            if title not in dinner:
                dinner.append(title)
        # Zustände dürfen höchstens 255 Zeichen lang sein.
        return ", ".join(dinner)[:255] if dinner else None

    @property
    def extra_state_attributes(self) -> dict[str, Any]:
        entries = self._today()
        return {
            "lunch": [entry_title(e) for e in entries if e.get("slot") == "lunch"],
            "coffee": [entry_title(e) for e in entries if e.get("slot") == "coffee"],
            "dishes": [
                {
                    "title": entry_title(e),
                    "slot": e.get("slot"),
                    "servings": e.get("servings"),
                    "status": e.get("status"),
                    "menu_id": e.get("menu_id"),
                }
                for e in entries
            ],
        }
