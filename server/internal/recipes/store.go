package recipes

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/database"
)

// Store liest und schreibt Rezepte in der Datenbank.
type Store struct {
	db *sql.DB
}

// NewStore erzeugt einen Store für die geöffnete Datenbank db.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Create speichert ein neues Rezept und gibt es so zurück, wie es nun in der
// Datenbank steht (mit IDs und Zeitstempeln). ID, Bildfelder und
// Zeitstempel aus r werden ignoriert.
func (s *Store) Create(ctx context.Context, r Recipe) (Recipe, error) {
	r.normalize()
	if err := r.validate(); err != nil {
		return Recipe{}, err
	}
	now := database.FormatTime(time.Now())

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Recipe{}, fmt.Errorf("rezept anlegen: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `
		INSERT INTO recipes (title, description, base_servings, prep_minutes, total_minutes,
		                     source_url, is_favorite, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.Title, r.Description, r.BaseServings, r.PrepMinutes, r.TotalMinutes,
		r.SourceURL, r.IsFavorite, r.Notes, now, now)
	if err != nil {
		return Recipe{}, fmt.Errorf("rezept anlegen: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Recipe{}, fmt.Errorf("rezept anlegen: %w", err)
	}

	if err := insertParts(ctx, tx, id, r, now); err != nil {
		return Recipe{}, err
	}
	if err := tx.Commit(); err != nil {
		return Recipe{}, fmt.Errorf("rezept anlegen: %w", err)
	}

	// Erst nach dem Commit lesen: Die einzige Verbindung ist dann wieder frei.
	return s.Get(ctx, id)
}

// insertParts speichert Zutaten, Schritte und Kategorien eines Rezepts.
// Die Reihenfolge in den Slices wird zur Position (beginnend bei 1).
func insertParts(ctx context.Context, tx *sql.Tx, recipeID int64, r Recipe, now string) error {
	for i, ing := range r.Ingredients {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO recipe_ingredients (recipe_id, position, group_name, quantity, unit,
			                                name, note, shopping_relevant, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			recipeID, i+1, ing.GroupName, ing.Quantity, ing.Unit,
			ing.Name, ing.Note, ing.ShoppingRelevant, now)
		if err != nil {
			return fmt.Errorf("zutat speichern: %w", err)
		}
	}

	for i, step := range r.Steps {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO recipe_steps (recipe_id, position, text, updated_at)
			VALUES (?, ?, ?, ?)`,
			recipeID, i+1, step.Text, now)
		if err != nil {
			return fmt.Errorf("schritt speichern: %w", err)
		}
	}

	// Eine Map merkt sich, welche Kategorien schon verknüpft sind, damit
	// doppelt übergebene IDs nicht zu einem Fehler führen.
	seen := make(map[int64]bool)
	for i, tag := range r.Tags {
		if seen[tag.ID] {
			continue
		}
		seen[tag.ID] = true

		// INSERT ... SELECT fügt nur dann eine Zeile ein, wenn es die
		// Kategorie gibt. So erkennen wir unbekannte IDs ohne extra Abfrage.
		res, err := tx.ExecContext(ctx, `
			INSERT INTO recipe_tags (recipe_id, tag_id, updated_at)
			SELECT ?, id, ? FROM tags WHERE id = ?`,
			recipeID, now, tag.ID)
		if err != nil {
			return fmt.Errorf("kategorie speichern: %w", err)
		}
		if n, err := res.RowsAffected(); err != nil {
			return fmt.Errorf("kategorie speichern: %w", err)
		} else if n == 0 {
			return &ValidationError{fmt.Sprintf("tags[%d].id", i), fmt.Sprintf("kategorie %d gibt es nicht", tag.ID)}
		}
	}
	return nil
}

// Get lädt ein Rezept mit allen Bestandteilen. Gibt es das Rezept nicht,
// ist der Fehler ErrNotFound.
func (s *Store) Get(ctx context.Context, id int64) (Recipe, error) {
	var r Recipe
	var createdAt, updatedAt string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, title, description, base_servings, prep_minutes, total_minutes,
		       source_url, image, thumbnail, is_favorite, notes, created_at, updated_at
		FROM recipes WHERE id = ?`, id).Scan(
		&r.ID, &r.Title, &r.Description, &r.BaseServings, &r.PrepMinutes, &r.TotalMinutes,
		&r.SourceURL, &r.Image, &r.Thumbnail, &r.IsFavorite, &r.Notes, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Recipe{}, ErrNotFound
	}
	if err != nil {
		return Recipe{}, fmt.Errorf("rezept %d laden: %w", id, err)
	}

	if r.CreatedAt, err = database.ParseTime(createdAt); err != nil {
		return Recipe{}, fmt.Errorf("rezept %d laden: %w", id, err)
	}
	if r.UpdatedAt, err = database.ParseTime(updatedAt); err != nil {
		return Recipe{}, fmt.Errorf("rezept %d laden: %w", id, err)
	}
	if r.Ingredients, err = s.loadIngredients(ctx, id); err != nil {
		return Recipe{}, err
	}
	if r.Steps, err = s.loadSteps(ctx, id); err != nil {
		return Recipe{}, err
	}
	if r.Tags, err = s.loadTags(ctx, id); err != nil {
		return Recipe{}, err
	}
	return r, nil
}

func (s *Store) loadIngredients(ctx context.Context, recipeID int64) ([]Ingredient, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, group_name, quantity, unit, name, note, shopping_relevant
		FROM recipe_ingredients WHERE recipe_id = ? ORDER BY position`, recipeID)
	if err != nil {
		return nil, fmt.Errorf("zutaten laden: %w", err)
	}
	defer rows.Close()

	// Leerer Slice statt nil, damit im JSON [] statt null steht.
	ingredients := []Ingredient{}
	for rows.Next() {
		var ing Ingredient
		if err := rows.Scan(&ing.ID, &ing.GroupName, &ing.Quantity, &ing.Unit,
			&ing.Name, &ing.Note, &ing.ShoppingRelevant); err != nil {
			return nil, fmt.Errorf("zutaten laden: %w", err)
		}
		ingredients = append(ingredients, ing)
	}
	// rows.Err meldet Fehler, die während der Schleife aufgetreten sind.
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("zutaten laden: %w", err)
	}
	return ingredients, nil
}

func (s *Store) loadSteps(ctx context.Context, recipeID int64) ([]Step, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, text FROM recipe_steps WHERE recipe_id = ? ORDER BY position`, recipeID)
	if err != nil {
		return nil, fmt.Errorf("schritte laden: %w", err)
	}
	defer rows.Close()

	steps := []Step{}
	for rows.Next() {
		var step Step
		if err := rows.Scan(&step.ID, &step.Text); err != nil {
			return nil, fmt.Errorf("schritte laden: %w", err)
		}
		steps = append(steps, step)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("schritte laden: %w", err)
	}
	return steps, nil
}

func (s *Store) loadTags(ctx context.Context, recipeID int64) ([]Tag, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT t.id, t.name, t.tag_group, t.position
		FROM recipe_tags rt JOIN tags t ON t.id = rt.tag_id
		WHERE rt.recipe_id = ?
		ORDER BY t.tag_group, t.position`, recipeID)
	if err != nil {
		return nil, fmt.Errorf("kategorien laden: %w", err)
	}
	defer rows.Close()

	tags := []Tag{}
	for rows.Next() {
		var tag Tag
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.Group, &tag.Position); err != nil {
			return nil, fmt.Errorf("kategorien laden: %w", err)
		}
		tags = append(tags, tag)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("kategorien laden: %w", err)
	}
	return tags, nil
}
