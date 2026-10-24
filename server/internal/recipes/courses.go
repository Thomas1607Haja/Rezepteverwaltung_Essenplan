package recipes

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/database"
)

// Gänge vorschlagen (Schritt 9.1): Für Rezepte ohne Gang schlägt der Server
// einen vor – nach Wörtern im Titel, Zutaten und vorhandenen Kategorien.
// Bewusst einfache Regeln statt eines KI-Modells: Der Nutzer prüft die
// Vorschläge ohnehin und übernimmt sie gesammelt (nur Weboberfläche).

// CourseSuggestion ist ein Vorschlag für ein Rezept.
type CourseSuggestion struct {
	RecipeID  int64  `json:"recipe_id"`
	Title     string `json:"title"`
	Thumbnail string `json:"thumbnail"`
	TagID     int64  `json:"tag_id"` // vorgeschlagener Gang
	Course    string `json:"course"`
	Reason    string `json:"reason"` // z. B. „Titel enthält „Kuchen““
}

// CourseAssignment ordnet einem Rezept einen Gang zu.
type CourseAssignment struct {
	RecipeID int64 `json:"recipe_id"`
	TagID    int64 `json:"tag_id"`
}

// ErrNoCourseGroup: Es ist keine Kategoriegruppe als Gang markiert.
var ErrNoCourseGroup = errors.New("keine gang-gruppe – in der kategorienverwaltung eine gruppe als gang markieren")

// courseRule: Wörter im (gefalteten) Titel → Gang (Name wie in der
// Startliste). Die erste passende Regel gewinnt; längere, eindeutige
// Wörter stehen daher vorn („kartoffelsalat“ ist Salat, nicht Beilage).
var courseRules = []struct {
	course string
	words  []string
}{
	{"Kuchen & Torten", []string{"kuchen", "torte", "tarte", "cheesecake", "strudel", "gugelhupf", "blechkuchen"}},
	{"Gebäck", []string{"kekse", "keks", "platzchen", "muffin", "cookie", "brownie", "waffel", "hefezopf", "schnecke", "croissant", "scone", "cupcake", "makrone", "berliner", "krapfen"}},
	{"Brot", []string{"brotchen", "brot", "baguette", "focaccia", "ciabatta", "fladen", "bagel", "laugen"}},
	{"Suppe/Eintopf", []string{"suppe", "eintopf", "bruhe", "chowder", "ramen", "gulaschsuppe", "minestrone"}},
	{"Salat", []string{"salat", "bowl"}},
	{"Nachspeise", []string{"dessert", "pudding", "mousse", "tiramisu", "panna cotta", "parfait", "crumble", "grutze", "kompott", "eis", "creme brulee", "milchreis", "quarkspeise"}},
	{"Vorspeise", []string{"vorspeise", "carpaccio", "bruschetta", "antipasti", "tatar"}},
	{"Snack", []string{"snack", "dip", "fingerfood", "happchen", "chips", "wrap"}},
	{"Beilage", []string{"beilage", "pommes", "puree", "kroketten", "rosti", "spatzle", "knodel"}},
}

// suggestCourse liefert den vorgeschlagenen Gang (Name) und den Grund.
func suggestCourse(title string, ingredients []string, tags []string) (string, string) {
	folded := database.Fold(title)
	words := strings.FieldsFunc(folded, func(r rune) bool { return !unicode.IsLetter(r) })
	for _, rule := range courseRules {
		for _, kw := range rule.words {
			if titleHas(folded, words, kw) {
				return rule.course, fmt.Sprintf("Titel enthält „%s“", kw)
			}
		}
	}
	has := func(words ...string) bool {
		for _, ing := range ingredients {
			f := database.Fold(ing)
			for _, w := range words {
				if strings.Contains(f, w) {
					return true
				}
			}
		}
		return false
	}
	sweet := slices.ContainsFunc(tags, func(t string) bool { return database.Fold(t) == "suss" })
	if has("mehl") && has("backpulver", "hefe", "natron") && !has("hackfleisch", "hahnchen", "schwein", "rind", "speck", "schinken") {
		if sweet || has("zucker") {
			return "Kuchen & Torten", "Mehl, Triebmittel und Zucker"
		}
		return "Brot", "Mehl und Triebmittel ohne Zucker"
	}
	if sweet {
		return "Nachspeise", "Kategorie „Süß“"
	}
	return "Hauptgericht", "keine besonderen Hinweise"
}

// titleHas: Kommt das Stichwort kw im Titel vor? Mehrwortige Stichwörter
// („panna cotta“) im ganzen Titel, Stichwörter ab 4 Buchstaben auch als
// Teil eines Worts (deutsche Zusammensetzungen: „Apfelkuchen“,
// „Kartoffelsalat“), kürzere nur als ganzes Wort – sonst wäre „Reis“ ein Eis.
func titleHas(folded string, words []string, kw string) bool {
	if strings.Contains(kw, " ") {
		return strings.Contains(folded, kw)
	}
	for _, w := range words {
		if w == kw || (len([]rune(kw)) >= 4 && strings.Contains(w, kw)) {
			return true
		}
	}
	return false
}

// courseGroup liefert die Gänge der Gang-Gruppe (gefalteter Name → Kategorie).
func (s *Store) courseGroup(ctx context.Context) (map[string]Tag, error) {
	groups, err := s.ListTags(ctx)
	if err != nil {
		return nil, err
	}
	for _, g := range groups {
		if g.Role == RoleCourse {
			byName := make(map[string]Tag, len(g.Tags))
			for _, t := range g.Tags {
				byName[database.Fold(t.Name)] = t
			}
			return byName, nil
		}
	}
	// Leere Gang-Gruppe (ListTags lässt leere Gruppen weg) oder keine.
	return nil, ErrNoCourseGroup
}

// CourseSuggestions schlägt für alle Rezepte ohne Gang einen vor. Gibt es
// den vorgeschlagenen Gang nicht (umbenannt oder gelöscht), entfällt der
// Vorschlag.
func (s *Store) CourseSuggestions(ctx context.Context) ([]CourseSuggestion, error) {
	courses, err := s.courseGroup(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT r.id, r.title, r.thumbnail,
		       coalesce((SELECT group_concat(name, char(31)) FROM recipe_ingredients WHERE recipe_id = r.id), ''),
		       coalesce((SELECT group_concat(t.name, char(31)) FROM recipe_tags rt JOIN tags t ON t.id = rt.tag_id
		                 WHERE rt.recipe_id = r.id), '')
		FROM recipes r
		WHERE NOT EXISTS (SELECT 1 FROM recipe_tags rt JOIN tags t ON t.id = rt.tag_id
		                  JOIN tag_groups g ON g.id = t.group_id
		                  WHERE rt.recipe_id = r.id AND g.role = ?)
		ORDER BY fold(r.title), r.id`, RoleCourse)
	if err != nil {
		return nil, fmt.Errorf("gänge vorschlagen: %w", err)
	}
	defer rows.Close()
	list := []CourseSuggestion{}
	for rows.Next() {
		var sug CourseSuggestion
		var ings, tags string
		if err := rows.Scan(&sug.RecipeID, &sug.Title, &sug.Thumbnail, &ings, &tags); err != nil {
			return nil, fmt.Errorf("gänge vorschlagen: %w", err)
		}
		name, reason := suggestCourse(sug.Title, strings.Split(ings, "\x1f"), strings.Split(tags, "\x1f"))
		tag, ok := courses[database.Fold(name)]
		if !ok {
			continue
		}
		sug.TagID, sug.Course, sug.Reason = tag.ID, tag.Name, reason
		list = append(list, sug)
	}
	return list, rows.Err()
}

// AssignCourses fügt den Rezepten die Gänge hinzu (vorhandene Kategorien
// bleiben). Nur Kategorien der Gang-Gruppe sind erlaubt.
func (s *Store) AssignCourses(ctx context.Context, list []CourseAssignment) (int, error) {
	courses, err := s.courseGroup(ctx)
	if err != nil {
		return 0, err
	}
	allowed := map[int64]bool{}
	for _, t := range courses {
		allowed[t.ID] = true
	}
	n := 0
	err = s.inTx(ctx, "gänge zuordnen", func(tx *sql.Tx, now string) error {
		for _, a := range list {
			if !allowed[a.TagID] {
				return &ValidationError{"tag_id", fmt.Sprintf("kategorie %d ist kein gang", a.TagID)}
			}
			res, err := tx.ExecContext(ctx, `
				INSERT OR IGNORE INTO recipe_tags (recipe_id, tag_id, updated_at)
				SELECT id, ?, ? FROM recipes WHERE id = ?`, a.TagID, now, a.RecipeID)
			if err != nil {
				return err
			}
			if k, _ := res.RowsAffected(); k > 0 {
				n++
				if _, err := tx.ExecContext(ctx, "UPDATE recipes SET updated_at = ? WHERE id = ?", now, a.RecipeID); err != nil {
					return err
				}
			}
		}
		return nil
	})
	return n, err
}
