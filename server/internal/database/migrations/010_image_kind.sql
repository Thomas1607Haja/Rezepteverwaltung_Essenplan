-- Art des Rezeptbilds. Siehe docs/importformat.md (Version 2).
--
-- 'photo' = Foto, 'illustration' = gezeichneter Ersatz (Claude-Skill, wenn
-- die Vorlage kein brauchbares Foto hatte), '' = kein Bild. Illustrierte
-- Rezepte lassen sich so finden und durch echte Fotos ersetzen. Bestehende
-- Bilder sind Fotos.
ALTER TABLE recipes ADD COLUMN image_kind TEXT NOT NULL DEFAULT '';
UPDATE recipes SET image_kind = 'photo' WHERE image <> '';
