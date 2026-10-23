-- Quelle ohne Webadresse. Siehe docs/PLANUNG.md, Abschnitt 4.
--
-- source_url nimmt nur http/https an (Sicherheit). Rezepte aus Kochbüchern
-- oder von Rezeptkarten (Datei-Import, Claude-Skill) haben stattdessen eine
-- Quelle wie „HelloFresh-Karte“ oder „Kochbuch“.
ALTER TABLE recipes ADD COLUMN source_name TEXT NOT NULL DEFAULT '';
