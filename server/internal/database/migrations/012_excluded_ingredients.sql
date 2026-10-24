-- Abgewählte Zutaten je Planeintrag („habe ich noch“). Siehe docs/PLANUNG.md,
-- Abschnitt 3 „Einkaufsliste und Bring“ (Schritt 8.E).
--
-- JSON-Liste von Zutatenschlüsseln (Paket itemkey, z. B. ["zwiebel","sahn"]).
-- Schlüssel statt Zutaten-IDs, weil das Speichern eines Rezepts die Zutaten
-- neu anlegt (neue IDs); der Schlüssel überdauert das. Eine Spalte statt
-- einer eigenen Tabelle: Die Abwahl gehört fest zum Eintrag, wandert beim
-- Verschieben und Tauschen mit und kommt über updated_at in die App.
ALTER TABLE meal_plan_entries ADD COLUMN excluded_ingredients TEXT NOT NULL DEFAULT '[]';

-- Die Rezepte liefern ab jetzt je Zutat einen item_key. Die App holt beim
-- Abgleich nur geänderte Rezepte; damit sie die Schlüssel für alle hat
-- (sonst bliebe die Kurzvorschau dort aus), gelten alle einmal als geändert.
UPDATE recipes SET updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now');
