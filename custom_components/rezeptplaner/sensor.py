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
        dinner = [entry_title(e) for e in self._today() if e.get("slot") != "lunch"]
        # Zustände dürfen höchstens 255 Zeichen lang sein.
        return ", ".join(dinner)[:255] if dinner else None

    @property
    def extra_state_attributes(self) -> dict[str, Any]:
        entries = self._today()
        return {
            "lunch": [entry_title(e) for e in entries if e.get("slot") == "lunch"],
            "dishes": [
                {
                    "title": entry_title(e),
                    "slot": e.get("slot"),
                    "servings": e.get("servings"),
                    "status": e.get("status"),
                }
                for e in entries
            ],
        }
