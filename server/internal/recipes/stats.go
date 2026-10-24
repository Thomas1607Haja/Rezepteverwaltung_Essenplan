package recipes

import (
	"context"
	"fmt"
	"time"
)

// Statistik (Schritt 8.D). Wird bei jedem Aufruf aus den Tabellen
// berechnet, nichts davon wird gespeichert. Regeln (docs/PLANUNG.md,
// Abschnitt 3 „Statistik und Statushinweis“):
//   - Häufigkeiten zählen Planeinträge bis einschließlich heute; Geplantes
//     zählt erst, wenn der Tag da ist (wie plan_count).
//   - „Noch nie eingeplant“ heißt: kein einziger Eintrag, auch kein
//     zukünftiger (passt zum Filter planned=never der Rezeptliste).
//   - Varianten zählen als eigene Rezepte, wie in der Rezeptliste.
//   - Einträge ohne Rezept („Essen gehen“) zählen nicht als Rezept,
//     erscheinen aber gesondert.

// Zeiträume der Statistik.
const (
	Period3Months  = "3m"
	Period12Months = "12m"
	PeriodAll      = "all"
)

// waitingAfterDays: Ab so vielen Tagen ohne Termin gilt ein Lieblingsgericht
// als „lange nicht dran“.
const waitingAfterDays = 56

// StatRecipe ist ein Rezept in einer Liste der Statistik.
type StatRecipe struct {
	ID          int64   `json:"id"`
	Title       string  `json:"title"`
	Thumbnail   string  `json:"thumbnail"`
	Count       int     `json:"count"`        // Termine im Zeitraum (nur bei top)
	LastPlanned *string `json:"last_planned"` // letzter Termin bis heute
	CreatedAt   string  `json:"created_at"`   // nur bei recently_added
}

// StatTag ist eine Kategorie mit der Zahl ihrer Rezepte.
type StatTag struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// StatGroup ist eine Gruppe der Kategorienverteilung.
type StatGroup struct {
	Name string    `json:"name"`
	Tags []StatTag `json:"tags"`
}

// StatNote zählt Einträge ohne Rezept mit gleichem Text.
type StatNote struct {
	Note  string `json:"note"`
	Count int    `json:"count"`
}

// Statistics ist die Antwort von GET /api/v1/statistics.
type Statistics struct {
	Period string  `json:"period"`
	From   *string `json:"from"` // erster Tag des Zeitraums, nil = gesamt
	Today  string  `json:"today"`

	Recipes      int `json:"recipes"`
	Planned      int `json:"planned"`       // mindestens ein Eintrag
	NeverPlanned int `json:"never_planned"` // kein Eintrag
	Illustrated  int `json:"illustrated"`
	Favorites    int `json:"favorites"`
	// Entries: Einträge mit Rezept im Zeitraum bis heute.
	Entries int `json:"entries"`

	Top              []StatRecipe `json:"top"`               // 20 häufigste im Zeitraum
	FavoritesWaiting []StatRecipe `json:"favorites_waiting"` // Favoriten, lange nicht dran
	RecentlyAdded    []StatRecipe `json:"recently_added"`    // 5 neueste
	Categories       []StatGroup  `json:"categories"`
	WithoutCategory  int          `json:"without_category"`
	WithoutRecipe    []StatNote   `json:"without_recipe"` // im Zeitraum bis heute
}

// periodStart liefert den ersten Tag des Zeitraums oder "" für gesamt.
func periodStart(period string, today time.Time) (string, error) {
	switch period {
	case PeriodAll:
		return "", nil
	case Period3Months:
		return today.AddDate(0, -3, 0).Format(time.DateOnly), nil
	case Period12Months:
		return today.AddDate(-1, 0, 0).Format(time.DateOnly), nil
	}
	return "", &ValidationError{"period", "erwartet 3m, 12m oder all"}
}

// Statistics berechnet die Statistik für den Zeitraum period.
func (s *Store) Statistics(ctx context.Context, period string) (Statistics, error) {
	todayDate, err := time.Parse(time.DateOnly, s.today())
	if err != nil {
		return Statistics{}, err
	}
	from, err := periodStart(period, todayDate)
	if err != nil {
		return Statistics{}, err
	}
	today := s.today()
	st := Statistics{Period: period, Today: today}
	if from != "" {
		st.From = &from
	}
	// Bedingung „im Zeitraum bis heute“ für meal_plan_entries e; from = ""
	// ist kleiner als jedes Datum und heißt damit „gesamt“.
	const inPeriod = `e.date >= ? AND e.date <= ?`

	err = s.db.QueryRowContext(ctx, `
		SELECT count(*),
		       count(CASE WHEN EXISTS (SELECT 1 FROM meal_plan_entries e WHERE e.recipe_id = r.id) THEN 1 END),
		       count(CASE WHEN r.image_kind = 'illustration' THEN 1 END),
		       count(CASE WHEN r.is_favorite = 1 THEN 1 END),
		       count(CASE WHEN NOT EXISTS (SELECT 1 FROM recipe_tags rt WHERE rt.recipe_id = r.id) THEN 1 END)
		FROM recipes r`).Scan(&st.Recipes, &st.Planned, &st.Illustrated, &st.Favorites, &st.WithoutCategory)
	if err != nil {
		return Statistics{}, fmt.Errorf("statistik: %w", err)
	}
	st.NeverPlanned = st.Recipes - st.Planned

	err = s.db.QueryRowContext(ctx, `
		SELECT count(*) FROM meal_plan_entries e WHERE e.recipe_id IS NOT NULL AND `+inPeriod, from, today).Scan(&st.Entries)
	if err != nil {
		return Statistics{}, fmt.Errorf("statistik: %w", err)
	}

	if st.Top, err = s.statRecipes(ctx, `
		SELECT r.id, r.title, r.thumbnail, count(*),
		       (SELECT max(date) FROM meal_plan_entries WHERE recipe_id = r.id AND date <= ?), ''
		FROM meal_plan_entries e JOIN recipes r ON r.id = e.recipe_id
		WHERE `+inPeriod+`
		GROUP BY r.id
		ORDER BY count(*) DESC, fold(r.title), r.id
		LIMIT 20`, today, from, today); err != nil {
		return Statistics{}, err
	}

	waitingBefore := todayDate.AddDate(0, 0, -waitingAfterDays).Format(time.DateOnly)
	if st.FavoritesWaiting, err = s.statRecipes(ctx, `
		SELECT r.id, r.title, r.thumbnail, 0, last, ''
		FROM (SELECT r.*, (SELECT max(date) FROM meal_plan_entries WHERE recipe_id = r.id AND date <= ?) AS last
		      FROM recipes r WHERE r.is_favorite = 1) r
		WHERE NOT EXISTS (SELECT 1 FROM meal_plan_entries e WHERE e.recipe_id = r.id AND e.date > ?)
		  AND (last IS NULL OR last < ?)
		ORDER BY last NULLS FIRST, fold(r.title), r.id
		LIMIT 10`, today, today, waitingBefore); err != nil {
		return Statistics{}, err
	}

	if st.RecentlyAdded, err = s.statRecipes(ctx, `
		SELECT r.id, r.title, r.thumbnail, 0,
		       (SELECT max(date) FROM meal_plan_entries WHERE recipe_id = r.id AND date <= ?), r.created_at
		FROM recipes r ORDER BY r.created_at DESC, r.id DESC LIMIT 5`, today); err != nil {
		return Statistics{}, err
	}

	if st.Categories, err = s.statCategories(ctx); err != nil {
		return Statistics{}, err
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT min(trim(e.note)), count(*)
		FROM meal_plan_entries e
		WHERE e.recipe_id IS NULL AND trim(e.note) <> '' AND `+inPeriod+`
		GROUP BY fold(trim(e.note))
		ORDER BY count(*) DESC, fold(min(trim(e.note)))
		LIMIT 10`, from, today)
	if err != nil {
		return Statistics{}, fmt.Errorf("statistik: %w", err)
	}
	defer rows.Close()
	st.WithoutRecipe = []StatNote{}
	for rows.Next() {
		var n StatNote
		if err := rows.Scan(&n.Note, &n.Count); err != nil {
			return Statistics{}, fmt.Errorf("statistik: %w", err)
		}
		st.WithoutRecipe = append(st.WithoutRecipe, n)
	}
	if err := rows.Err(); err != nil {
		return Statistics{}, fmt.Errorf("statistik: %w", err)
	}
	return st, nil
}

// statRecipes liest StatRecipe-Zeilen (id, title, thumbnail, count,
// last_planned, created_at).
func (s *Store) statRecipes(ctx context.Context, query string, args ...any) ([]StatRecipe, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("statistik: %w", err)
	}
	defer rows.Close()
	list := []StatRecipe{}
	for rows.Next() {
		var r StatRecipe
		if err := rows.Scan(&r.ID, &r.Title, &r.Thumbnail, &r.Count, &r.LastPlanned, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("statistik: %w", err)
		}
		list = append(list, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("statistik: %w", err)
	}
	return list, nil
}

// statCategories zählt die Rezepte je Kategorie, in der Reihenfolge der
// Verwaltung. Leere Gruppen fehlen.
func (s *Store) statCategories(ctx context.Context) ([]StatGroup, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT g.name, t.id, t.name, (SELECT count(*) FROM recipe_tags rt WHERE rt.tag_id = t.id)
		FROM tags t `+tagJoin+`
		ORDER BY `+tagOrder)
	if err != nil {
		return nil, fmt.Errorf("statistik: %w", err)
	}
	defer rows.Close()
	groups := []StatGroup{}
	for rows.Next() {
		var group string
		var tag StatTag
		if err := rows.Scan(&group, &tag.ID, &tag.Name, &tag.Count); err != nil {
			return nil, fmt.Errorf("statistik: %w", err)
		}
		if len(groups) == 0 || groups[len(groups)-1].Name != group {
			groups = append(groups, StatGroup{Name: group})
		}
		last := &groups[len(groups)-1]
		last.Tags = append(last.Tags, tag)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("statistik: %w", err)
	}
	return groups, nil
}
