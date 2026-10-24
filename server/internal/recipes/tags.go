package recipes

import (
	"context"
	"fmt"
)

// tagJoin hängt an Abfragen auf tags (Name t) die Gruppe (Name g) an.
const tagJoin = `JOIN tag_groups g ON g.id = t.group_id`

// tagOrder sortiert Kategorien für die Anzeige: Gruppen in ihrer
// Reihenfolge, darin die Kategorien nach position. Erwartet tagJoin.
const tagOrder = `g.position, g.id, t.position, t.id`

// TagGroup ist eine Gruppe von Kategorien, z. B. "Hauptzutat".
type TagGroup struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Tags []Tag  `json:"tags"`
}

// ListTags liefert alle Kategorien nach Gruppen geordnet. Leere Gruppen
// fehlen – sie haben nur in der Verwaltung (AdminTags) eine Bedeutung.
func (s *Store) ListTags(ctx context.Context) ([]TagGroup, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT t.id, t.name, g.id, g.name, t.position
		FROM tags t `+tagJoin+`
		ORDER BY `+tagOrder)
	if err != nil {
		return nil, fmt.Errorf("kategorien laden: %w", err)
	}
	defer rows.Close()

	groups := []TagGroup{}
	for rows.Next() {
		var tag Tag
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.GroupID, &tag.Group, &tag.Position); err != nil {
			return nil, fmt.Errorf("kategorien laden: %w", err)
		}
		// Die Zeilen kommen nach Gruppen sortiert: Eine neue Gruppe beginnt,
		// sobald sich die Gruppe ändert.
		if len(groups) == 0 || groups[len(groups)-1].ID != tag.GroupID {
			groups = append(groups, TagGroup{ID: tag.GroupID, Name: tag.Group})
		}
		last := &groups[len(groups)-1] // Zeiger, damit append im Slice landet
		last.Tags = append(last.Tags, tag)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("kategorien laden: %w", err)
	}
	return groups, nil
}
