package pantry

import (
	"context"
	"fmt"
	"time"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/database"
)

// Change ist eine Zutat, deren Häkchen "einkaufsrelevant" nicht zur
// Vorratsliste passt.
type Change struct {
	RecipeID       int64  `json:"recipe_id"`
	RecipeTitle    string `json:"recipe_title"`
	IngredientID   int64  `json:"ingredient_id"`
	IngredientName string `json:"ingredient_name"`
	// ShoppingRelevant ist der neue Wert: false = kommt in den Vorrat,
	// true = wird wieder eingekauft.
	ShoppingRelevant bool `json:"shopping_relevant"`
}

// ApplyResult beschreibt, was das Übertragen der Vorratsliste auf alle
// Rezepte ändert bzw. geändert hat.
type ApplyResult struct {
	Applied bool     `json:"applied"` // false = nur Vorschau
	Changes []Change `json:"changes"` // nach Rezepttitel sortiert
	Recipes int      `json:"recipes"` // Anzahl betroffener Rezepte
}

// ApplyToRecipes gleicht das Häkchen "einkaufsrelevant" aller Zutaten mit
// der Vorratsliste ab: Zutaten im Vorrat werden abgehakt, alle anderen
// wieder einkaufsrelevant. Abweichungen, die von Hand in einem Rezept
// gesetzt wurden, gehen dabei verloren – daher erst mit confirm = false die
// Vorschau holen und nur nach Bestätigung mit confirm = true speichern.
func (s *Store) ApplyToRecipes(ctx context.Context, confirm bool) (ApplyResult, error) {
	m, err := s.Matcher(ctx)
	if err != nil {
		return ApplyResult{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ApplyResult{}, fmt.Errorf("vorrat übertragen: %w", err)
	}
	defer tx.Rollback()

	// Vorschau und Speichern rechnen in derselben Transaktion, damit
	// zwischen Lesen und Schreiben niemand ein Rezept ändert.
	rows, err := tx.QueryContext(ctx, `
		SELECT r.id, r.title, i.id, i.name, i.shopping_relevant
		FROM recipe_ingredients i JOIN recipes r ON r.id = i.recipe_id
		ORDER BY fold(r.title), r.id, i.position`)
	if err != nil {
		return ApplyResult{}, fmt.Errorf("vorrat übertragen: %w", err)
	}
	result := ApplyResult{Changes: []Change{}}
	for rows.Next() {
		var c Change
		var relevant bool
		if err := rows.Scan(&c.RecipeID, &c.RecipeTitle, &c.IngredientID, &c.IngredientName, &relevant); err != nil {
			rows.Close()
			return ApplyResult{}, fmt.Errorf("vorrat übertragen: %w", err)
		}
		c.ShoppingRelevant = !m.Matches(c.IngredientName)
		if c.ShoppingRelevant != relevant {
			result.Changes = append(result.Changes, c)
		}
	}
	// Erst schließen, dann prüfen: rows.Err meldet Fehler beim Durchlaufen.
	rows.Close()
	if err := rows.Err(); err != nil {
		return ApplyResult{}, fmt.Errorf("vorrat übertragen: %w", err)
	}

	recipes := map[int64]bool{}
	for _, c := range result.Changes {
		recipes[c.RecipeID] = true
	}
	result.Recipes = len(recipes)
	if !confirm || len(result.Changes) == 0 {
		return result, nil
	}

	now := database.FormatTime(time.Now())
	for _, c := range result.Changes {
		_, err := tx.ExecContext(ctx, `
			UPDATE recipe_ingredients SET shopping_relevant = ?, updated_at = ? WHERE id = ?`,
			c.ShoppingRelevant, now, c.IngredientID)
		if err != nil {
			return ApplyResult{}, fmt.Errorf("vorrat übertragen: %w", err)
		}
	}
	// Auch das Rezept selbst gilt als geändert, damit die App es beim
	// Abgleich neu holt.
	for id := range recipes {
		if _, err := tx.ExecContext(ctx, "UPDATE recipes SET updated_at = ? WHERE id = ?", now, id); err != nil {
			return ApplyResult{}, fmt.Errorf("vorrat übertragen: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return ApplyResult{}, fmt.Errorf("vorrat übertragen: %w", err)
	}
	result.Applied = true
	return result, nil
}
