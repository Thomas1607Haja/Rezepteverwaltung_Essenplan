-- Vorratsliste für Phase 5. Siehe docs/PLANUNG.md, Abschnitte 3 und 4.
--
-- Zutaten, deren Name hier (oder unter den Synonymen) steht, werden beim
-- Anlegen und Importieren als nicht einkaufsrelevant vorgeschlagen.

CREATE TABLE pantry_items (
    id         INTEGER PRIMARY KEY,
    name       TEXT    NOT NULL CHECK (name <> ''),
    -- Kommagetrennt, z. B. "Olivenöl, Rapsöl"; leer = keine Synonyme
    synonyms   TEXT    NOT NULL DEFAULT '',
    updated_at TEXT    NOT NULL
) STRICT;

-- Startbestand: Was praktisch jeder Haushalt immer da hat. Lässt sich in
-- den Einstellungen ändern.
INSERT INTO pantry_items (name, synonyms, updated_at)
SELECT column1, column2, strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
FROM (
    VALUES
    ('Salz', 'Meersalz, Jodsalz'),
    ('Pfeffer', 'schwarzer Pfeffer, Pfeffer aus der Mühle'),
    ('Öl', 'Olivenöl, Rapsöl, Sonnenblumenöl, Pflanzenöl, Speiseöl'),
    ('Zucker', ''),
    ('Wasser', '')
);
