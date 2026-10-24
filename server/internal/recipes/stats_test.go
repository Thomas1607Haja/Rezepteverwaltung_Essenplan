package recipes

import (
	"context"
	"testing"
	"time"
)

func TestStatistics(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	s.now = func() time.Time { return time.Date(2026, 10, 24, 12, 0, 0, 0, time.Local) }

	ids := map[string]int64{}
	for _, r := range []Recipe{
		{Title: "Spaghetti", BaseServings: 4, IsFavorite: true, Tags: []Tag{{ID: tagID(t, s, "Pasta")}}},
		{Title: "Risotto", BaseServings: 4, IsFavorite: true},
		{Title: "Steak", BaseServings: 2, Tags: []Tag{{ID: tagID(t, s, "Rind")}}},
		{Title: "Neu", BaseServings: 2, IsFavorite: true},
	} {
		created, err := s.Create(ctx, r)
		if err != nil {
			t.Fatal(err)
		}
		ids[r.Title] = created.ID
	}
	if _, err := s.db.Exec(`UPDATE recipes SET image_kind = 'illustration', image = 'x.jpg' WHERE id = ?`, ids["Steak"]); err != nil {
		t.Fatal(err)
	}
	add := func(date string, recipe int64, note string) {
		t.Helper()
		var rid any
		if recipe > 0 {
			rid = recipe
		}
		if _, err := s.db.Exec(`INSERT INTO meal_plan_entries (date, recipe_id, servings, note, created_at, updated_at)
			VALUES (?, ?, 2, ?, '2026-01-01T00:00:00.000Z', '2026-01-01T00:00:00.000Z')`, date, rid, note); err != nil {
			t.Fatal(err)
		}
	}
	add("2025-12-01", ids["Spaghetti"], "")
	add("2026-10-01", ids["Spaghetti"], "")
	add("2026-10-24", ids["Spaghetti"], "") // heute zählt
	add("2026-03-01", ids["Risotto"], "")   // Favorit, lange nicht dran
	add("2026-10-20", ids["Steak"], "")
	add("2026-10-30", ids["Steak"], "") // Zukunft zählt nicht
	add("2026-10-10", 0, "Essen gehen")
	add("2026-10-17", 0, "essen gehen ")
	add("2026-10-25", 0, "Essen gehen") // Zukunft

	st, err := s.Statistics(ctx, Period3Months)
	if err != nil {
		t.Fatal(err)
	}
	if st.Recipes != 4 || st.Planned != 3 || st.NeverPlanned != 1 || st.Illustrated != 1 || st.Favorites != 3 {
		t.Errorf("Zahlen: %+v", st)
	}
	if st.From == nil || *st.From != "2026-07-24" || st.Entries != 3 {
		t.Errorf("Zeitraum: from=%v entries=%d", st.From, st.Entries)
	}
	if len(st.Top) != 2 || st.Top[0].Title != "Spaghetti" || st.Top[0].Count != 2 || st.Top[1].Count != 1 {
		t.Errorf("Top 3 Monate: %+v", st.Top)
	}
	if len(st.FavoritesWaiting) != 2 || st.FavoritesWaiting[0].Title != "Neu" || st.FavoritesWaiting[1].Title != "Risotto" {
		t.Errorf("Lange nicht dran: %+v", st.FavoritesWaiting)
	}
	if len(st.WithoutRecipe) != 1 || st.WithoutRecipe[0].Count != 2 {
		t.Errorf("Ohne Rezept: %+v", st.WithoutRecipe)
	}
	if st.WithoutCategory != 2 || len(st.RecentlyAdded) != 4 || st.RecentlyAdded[0].Title != "Neu" {
		t.Errorf("ohne Kategorie %d, neu %+v", st.WithoutCategory, st.RecentlyAdded)
	}

	all, err := s.Statistics(ctx, PeriodAll)
	if err != nil {
		t.Fatal(err)
	}
	if all.From != nil || all.Top[0].Count != 3 || len(all.Top) != 3 {
		t.Errorf("gesamt: %+v", all.Top)
	}
	if _, err := s.Statistics(ctx, "1w"); err == nil {
		t.Error("unbekannter Zeitraum ohne Fehler")
	}
}
