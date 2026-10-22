-- Rezeptvarianten. Siehe docs/PLANUNG.md, Abschnitte 3 und 4.
--
-- Eine Variante ist ein eigenständiges Rezept mit Verweis auf ihr
-- Ursprungsrezept. Es gibt nur eine Ebene („Rezeptfamilie“): Varianten
-- hängen immer am Ursprungsrezept, nie an einer anderen Variante. Beim
-- Löschen des Ursprungsrezepts wird die älteste Variante zum neuen
-- Ursprungsrezept (recipes.Store.Delete); ON DELETE SET NULL ist nur die
-- Absicherung.
ALTER TABLE recipes ADD COLUMN parent_recipe_id INTEGER REFERENCES recipes (id) ON DELETE SET NULL;

CREATE INDEX recipes_parent ON recipes (parent_recipe_id) WHERE parent_recipe_id IS NOT NULL;
