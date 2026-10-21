"""Einrichtung der Integration: nur über die Discovery der Rezeptplaner-App."""

from __future__ import annotations

from typing import Any

from homeassistant.config_entries import ConfigFlow, ConfigFlowResult
from homeassistant.helpers.service_info.hassio import HassioServiceInfo

from .const import CONF_HOST, CONF_PORT, CONF_SLUG, CONF_TOKEN, DOMAIN


class RezeptplanerConfigFlow(ConfigFlow, domain=DOMAIN):
    """Die App meldet Adresse und Schlüssel; der Nutzer bestätigt nur."""

    VERSION = 1

    def __init__(self) -> None:
        self._data: dict[str, Any] = {}

    async def async_step_hassio(self, discovery_info: HassioServiceInfo) -> ConfigFlowResult:
        """Meldung der App über den Supervisor."""
        config = discovery_info.config
        self._data = {
            CONF_HOST: config[CONF_HOST],
            CONF_PORT: config[CONF_PORT],
            CONF_TOKEN: config[CONF_TOKEN],
            CONF_SLUG: discovery_info.slug,
        }
        # Es gibt nur eine Rezeptplaner-App. Ist die Integration schon
        # eingerichtet, werden nur Adresse und Schlüssel aktualisiert (z. B.
        # nach einer Neuinstallation der App) und die Integration neu geladen.
        await self.async_set_unique_id(DOMAIN)
        self._abort_if_unique_id_configured(updates=self._data)
        return await self.async_step_hassio_confirm()

    async def async_step_hassio_confirm(self, user_input: dict[str, Any] | None = None) -> ConfigFlowResult:
        """Bestätigung durch den Nutzer."""
        if user_input is not None:
            return self.async_create_entry(title="Rezeptplaner", data=self._data)
        return self.async_show_form(step_id="hassio_confirm")

    async def async_step_user(self, user_input: dict[str, Any] | None = None) -> ConfigFlowResult:
        """Von Hand geht es nicht: Adresse und Schlüssel kennt nur die App."""
        return self.async_abort(reason="discovery_only")
