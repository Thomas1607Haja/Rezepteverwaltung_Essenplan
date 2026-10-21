-- Essensplan für Phase 4. Siehe docs/PLANUNG.md, Abschnitte 3 und 4.
--
-- Pro Datum und Slot sind mehrere Einträge erlaubt (z. B. Hauptgericht
-- und Nachtisch); position legt ihre Reihenfolge fest.

CREATE TABLE meal_plan_entries (
    id             INTEGER PRIMARY KEY,
    -- Kalendertag ohne Uhrzeit, z. B. 2026-10-24
    date           TEXT    NOT NULL CHECK (date GLOB '[0-9][0-9][0-9][0-9]-[0-1][0-9]-[0-3][0-9]'),
    slot           TEXT    NOT NULL DEFAULT 'dinner' CHECK (slot IN ('dinner', 'lunch')),
    position       INTEGER NOT NULL DEFAULT 0,
    -- NULL bei Einträgen ohne Rezept (z. B. „Essen gehen“)
    recipe_id      INTEGER REFERENCES recipes (id) ON DELETE SET NULL,
    servings       INTEGER NOT NULL CHECK (servings >= 1),
    status         TEXT    NOT NULL DEFAULT 'planned'
                           CHECK (status IN ('planned', 'transferred', 'have_ingredients')),
    transferred_at TEXT,
    note           TEXT    NOT NULL DEFAULT '',
    created_at     TEXT    NOT NULL,
    updated_at     TEXT    NOT NULL,
    -- Ein Eintrag ohne Rezept braucht wenigstens eine Notiz.
    CHECK (recipe_id IS NOT NULL OR note <> '')
) STRICT;

-- Planansicht: Einträge eines Zeitraums in Anzeigereihenfolge
CREATE INDEX meal_plan_entries_date ON meal_plan_entries (date, slot, position);
-- Statushinweise und Statistik je Rezept
CREATE INDEX meal_plan_entries_recipe ON meal_plan_entries (recipe_id, date);

-- Wird ein Rezept gelöscht, bleiben seine Planeinträge erhalten: Der Titel
-- wandert in die Notiz (falls dort nichts steht), danach setzt der
-- Fremdschlüssel recipe_id auf NULL. So zeigt der Plan weiterhin, was es
-- an dem Tag gab bzw. geben sollte.
CREATE TRIGGER meal_plan_keep_title BEFORE DELETE ON recipes
BEGIN
    UPDATE meal_plan_entries
    SET note = CASE WHEN note = '' THEN OLD.title ELSE note END,
        updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
    WHERE recipe_id = OLD.id;
END;
