-- Gang eines Gerichts (Schritt 9.1). Siehe docs/PLANUNG.md, Phase 9.
--
-- Der Gang ist eine ganz normale Kategoriegruppe; das Kennzeichen
-- role = 'course' sagt dem Code, welche Gruppe es ist (Schnellfilter,
-- Gänge im Menü). Höchstens eine Gruppe trägt es (prüft der Server).
--
-- Vorsicht: Kategorien sind seit 0.9.0 frei änderbar. Was es schon gibt,
-- wird nicht doppelt angelegt; „Suppe/Eintopf“ und „Salat“ wandern aus
-- ihrer bisherigen Gruppe in „Gang“, ihre Zuordnungen bleiben.

ALTER TABLE tag_groups ADD COLUMN role TEXT NOT NULL DEFAULT '';

-- Gruppe „Gang“ vorn einfügen (oder eine vorhandene gleichen Namens nehmen).
UPDATE tag_groups SET position = position + 1
WHERE NOT EXISTS (SELECT 1 FROM tag_groups WHERE name = 'Gang');

INSERT INTO tag_groups (name, position, updated_at, role)
SELECT 'Gang', 1, strftime('%Y-%m-%dT%H:%M:%fZ', 'now'), 'course'
WHERE NOT EXISTS (SELECT 1 FROM tag_groups WHERE name = 'Gang');

UPDATE tag_groups SET role = 'course' WHERE name = 'Gang';

-- Vorhandene Kategorien verschieben; die Rezepte mit ihnen gelten als
-- geändert, damit die App die neue Gruppe sieht.
UPDATE recipes SET updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
WHERE id IN (SELECT rt.recipe_id FROM recipe_tags rt JOIN tags t ON t.id = rt.tag_id
             WHERE t.name IN ('Suppe/Eintopf', 'Salat'));

UPDATE tags SET group_id = (SELECT id FROM tag_groups WHERE name = 'Gang'),
                updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
WHERE name IN ('Suppe/Eintopf', 'Salat');

-- Fehlende Gänge anlegen, in der Reihenfolge der Startliste.
INSERT INTO tags (name, group_id, position, updated_at)
SELECT column1, (SELECT id FROM tag_groups WHERE name = 'Gang'), column2,
       strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
FROM (VALUES
    ('Vorspeise', 1),
    ('Suppe/Eintopf', 2),
    ('Salat', 3),
    ('Hauptgericht', 4),
    ('Beilage', 5),
    ('Nachspeise', 6),
    ('Kuchen & Torten', 7),
    ('Gebäck', 8),
    ('Brot', 9),
    ('Snack', 10)
)
WHERE NOT EXISTS (SELECT 1 FROM tags t WHERE lower(t.name) = lower(column1));

-- Reihenfolge innerhalb der Gruppe wie in der Startliste.
UPDATE tags SET position = CASE name
    WHEN 'Vorspeise' THEN 1 WHEN 'Suppe/Eintopf' THEN 2 WHEN 'Salat' THEN 3
    WHEN 'Hauptgericht' THEN 4 WHEN 'Beilage' THEN 5 WHEN 'Nachspeise' THEN 6
    WHEN 'Kuchen & Torten' THEN 7 WHEN 'Gebäck' THEN 8 WHEN 'Brot' THEN 9
    WHEN 'Snack' THEN 10 ELSE 20 + position END
WHERE group_id = (SELECT id FROM tag_groups WHERE name = 'Gang');
