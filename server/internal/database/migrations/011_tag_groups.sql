-- Gruppen der Kategorien als eigene Tabelle. Siehe docs/PLANUNG.md,
-- Abschnitte 4 und 10 (Schritt 8.C).
--
-- Bisher stand die Gruppe als Text in tags.tag_group; ihre Reihenfolge
-- ergab sich aus der kleinsten ID. Für die Verwaltung in den Einstellungen
-- braucht eine Gruppe eine eigene Reihenfolge, einen Namen, den man an
-- einer Stelle ändert, und sie muss auch leer bestehen können.
--
-- Namen sind über alle Gruppen eindeutig, ohne Rücksicht auf Groß- und
-- Kleinschreibung und Umlaute. Das prüft der Server (database.Fold); die
-- UNIQUE-Angaben hier fangen nur exakt gleiche Namen ab.

CREATE TABLE tag_groups (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT    NOT NULL CHECK (name <> ''),
    position   INTEGER NOT NULL,
    updated_at TEXT    NOT NULL,
    UNIQUE (name)
) STRICT;

-- Vorhandene Gruppen in ihrer bisherigen Reihenfolge übernehmen.
INSERT INTO tag_groups (name, position, updated_at)
SELECT tag_group, row_number() OVER (ORDER BY min(id)), strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
FROM tags
GROUP BY tag_group
ORDER BY min(id);

-- tags mit group_id statt tag_group neu anlegen (Verfahren wie in 008).
CREATE TABLE new_tags (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT    NOT NULL CHECK (name <> ''),
    group_id   INTEGER NOT NULL REFERENCES tag_groups (id),
    position   INTEGER NOT NULL,
    updated_at TEXT    NOT NULL,
    UNIQUE (name)
) STRICT;

INSERT INTO new_tags (id, name, group_id, position, updated_at)
SELECT t.id, t.name, g.id, t.position, t.updated_at
FROM tags t JOIN tag_groups g ON g.name = t.tag_group;

-- Den Zähler übernehmen, damit gelöschte IDs nicht wiederkommen.
UPDATE sqlite_sequence
SET seq = max(seq, coalesce((SELECT seq FROM sqlite_sequence WHERE name = 'tags'), 0))
WHERE name = 'new_tags';

DROP TABLE tags;
ALTER TABLE new_tags RENAME TO tags;

CREATE INDEX tags_group ON tags (group_id, position);
