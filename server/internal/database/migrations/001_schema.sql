-- Grundschema für Phase 1: Rezepte, Kategorien, Einstellungen, Löschprotokoll.
-- Siehe docs/PLANUNG.md, Abschnitt 4.
--
-- Konventionen:
-- * Zeitstempel als Text in UTC mit Millisekunden und fester Länge
--   (z. B. 2026-10-19T18:30:00.000Z), damit Textvergleich = Zeitvergleich.
-- * Wahrheitswerte als INTEGER 0/1.
-- * Optionale Texte sind leer (''), nicht NULL. NULL gibt es nur bei
--   optionalen Zahlen, wo "leer" etwas anderes bedeutet als 0.
-- * STRICT lässt SQLite die Spaltentypen streng prüfen.

CREATE TABLE recipes (
    id            INTEGER PRIMARY KEY,
    title         TEXT    NOT NULL,
    description   TEXT    NOT NULL DEFAULT '',
    base_servings INTEGER NOT NULL CHECK (base_servings > 0),
    prep_minutes  INTEGER CHECK (prep_minutes >= 0),
    total_minutes INTEGER CHECK (total_minutes >= 0),
    source_url    TEXT    NOT NULL DEFAULT '',
    image         TEXT    NOT NULL DEFAULT '',
    thumbnail     TEXT    NOT NULL DEFAULT '',
    is_favorite   INTEGER NOT NULL DEFAULT 0 CHECK (is_favorite IN (0, 1)),
    notes         TEXT    NOT NULL DEFAULT '',
    created_at    TEXT    NOT NULL,
    updated_at    TEXT    NOT NULL
) STRICT;

-- Dublettenprüfung beim URL-Import (nur Rezepte mit Quell-URL).
CREATE INDEX recipes_source_url ON recipes (source_url) WHERE source_url <> '';
-- Abgleich mit der App: "alles seit Zeitpunkt X".
CREATE INDEX recipes_updated_at ON recipes (updated_at);

CREATE TABLE recipe_ingredients (
    id                INTEGER PRIMARY KEY,
    recipe_id         INTEGER NOT NULL REFERENCES recipes (id) ON DELETE CASCADE,
    position          INTEGER NOT NULL,
    group_name        TEXT    NOT NULL DEFAULT '',
    quantity          REAL    CHECK (quantity >= 0), -- NULL = nicht umrechnen
    unit              TEXT    NOT NULL DEFAULT '',
    name              TEXT    NOT NULL,
    note              TEXT    NOT NULL DEFAULT '',
    shopping_relevant INTEGER NOT NULL DEFAULT 1 CHECK (shopping_relevant IN (0, 1)),
    updated_at        TEXT    NOT NULL
) STRICT;

CREATE INDEX recipe_ingredients_recipe ON recipe_ingredients (recipe_id, position);

CREATE TABLE recipe_steps (
    id         INTEGER PRIMARY KEY,
    recipe_id  INTEGER NOT NULL REFERENCES recipes (id) ON DELETE CASCADE,
    position   INTEGER NOT NULL,
    text       TEXT    NOT NULL,
    updated_at TEXT    NOT NULL
) STRICT;

CREATE INDEX recipe_steps_recipe ON recipe_steps (recipe_id, position);

CREATE TABLE tags (
    id         INTEGER PRIMARY KEY,
    name       TEXT    NOT NULL,
    tag_group  TEXT    NOT NULL,
    position   INTEGER NOT NULL,
    updated_at TEXT    NOT NULL,
    UNIQUE (tag_group, name)
) STRICT;

CREATE TABLE recipe_tags (
    recipe_id  INTEGER NOT NULL REFERENCES recipes (id) ON DELETE CASCADE,
    tag_id     INTEGER NOT NULL REFERENCES tags (id) ON DELETE CASCADE,
    updated_at TEXT    NOT NULL,
    PRIMARY KEY (recipe_id, tag_id)
) STRICT;

-- Für den Kategorie-Filter: "alle Rezepte mit Kategorie X".
CREATE INDEX recipe_tags_tag ON recipe_tags (tag_id);

-- Gelöschte Datensätze, damit die App beim Abgleich weiß, was weg muss.
CREATE TABLE deletions (
    id         INTEGER PRIMARY KEY,
    table_name TEXT    NOT NULL,
    record_id  INTEGER NOT NULL,
    deleted_at TEXT    NOT NULL
) STRICT;

CREATE INDEX deletions_deleted_at ON deletions (deleted_at);

-- Schlüssel-Wert-Speicher (z. B. family_size, shopping_day, bring_entity).
CREATE TABLE settings (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL,
    updated_at TEXT NOT NULL
) STRICT;
