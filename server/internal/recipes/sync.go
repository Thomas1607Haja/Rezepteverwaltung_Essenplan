package recipes

import (
	"context"
	"fmt"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/database"
)

// ChangedSince lädt alle Rezepte, deren updated_at nicht vor since liegt
// (Datenbankformat, leer = alle), mit Zutaten, Schritten und Kategorien –
// für den Abgleich mit der App.
//
// Anders als Get lädt es nicht jedes Rezept einzeln, sondern jede Tabelle
// mit einer einzigen Abfrage; bei einigen hundert Rezepten sind das vier
// statt über tausend Abfragen. PlanInfo und Family bleiben leer: Die App
// bekommt die Termine gesondert (PlanInfos) und bildet die Familien selbst
// aus parent_recipe_id.
func (s *Store) ChangedSince(ctx context.Context, since string) ([]Recipe, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, title, description, base_servings, prep_minutes, total_minutes,
		       source_url, source_name, image, thumbnail, image_kind, is_favorite, notes,
		       created_at, updated_at, parent_recipe_id
		FROM recipes WHERE updated_at >= ? ORDER BY id`, since)
	if err != nil {
		return nil, fmt.Errorf("rezepte für den abgleich laden: %w", err)
	}
	defer rows.Close()

	list := []Recipe{}
	// byID zeigt auf die Rezepte im Slice, damit die Bestandteile unten
	// direkt beim richtigen Rezept landen. Die Zeiger sind erst gültig,
	// wenn der Slice nicht mehr wächst – daher erst nach der Schleife.
	for rows.Next() {
		var r Recipe
		var createdAt, updatedAt string
		if err := rows.Scan(&r.ID, &r.Title, &r.Description, &r.BaseServings, &r.PrepMinutes, &r.TotalMinutes,
			&r.SourceURL, &r.SourceName, &r.Image, &r.Thumbnail, &r.ImageKind, &r.IsFavorite, &r.Notes,
			&createdAt, &updatedAt, &r.ParentRecipeID); err != nil {
			return nil, fmt.Errorf("rezepte für den abgleich laden: %w", err)
		}
		if r.CreatedAt, err = database.ParseTime(createdAt); err != nil {
			return nil, fmt.Errorf("rezept %d: %w", r.ID, err)
		}
		if r.UpdatedAt, err = database.ParseTime(updatedAt); err != nil {
			return nil, fmt.Errorf("rezept %d: %w", r.ID, err)
		}
		r.Ingredients, r.Steps, r.Tags, r.Family = []Ingredient{}, []Step{}, []Tag{}, []FamilyMember{}
		list = append(list, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rezepte für den abgleich laden: %w", err)
	}
	rows.Close()
	if len(list) == 0 {
		return list, nil
	}
	byID := make(map[int64]*Recipe, len(list))
	for i := range list {
		byID[list[i].ID] = &list[i]
	}

	// Die Unterabfrage wählt dieselben Rezepte wie oben aus.
	const changed = `(SELECT id FROM recipes WHERE updated_at >= ?)`

	rows, err = s.db.QueryContext(ctx, `
		SELECT recipe_id, id, group_name, quantity, unit, name, note, shopping_relevant
		FROM recipe_ingredients WHERE recipe_id IN `+changed+`
		ORDER BY recipe_id, position`, since)
	if err != nil {
		return nil, fmt.Errorf("zutaten für den abgleich laden: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var recipeID int64
		var ing Ingredient
		if err := rows.Scan(&recipeID, &ing.ID, &ing.GroupName, &ing.Quantity, &ing.Unit,
			&ing.Name, &ing.Note, &ing.ShoppingRelevant); err != nil {
			return nil, fmt.Errorf("zutaten für den abgleich laden: %w", err)
		}
		if r := byID[recipeID]; r != nil {
			r.Ingredients = append(r.Ingredients, ing)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("zutaten für den abgleich laden: %w", err)
	}
	rows.Close()

	rows, err = s.db.QueryContext(ctx, `
		SELECT recipe_id, id, text FROM recipe_steps WHERE recipe_id IN `+changed+`
		ORDER BY recipe_id, position`, since)
	if err != nil {
		return nil, fmt.Errorf("schritte für den abgleich laden: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var recipeID int64
		var step Step
		if err := rows.Scan(&recipeID, &step.ID, &step.Text); err != nil {
			return nil, fmt.Errorf("schritte für den abgleich laden: %w", err)
		}
		if r := byID[recipeID]; r != nil {
			r.Steps = append(r.Steps, step)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("schritte für den abgleich laden: %w", err)
	}
	rows.Close()

	rows, err = s.db.QueryContext(ctx, `
		SELECT rt.recipe_id, t.id, t.name, g.name, g.id, t.position
		FROM recipe_tags rt JOIN tags t ON t.id = rt.tag_id `+tagJoin+`
		WHERE rt.recipe_id IN `+changed+`
		ORDER BY rt.recipe_id, `+tagOrder, since)
	if err != nil {
		return nil, fmt.Errorf("kategorien für den abgleich laden: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var recipeID int64
		var tag Tag
		if err := rows.Scan(&recipeID, &tag.ID, &tag.Name, &tag.Group, &tag.GroupID, &tag.Position); err != nil {
			return nil, fmt.Errorf("kategorien für den abgleich laden: %w", err)
		}
		if r := byID[recipeID]; r != nil {
			r.Tags = append(r.Tags, tag)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("kategorien für den abgleich laden: %w", err)
	}
	return list, nil
}

// RecipePlanInfo sind die Termine eines Rezepts im Essensplan.
type RecipePlanInfo struct {
	RecipeID int64 `json:"recipe_id"`
	PlanInfo
}

// PlanInfos liefert die Termine aller Rezepte, die schon einmal eingeplant
// waren. Die App hat nur einen Ausschnitt des Plans und kann daraus den
// Statushinweis nicht selbst berechnen; die Liste kommt deshalb bei jedem
// Abgleich vollständig mit (ein paar Hundert kurze Zeilen).
func (s *Store) PlanInfos(ctx context.Context) ([]RecipePlanInfo, error) {
	today := s.today()
	rows, err := s.db.QueryContext(ctx, `
		SELECT recipe_id,
		       MIN(CASE WHEN date >= ? THEN date END),
		       MAX(CASE WHEN date < ? THEN date END),
		       COUNT(*)
		FROM meal_plan_entries WHERE recipe_id IS NOT NULL
		GROUP BY recipe_id ORDER BY recipe_id`, today, today)
	if err != nil {
		return nil, fmt.Errorf("termine laden: %w", err)
	}
	defer rows.Close()

	infos := []RecipePlanInfo{}
	for rows.Next() {
		var info RecipePlanInfo
		if err := rows.Scan(&info.RecipeID, &info.NextPlanned, &info.LastPlanned, &info.PlanCount); err != nil {
			return nil, fmt.Errorf("termine laden: %w", err)
		}
		infos = append(infos, info)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("termine laden: %w", err)
	}
	return infos, nil
}
