-- IDs nie wieder vergeben. Siehe docs/PLANUNG.md, Abschnitte 4 und 11.
--
-- Ohne AUTOINCREMENT vergibt SQLite nach dem Löschen des neuesten
-- Datensatzes dessen ID erneut. Beim Abgleich mit der App träfen dann
-- „Rezept 12 gelöscht“ (deletions) und „Rezept 12 neu“ aufeinander.
-- Betroffen sind die Tabellen, die die App einzeln abgleicht: recipes,
-- meal_plan_entries, pantry_items und tags. Zutaten und Schritte kommen
-- immer zusammen mit ihrem Rezept und brauchen das nicht.
--
-- SQLite kann AUTOINCREMENT nicht nachträglich setzen; die Tabellen werden
-- nach dem Verfahren aus der SQLite-Doku neu angelegt, umkopiert und
-- umbenannt. Das geht nur mit abgeschalteten Fremdschlüsseln, sonst
-- löscht DROP TABLE per ON DELETE CASCADE die Zutaten mit (siehe
-- applyMigration). Spalten, Prüfungen, Indizes und Trigger bleiben gleich.

-- Der Trigger verweist auf meal_plan_entries; ohne ihn lässt sich die
-- Tabelle gefahrlos umbenennen. Er wird am Ende neu angelegt.
DROP TRIGGER meal_plan_keep_title;

-- recipes ------------------------------------------------------------------

CREATE TABLE new_recipes (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    title            TEXT    NOT NULL,
    description      TEXT    NOT NULL DEFAULT '',
    base_servings    INTEGER NOT NULL CHECK (base_servings > 0),
    prep_minutes     INTEGER CHECK (prep_minutes >= 0),
    total_minutes    INTEGER CHECK (total_minutes >= 0),
    source_url       TEXT    NOT NULL DEFAULT '',
    image            TEXT    NOT NULL DEFAULT '',
    thumbnail        TEXT    NOT NULL DEFAULT '',
    is_favorite      INTEGER NOT NULL DEFAULT 0 CHECK (is_favorite IN (0, 1)),
    notes            TEXT    NOT NULL DEFAULT '',
    created_at       TEXT    NOT NULL,
    updated_at       TEXT    NOT NULL,
    parent_recipe_id INTEGER REFERENCES recipes (id) ON DELETE SET NULL
) STRICT;

INSERT INTO new_recipes (id, title, description, base_servings, prep_minutes,
    total_minutes, source_url, image, thumbnail, is_favorite, notes,
    created_at, updated_at, parent_recipe_id)
SELECT id, title, description, base_servings, prep_minutes,
    total_minutes, source_url, image, thumbnail, is_favorite, notes,
    created_at, updated_at, parent_recipe_id
FROM recipes;

DROP TABLE recipes;
ALTER TABLE new_recipes RENAME TO recipes;

CREATE INDEX recipes_source_url ON recipes (source_url) WHERE source_url <> '';
CREATE INDEX recipes_updated_at ON recipes (updated_at);
CREATE INDEX recipes_parent ON recipes (parent_recipe_id) WHERE parent_recipe_id IS NOT NULL;

-- meal_plan_entries --------------------------------------------------------

CREATE TABLE new_meal_plan_entries (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    date           TEXT    NOT NULL CHECK (date GLOB '[0-9][0-9][0-9][0-9]-[0-1][0-9]-[0-3][0-9]'),
    slot           TEXT    NOT NULL DEFAULT 'dinner' CHECK (slot IN ('dinner', 'lunch')),
    position       INTEGER NOT NULL DEFAULT 0,
    recipe_id      INTEGER REFERENCES recipes (id) ON DELETE SET NULL,
    servings       INTEGER NOT NULL CHECK (servings >= 1),
    status         TEXT    NOT NULL DEFAULT 'planned'
                           CHECK (status IN ('planned', 'transferred', 'have_ingredients')),
    transferred_at TEXT,
    note           TEXT    NOT NULL DEFAULT '',
    created_at     TEXT    NOT NULL,
    updated_at     TEXT    NOT NULL,
    CHECK (recipe_id IS NOT NULL OR note <> '')
) STRICT;

INSERT INTO new_meal_plan_entries (id, date, slot, position, recipe_id,
    servings, status, transferred_at, note, created_at, updated_at)
SELECT id, date, slot, position, recipe_id,
    servings, status, transferred_at, note, created_at, updated_at
FROM meal_plan_entries;

DROP TABLE meal_plan_entries;
ALTER TABLE new_meal_plan_entries RENAME TO meal_plan_entries;

CREATE INDEX meal_plan_entries_date ON meal_plan_entries (date, slot, position);
CREATE INDEX meal_plan_entries_recipe ON meal_plan_entries (recipe_id, date);

-- pantry_items -------------------------------------------------------------

CREATE TABLE new_pantry_items (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT    NOT NULL CHECK (name <> ''),
    synonyms   TEXT    NOT NULL DEFAULT '',
    updated_at TEXT    NOT NULL
) STRICT;

INSERT INTO new_pantry_items (id, name, synonyms, updated_at)
SELECT id, name, synonyms, updated_at FROM pantry_items;

DROP TABLE pantry_items;
ALTER TABLE new_pantry_items RENAME TO pantry_items;

-- tags ---------------------------------------------------------------------

CREATE TABLE new_tags (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT    NOT NULL,
    tag_group  TEXT    NOT NULL,
    position   INTEGER NOT NULL,
    updated_at TEXT    NOT NULL,
    UNIQUE (tag_group, name)
) STRICT;

INSERT INTO new_tags (id, name, tag_group, position, updated_at)
SELECT id, name, tag_group, position, updated_at FROM tags;

DROP TABLE tags;
ALTER TABLE new_tags RENAME TO tags;

-- Trigger wie in 003 ---------------------------------------------------------

CREATE TRIGGER meal_plan_keep_title BEFORE DELETE ON recipes
BEGIN
    UPDATE meal_plan_entries
    SET note = CASE WHEN note = '' THEN OLD.title ELSE note END,
        updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
    WHERE recipe_id = OLD.id;
END;

-- Zähler -------------------------------------------------------------------

-- Das Umkopieren setzt den Zähler auf die größte vorhandene ID. War der
-- neueste Datensatz schon gelöscht, steht seine ID nur noch in deletions;
-- der Zähler beginnt dann dahinter, damit auch sie nicht wiederkommt.
-- Leere Tabellen haben noch keine Zeile in sqlite_sequence.
INSERT INTO sqlite_sequence (name, seq)
SELECT column1, 0
FROM (VALUES ('recipes'), ('meal_plan_entries'), ('pantry_items'), ('tags'))
WHERE NOT EXISTS (SELECT 1 FROM sqlite_sequence s WHERE s.name = column1);

UPDATE sqlite_sequence
SET seq = max(seq, (SELECT coalesce(max(d.record_id), 0)
                    FROM deletions d WHERE d.table_name = sqlite_sequence.name))
WHERE name IN ('recipes', 'meal_plan_entries', 'pantry_items', 'tags');
