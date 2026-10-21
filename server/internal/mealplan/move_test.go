package mealplan

import (
	"context"
	"errors"
	"testing"
)

// Heute ist Mittwoch, 21.10.2026; mit Einkaufstag Samstag läuft der
// Einkaufszeitraum vom 17. bis 23.10. (siehe testNow).

// addEntry legt einen Eintrag mit Rezept 1 an und setzt seinen Status.
func addEntry(t *testing.T, s *Store, date, status string) Entry {
	t.Helper()
	ctx := context.Background()
	e, err := s.Create(ctx, NewEntry{Date: date, RecipeID: ptr(int64(1))})
	if err != nil {
		t.Fatal(err)
	}
	// "transferred" lässt sich über die API nicht setzen, daher direkt.
	if _, err := s.db.ExecContext(ctx, "UPDATE meal_plan_entries SET status = ? WHERE id = ?", status, e.ID); err != nil {
		t.Fatal(err)
	}
	e, err = s.Get(ctx, e.ID)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func TestMoveWarnings(t *testing.T) {
	tests := []struct {
		name         string
		from, to     string // Datum des gezogenen Eintrags vorher/nachher
		status       string
		swapDate     string // falls gesetzt: Tausch mit einem Eintrag an diesem Tag
		swapStatus   string
		wantWarnings []string
	}{
		{"geplant in den Zeitraum", "2026-10-26", "2026-10-22", StatusPlanned, "", "", []string{WarnPlannedIntoPeriod}},
		{"geplant innerhalb des Zeitraums", "2026-10-18", "2026-10-22", StatusPlanned, "", "", nil},
		{"geplant außerhalb", "2026-10-26", "2026-10-28", StatusPlanned, "", "", nil},
		{"geplant aus dem Zeitraum hinaus", "2026-10-22", "2026-10-26", StatusPlanned, "", "", nil},
		{"Zutaten vorhanden, in den Zeitraum", "2026-10-26", "2026-10-22", StatusHaveIngredients, "", "", nil},
		{"übertragen aus dem Zeitraum", "2026-10-22", "2026-10-26", StatusTransferred, "", "", []string{WarnTransferredOutOfPeriod}},
		{"übertragen in den Zeitraum", "2026-10-26", "2026-10-22", StatusTransferred, "", "", nil},
		{"Tausch zweier übertragener im Zeitraum", "2026-10-20", "2026-10-22", StatusTransferred, "2026-10-22", StatusTransferred, nil},
		{"Tausch: geplantes rutscht hinein, anderes hinaus", "2026-10-26", "2026-10-22", StatusPlanned, "2026-10-22", StatusPlanned,
			[]string{WarnPlannedIntoPeriod}},
		{"Tausch: übertragenes hinaus, geplantes hinein", "2026-10-26", "2026-10-22", StatusPlanned, "2026-10-22", StatusTransferred,
			[]string{WarnPlannedIntoPeriod, WarnTransferredOutOfPeriod}},
	}
	for _, tt := range tests {
		// t.Run führt jeden Fall als eigenen Untertest aus; ein Fehler nennt
		// dann den Namen des Falls.
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			s := newTestStore(t)
			e := addEntry(t, s, tt.from, tt.status)
			m := Move{Date: tt.to, Slot: SlotDinner}
			if tt.swapDate != "" {
				other := addEntry(t, s, tt.swapDate, tt.swapStatus)
				m.SwapWith = &other.ID
			}

			res, err := s.Move(ctx, e.ID, m)
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			for _, w := range res.Warnings {
				got = append(got, w.Type)
			}
			if len(got) != len(tt.wantWarnings) {
				t.Fatalf("Warnungen %v, erwartet %v", got, tt.wantWarnings)
			}
			for i := range got {
				if got[i] != tt.wantWarnings[i] {
					t.Errorf("Warnungen %v, erwartet %v", got, tt.wantWarnings)
				}
			}

			// Mit Warnungen wird nichts gespeichert, ohne sofort.
			after, _ := s.Get(ctx, e.ID)
			if wantApplied := len(tt.wantWarnings) == 0; res.Applied != wantApplied {
				t.Errorf("applied = %v, erwartet %v", res.Applied, wantApplied)
			}
			if res.Applied && after.Date != tt.to || !res.Applied && after.Date != tt.from {
				t.Errorf("Datum nach Move: %s (applied %v)", after.Date, res.Applied)
			}
		})
	}
}

func TestMoveConfirm(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	e := addEntry(t, s, "2026-10-26", StatusPlanned)

	res, err := s.Move(ctx, e.ID, Move{Date: "2026-10-22", Confirm: true, HaveIngredients: []int64{e.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Applied || len(res.Entries) != 1 {
		t.Fatalf("bestätigt, aber nicht gespeichert: %+v", res)
	}
	got := res.Entries[0]
	if got.Date != "2026-10-22" || got.Status != StatusHaveIngredients {
		t.Errorf("nach Bestätigung: Datum %s, Status %s", got.Date, got.Status)
	}
	if len(res.Warnings) != 1 {
		t.Errorf("Warnungen sollen auch nach Bestätigung im Ergebnis stehen: %v", res.Warnings)
	}
}

func TestMoveSwapAndAppend(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	a := addEntry(t, s, "2026-10-20", StatusPlanned)
	b := addEntry(t, s, "2026-10-22", StatusPlanned) // Position 0
	c := addEntry(t, s, "2026-10-22", StatusPlanned) // Position 1
	d := addEntry(t, s, "2026-10-20", StatusPlanned) // Position 1 am 20.

	// a tauscht mit c: a bekommt Platz 1 am 22., c Platz 0 am 20.
	res, err := s.Move(ctx, a.ID, Move{Date: "2026-10-22", SwapWith: &c.ID})
	if err != nil || !res.Applied {
		t.Fatalf("Tausch: %v %+v", err, res)
	}
	a, _ = s.Get(ctx, a.ID)
	c, _ = s.Get(ctx, c.ID)
	if a.Date != "2026-10-22" || a.Position != 1 || c.Date != "2026-10-20" || c.Position != 0 {
		t.Errorf("nach Tausch: a=%s/%d c=%s/%d", a.Date, a.Position, c.Date, c.Position)
	}

	// d wird an den 22. angehängt: hinter b (0) und a (1)
	if _, err := s.Move(ctx, d.ID, Move{Date: "2026-10-22"}); err != nil {
		t.Fatal(err)
	}
	d, _ = s.Get(ctx, d.ID)
	if d.Date != "2026-10-22" || d.Position != 2 {
		t.Errorf("angehängt: %s/%d, erwartet 2026-10-22/2", d.Date, d.Position)
	}

	// Zum Mittagessen verschieben
	if _, err := s.Move(ctx, b.ID, Move{Date: "2026-10-22", Slot: SlotLunch}); err != nil {
		t.Fatal(err)
	}
	b, _ = s.Get(ctx, b.ID)
	if b.Slot != SlotLunch || b.Position != 0 {
		t.Errorf("Mittagessen: %s/%d", b.Slot, b.Position)
	}
}

func TestMoveValidation(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	a := addEntry(t, s, "2026-10-20", StatusPlanned)
	b := addEntry(t, s, "2026-10-22", StatusPlanned)

	tests := []struct {
		name  string
		m     Move
		field string
	}{
		{"mit sich selbst", Move{Date: "2026-10-20", SwapWith: &a.ID}, "swap_with"},
		{"Partner steht woanders", Move{Date: "2026-10-23", SwapWith: &b.ID}, "swap_with"},
		{"Partner fehlt", Move{Date: "2026-10-22", SwapWith: ptr(int64(999))}, "swap_with"},
		{"kein Datum", Move{Date: ""}, "date"},
		{"falscher Slot", Move{Date: "2026-10-22", Slot: "brunch"}, "slot"},
	}
	for _, tt := range tests {
		_, err := s.Move(ctx, a.ID, tt.m)
		var verr *ValidationError
		if !errors.As(err, &verr) || verr.Field != tt.field {
			t.Errorf("%s: erwartet Fehler im Feld %q, bekommen %v", tt.name, tt.field, err)
		}
	}
	if _, err := s.Move(ctx, 999, Move{Date: "2026-10-22"}); !errors.Is(err, ErrNotFound) {
		t.Errorf("unbekannter Eintrag: %v", err)
	}
}
