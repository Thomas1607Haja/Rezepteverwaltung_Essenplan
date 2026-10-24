package recipes

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/database"
)

// Verwaltung der Kategorien und Gruppen (Schritt 8.C, nur Weboberfläche;
// apikeys.Require sperrt die Endpunkte für die App).
//
// Abgleich mit der App: Die Kategorieliste kommt bei jedem Abgleich
// vollständig mit. Die Rezepte tragen ihre Kategorien aber mit Name und
// Gruppe in sich; deshalb bekommt jedes betroffene Rezept bei Umbenennen,
// Verschieben in eine andere Gruppe, Löschen und Zusammenführen ein neues
// updated_at und kommt so mit dem nächsten Abgleich neu. Reine
// Umsortierungen ändern an den Rezepten nichts.

// maxTagNameLength begrenzt Namen von Kategorien und Gruppen.
const maxTagNameLength = 50

// ErrTagNotFound: Die Kategorie oder Gruppe gibt es nicht.
var ErrTagNotFound = errors.New("kategorie oder gruppe nicht gefunden")

// ErrGroupNotEmpty: Eine Gruppe mit Kategorien soll gelöscht werden, ohne
// zu sagen, was mit ihnen geschieht.
var ErrGroupNotEmpty = errors.New("die gruppe enthält noch kategorien – verschieben oder mitlöschen")

// AdminTag ist eine Kategorie in der Verwaltung, mit der Zahl ihrer Rezepte.
type AdminTag struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Position    int    `json:"position"`
	RecipeCount int    `json:"recipe_count"`
}

// AdminGroup ist eine Gruppe in der Verwaltung, auch leer.
type AdminGroup struct {
	ID       int64      `json:"id"`
	Name     string     `json:"name"`
	Position int        `json:"position"`
	Tags     []AdminTag `json:"tags"`
}

// AdminTags liefert alle Gruppen (auch leere) mit ihren Kategorien.
func (s *Store) AdminTags(ctx context.Context) ([]AdminGroup, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT g.id, g.name, g.position, t.id, t.name, t.position,
		       (SELECT count(*) FROM recipe_tags rt WHERE rt.tag_id = t.id)
		FROM tag_groups g LEFT JOIN tags t ON t.group_id = g.id
		ORDER BY g.position, g.id, t.position, t.id`)
	if err != nil {
		return nil, fmt.Errorf("kategorien laden: %w", err)
	}
	defer rows.Close()

	groups := []AdminGroup{}
	for rows.Next() {
		var g AdminGroup
		// Bei einer leeren Gruppe liefert der LEFT JOIN NULL für die Kategorie.
		var tagID sql.NullInt64
		var tagName sql.NullString
		var tagPos sql.NullInt64
		var count int
		if err := rows.Scan(&g.ID, &g.Name, &g.Position, &tagID, &tagName, &tagPos, &count); err != nil {
			return nil, fmt.Errorf("kategorien laden: %w", err)
		}
		if len(groups) == 0 || groups[len(groups)-1].ID != g.ID {
			g.Tags = []AdminTag{}
			groups = append(groups, g)
		}
		if tagID.Valid {
			last := &groups[len(groups)-1]
			last.Tags = append(last.Tags, AdminTag{ID: tagID.Int64, Name: tagName.String, Position: int(tagPos.Int64), RecipeCount: count})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("kategorien laden: %w", err)
	}
	return groups, nil
}

// cleanName prüft und kürzt einen Namen.
func cleanName(name string) (string, error) {
	name = strings.Join(strings.Fields(name), " ")
	switch {
	case name == "":
		return "", &ValidationError{"name", "name fehlt"}
	case utf8.RuneCountInString(name) > maxTagNameLength:
		return "", &ValidationError{"name", fmt.Sprintf("name zu lang (höchstens %d zeichen)", maxTagNameLength)}
	}
	return name, nil
}

// checkNameFree meldet einen Fehler, wenn es in table (tags oder
// tag_groups) schon einen Eintrag mit diesem Namen gibt – ohne Rücksicht
// auf Groß-/Kleinschreibung und Umlaute. exceptID ist der Eintrag selbst
// (beim Umbenennen).
func checkNameFree(ctx context.Context, tx *sql.Tx, table, name string, exceptID int64) error {
	rows, err := tx.QueryContext(ctx, "SELECT id, name FROM "+table)
	if err != nil {
		return err
	}
	defer rows.Close()
	want := database.Fold(name)
	for rows.Next() {
		var id int64
		var other string
		if err := rows.Scan(&id, &other); err != nil {
			return err
		}
		if id != exceptID && database.Fold(other) == want {
			what := "eine kategorie"
			if table == "tag_groups" {
				what = "eine gruppe"
			}
			return &ValidationError{"name", fmt.Sprintf("es gibt schon %s „%s“", what, other)}
		}
	}
	return rows.Err()
}

// touchRecipesWithTags setzt updated_at aller Rezepte mit einer der
// Kategorien, damit sie mit dem nächsten Abgleich neu in die App kommen.
func touchRecipesWithTags(ctx context.Context, tx *sql.Tx, now string, tagIDs ...int64) error {
	if len(tagIDs) == 0 {
		return nil
	}
	args := []any{now}
	for _, id := range tagIDs {
		args = append(args, id)
	}
	_, err := tx.ExecContext(ctx, `
		UPDATE recipes SET updated_at = ?
		WHERE id IN (SELECT recipe_id FROM recipe_tags WHERE tag_id IN (`+placeholders(len(tagIDs))+`))`, args...)
	return err
}

// tagIDsOfGroup liefert die Kategorien einer Gruppe.
func tagIDsOfGroup(ctx context.Context, tx *sql.Tx, groupID int64) ([]int64, error) {
	rows, err := tx.QueryContext(ctx, "SELECT id FROM tags WHERE group_id = ? ORDER BY position, id", groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// exists prüft, ob es in table einen Eintrag mit id gibt.
func exists(ctx context.Context, tx *sql.Tx, table string, id int64) error {
	var n int
	if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM "+table+" WHERE id = ?", id).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		return ErrTagNotFound
	}
	return nil
}

// inTx führt fn in einer Transaktion aus; now ist der Zeitstempel für
// updated_at und deletions.
func (s *Store) inTx(ctx context.Context, what string, fn func(tx *sql.Tx, now string) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: %w", what, err)
	}
	defer tx.Rollback()
	if err := fn(tx, database.FormatTime(s.now())); err != nil {
		var verr *ValidationError
		if errors.As(err, &verr) || errors.Is(err, ErrTagNotFound) || errors.Is(err, ErrGroupNotEmpty) {
			return err
		}
		return fmt.Errorf("%s: %w", what, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%s: %w", what, err)
	}
	return nil
}

// CreateTagGroup legt eine Gruppe am Ende an.
func (s *Store) CreateTagGroup(ctx context.Context, name string) (int64, error) {
	name, err := cleanName(name)
	if err != nil {
		return 0, err
	}
	var id int64
	err = s.inTx(ctx, "gruppe anlegen", func(tx *sql.Tx, now string) error {
		if err := checkNameFree(ctx, tx, "tag_groups", name, 0); err != nil {
			return err
		}
		res, err := tx.ExecContext(ctx, `
			INSERT INTO tag_groups (name, position, updated_at)
			VALUES (?, (SELECT coalesce(max(position), 0) + 1 FROM tag_groups), ?)`, name, now)
		if err != nil {
			return err
		}
		id, err = res.LastInsertId()
		return err
	})
	return id, err
}

// RenameTagGroup benennt eine Gruppe um.
func (s *Store) RenameTagGroup(ctx context.Context, id int64, name string) error {
	name, err := cleanName(name)
	if err != nil {
		return err
	}
	return s.inTx(ctx, "gruppe umbenennen", func(tx *sql.Tx, now string) error {
		if err := exists(ctx, tx, "tag_groups", id); err != nil {
			return err
		}
		if err := checkNameFree(ctx, tx, "tag_groups", name, id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "UPDATE tag_groups SET name = ?, updated_at = ? WHERE id = ?", name, now, id); err != nil {
			return err
		}
		ids, err := tagIDsOfGroup(ctx, tx, id)
		if err != nil {
			return err
		}
		return touchRecipesWithTags(ctx, tx, now, ids...)
	})
}

// OrderTagGroups setzt die Reihenfolge der Gruppen; ids muss alle Gruppen
// genau einmal enthalten.
func (s *Store) OrderTagGroups(ctx context.Context, ids []int64) error {
	return s.inTx(ctx, "gruppen sortieren", func(tx *sql.Tx, now string) error {
		var n int
		if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM tag_groups").Scan(&n); err != nil {
			return err
		}
		if len(ids) != n || !distinct(ids) {
			return &ValidationError{"ids", "alle gruppen genau einmal angeben"}
		}
		for i, id := range ids {
			res, err := tx.ExecContext(ctx, "UPDATE tag_groups SET position = ?, updated_at = ? WHERE id = ?", i+1, now, id)
			if err != nil {
				return err
			}
			if k, _ := res.RowsAffected(); k == 0 {
				return &ValidationError{"ids", fmt.Sprintf("gruppe %d gibt es nicht", id)}
			}
		}
		return nil
	})
}

// DeleteTagGroup löscht eine Gruppe. Hat sie Kategorien, kommen sie nach
// moveTo (> 0, ans Ende) oder werden mit gelöscht (withTags); sonst
// ErrGroupNotEmpty.
func (s *Store) DeleteTagGroup(ctx context.Context, id, moveTo int64, withTags bool) error {
	return s.inTx(ctx, "gruppe löschen", func(tx *sql.Tx, now string) error {
		if err := exists(ctx, tx, "tag_groups", id); err != nil {
			return err
		}
		ids, err := tagIDsOfGroup(ctx, tx, id)
		if err != nil {
			return err
		}
		switch {
		case len(ids) == 0:
		case moveTo > 0:
			if moveTo == id {
				return &ValidationError{"move_to", "die kategorien müssen in eine andere gruppe"}
			}
			if err := exists(ctx, tx, "tag_groups", moveTo); err != nil {
				return err
			}
			if err := appendToGroup(ctx, tx, now, moveTo, ids); err != nil {
				return err
			}
		case withTags:
			for _, tagID := range ids {
				if err := deleteTag(ctx, tx, now, tagID); err != nil {
					return err
				}
			}
		default:
			return ErrGroupNotEmpty
		}
		_, err = tx.ExecContext(ctx, "DELETE FROM tag_groups WHERE id = ?", id)
		return err
	})
}

// appendToGroup hängt Kategorien in ihrer Reihenfolge ans Ende einer Gruppe.
func appendToGroup(ctx context.Context, tx *sql.Tx, now string, groupID int64, tagIDs []int64) error {
	if err := touchRecipesWithTags(ctx, tx, now, tagIDs...); err != nil {
		return err
	}
	for _, tagID := range tagIDs {
		_, err := tx.ExecContext(ctx, `
			UPDATE tags SET group_id = ?, updated_at = ?,
			       position = (SELECT coalesce(max(position), 0) + 1 FROM tags WHERE group_id = ?)
			WHERE id = ?`, groupID, now, groupID, tagID)
		if err != nil {
			return err
		}
	}
	return nil
}

// CreateTag legt eine Kategorie am Ende der Gruppe an.
func (s *Store) CreateTag(ctx context.Context, name string, groupID int64) (Tag, error) {
	name, err := cleanName(name)
	if err != nil {
		return Tag{}, err
	}
	var tag Tag
	err = s.inTx(ctx, "kategorie anlegen", func(tx *sql.Tx, now string) error {
		if err := exists(ctx, tx, "tag_groups", groupID); err != nil {
			return err
		}
		if err := checkNameFree(ctx, tx, "tags", name, 0); err != nil {
			return err
		}
		res, err := tx.ExecContext(ctx, `
			INSERT INTO tags (name, group_id, position, updated_at)
			VALUES (?, ?, (SELECT coalesce(max(position), 0) + 1 FROM tags WHERE group_id = ?), ?)`,
			name, groupID, groupID, now)
		if err != nil {
			return err
		}
		id, err := res.LastInsertId()
		if err != nil {
			return err
		}
		return tx.QueryRowContext(ctx, `
			SELECT t.id, t.name, g.name, g.id, t.position FROM tags t `+tagJoin+` WHERE t.id = ?`, id).
			Scan(&tag.ID, &tag.Name, &tag.Group, &tag.GroupID, &tag.Position)
	})
	return tag, err
}

// UpdateTag benennt eine Kategorie um und/oder verschiebt sie in die
// Gruppe groupID (dort ans Ende).
func (s *Store) UpdateTag(ctx context.Context, id int64, name string, groupID int64) error {
	name, err := cleanName(name)
	if err != nil {
		return err
	}
	return s.inTx(ctx, "kategorie ändern", func(tx *sql.Tx, now string) error {
		var oldName string
		var oldGroup int64
		err := tx.QueryRowContext(ctx, "SELECT name, group_id FROM tags WHERE id = ?", id).Scan(&oldName, &oldGroup)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrTagNotFound
		}
		if err != nil {
			return err
		}
		if err := exists(ctx, tx, "tag_groups", groupID); err != nil {
			return err
		}
		if err := checkNameFree(ctx, tx, "tags", name, id); err != nil {
			return err
		}
		if name == oldName && groupID == oldGroup {
			return nil
		}
		if err := touchRecipesWithTags(ctx, tx, now, id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "UPDATE tags SET name = ?, updated_at = ? WHERE id = ?", name, now, id); err != nil {
			return err
		}
		if groupID != oldGroup {
			return appendToGroup(ctx, tx, now, groupID, []int64{id})
		}
		return nil
	})
}

// OrderTags setzt Inhalt und Reihenfolge einer Gruppe: Die Kategorien ids
// gehören danach in dieser Reihenfolge zur Gruppe groupID – auch solche,
// die vorher in einer anderen Gruppe waren (Ziehen zwischen Gruppen). Alle
// bisherigen Kategorien der Gruppe müssen dabei sein.
func (s *Store) OrderTags(ctx context.Context, groupID int64, ids []int64) error {
	return s.inTx(ctx, "kategorien sortieren", func(tx *sql.Tx, now string) error {
		if err := exists(ctx, tx, "tag_groups", groupID); err != nil {
			return err
		}
		if !distinct(ids) {
			return &ValidationError{"ids", "jede kategorie nur einmal angeben"}
		}
		current, err := tagIDsOfGroup(ctx, tx, groupID)
		if err != nil {
			return err
		}
		inList := make(map[int64]bool, len(ids))
		for _, id := range ids {
			inList[id] = true
		}
		for _, id := range current {
			if !inList[id] {
				return &ValidationError{"ids", fmt.Sprintf("kategorie %d fehlt – erst in eine andere gruppe verschieben", id)}
			}
		}
		for i, id := range ids {
			var oldGroup int64
			err := tx.QueryRowContext(ctx, "SELECT group_id FROM tags WHERE id = ?", id).Scan(&oldGroup)
			if errors.Is(err, sql.ErrNoRows) {
				return &ValidationError{"ids", fmt.Sprintf("kategorie %d gibt es nicht", id)}
			}
			if err != nil {
				return err
			}
			if oldGroup != groupID {
				if err := touchRecipesWithTags(ctx, tx, now, id); err != nil {
					return err
				}
			}
			if _, err := tx.ExecContext(ctx, "UPDATE tags SET group_id = ?, position = ?, updated_at = ? WHERE id = ?",
				groupID, i+1, now, id); err != nil {
				return err
			}
		}
		return nil
	})
}

// DeleteTag löscht eine Kategorie; die Rezepte verlieren nur die Zuordnung.
func (s *Store) DeleteTag(ctx context.Context, id int64) error {
	return s.inTx(ctx, "kategorie löschen", func(tx *sql.Tx, now string) error {
		if err := exists(ctx, tx, "tags", id); err != nil {
			return err
		}
		return deleteTag(ctx, tx, now, id)
	})
}

func deleteTag(ctx context.Context, tx *sql.Tx, now string, id int64) error {
	if err := touchRecipesWithTags(ctx, tx, now, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM recipe_tags WHERE tag_id = ?", id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM tags WHERE id = ?", id); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO deletions (table_name, record_id, deleted_at) VALUES ('tags', ?, ?)`, id, now)
	return err
}

// MergeTag führt die Kategorie id in into zusammen: Alle Rezepte von id
// bekommen into, danach wird id gelöscht.
func (s *Store) MergeTag(ctx context.Context, id, into int64) error {
	if id == into {
		return &ValidationError{"into", "eine kategorie lässt sich nicht mit sich selbst zusammenführen"}
	}
	return s.inTx(ctx, "kategorien zusammenführen", func(tx *sql.Tx, now string) error {
		if err := exists(ctx, tx, "tags", id); err != nil {
			return err
		}
		if err := exists(ctx, tx, "tags", into); err != nil {
			return err
		}
		// OR IGNORE: Rezepte, die schon beide haben, behalten into einmal.
		if _, err := tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO recipe_tags (recipe_id, tag_id, updated_at)
			SELECT recipe_id, ?, ? FROM recipe_tags WHERE tag_id = ?`, into, now, id); err != nil {
			return err
		}
		return deleteTag(ctx, tx, now, id)
	})
}

// distinct prüft, ob jede ID nur einmal vorkommt.
func distinct(ids []int64) bool {
	seen := make(map[int64]bool, len(ids))
	for _, id := range ids {
		if seen[id] {
			return false
		}
		seen[id] = true
	}
	return true
}
