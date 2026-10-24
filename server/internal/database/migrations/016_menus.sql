-- Menüs (Schritt 9.4). Siehe docs/PLANUNG.md, Phase 9.
--
-- Ein Menü ist eine Klammer um normale Planeinträge: Jedes Gericht bleibt
-- ein eigener Eintrag (Einkauf, Bring, Abwahl und Statistik wie bisher)
-- und verweist mit menu_id auf das Menü. Alle Einträge eines Menüs stehen
-- an dessen Tag und Slot (prüft der Server). course_tag_id ist der Gang
-- des Gerichts im Menü (eine Kategorie der Gang-Gruppe, ohne
-- Fremdschlüssel: wird die Kategorie gelöscht, bleibt nur die Zahl).

CREATE TABLE meal_plan_menus (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    date       TEXT    NOT NULL CHECK (date GLOB '[0-9][0-9][0-9][0-9]-[0-1][0-9]-[0-3][0-9]'),
    slot       TEXT    NOT NULL DEFAULT 'dinner' CHECK (slot IN ('dinner', 'lunch', 'coffee')),
    position   INTEGER NOT NULL DEFAULT 0,
    -- Leer = automatisch (Titel des Hauptgerichts)
    title      TEXT    NOT NULL DEFAULT '',
    note       TEXT    NOT NULL DEFAULT '',
    servings   INTEGER NOT NULL CHECK (servings >= 1),
    created_at TEXT    NOT NULL,
    updated_at TEXT    NOT NULL
) STRICT;

CREATE INDEX meal_plan_menus_date ON meal_plan_menus (date, slot, position);
CREATE INDEX meal_plan_menus_updated ON meal_plan_menus (updated_at);

ALTER TABLE meal_plan_entries ADD COLUMN menu_id INTEGER REFERENCES meal_plan_menus (id) ON DELETE SET NULL;
ALTER TABLE meal_plan_entries ADD COLUMN course_tag_id INTEGER;

CREATE INDEX meal_plan_entries_menu ON meal_plan_entries (menu_id) WHERE menu_id IS NOT NULL;
