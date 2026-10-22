"""Rezeptplaner: Essensplan als Kalender und Sensor, dazu die Dashboard-Karte.

Die Integration verbindet sich mit der Rezeptplaner-App im Supervisor. Die
App meldet sich selbst an (Discovery); Adresse und Schlüssel stehen danach
im Config-Entry.
"""

from __future__ import annotations

import hashlib
from dataclasses import dataclass
from pathlib import Path

from homeassistant.components.frontend import add_extra_js_url
from homeassistant.components.http import StaticPathConfig
from homeassistant.config_entries import ConfigEntry
from homeassistant.const import Platform
from homeassistant.core import HomeAssistant
from homeassistant.helpers import config_validation as cv
from homeassistant.helpers.aiohttp_client import async_get_clientsession
from homeassistant.helpers.typing import ConfigType
from homeassistant.loader import async_get_integration

from .api import RezeptplanerClient
from .const import CARD_URL, CONF_HOST, CONF_PORT, CONF_SLUG, CONF_TOKEN, DOMAIN
from .coordinator import RezeptplanerCoordinator
from .views import ImageView, PlanView, RecipeView

PLATFORMS = [Platform.CALENDAR, Platform.SENSOR]

# Keine Einstellungen in configuration.yaml, nur über die Oberfläche.
CONFIG_SCHEMA = cv.config_entry_only_config_schema(DOMAIN)


@dataclass
class RuntimeData:
    """Was die Integration zur Laufzeit braucht."""

    client: RezeptplanerClient
    coordinator: RezeptplanerCoordinator
    slug: str  # App-ID, z. B. "a1b2c3d4_rezeptplaner", für den Link zur Weboberfläche


# ConfigEntry[RuntimeData] sagt dem Typprüfer, was in entry.runtime_data steckt.
type RezeptplanerConfigEntry = ConfigEntry[RuntimeData]


async def async_setup(hass: HomeAssistant, config: ConfigType) -> bool:
    """Einmal beim Start: Dashboard-Karte und Endpunkte für die Karte anmelden."""
    integration = await async_get_integration(hass, DOMAIN)
    card = Path(__file__).parent / "frontend" / "rezeptplaner-card.js"
    card_hash = await hass.async_add_executor_job(_file_hash, card)
    # cache_headers=True: Home Assistant schickt ein langes Cache-Control mit.
    # Das ist sicher, weil sich die Adresse mit jedem Inhalt ändert (siehe
    # unten). Ohne eigenen Header setzt ein vorgeschalteter Cloudflare-Tunnel
    # seine Vorgabe (4 Stunden) ein.
    await hass.http.async_register_static_paths([StaticPathConfig(CARD_URL, str(card), True)])
    # Lädt die Karte in jedem Dashboard. "?v=" aus Version und Inhalt sorgt
    # dafür, dass Browser nach jeder Änderung die neue Datei holen.
    add_extra_js_url(hass, f"{CARD_URL}?v={integration.version}-{card_hash}")
    hass.http.register_view(PlanView())
    hass.http.register_view(RecipeView())
    hass.http.register_view(ImageView())
    return True


async def async_setup_entry(hass: HomeAssistant, entry: RezeptplanerConfigEntry) -> bool:
    client = RezeptplanerClient(
        async_get_clientsession(hass),
        entry.data[CONF_HOST],
        entry.data[CONF_PORT],
        entry.data[CONF_TOKEN],
    )
    coordinator = RezeptplanerCoordinator(hass, entry, client)
    entry.runtime_data = RuntimeData(client, coordinator, entry.data[CONF_SLUG])
    # Die Endpunkte der Karte finden die Verbindung hier – schon vor dem
    # ersten Abruf, damit die Karte nicht auf den Coordinator warten muss.
    hass.data[DOMAIN] = entry.runtime_data
    try:
        await coordinator.async_config_entry_first_refresh()
    except Exception:
        hass.data.pop(DOMAIN, None)
        raise

    await hass.config_entries.async_forward_entry_setups(entry, PLATFORMS)
    return True


async def async_unload_entry(hass: HomeAssistant, entry: RezeptplanerConfigEntry) -> bool:
    unloaded = await hass.config_entries.async_unload_platforms(entry, PLATFORMS)
    if unloaded:
        hass.data.pop(DOMAIN, None)
    return unloaded


def _file_hash(path: Path) -> str:
    """Kurze Prüfsumme des Dateiinhalts für den Versionsparameter."""
    return hashlib.sha256(path.read_bytes()).hexdigest()[:8]
