package recipes

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/database"
)

// Sortierungen der Rezeptliste. "Zuletzt eingeplant" und "am häufigsten
// eingeplant" folgen mit dem Essensplan in Phase 4.
const (
	SortName   = "name"   // alphabetisch nach Titel (Standard)
	SortNewest = "newest" // zuletzt angelegte zuerst
)

// ListFilter beschreibt, welche Rezepte die Liste in welcher Reihenfolge
// enthält. Der Nullwert liefert alle Rezepte nach Namen sortiert.
type ListFilter struct {
	// Query ist der Suchtext. Jedes Wort muss im Titel, in der Beschreibung
	// oder in einer Zutat vorkommen.
	Query string
	// TagIDs filtert nach Kategorien. Innerhalb einer Gruppe genügt eine
	// der Kategorien (Pasta oder Reis), über Gruppen hinweg müssen alle
	// passen (Vegetarisch und zugleich Pasta oder Reis).
	TagIDs        []int64
	FavoritesOnly bool
	Sort          string // SortName, SortNewest oder leer (= SortName)
}

// Summary ist die Kurzfassung eines Rezepts für Listen und Kacheln.
type Summary struct {
	ID           int64  `json:"id"`
	Title        string `json:"title"`
	PrepMinutes  *int   `json:"prep_minutes"`
	TotalMinutes *int   `json:"total_minutes"`
	Thumbnail    string `json:"thumbnail"`
	IsFavorite   bool   `json:"is_favorite"`
}

// List liefert die Rezepte, die zum Filter f passen.
func (s *Store) List(ctx context.Context, f ListFilter) ([]Summary, error) {
	var orderBy string
	switch f.Sort {
	case "", SortName:
		// r.id zuletzt, damit gleiche Titel immer gleich sortiert sind.
		orderBy = "fold(r.title), r.title, r.id"
	case SortNewest:
		orderBy = "r.created_at DESC, r.id DESC"
	default:
		return nil, &ValidationError{"sort", fmt.Sprintf("unbekannte sortierung %q", f.Sort)}
	}

	// Die Bedingungen werden einzeln gesammelt und am Ende mit AND
	// verbunden; args enthält die Werte für die ?-Platzhalter in derselben
	// Reihenfolge.
	var where []string
	var args []any

	for _, word := range strings.Fields(database.Fold(f.Query)) {
		pattern := "%" + escapeLike(word) + "%"
		where = append(where, `(fold(r.title) LIKE ? ESCAPE '\'
			OR fold(r.description) LIKE ? ESCAPE '\'
			OR EXISTS (SELECT 1 FROM recipe_ingredients i
			           WHERE i.recipe_id = r.id AND fold(i.name) LIKE ? ESCAPE '\'))`)
		args = append(args, pattern, pattern, pattern)
	}

	if f.FavoritesOnly {
		where = append(where, "r.is_favorite = 1")
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

	query := `SELECT r.id, r.title, r.prep_minutes, r.total_minutes, r.thumbnail, r.is_favorite
		FROM recipes r`
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
			&sum.Thumbnail, &sum.IsFavorite); err != nil {
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
func (s *Store) tagIDsByGroup(ctx context.Context, tagIDs []int64) (map[string][]int64, error) {
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
		"SELECT id, tag_group FROM tags WHERE id IN ("+placeholders(len(ids))+")", args...)
	if err != nil {
		return nil, fmt.Errorf("kategorien laden: %w", err)
	}
	defer rows.Close()

	groups := make(map[string][]int64)
	known := make(map[int64]bool)
	for rows.Next() {
		var id int64
		var group string
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
