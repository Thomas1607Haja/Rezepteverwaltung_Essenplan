package recipes

import (
	"context"
	"slices"
	"testing"
	"time"
)

// TestPlanInfo prüft Statushinweis-Angaben und die Sortierungen nach dem
// Essensplan. Heute ist der 21.10.2026.
func TestPlanInfo(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	s.now = func() time.Time { return time.Date(2026, 10, 21, 12, 0, 0, 0, time.Local) }
	seedList(t, s)

	ids := make(map[string]int64)
	all, err := s.List(ctx, ListFilter{})
	if err != nil {
		t.Fatal(err)
	}
	for _, sum := range all {
		ids[sum.Title] = sum.ID
	}

	// Spaghetti: zweimal in der Vergangenheit, einmal heute
	// Risotto: einmal vor langer Zeit, einmal in 5 Tagen
	// Steak: einmal gestern; Äpfel: nie
	plan := []struct {
		title, date string
	}{
		{"Spaghetti Napoli", "2026-09-01"},
		{"Spaghetti Napoli", "2026-10-10"},
		{"Spaghetti Napoli", "2026-10-21"},
		{"Pilzrisotto", "2026-03-01"},
		{"Pilzrisotto", "2026-10-26"},
		{"Steak mit Tomaten", "2026-10-20"},
	}
	for _, p := range plan {
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO meal_plan_entries (date, recipe_id, servings, created_at, updated_at)
			VALUES (?, ?, 4, '2026-10-01T10:00:00.000Z', '2026-10-01T10:00:00.000Z')`,
			p.date, ids[p.title])
		if err != nil {
			t.Fatal(err)
		}
	}

	str := func(p *string) string {
		if p == nil {
			return "-"
		}
		return *p
	}
	want := map[string]struct {
		next, last string
		count      int
	}{
		"Spaghetti Napoli":    {"2026-10-21", "2026-10-10", 3}, // heute zählt als nächster Termin
		"Pilzrisotto":         {"2026-10-26", "2026-03-01", 1}, // Geplantes zählt erst, wenn der Tag da ist
		"Steak mit Tomaten":   {"-", "2026-10-20", 1},
		"Äpfel im Schlafrock": {"-", "-", 0},
	}
	all, err = s.List(ctx, ListFilter{})
	if err != nil {
		t.Fatal(err)
	}
	for _, sum := range all {
		w := want[sum.Title]
		if str(sum.NextPlanned) != w.next || str(sum.LastPlanned) != w.last || sum.PlanCount != w.count {
			t.Errorf("%s: next=%s last=%s count=%d, erwartet %+v",
				sum.Title, str(sum.NextPlanned), str(sum.LastPlanned), sum.PlanCount, w)
		}
	}

	// Das volle Rezept liefert dieselben Angaben.
	r, err := s.Get(ctx, ids["Pilzrisotto"])
	if err != nil {
		t.Fatal(err)
	}
	if str(r.NextPlanned) != "2026-10-26" || r.PlanCount != 1 {
		t.Errorf("Get: next=%s count=%d", str(r.NextPlanned), r.PlanCount)
	}

	// Noch nie eingeplant: Risotto steht nur zukünftig drin und zählt nicht dazu.
	never, err := s.List(ctx, ListFilter{NeverPlanned: true})
	if err != nil || len(never) != 1 || never[0].Title != "Äpfel im Schlafrock" {
		t.Errorf("NeverPlanned: %+v %v", never, err)
	}

	sorts := map[string][]string{
		// zuletzt gegessen zuerst, nie eingeplant am Ende
		SortLastPlanned: {"Steak mit Tomaten", "Spaghetti Napoli", "Pilzrisotto", "Äpfel im Schlafrock"},
		SortMostPlanned: {"Spaghetti Napoli", "Pilzrisotto", "Steak mit Tomaten", "Äpfel im Schlafrock"},
	}
	for sort, wantOrder := range sorts {
		list, err := s.List(ctx, ListFilter{Sort: sort})
		if err != nil {
			t.Fatal(err)
		}
		if got := titles(list); !slices.Equal(got, wantOrder) {
			t.Errorf("Sortierung %s: %v, erwartet %v", sort, got, wantOrder)
		}
	}
}
