-- Slot „Kaffee & Kuchen“ (coffee) zwischen Mittag und Abend (Schritt 9.3).
-- Siehe docs/PLANUNG.md, Phase 9.
--
-- Der Slot ist per CHECK auf dinner/lunch begrenzt; SQLite kann eine
-- CHECK-Bedingung nicht ändern. Die Tabelle wird deshalb wie in Migration
-- 008 neu angelegt, umkopiert und umbenannt (Fremdschlüssel sind dabei
-- abgeschaltet, siehe applyMigration). Spalten, IDs, Indizes, Trigger und
-- der AUTOINCREMENT-Zähler bleiben.

DROP TRIGGER meal_plan_keep_title;

CREATE TABLE new_meal_plan_entries (
    id                   INTEGER PRIMARY KEY AUTOINCREMENT,
    date                 TEXT    NOT NULL CHECK (date GLOB '[0-9][0-9][0-9][0-9]-[0-1][0-9]-[0-3][0-9]'),
    slot                 TEXT    NOT NULL DEFAULT 'dinner' CHECK (slot IN ('dinner', 'lunch', 'coffee')),
    position             INTEGER NOT NULL DEFAULT 0,
    recipe_id            INTEGER REFERENCES recipes (id) ON DELETE SET NULL,
    servings             INTEGER NOT NULL CHECK (servings >= 1),
    status               TEXT    NOT NULL DEFAULT 'planned'
                                 CHECK (status IN ('planned', 'transferred', 'have_ingredients')),
    transferred_at       TEXT,
    note                 TEXT    NOT NULL DEFAULT '',
    created_at           TEXT    NOT NULL,
    updated_at           TEXT    NOT NULL,
    excluded_ingredients TEXT    NOT NULL DEFAULT '[]',
    CHECK (recipe_id IS NOT NULL OR note <> '')
) STRICT;

INSERT INTO new_meal_plan_entries (id, date, slot, position, recipe_id, servings, status,
    transferred_at, note, created_at, updated_at, excluded_ingredients)
SELECT id, date, slot, position, recipe_id, servings, status,
    transferred_at, note, created_at, updated_at, excluded_ingredients
FROM meal_plan_entries;

-- Zähler übernehmen, damit gelöschte IDs nicht wiederkommen.
INSERT INTO sqlite_sequence (name, seq)
SELECT 'new_meal_plan_entries', coalesce((SELECT seq FROM sqlite_sequence WHERE name = 'meal_plan_entries'), 0)
WHERE NOT EXISTS (SELECT 1 FROM sqlite_sequence WHERE name = 'new_meal_plan_entries');
UPDATE sqlite_sequence
SET seq = max(seq, coalesce((SELECT seq FROM sqlite_sequence WHERE name = 'meal_plan_entries'), 0))
WHERE name = 'new_meal_plan_entries';

DROP TABLE meal_plan_entries;
ALTER TABLE new_meal_plan_entries RENAME TO meal_plan_entries;

CREATE INDEX meal_plan_entries_date ON meal_plan_entries (date, slot, position);
CREATE INDEX meal_plan_entries_recipe ON meal_plan_entries (recipe_id, date);

CREATE TRIGGER meal_plan_keep_title BEFORE DELETE ON recipes
BEGIN
    UPDATE meal_plan_entries
    SET note = CASE WHEN note = '' THEN OLD.title ELSE note END,
        updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
    WHERE recipe_id = OLD.id;
END;
