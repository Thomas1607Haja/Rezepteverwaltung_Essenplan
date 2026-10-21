"""Kalender „Essensplan“: jedes Gericht als ganztägiger Termin."""

from __future__ import annotations

from datetime import date, datetime, timedelta
from typing import Any

from homeassistant.components.calendar import CalendarEntity, CalendarEvent
from homeassistant.core import HomeAssistant
from homeassistant.exceptions import HomeAssistantError
from homeassistant.helpers.entity_platform import AddConfigEntryEntitiesCallback

from . import RezeptplanerConfigEntry
from .api import RezeptplanerError
from .coordinator import RezeptplanerCoordinator, entry_title
from .entity import RezeptplanerEntity

# Grenze des Servers für einen Abruf (gut ein Jahr)
MAX_DAYS = 400


async def async_setup_entry(
    hass: HomeAssistant,
    entry: RezeptplanerConfigEntry,
    async_add_entities: AddConfigEntryEntitiesCallback,
) -> None:
    async_add_entities([MealPlanCalendar(entry.runtime_data.coordinator)])


def to_event(entry: dict[str, Any]) -> CalendarEvent:
    """Planeintrag als ganztägiger Termin; Mittagessen mit Hinweis im Titel."""
    day = date.fromisoformat(entry["date"])
    summary = entry_title(entry)
    if entry.get("slot") == "lunch":
        summary = f"Mittag: {summary}"
    return CalendarEvent(
        start=day,
        end=day + timedelta(days=1),
        summary=summary,
        description=f"{entry['servings']} Portionen",
        uid=str(entry["id"]),
    )


class MealPlanCalendar(RezeptplanerEntity, CalendarEntity):
    """Der Essensplan als Kalender."""

    def __init__(self, coordinator: RezeptplanerCoordinator) -> None:
        super().__init__(coordinator, "meal_plan")

    @property
    def event(self) -> CalendarEvent | None:
        """Das heutige oder nächste geplante Gericht."""
        data = self.coordinator.data or {}
        today = data.get("today", "")
        upcoming = [e for e in data.get("entries", []) if e["date"] >= today]
        return to_event(upcoming[0]) if upcoming else None

    async def async_get_events(self, hass: HomeAssistant, start_date: datetime, end_date: datetime) -> list[CalendarEvent]:
        """Termine eines Zeitraums, direkt vom Server (die Kalenderansicht blättert frei)."""
        first = start_date.date()
        # end_date gehört nicht mehr dazu (z. B. Mitternacht des Folgetags).
        last = (end_date - timedelta(microseconds=1)).date()
        last = min(last, first + timedelta(days=MAX_DAYS))
        try:
            data = await self.coordinator.client.meal_plan(first.isoformat(), last.isoformat())
        except RezeptplanerError as err:
            raise HomeAssistantError(str(err)) from err
        return [to_event(e) for e in data.get("entries", [])]
