package recipes

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/apierr"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/database"
)

// Gänge vorschlagen (Schritt 9.1): Für Rezepte ohne Gang schlägt der Server
// einen vor – nach Wörtern im Titel, Zutaten und vorhandenen Kategorien.
// Bewusst einfache Regeln statt eines KI-Modells: Der Nutzer prüft die
// Vorschläge ohnehin und übernimmt sie gesammelt (nur Weboberfläche).

// CourseSuggestion ist ein Vorschlag für ein Rezept. Der Grund steht als
// Code mit Parametern in ReasonCode/ReasonParams (die Oberfläche übersetzt
// ihn); Reason ist der englische Text dazu.
type CourseSuggestion struct {
	RecipeID     int64          `json:"recipe_id"`
	Title        string         `json:"title"`
	Thumbnail    string         `json:"thumbnail"`
	TagID        int64          `json:"tag_id"` // vorgeschlagener Gang
	Course       string         `json:"course"`
	Reason       string         `json:"reason"`
	ReasonCode   string         `json:"reason_code"`
	ReasonParams map[string]any `json:"reason_params,omitempty"`
}

// CourseAssignment ordnet einem Rezept einen Gang zu.
type CourseAssignment struct {
	RecipeID int64 `json:"recipe_id"`
	TagID    int64 `json:"tag_id"`
}

// ErrNoCourseGroup: Es ist keine Kategoriegruppe als Gang markiert.
var ErrNoCourseGroup = apierr.New("no_course_group")

// courseRules: Wörter im Titel → Gang (course_key). Die erste passende
// Regel gewinnt; längere, eindeutige Wörter stehen daher vorn
// („kartoffelsalat“ ist Salat, nicht Beilage). Verglichen wird mit
// database.Plain (ohne Akzente).
//
// words sind deutsche Stichwörter: Ab 4 Buchstaben zählen sie auch als
// Teil eines Worts (Zusammensetzungen wie „Apfelkuchen“). exact gilt für
// die anderen Sprachen und nur als ganzes Wort bzw. ganze Wortfolge – sonst
// wäre „meatloaf“ ein Brot und „steak tartare“ ein Kuchen.
var courseRules = []struct {
	course string
	words  []string
	exact  []string
}{
	{"cake", []string{"kuchen", "torte", "tarte", "cheesecake", "strudel", "gugelhupf", "blechkuchen"},
		[]string{"cake", "pie", "gateau", "tarta", "pastel", "bizcocho", "torta", "crostata", "ciambella"}},
	{"pastry", []string{"kekse", "keks", "platzchen", "muffin", "cookie", "brownie", "waffel", "hefezopf", "schnecke",
		"croissant", "scone", "cupcake", "makrone", "berliner", "krapfen"},
		[]string{"cookies", "biscuit", "biscuits", "pastry", "pastries", "waffle", "waffles", "macaron", "macarons",
			"madeleine", "madeleines", "galletas", "magdalenas", "biscotti", "cornetti"}},
	{"bread", []string{"brotchen", "brot", "baguette", "focaccia", "ciabatta", "fladen", "bagel", "laugen"},
		[]string{"bread", "pain", "pan", "pane", "panini"}},
	{"soup", []string{"suppe", "eintopf", "bruhe", "chowder", "ramen", "gulaschsuppe", "minestrone"},
		[]string{"soup", "stew", "soupe", "veloute", "potage", "sopa", "crema", "guiso", "potaje", "gazpacho",
			"zuppa", "brodo", "vellutata", "stufato", "minestra"}},
	{"salad", []string{"salat", "bowl"}, []string{"salad", "salade", "ensalada", "insalata"}},
	{"dessert", []string{"dessert", "pudding", "mousse", "tiramisu", "panna cotta", "parfait", "crumble", "grutze",
		"kompott", "eis", "creme brulee", "milchreis", "quarkspeise"},
		[]string{"ice cream", "sorbet", "flan", "postre", "helado", "natillas", "arroz con leche", "gelato", "semifreddo"}},
	{"starter", []string{"vorspeise", "carpaccio", "bruschetta", "antipasti", "tatar"},
		[]string{"starter", "appetizer", "entree", "entrante", "antipasto"}},
	{"snack", []string{"snack", "dip", "fingerfood", "happchen", "chips", "wrap"}, []string{"tapa", "tapas", "spuntino"}},
	{"side", []string{"beilage", "pommes", "puree", "kroketten", "rosti", "spatzle", "knodel"},
		[]string{"side dish", "mash", "fries", "gratin", "contorno"}},
}

// Zutaten-Stichwörter für Gebäck ohne passenden Titel (alle Sprachen).
var (
	flourWords  = []string{"mehl", "flour", "farine", "harina", "farina"}
	leavenWords = []string{"backpulver", "hefe", "natron", "baking powder", "baking soda", "yeast",
		"levure", "bicarbonate", "levadura", "polvo de hornear", "bicarbonato", "lievito"}
	meatWords = []string{"hackfleisch", "hahnchen", "schwein", "rind", "speck", "schinken",
		"beef", "chicken", "pork", "bacon", "ham", "mince", "boeuf", "poulet", "porc", "lardon", "jambon",
		"carne", "pollo", "cerdo", "tocino", "jamon", "manzo", "maiale", "pancetta", "prosciutto", "guanciale"}
	sugarWords = []string{"zucker", "sugar", "sucre", "azucar", "zucchero"}
	// Kategorie „Süß“ des Nutzers (Gruppe Geschmack) in den Startlisten.
	sweetTags = []string{"suss", "sweet", "sucre", "dulce", "dolce"}
)

// suggestCourse liefert den vorgeschlagenen Gang (course_key) und den Grund.
func suggestCourse(title string, ingredients []string, tags []string) (string, *apierr.Error) {
	plain := database.Plain(title)
	words := strings.FieldsFunc(plain, func(r rune) bool { return !unicode.IsLetter(r) })
	for _, rule := range courseRules {
		for _, kw := range rule.words {
			if titleHas(plain, words, kw) {
				return rule.course, apierr.New("course_reason_title", "word", kw)
			}
		}
		for _, kw := range rule.exact {
			if hasPhrase(words, kw) {
				return rule.course, apierr.New("course_reason_title", "word", kw)
			}
		}
	}
	has := func(words []string) bool {
		for _, ing := range ingredients {
			f := database.Plain(ing)
			for _, w := range words {
				if strings.Contains(f, w) {
					return true
				}
			}
		}
		return false
	}
	sweet := slices.ContainsFunc(tags, func(t string) bool { return slices.Contains(sweetTags, database.Plain(t)) })
	if has(flourWords) && has(leavenWords) && !has(meatWords) {
		if sweet || has(sugarWords) {
			return "cake", apierr.New("course_reason_baking")
		}
		return "bread", apierr.New("course_reason_bread")
	}
	if sweet {
		return "dessert", apierr.New("course_reason_sweet")
	}
	return "main", apierr.New("course_reason_none")
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

// hasPhrase: Kommt die Wortfolge phrase („ice cream“) als ganze Wörter vor?
func hasPhrase(words []string, phrase string) bool {
	want := strings.Fields(phrase)
	for i := 0; i+len(want) <= len(words); i++ {
		if slices.Equal(words[i:i+len(want)], want) {
			return true
		}
	}
	return false
}

// courseGroup liefert die Gänge der Gang-Gruppe mit Schlüssel (course_key → Kategorie).
func (s *Store) courseGroup(ctx context.Context) (map[string]Tag, error) {
	groups, err := s.ListTags(ctx)
	if err != nil {
		return nil, err
	}
	for _, g := range groups {
		if g.Role == RoleCourse {
			byKey := make(map[string]Tag, len(g.Tags))
			for _, t := range g.Tags {
				if t.CourseKey != "" {
					byKey[t.CourseKey] = t
				}
			}
			return byKey, nil
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
		key, reason := suggestCourse(sug.Title, strings.Split(ings, "\x1f"), strings.Split(tags, "\x1f"))
		tag, ok := courses[key]
		if !ok {
			continue
		}
		sug.TagID, sug.Course = tag.ID, tag.Name
		sug.Reason, sug.ReasonCode, sug.ReasonParams = reason.Message(), reason.Code, reason.Params
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
				return apierr.Field("tag_id", "not_a_course", "id", a.TagID)
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
