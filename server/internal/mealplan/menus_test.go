package mealplan

import (
	"context"
	"errors"
	"testing"
)

// Menüs (Schritt 9.4): Klammer um Einträge, Block beim Verschieben.
func TestMenus(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	menu, err := s.CreateMenu(ctx, MenuInput{Date: "2026-10-31", Slot: SlotDinner, Note: "Suppe vorher kochen", Servings: ptr(6)})
	if err != nil {
		t.Fatal(err)
	}
	if menu.DisplayTitle != "Menü" || menu.Servings != 6 {
		t.Errorf("leeres Menü: %+v", menu)
	}
	// Zwei Gänge: Portionen vom Menü, Tag/Slot vom Menü (Angaben im Eintrag zählen nicht).
	suppe, err := s.Create(ctx, NewEntry{Date: "2020-01-01", Slot: SlotLunch, RecipeID: ptr(int64(2)), MenuID: &menu.ID})
	if err != nil {
		t.Fatal(err)
	}
	haupt, err := s.Create(ctx, NewEntry{RecipeID: ptr(int64(1)), MenuID: &menu.ID})
	if err != nil {
		t.Fatal(err)
	}
	if suppe.Date != "2026-10-31" || suppe.Slot != SlotDinner || suppe.Servings != 6 || *suppe.MenuID != menu.ID {
		t.Errorf("Gang: %+v", suppe)
	}
	if m, _ := s.GetMenu(ctx, menu.ID); m.DisplayTitle != "Linsensuppe" {
		t.Errorf("automatischer Titel: %q", m.DisplayTitle)
	}

	// Einzeleintrag am Tag davor; Verschieben des Menüs per Gang = ganzer Block.
	single, err := s.Create(ctx, NewEntry{Date: "2026-10-30", RecipeID: ptr(int64(1))})
	if err != nil {
		t.Fatal(err)
	}
	res, err := s.Move(ctx, haupt.ID, Move{Date: "2026-11-01", Slot: SlotLunch, Confirm: true})
	if err != nil || !res.Applied {
		t.Fatalf("Move: %+v %v", res, err)
	}
	if len(res.Entries) != 2 || len(res.Menus) != 1 || res.Menus[0].Date != "2026-11-01" || res.Menus[0].Slot != SlotLunch {
		t.Errorf("Block verschoben: %+v", res)
	}
	for _, e := range res.Entries {
		if e.Date != "2026-11-01" || e.Slot != SlotLunch {
			t.Errorf("Gang nicht mitgewandert: %+v", e)
		}
	}

	// Tauschen: Einzeleintrag gegen das Menü.
	res, err = s.Move(ctx, single.ID, Move{Date: "2026-11-01", Slot: SlotLunch, SwapWithMenu: &menu.ID, Confirm: true})
	if err != nil || !res.Applied {
		t.Fatalf("Tausch: %+v %v", res, err)
	}
	if m, _ := s.GetMenu(ctx, menu.ID); m.Date != "2026-10-30" || m.Slot != SlotDinner {
		t.Errorf("Menü nach Tausch: %+v", m)
	}
	if e, _ := s.Get(ctx, single.ID); e.Date != "2026-11-01" || e.Slot != SlotLunch {
		t.Errorf("Einzeleintrag nach Tausch: %+v", e)
	}
	if e, _ := s.Get(ctx, suppe.ID); e.Date != "2026-10-30" {
		t.Errorf("Gang nach Tausch: %+v", e)
	}

	// Kuchen im Menü: Grundmenge statt Menü-Portionen, und Übertragen der
	// Portionen lässt ihn aus.
	if _, err := s.db.Exec(`INSERT INTO recipes (id, title, base_servings, yield_unit, created_at, updated_at)
		VALUES (9, 'Apfelkuchen', 1, 'Kuchen', '2026-01-01T00:00:00.000Z', '2026-01-01T00:00:00.000Z')`); err != nil {
		t.Fatal(err)
	}
	kuchen, err := s.Create(ctx, NewEntry{RecipeID: ptr(int64(9)), MenuID: &menu.ID})
	if err != nil || kuchen.Servings != 1 {
		t.Fatalf("Kuchen im Menü: %+v %v", kuchen, err)
	}

	// Portionen auf alle Gänge übertragen.
	if _, err := s.UpdateMenu(ctx, menu.ID, MenuInput{Title: "Herbstmenü", Servings: ptr(8), ApplyServings: true}); err != nil {
		t.Fatal(err)
	}
	if e, _ := s.Get(ctx, haupt.ID); e.Servings != 8 {
		t.Errorf("Portionen: %d", e.Servings)
	}
	if e, _ := s.Get(ctx, kuchen.ID); e.Servings != 1 {
		t.Errorf("Kuchen nach Übertragen: %d", e.Servings)
	}
	if m, _ := s.GetMenu(ctx, menu.ID); m.DisplayTitle != "Herbstmenü" {
		t.Errorf("Titel: %q", m.DisplayTitle)
	}

	// Alles vorhanden, Gang lösen, Menü auflösen.
	if err := s.MenuAllAvailable(ctx, menu.ID); err != nil {
		t.Fatal(err)
	}
	if e, _ := s.Get(ctx, suppe.ID); e.Status != StatusHaveIngredients {
		t.Errorf("Status: %s", e.Status)
	}
	if e, err := s.Detach(ctx, suppe.ID); err != nil || e.MenuID != nil || e.Date != "2026-10-30" {
		t.Errorf("Lösen: %+v %v", e, err)
	}
	if err := s.DeleteMenu(ctx, menu.ID, false); err != nil {
		t.Fatal(err)
	}
	if e, err := s.Get(ctx, haupt.ID); err != nil || e.MenuID != nil {
		t.Errorf("nach Auflösen: %+v %v", e, err)
	}
	if _, err := s.GetMenu(ctx, menu.ID); !errors.Is(err, ErrMenuNotFound) {
		t.Errorf("Menü noch da: %v", err)
	}

	// Zu einem Menü erweitern, dann mit Gängen löschen.
	m2, err := s.ToMenu(ctx, haupt.ID, "", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if m2.Date != "2026-10-30" || m2.Servings != 8 || m2.DisplayTitle != "Lasagne" {
		t.Errorf("ToMenu: %+v", m2)
	}
	if _, err := s.ToMenu(ctx, haupt.ID, "", "", 0); err == nil {
		t.Error("zweimal erweitert")
	}

	// Erneut erweitern: ein neuer Gang gilt, ohne Angabe bleibt der alte.
	for _, c := range []struct{ give, want int64 }{{7, 7}, {9, 9}, {0, 9}} {
		m, err := s.ToMenu(ctx, suppe.ID, "", "", c.give)
		if err != nil {
			t.Fatal(err)
		}
		if e, _ := s.Get(ctx, suppe.ID); e.CourseTagID == nil || *e.CourseTagID != c.want {
			t.Errorf("Gang nach ToMenu(%d): %v", c.give, e.CourseTagID)
		}
		if err := s.DeleteMenu(ctx, m.ID, false); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.DeleteMenu(ctx, m2.ID, true); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get(ctx, haupt.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("Gang nicht mitgelöscht: %v", err)
	}
	var n int
	s.db.QueryRow("SELECT count(*) FROM deletions WHERE table_name = 'meal_plan_menus'").Scan(&n)
	if n != 5 { // 2 + die drei aus „erneut erweitern“
		t.Errorf("Löschungen Menüs: %d", n)
	}
}
