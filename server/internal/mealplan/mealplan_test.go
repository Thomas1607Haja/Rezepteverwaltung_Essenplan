package mealplan

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/database"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/settings"
)

// Mittwoch, 21.10.2026, 12:00 – fester "heutiger" Tag aller Tests.
var testNow = time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)

// newTestStore liefert einen Store mit leerer Datenbank und zwei Rezepten
// (IDs 1 und 2).
func newTestStore(t *testing.T) *Store {
	t.Helper()
	ctx := context.Background()
	db, err := database.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := database.Migrate(ctx, db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	_, err = db.ExecContext(ctx, `
		INSERT INTO recipes (id, title, base_servings, thumbnail, is_favorite, created_at, updated_at) VALUES
		(1, 'Lasagne', 4, 'lasagne-thumb.jpg', 1, '2026-10-01T10:00:00.000Z', '2026-10-01T10:00:00.000Z'),
		(2, 'Linsensuppe', 4, '', 0, '2026-10-01T10:00:00.000Z', '2026-10-01T10:00:00.000Z')`)
	if err != nil {
		t.Fatalf("Rezepte anlegen: %v", err)
	}
	s := NewStore(db, settings.NewStore(db))
	s.now = func() time.Time { return testNow }
	return s
}

func ptr[T any](v T) *T { return &v }

func TestShoppingPeriod(t *testing.T) {
	tests := []struct {
		today       string
		shoppingDay time.Weekday
		want        Period
	}{
		{"2026-10-21", time.Saturday, Period{"2026-10-17", "2026-10-23"}}, // Mittwoch
		{"2026-10-24", time.Saturday, Period{"2026-10-24", "2026-10-30"}}, // am Einkaufstag selbst
		{"2026-10-23", time.Saturday, Period{"2026-10-17", "2026-10-23"}}, // letzter Tag
		{"2026-10-21", time.Wednesday, Period{"2026-10-21", "2026-10-27"}},
		{"2026-10-21", time.Thursday, Period{"2026-10-15", "2026-10-21"}},
		{"2026-10-25", time.Monday, Period{"2026-10-19", "2026-10-25"}},   // Sonntag
		{"2026-12-29", time.Saturday, Period{"2026-12-26", "2027-01-01"}}, // über den Jahreswechsel
	}
	for _, tt := range tests {
		today, _ := time.Parse(dateLayout, tt.today)
		if got := ShoppingPeriod(today, tt.shoppingDay, 0); got != tt.want {
			t.Errorf("ShoppingPeriod(%s, %v) = %+v, erwartet %+v", tt.today, tt.shoppingDay, got, tt.want)
		}
	}
}

// Mit Uhrzeit: Am Einkaufstag gilt vor der Uhrzeit noch die vorige Woche
// als eingekauft, danach die neue. Einkaufstag Donnerstag, 22.10.2026.
func TestShoppingPeriodWithTime(t *testing.T) {
	cutoff := 18 * time.Hour
	tests := []struct {
		now  string
		want Period
	}{
		{"2026-10-21 23:59", Period{"2026-10-15", "2026-10-21"}}, // Mittwoch
		{"2026-10-22 09:30", Period{"2026-10-15", "2026-10-21"}}, // Donnerstag vor 18 Uhr
		{"2026-10-22 17:59", Period{"2026-10-15", "2026-10-21"}},
		{"2026-10-22 18:00", Period{"2026-10-22", "2026-10-28"}}, // ab 18 Uhr eingekauft
		{"2026-10-23 08:00", Period{"2026-10-22", "2026-10-28"}}, // Freitag
	}
	for _, tt := range tests {
		now, _ := time.Parse("2006-01-02 15:04", tt.now)
		if got := ShoppingPeriod(now, time.Thursday, cutoff); got != tt.want {
			t.Errorf("ShoppingPeriod(%s) = %+v, erwartet %+v", tt.now, got, tt.want)
		}
	}
	if next := (Period{"2026-10-22", "2026-10-28"}).Next(); next != (Period{"2026-10-29", "2026-11-04"}) {
		t.Errorf("Next = %+v", next)
	}
}

func TestCreateAndList(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	lasagne, err := s.Create(ctx, NewEntry{Date: "2026-10-24", RecipeID: ptr(int64(1))})
	if err != nil {
		t.Fatal(err)
	}
	// Standardwerte: Abendessen, Familiengröße, geplant
	if lasagne.Slot != SlotDinner || lasagne.Servings != 4 || lasagne.Status != StatusPlanned || lasagne.Position != 0 {
		t.Errorf("Standardwerte falsch: %+v", lasagne)
	}
	if lasagne.Recipe == nil || lasagne.Recipe.Title != "Lasagne" || !lasagne.Recipe.IsFavorite {
		t.Errorf("Rezeptangaben fehlen: %+v", lasagne.Recipe)
	}

	// Zweites Gericht am selben Abend kommt dahinter.
	soup, err := s.Create(ctx, NewEntry{Date: "2026-10-24", RecipeID: ptr(int64(2)), Servings: ptr(2)})
	if err != nil {
		t.Fatal(err)
	}
	if soup.Position != 1 || soup.Servings != 2 {
		t.Errorf("zweites Gericht: %+v", soup)
	}

	// Mittagessen und Eintrag ohne Rezept
	if _, err := s.Create(ctx, NewEntry{Date: "2026-10-24", Slot: SlotLunch, Note: "Reste"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create(ctx, NewEntry{Date: "2026-11-01", Note: "Essen gehen"}); err != nil {
		t.Fatal(err)
	}

	entries, err := s.List(ctx, "2026-10-24", "2026-10-30")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, e := range entries {
		label := e.Note
		if e.Recipe != nil {
			label = e.Recipe.Title
		}
		got = append(got, label)
	}
	// Mittag vor Abend, im Abend nach Position; der 1.11. liegt außerhalb.
	if strings.Join(got, ",") != "Reste,Lasagne,Linsensuppe" {
		t.Errorf("Reihenfolge: %v", got)
	}
}

func TestCreateValidation(t *testing.T) {
	s := newTestStore(t)
	tests := []struct {
		in    NewEntry
		field string
	}{
		{NewEntry{Date: "24.10.2026", RecipeID: ptr(int64(1))}, "date"},
		{NewEntry{Date: "2026-02-30", RecipeID: ptr(int64(1))}, "date"},
		{NewEntry{Date: "2026-10-24", Slot: "breakfast", RecipeID: ptr(int64(1))}, "slot"},
		{NewEntry{Date: "2026-10-24"}, "note"},              // weder Rezept noch Notiz
		{NewEntry{Date: "2026-10-24", Note: "   "}, "note"}, // Notiz nur aus Leerzeichen
		{NewEntry{Date: "2026-10-24", RecipeID: ptr(int64(99))}, "recipe_id"},
		{NewEntry{Date: "2026-10-24", RecipeID: ptr(int64(1)), Servings: ptr(0)}, "servings"},
	}
	for _, tt := range tests {
		_, err := s.Create(context.Background(), tt.in)
		var verr *ValidationError
		if !errors.As(err, &verr) || verr.Field != tt.field {
			t.Errorf("%+v: erwartet Fehler im Feld %q, bekommen %v", tt.in, tt.field, err)
		}
	}
}

func TestUpdate(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	e, err := s.Create(ctx, NewEntry{Date: "2026-10-24", RecipeID: ptr(int64(1))})
	if err != nil {
		t.Fatal(err)
	}

	// Rezept ersetzen, Portionen und Status ändern
	got, err := s.Update(ctx, e.ID, Changes{RecipeID: ptr(int64(2)), Servings: 6, Status: StatusHaveIngredients})
	if err != nil {
		t.Fatal(err)
	}
	if got.Recipe.Title != "Linsensuppe" || got.Servings != 6 || got.Status != StatusHaveIngredients {
		t.Errorf("nach Update: %+v", got)
	}

	// Rezept entfernen geht nur mit Notiz
	if _, err := s.Update(ctx, e.ID, Changes{Servings: 4, Status: StatusPlanned}); err == nil {
		t.Error("Eintrag ohne Rezept und Notiz wurde angenommen")
	}
	got, err = s.Update(ctx, e.ID, Changes{Servings: 4, Note: "Pizza bestellen", Status: StatusPlanned})
	if err != nil {
		t.Fatal(err)
	}
	if got.RecipeID != nil || got.Recipe != nil || got.Note != "Pizza bestellen" {
		t.Errorf("ohne Rezept: %+v", got)
	}

	// "transferred" darf man nicht selbst setzen
	_, err = s.Update(ctx, e.ID, Changes{Servings: 4, Note: "x", Status: StatusTransferred})
	var verr *ValidationError
	if !errors.As(err, &verr) || verr.Field != "status" {
		t.Errorf("transferred gesetzt: %v", err)
	}

	if _, err := s.Update(ctx, 999, Changes{Servings: 4, Note: "x", Status: StatusPlanned}); !errors.Is(err, ErrNotFound) {
		t.Errorf("unbekannter Eintrag: %v", err)
	}
}

func TestDeleteRecordsDeletion(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	e, err := s.Create(ctx, NewEntry{Date: "2026-10-24", Note: "Reste"})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(ctx, e.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get(ctx, e.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("nach dem Löschen: %v", err)
	}
	var n int
	if err := s.db.QueryRowContext(ctx,
		"SELECT count(*) FROM deletions WHERE table_name = 'meal_plan_entries' AND record_id = ?", e.ID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("deletions: %d Einträge, erwartet 1", n)
	}
	if err := s.Delete(ctx, e.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("zweites Löschen: %v", err)
	}
}

func TestHandlers(t *testing.T) {
	mux := http.NewServeMux()
	NewHandler(newTestStore(t)).Register(mux)
	do := func(method, path, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
		return rec
	}

	if rec := do("POST", "/api/v1/meal-plan", `{"date":"2026-10-22","recipe_id":1}`); rec.Code != http.StatusCreated {
		t.Fatalf("POST: %d %s", rec.Code, rec.Body)
	}
	// Ohne from/to: aktueller Einkaufszeitraum (Standard Samstag)
	rec := do("GET", "/api/v1/meal-plan", "")
	body := rec.Body.String()
	for _, want := range []string{`"today":"2026-10-21"`, `"from":"2026-10-17"`, `"to":"2026-10-23"`, `"title":"Lasagne"`} {
		if !strings.Contains(body, want) {
			t.Errorf("GET: %s fehlt in %s", want, body)
		}
	}
	if rec := do("GET", "/api/v1/meal-plan?from=2026-10-25&to=2026-10-20", ""); rec.Code != http.StatusBadRequest {
		t.Errorf("verdrehter Zeitraum: %d", rec.Code)
	}
	if rec := do("PUT", "/api/v1/meal-plan/1", `{"recipe_id":1,"servings":3,"note":"","status":"planned"}`); rec.Code != http.StatusOK {
		t.Errorf("PUT: %d %s", rec.Code, rec.Body)
	}
	if rec := do("DELETE", "/api/v1/meal-plan/1", ""); rec.Code != http.StatusNoContent {
		t.Errorf("DELETE: %d", rec.Code)
	}
	if rec := do("GET", "/api/v1/meal-plan/1", ""); rec.Code != http.StatusNotFound {
		t.Errorf("GET nach DELETE: %d", rec.Code)
	}
}
