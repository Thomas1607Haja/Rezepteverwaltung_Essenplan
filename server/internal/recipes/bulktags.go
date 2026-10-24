package recipes

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"slices"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/httpx"
)

// Sammeländerung von Kategorien (Zuweisungsmodus der Weboberfläche): viele
// Rezepte bekommen eine Kategorie (z. B. einen Gang) oder verlieren sie,
// alles in einer Transaktion. Das Ergebnis nennt nur, was sich wirklich
// geändert hat – mit vertauschten Listen zurückgeschickt macht es die
// Änderung rückgängig.

// MaxBulkTagChanges begrenzt eine Sammeländerung (Hinzufügen + Entfernen).
const MaxBulkTagChanges = 5000

// TagChange ist eine Kategorie an einem Rezept.
type TagChange struct {
	RecipeID int64 `json:"recipe_id"`
	TagID    int64 `json:"tag_id"`
}

// BulkTags sind die gewünschten Änderungen; erst wird entfernt, dann
// hinzugefügt.
type BulkTags struct {
	Add    []TagChange `json:"add"`
	Remove []TagChange `json:"remove"`
}

// BulkTagsResult: tatsächlich hinzugefügt bzw. entfernt (schon vorhandene
// bzw. fehlende Paare fehlen hier) und die Zahl der geänderten Rezepte.
type BulkTagsResult struct {
	Added   []TagChange `json:"added"`
	Removed []TagChange `json:"removed"`
	Recipes int         `json:"recipes"`
}

// BulkTags führt die Änderungen aus. Unbekannte Rezepte oder Kategorien
// sind ein Eingabefehler; dann ändert sich nichts. Geänderte Rezepte
// bekommen ein neues updated_at, damit die App sie beim Abgleich holt.
func (s *Store) BulkTags(ctx context.Context, in BulkTags) (BulkTagsResult, error) {
	result := BulkTagsResult{Added: []TagChange{}, Removed: []TagChange{}}
	if len(in.Add)+len(in.Remove) > MaxBulkTagChanges {
		return result, &ValidationError{"add", fmt.Sprintf("höchstens %d änderungen auf einmal", MaxBulkTagChanges)}
	}
	var recipeIDs, tagIDs []int64
	for _, c := range append(slices.Clone(in.Add), in.Remove...) {
		recipeIDs = append(recipeIDs, c.RecipeID)
		tagIDs = append(tagIDs, c.TagID)
	}
	changed := map[int64]bool{}
	err := s.inTx(ctx, "kategorien ändern", func(tx *sql.Tx, now string) error {
		if err := checkIDs(ctx, tx, "recipes", "recipe_id", "rezept", recipeIDs); err != nil {
			return err
		}
		if err := checkIDs(ctx, tx, "tags", "tag_id", "kategorie", tagIDs); err != nil {
			return err
		}
		for _, c := range in.Remove {
			res, err := tx.ExecContext(ctx, "DELETE FROM recipe_tags WHERE recipe_id = ? AND tag_id = ?", c.RecipeID, c.TagID)
			if err != nil {
				return err
			}
			if n, _ := res.RowsAffected(); n > 0 {
				result.Removed = append(result.Removed, c)
				changed[c.RecipeID] = true
			}
		}
		for _, c := range in.Add {
			res, err := tx.ExecContext(ctx, "INSERT OR IGNORE INTO recipe_tags (recipe_id, tag_id, updated_at) VALUES (?, ?, ?)",
				c.RecipeID, c.TagID, now)
			if err != nil {
				return err
			}
			if n, _ := res.RowsAffected(); n > 0 {
				result.Added = append(result.Added, c)
				changed[c.RecipeID] = true
			}
		}
		for id := range changed {
			if _, err := tx.ExecContext(ctx, "UPDATE recipes SET updated_at = ? WHERE id = ?", now, id); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return BulkTagsResult{}, err
	}
	result.Recipes = len(changed)
	return result, nil
}

// checkIDs prüft, dass es alle ids in table gibt.
func checkIDs(ctx context.Context, tx *sql.Tx, table, field, what string, ids []int64) error {
	ids = slices.Clone(ids)
	slices.Sort(ids)
	ids = slices.Compact(ids)
	for start := 0; start < len(ids); start += 500 {
		part := ids[start:min(start+500, len(ids))]
		args := make([]any, len(part))
		for i, id := range part {
			args[i] = id
		}
		var n int
		if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM "+table+" WHERE id IN ("+placeholders(len(part))+")", args...).Scan(&n); err != nil {
			return err
		}
		if n != len(part) {
			return &ValidationError{field, fmt.Sprintf("unbekannte %s-id", what)}
		}
	}
	return nil
}

// bulkTags: {"add": [{recipe_id, tag_id}], "remove": [...]} → was sich
// geändert hat. Nur über die Weboberfläche (apikeys sperrt POST /tags/…).
func (h *Handler) bulkTags(w http.ResponseWriter, r *http.Request) {
	var in BulkTags
	if err := httpx.ReadJSON(w, r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.store.BulkTags(r.Context(), in)
	if err != nil {
		writeTagError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}
