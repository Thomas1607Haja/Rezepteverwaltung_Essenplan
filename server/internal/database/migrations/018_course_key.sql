-- Fester Schlüssel für die Gänge der Startliste (Mehrsprachigkeit, 0.20).
-- Siehe docs/PLANUNG.md und docs/glossar.md.
--
-- Gänge sind normale Kategorien; ihre Namen sind Daten des Nutzers und
-- werden nie übersetzt oder umbenannt. Der Code fand bestimmte Gänge aber
-- bisher über den deutschen Namen (Reiter „Hauptgerichte“, „Backen“ …,
-- Vorschlag „Hauptgericht“ im Menü, Gang-Vorschläge). Ab jetzt über
-- course_key: starter, soup, salad, main, side, dessert, cake, pastry,
-- bread, snack; leer = eigener Gang ohne feste Bedeutung.
--
-- Bestehende Installationen: Die Schlüssel werden über die Namen der
-- deutschen Startliste vergeben. Kein Name ändert sich. Wer einen Gang
-- umbenannt hat, behält dessen Schlüssel nicht – er kann ihn aber auch
-- nicht verlieren, was es vorher nicht gab.
ALTER TABLE tags ADD COLUMN course_key TEXT NOT NULL DEFAULT '';

UPDATE tags SET course_key = CASE lower(name)
        WHEN 'vorspeise' THEN 'starter'
        WHEN 'suppe/eintopf' THEN 'soup'
        WHEN 'salat' THEN 'salad'
        WHEN 'hauptgericht' THEN 'main'
        WHEN 'beilage' THEN 'side'
        WHEN 'nachspeise' THEN 'dessert'
        WHEN 'kuchen & torten' THEN 'cake'
        WHEN 'gebäck' THEN 'pastry'
        WHEN 'brot' THEN 'bread'
        WHEN 'snack' THEN 'snack'
    END,
    updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
WHERE group_id IN (SELECT id FROM tag_groups WHERE role = 'course')
  AND lower(name) IN ('vorspeise', 'suppe/eintopf', 'salat', 'hauptgericht', 'beilage',
                      'nachspeise', 'kuchen & torten', 'gebäck', 'brot', 'snack');

-- Rezepte mit diesen Gängen gelten als geändert, damit die App die
-- Schlüssel beim nächsten Abgleich bekommt.
UPDATE recipes SET updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
WHERE id IN (SELECT rt.recipe_id FROM recipe_tags rt JOIN tags t ON t.id = rt.tag_id
             WHERE t.course_key <> '');
