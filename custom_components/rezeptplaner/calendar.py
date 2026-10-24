"""Kalender „Essensplan“: jedes Gericht als Termin zur Uhrzeit seiner Mahlzeit.

Die Uhrzeiten (Mittag, Kaffee, Abend) stellt man in den Einstellungen des
Rezeptplaners ein; ältere Server ohne Uhrzeiten liefern ganztägige Termine."""

from __future__ import annotations

from datetime import date, datetime, time, timedelta
from typing import Any

from homeassistant.components.calendar import CalendarEntity, CalendarEvent
from homeassistant.core import HomeAssistant
from homeassistant.exceptions import HomeAssistantError
from homeassistant.helpers.entity_platform import AddConfigEntryEntitiesCallback
from homeassistant.util import dt as dt_util

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


# Vorsatz im Titel je Mahlzeit; Abendessen ohne.
SLOT_PREFIX = {"lunch": "Mittag: ", "coffee": "Kaffee: "}


def amount_text(entry: dict[str, Any]) -> str:
    """„4 Portionen“ bzw. mit Ergebnis-Einheit „1 Springform 26 cm“."""
    unit = (entry.get("recipe") or {}).get("yield_unit") or ""
    servings = entry.get("servings", 0)
    if unit:
        return f"{servings} {unit}"
    return "1 Portion" if servings == 1 else f"{servings} Portionen"


def to_event(entry: dict[str, Any], slot_times: dict[str, str] | None = None) -> CalendarEvent:
    """Planeintrag als Termin: zur Uhrzeit seiner Mahlzeit (eine Stunde),
    ohne Uhrzeiten ganztägig. Mittag und Kaffee mit Hinweis im Titel."""
    day = date.fromisoformat(entry["date"])
    slot = entry.get("slot", "dinner")
    summary = SLOT_PREFIX.get(slot, "") + entry_title(entry)
    description = amount_text(entry) if entry.get("recipe") else ""
    clock = (slot_times or {}).get(slot)
    if clock:
        hour, minute = (int(x) for x in clock.split(":"))
        start = datetime.combine(day, time(hour, minute), tzinfo=dt_util.get_default_time_zone())
        return CalendarEvent(start=start, end=start + timedelta(hours=1), summary=summary,
                             description=description, uid=str(entry["id"]))
    return CalendarEvent(start=day, end=day + timedelta(days=1), summary=summary,
                         description=description, uid=str(entry["id"]))


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
        return to_event(upcoming[0], data.get("slot_times")) if upcoming else None

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
        return [to_event(e, data.get("slot_times")) for e in data.get("entries", [])]
