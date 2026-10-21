"""Konstanten der Rezeptplaner-Integration."""

from datetime import timedelta

DOMAIN = "rezeptplaner"

# Schlüssel in den Daten des Config-Entrys (kommen aus der Discovery der App)
CONF_HOST = "host"
CONF_PORT = "port"
CONF_TOKEN = "token"
CONF_SLUG = "slug"

# Der Plan ändert sich selten; alle 15 Minuten eine kleine Anfrage genügt.
UPDATE_INTERVAL = timedelta(minutes=15)

# Unter dieser Adresse liefert Home Assistant die Dashboard-Karte aus.
CARD_URL = "/rezeptplaner_static/rezeptplaner-card.js"
