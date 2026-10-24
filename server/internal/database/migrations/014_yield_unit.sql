-- Ergebnis-Einheit eines Rezepts (Schritt 9.2). Siehe docs/PLANUNG.md,
-- Phase 9.
--
-- Leer = Portionen (wie bisher). Sonst z. B. "Stück", "Kuchen", "Blech",
-- "Springform 26 cm": base_servings zählt dann diese Einheit ("1
-- Springform 26 cm", "12 Stück"); umgerechnet wird weiter linear.
ALTER TABLE recipes ADD COLUMN yield_unit TEXT NOT NULL DEFAULT '';
