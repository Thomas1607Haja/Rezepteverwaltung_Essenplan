-- Startkategorien aus docs/PLANUNG.md, Abschnitt 4.
-- Sie können später in den Einstellungen geändert werden.

INSERT INTO tags (tag_group, position, name, updated_at)
-- SQLite benennt die Spalten einer VALUES-Liste fest column1, column2, ...
SELECT column1, column2, column3, strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
FROM (
    VALUES
    ('Hauptzutat', 1, 'Vegetarisch'),
    ('Hauptzutat', 2, 'Vegan'),
    ('Hauptzutat', 3, 'Rind'),
    ('Hauptzutat', 4, 'Schwein'),
    ('Hauptzutat', 5, 'Geflügel'),
    ('Hauptzutat', 6, 'Hackfleisch'),
    ('Hauptzutat', 7, 'Fisch'),
    ('Hauptzutat', 8, 'Meeresfrüchte'),

    ('Geschmack', 1, 'Herzhaft'),
    ('Geschmack', 2, 'Süß'),

    ('Art', 1, 'Pasta'),
    ('Art', 2, 'Reis'),
    ('Art', 3, 'Kartoffeln'),
    ('Art', 4, 'Suppe/Eintopf'),
    ('Art', 5, 'Salat'),
    ('Art', 6, 'Auflauf/Ofen'),
    ('Art', 7, 'Pfanne'),
    ('Art', 8, 'Grill'),

    ('Eigenschaften', 1, 'Schnell'),
    ('Eigenschaften', 2, 'Kinderliebling'),
    ('Eigenschaften', 3, 'Gut vorzubereiten'),
    ('Eigenschaften', 4, 'Resteverwertung'),
    ('Eigenschaften', 5, 'Für Gäste')
);
