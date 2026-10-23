package recipes

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/database"
)

// Rezeptvarianten: Eine Variante ist ein eigenständiges Rezept (eigene
// Zutaten, eigener Favoritenstatus, eigene Planeinträge) mit Verweis auf
// ihr Ursprungsrezept in parent_recipe_id. Es gibt nur eine Ebene: Wer von
// einer Variante wieder eine Variante speichert, hängt sie ans
// Ursprungsrezept. Ursprungsrezept und Varianten bilden die Rezeptfamilie.

// variantSuffix wird an den Titel der Kopie gehängt.
const variantSuffix = " (Variante)"

// FamilyMember ist ein anderes Rezept derselben Familie, für die Links in
// der Rezeptansicht.
type FamilyMember struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Thumbnail string `json:"thumbnail"`
	IsOrigin  bool   `json:"is_origin"` // das Ursprungsrezept der Familie
}

// variantTitle hängt " (Variante)" an – aber nicht doppelt, wenn schon
// eine Variante kopiert wird.
func variantTitle(title string) string {
	if strings.HasSuffix(title, variantSuffix) {
		return title
	}
	return title + variantSuffix
}

// CreateVariant legt eine Kopie des Rezepts sourceID als Variante an und
// gibt sie zurück. Kopiert werden alle Felder, Zutaten (mit Reihenfolge und
// Einkaufsrelevanz), Schritte und Kategorien; nicht der Favoritenstatus
// und keine Planeinträge. image und thumbnail sind die schon kopierten
// Bilddateien (leer = kein Bild), siehe images.Store.Copy.
// Gibt es das Rezept nicht, ist der Fehler ErrNotFound.
func (s *Store) CreateVariant(ctx context.Context, sourceID int64, image, thumbnail string) (Recipe, error) {
	now := database.FormatTime(time.Now())

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Recipe{}, fmt.Errorf("variante von %d anlegen: %w", sourceID, err)
	}
	defer tx.Rollback()

	var title string
	var parent sql.NullInt64
	err = tx.QueryRowContext(ctx, `SELECT title, parent_recipe_id FROM recipes WHERE id = ?`, sourceID).
		Scan(&title, &parent)
	if errors.Is(err, sql.ErrNoRows) {
		return Recipe{}, ErrNotFound
	}
	if err != nil {
		return Recipe{}, fmt.Errorf("variante von %d anlegen: %w", sourceID, err)
	}
	// Eine Ebene: Ist die Quelle selbst eine Variante, gilt ihr Ursprung.
	origin := sourceID
	if parent.Valid {
		origin = parent.Int64
	}

	// INSERT … SELECT kopiert die Zeile direkt in der Datenbank.
	res, err := tx.ExecContext(ctx, `
		INSERT INTO recipes (title, description, base_servings, prep_minutes, total_minutes,
		                     source_url, source_name, image, thumbnail, image_kind, is_favorite, notes,
		                     parent_recipe_id, created_at, updated_at)
		SELECT ?, description, base_servings, prep_minutes, total_minutes,
		       source_url, source_name, ?, ?, CASE WHEN ? = '' THEN '' ELSE image_kind END, 0, notes, ?, ?, ?
		FROM recipes WHERE id = ?`,
		variantTitle(title), image, thumbnail, image, origin, now, now, sourceID)
	if err != nil {
		return Recipe{}, fmt.Errorf("variante von %d anlegen: %w", sourceID, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Recipe{}, fmt.Errorf("variante von %d anlegen: %w", sourceID, err)
	}

	for _, query := range []string{
		`INSERT INTO recipe_ingredients (recipe_id, position, group_name, quantity, unit,
		                                 name, note, shopping_relevant, updated_at)
		 SELECT ?, position, group_name, quantity, unit, name, note, shopping_relevant, ?
		 FROM recipe_ingredients WHERE recipe_id = ? ORDER BY position`,
		`INSERT INTO recipe_steps (recipe_id, position, text, updated_at)
		 SELECT ?, position, text, ?
		 FROM recipe_steps WHERE recipe_id = ? ORDER BY position`,
		`INSERT INTO recipe_tags (recipe_id, tag_id, updated_at)
		 SELECT ?, tag_id, ?
		 FROM recipe_tags WHERE recipe_id = ?`,
	} {
		if _, err := tx.ExecContext(ctx, query, id, now, sourceID); err != nil {
			return Recipe{}, fmt.Errorf("variante von %d anlegen: %w", sourceID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return Recipe{}, fmt.Errorf("variante von %d anlegen: %w", sourceID, err)
	}
	return s.Get(ctx, id)
}

// loadFamily liefert die anderen Rezepte der Familie von r: das
// Ursprungsrezept zuerst, dann die Varianten nach Alter. Ohne Familie ist
// die Liste leer.
func (s *Store) loadFamily(ctx context.Context, r Recipe) ([]FamilyMember, error) {
	origin := r.ID
	if r.ParentRecipeID != nil {
		origin = *r.ParentRecipeID
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, title, thumbnail, parent_recipe_id IS NULL
		FROM recipes
		WHERE (id = ? OR parent_recipe_id = ?) AND id <> ?
		ORDER BY parent_recipe_id IS NOT NULL, id`, origin, origin, r.ID)
	if err != nil {
		return nil, fmt.Errorf("rezeptfamilie von %d laden: %w", r.ID, err)
	}
	defer rows.Close()

	family := []FamilyMember{}
	for rows.Next() {
		var m FamilyMember
		if err := rows.Scan(&m.ID, &m.Title, &m.Thumbnail, &m.IsOrigin); err != nil {
			return nil, fmt.Errorf("rezeptfamilie von %d laden: %w", r.ID, err)
		}
		family = append(family, m)
	}
	return family, rows.Err()
}

// promoteOldestVariant bereitet das Löschen des Rezepts id vor: Hat es
// Varianten, wird die älteste (kleinste ID) zum neuen Ursprungsrezept, und
// die übrigen hängen danach an ihr. updated_at ändert sich bei allen
// umgehängten Rezepten, damit die App sie beim Abgleich neu holt.
func promoteOldestVariant(ctx context.Context, tx *sql.Tx, id int64, now string) error {
	var oldest sql.NullInt64
	if err := tx.QueryRowContext(ctx,
		`SELECT MIN(id) FROM recipes WHERE parent_recipe_id = ?`, id).Scan(&oldest); err != nil {
		return err
	}
	if !oldest.Valid {
		return nil // keine Varianten
	}
	_, err := tx.ExecContext(ctx, `
		UPDATE recipes
		SET parent_recipe_id = CASE WHEN id = ? THEN NULL ELSE ? END, updated_at = ?
		WHERE parent_recipe_id = ?`, oldest.Int64, oldest.Int64, now, id)
	return err
}
