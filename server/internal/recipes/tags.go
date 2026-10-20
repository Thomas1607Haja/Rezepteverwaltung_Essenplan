package recipes

import (
	"context"
	"fmt"
)

// tagOrder sortiert Kategorien für die Anzeige: Gruppen in der Reihenfolge,
// in der sie angelegt wurden (kleinste ID der Gruppe), darin nach position.
// Erwartet den Tabellennamen t für tags.
const tagOrder = `(SELECT min(g.id) FROM tags g WHERE g.tag_group = t.tag_group), t.position, t.id`

// TagGroup ist eine Gruppe von Kategorien, z. B. "Hauptzutat".
type TagGroup struct {
	Name string `json:"name"`
	Tags []Tag  `json:"tags"`
}

// ListTags liefert alle Kategorien nach Gruppen geordnet.
func (s *Store) ListTags(ctx context.Context) ([]TagGroup, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT t.id, t.name, t.tag_group, t.position
		FROM tags t
		ORDER BY `+tagOrder)
	if err != nil {
		return nil, fmt.Errorf("kategorien laden: %w", err)
	}
	defer rows.Close()

	groups := []TagGroup{}
	for rows.Next() {
		var tag Tag
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.Group, &tag.Position); err != nil {
			return nil, fmt.Errorf("kategorien laden: %w", err)
		}
		// Die Zeilen kommen nach Gruppen sortiert: Eine neue Gruppe beginnt,
		// sobald sich der Gruppenname ändert.
		if len(groups) == 0 || groups[len(groups)-1].Name != tag.Group {
			groups = append(groups, TagGroup{Name: tag.Group})
		}
		last := &groups[len(groups)-1] // Zeiger, damit append im Slice landet
		last.Tags = append(last.Tags, tag)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("kategorien laden: %w", err)
	}
	return groups, nil
}
