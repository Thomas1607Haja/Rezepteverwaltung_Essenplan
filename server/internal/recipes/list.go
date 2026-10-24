package recipes

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/database"
)

// Sortierungen der Rezeptliste.
const (
	SortName        = "name"         // alphabetisch nach Titel (Standard)
	SortNewest      = "newest"       // zuletzt angelegte zuerst
	SortLastPlanned = "last_planned" // zuletzt gegessene zuerst, nie eingeplante am Ende
	SortMostPlanned = "most_planned" // am häufigsten eingeplante zuerst
)

// ListFilter beschreibt, welche Rezepte die Liste in welcher Reihenfolge
// enthält. Der Nullwert liefert alle Rezepte nach Namen sortiert.
type ListFilter struct {
	// Query ist der Suchtext. Jedes Wort muss im Titel, in der Beschreibung,
	// in einer Zutat oder in einem Zubereitungsschritt vorkommen.
	Query string
	// TagIDs filtert nach Kategorien. Innerhalb einer Gruppe genügt eine
	// der Kategorien (Pasta oder Reis), über Gruppen hinweg müssen alle
	// passen (Vegetarisch und zugleich Pasta oder Reis).
	TagIDs        []int64
	FavoritesOnly bool
	// IllustrationsOnly: nur Rezepte mit gezeichnetem Bild (zum Ersetzen
	// durch ein echtes Foto).
	IllustrationsOnly bool
	// NeverPlanned: nur Rezepte ohne jeden Planeintrag – auch ohne einen
	// zukünftigen (Ideen für Neues; Link aus der Statistik).
	NeverPlanned bool
	Sort         string // eine der Sort-Konstanten oder leer (= SortName)
}

// Summary ist die Kurzfassung eines Rezepts für Listen und Kacheln.
type Summary struct {
	ID           int64  `json:"id"`
	Title        string `json:"title"`
	PrepMinutes  *int   `json:"prep_minutes"`
	TotalMinutes *int   `json:"total_minutes"`
	Thumbnail    string `json:"thumbnail"`
	ImageKind    string `json:"image_kind"` // ImagePhoto, ImageIllustration oder leer
	IsFavorite   bool   `json:"is_favorite"`
	// Gesetzt bei einer Variante (Kennzeichen in der Übersicht).
	ParentRecipeID *int64 `json:"parent_recipe_id"`
	// Eingebettetes Struct: Seine Felder erscheinen im JSON direkt neben
	// den anderen (next_planned, last_planned, plan_count).
	PlanInfo
}

// PlanInfo sagt, wann ein Rezept im Essensplan steht. Daraus entsteht der
// Statushinweis („eingeplant in 5 Tagen“, „zuletzt vor 2 Wochen“). Wird
// bei jeder Abfrage berechnet, nicht gespeichert.
type PlanInfo struct {
	NextPlanned *string `json:"next_planned"` // nächster Termin ab heute, z. B. "2026-10-24"
	LastPlanned *string `json:"last_planned"` // letzter Termin vor heute
	// PlanCount: wie oft eingeplant, bis einschließlich heute – Geplantes
	// zählt erst, wenn der Tag da ist (wie in der Statistik).
	PlanCount int `json:"plan_count"`
}

// planInfoColumns berechnet PlanInfo je Rezept r. Die drei Platzhalter
// sind jeweils der heutige Tag (planInfoArgs). Der Index
// meal_plan_entries_recipe (recipe_id, date) macht die Unterabfragen schnell.
const planInfoColumns = `
	(SELECT MIN(date) FROM meal_plan_entries WHERE recipe_id = r.id AND date >= ?) AS next_planned,
	(SELECT MAX(date) FROM meal_plan_entries WHERE recipe_id = r.id AND date < ?) AS last_planned,
	(SELECT COUNT(*) FROM meal_plan_entries WHERE recipe_id = r.id AND date <= ?) AS plan_count`

// planInfoArgs liefert die Werte für die Platzhalter von planInfoColumns.
func planInfoArgs(today string) []any { return []any{today, today, today} }

// List liefert die Rezepte, die zum Filter f passen.
func (s *Store) List(ctx context.Context, f ListFilter) ([]Summary, error) {
	var orderBy string
	switch f.Sort {
	case "", SortName:
		// r.id zuletzt, damit gleiche Titel immer gleich sortiert sind.
		orderBy = "fold(r.title), r.title, r.id"
	case SortNewest:
		orderBy = "r.created_at DESC, r.id DESC"
	case SortLastPlanned:
		orderBy = "last_planned DESC NULLS LAST, fold(r.title), r.title, r.id"
	case SortMostPlanned:
		orderBy = "plan_count DESC, fold(r.title), r.title, r.id"
	default:
		return nil, &ValidationError{"sort", fmt.Sprintf("unbekannte sortierung %q", f.Sort)}
	}

	// Die Bedingungen werden einzeln gesammelt und am Ende mit AND
	// verbunden; args enthält die Werte für die ?-Platzhalter in derselben
	// Reihenfolge.
	var where []string
	today := s.today()
	args := planInfoArgs(today) // für planInfoColumns, die im SELECT zuerst kommen

	for _, word := range strings.Fields(database.Fold(f.Query)) {
		pattern := "%" + escapeLike(word) + "%"
		where = append(where, `(fold(r.title) LIKE ? ESCAPE '\'
			OR fold(r.description) LIKE ? ESCAPE '\'
			OR EXISTS (SELECT 1 FROM recipe_ingredients i
			           WHERE i.recipe_id = r.id AND fold(i.name) LIKE ? ESCAPE '\')
			OR EXISTS (SELECT 1 FROM recipe_steps st
			           WHERE st.recipe_id = r.id AND fold(st.text) LIKE ? ESCAPE '\'))`)
		args = append(args, pattern, pattern, pattern, pattern)
	}

	if f.FavoritesOnly {
		where = append(where, "r.is_favorite = 1")
	}
	if f.IllustrationsOnly {
		where = append(where, "r.image_kind = 'illustration'")
	}
	if f.NeverPlanned {
		where = append(where, "NOT EXISTS (SELECT 1 FROM meal_plan_entries e WHERE e.recipe_id = r.id)")
	}

	groups, err := s.tagIDsByGroup(ctx, f.TagIDs)
	if err != nil {
		return nil, err
	}
	for _, ids := range groups {
		where = append(where, `EXISTS (SELECT 1 FROM recipe_tags rt
			WHERE rt.recipe_id = r.id AND rt.tag_id IN (`+placeholders(len(ids))+`))`)
		for _, id := range ids {
			args = append(args, id)
		}
	}

	query := `SELECT r.id, r.title, r.prep_minutes, r.total_minutes, r.thumbnail, r.image_kind, r.is_favorite,
		r.parent_recipe_id,` +
		planInfoColumns + ` FROM recipes r`
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY " + orderBy

	// args... übergibt die Elemente des Slices als einzelne Argumente.
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("rezeptliste laden: %w", err)
	}
	defer rows.Close()

	list := []Summary{}
	for rows.Next() {
		var sum Summary
		if err := rows.Scan(&sum.ID, &sum.Title, &sum.PrepMinutes, &sum.TotalMinutes,
			&sum.Thumbnail, &sum.ImageKind, &sum.IsFavorite, &sum.ParentRecipeID,
			&sum.NextPlanned, &sum.LastPlanned, &sum.PlanCount); err != nil {
			return nil, fmt.Errorf("rezeptliste laden: %w", err)
		}
		list = append(list, sum)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rezeptliste laden: %w", err)
	}
	return list, nil
}

// tagIDsByGroup ordnet die Kategorie-IDs ihren Gruppen zu. Unbekannte IDs
// sind ein Eingabefehler; doppelte IDs werden zusammengefasst.
func (s *Store) tagIDsByGroup(ctx context.Context, tagIDs []int64) (map[int64][]int64, error) {
	if len(tagIDs) == 0 {
		return nil, nil
	}
	// Kopie sortieren (der Slice des Aufrufers bleibt unverändert), dann
	// entfernt slices.Compact direkt aufeinanderfolgende Doppelte.
	ids := slices.Clone(tagIDs)
	slices.Sort(ids)
	ids = slices.Compact(ids)

	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, group_id FROM tags WHERE id IN ("+placeholders(len(ids))+")", args...)
	if err != nil {
		return nil, fmt.Errorf("kategorien laden: %w", err)
	}
	defer rows.Close()

	groups := make(map[int64][]int64)
	known := make(map[int64]bool)
	for rows.Next() {
		var id int64
		var group int64
		if err := rows.Scan(&id, &group); err != nil {
			return nil, fmt.Errorf("kategorien laden: %w", err)
		}
		groups[group] = append(groups[group], id)
		known[id] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("kategorien laden: %w", err)
	}

	for _, id := range ids {
		if !known[id] {
			return nil, &ValidationError{"tag", fmt.Sprintf("kategorie %d gibt es nicht", id)}
		}
	}
	return groups, nil
}

// placeholders liefert n durch Komma getrennte ?-Platzhalter für IN (...).
func placeholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?, ", n), ", ")
}

// escapeLike schützt die Sonderzeichen von LIKE, damit z. B. eine Suche
// nach "100%" nicht als Muster "100 und beliebig viel danach" gilt.
// Passt zu ESCAPE '\' in der Abfrage.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func escapeLike(s string) string {
	return likeEscaper.Replace(s)
}

// FindDuplicates liefert die Rezepte mit demselben Titel (ohne Rücksicht
// auf Groß-/Kleinschreibung und Umlaute) oder derselben Quell-URL – für den
// Dublettenhinweis beim Datei-Import.
func (s *Store) FindDuplicates(ctx context.Context, title, sourceURL string) ([]Summary, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, title, prep_minutes, total_minutes, thumbnail, is_favorite, parent_recipe_id
		FROM recipes WHERE fold(title) = ? OR (? <> '' AND source_url = ?) ORDER BY id`,
		database.Fold(strings.TrimSpace(title)), sourceURL, sourceURL)
	if err != nil {
		return nil, fmt.Errorf("dubletten suchen: %w", err)
	}
	defer rows.Close()
	found := []Summary{}
	for rows.Next() {
		var sum Summary
		if err := rows.Scan(&sum.ID, &sum.Title, &sum.PrepMinutes, &sum.TotalMinutes,
			&sum.Thumbnail, &sum.IsFavorite, &sum.ParentRecipeID); err != nil {
			return nil, fmt.Errorf("dubletten suchen: %w", err)
		}
		found = append(found, sum)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dubletten suchen: %w", err)
	}
	return found, nil
}

// FindBySourceURL liefert die Rezepte mit genau dieser Quell-URL – für den
// Dublettenhinweis beim Import. Eine leere Adresse findet nichts.
func (s *Store) FindBySourceURL(ctx context.Context, sourceURL string) ([]Summary, error) {
	found := []Summary{}
	if sourceURL == "" {
		return found, nil
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, title, prep_minutes, total_minutes, thumbnail, is_favorite, parent_recipe_id
		FROM recipes WHERE source_url = ? ORDER BY id`, sourceURL)
	if err != nil {
		return nil, fmt.Errorf("dubletten suchen: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var sum Summary
		if err := rows.Scan(&sum.ID, &sum.Title, &sum.PrepMinutes, &sum.TotalMinutes,
			&sum.Thumbnail, &sum.IsFavorite, &sum.ParentRecipeID); err != nil {
			return nil, fmt.Errorf("dubletten suchen: %w", err)
		}
		found = append(found, sum)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dubletten suchen: %w", err)
	}
	return found, nil
}
