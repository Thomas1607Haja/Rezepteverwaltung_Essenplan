-- Protokoll der an Bring übertragenen Artikel (Phase 5). Siehe
-- docs/PLANUNG.md, Abschnitte 3 und 4.
--
-- Je Planeintrag und Zutat eine Zeile mit der umgerechneten Menge. So lässt
-- sich später vergleichen, was ein geänderter Plan braucht und was schon
-- auf der Liste steht (Differenz nachsenden, zu viel Gekauftes anzeigen).

CREATE TABLE transferred_items (
    id              INTEGER PRIMARY KEY,
    -- NULL, wenn der Planeintrag inzwischen gelöscht ist: Die Zeile bleibt,
    -- damit die Einkaufsseite das Gekaufte weiter als "zu viel" zeigen kann.
    entry_id        INTEGER REFERENCES meal_plan_entries (id) ON DELETE SET NULL,
    -- Beginn des Einkaufszeitraums, für den übertragen wurde, z. B. 2026-10-24
    period_from     TEXT    NOT NULL,
    -- Vergleichsschlüssel des Artikels (siehe shopping.Item.Key)
    item_key        TEXT    NOT NULL,
    ingredient_name TEXT    NOT NULL,
    quantity        REAL,             -- NULL = ohne Mengenangabe
    unit            TEXT    NOT NULL DEFAULT '',
    -- 1 = an Bring gesendet, 0 = beim Senden abgewählt (z. B. schon im Haus)
    sent            INTEGER NOT NULL CHECK (sent IN (0, 1)),
    transferred_at  TEXT    NOT NULL
) STRICT;

CREATE INDEX transferred_items_period ON transferred_items (period_from, item_key);
CREATE INDEX transferred_items_entry ON transferred_items (entry_id);
