-- API-Schlüssel für den App-Zugang (Android-App). Siehe docs/PLANUNG.md,
-- Abschnitte 2 und 4.
--
-- Gespeichert wird nur der SHA-256-Hash des Schlüssels. Wer die Datenbank
-- liest (z. B. aus einem Backup), kann damit keine Anfrage stellen.

CREATE TABLE api_keys (
    id           INTEGER PRIMARY KEY,
    name         TEXT    NOT NULL CHECK (name <> ''),
    key_hash     TEXT    NOT NULL UNIQUE,
    created_at   TEXT    NOT NULL,
    last_used_at TEXT
) STRICT;
