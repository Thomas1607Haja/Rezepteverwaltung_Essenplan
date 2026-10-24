package appsync

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/database"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/mealplan"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/pantry"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/recipes"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/settings"
)

type fixture struct {
	db      *sql.DB
	handler *Handler
	recipes *recipes.Store
	plan    *mealplan.Store
}

func newFixture(t *testing.T) fixture {
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
	rs := recipes.NewStore(db)
	ss := settings.NewStore(db)
	ps := mealplan.NewStore(db, ss)
	return fixture{db: db, handler: NewHandler(db, rs, ps, pantry.NewStore(db), ss), recipes: rs, plan: ps}
}

// get ruft den Endpunkt über den Router auf und liefert Status und Antwort.
func (f fixture) get(t *testing.T, query string) (int, Response) {
	t.Helper()
	mux := http.NewServeMux()
	f.handler.Register(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/sync"+query, nil))
	var resp Response
	if rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Antwort: %v", err)
		}
	}
	return rec.Code, resp
}

func ptr[T any](v T) *T { return &v }

func TestSyncFullAndIncremental(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)

	lasagne, err := f.recipes.Create(ctx, recipes.Recipe{
		Title: "Lasagne", BaseServings: 4, SourceName: "Kochbuch",
		Ingredients: []recipes.Ingredient{{Name: "Nudelplatten", Quantity: ptr(250.0), Unit: "g", ShoppingRelevant: true}, {Name: "Salz"}},
		Steps:       []recipes.Step{{Text: "Schichten"}, {Text: "Backen"}},
		Tags:        []recipes.Tag{{ID: 11}}, // Pasta
	})
	if err != nil {
		t.Fatal(err)
	}
	soup, err := f.recipes.Create(ctx, recipes.Recipe{Title: "Linsensuppe", BaseServings: 4,
		Ingredients: []recipes.Ingredient{{Name: "Linsen"}}})
	if err != nil {
		t.Fatal(err)
	}
	today := f.plan.Today().Format("2006-01-02")
	if _, err := f.plan.Create(ctx, mealplan.NewEntry{Date: today, RecipeID: &lasagne.ID}); err != nil {
		t.Fatal(err)
	}

	// Vollständig
	code, full := f.get(t, "")
	if code != http.StatusOK || !full.Full {
		t.Fatalf("status %d, full %v", code, full.Full)
	}
	if len(full.Recipes) != 2 {
		t.Fatalf("%d Rezepte, erwartet 2", len(full.Recipes))
	}
	got := full.Recipes[0]
	if got.Title != "Lasagne" || got.SourceName != "Kochbuch" || len(got.Ingredients) != 2 ||
		got.Ingredients[0].Name != "Nudelplatten" || len(got.Steps) != 2 || got.Steps[1].Text != "Backen" ||
		len(got.Tags) != 1 || got.Tags[0].Name != "Pasta" {
		t.Errorf("Lasagne unvollständig: %+v", got)
	}
	if len(full.Recipes[1].Steps) != 0 || full.Recipes[1].Tags == nil {
		t.Errorf("Linsensuppe: Schritte %v, Kategorien %v (erwartet leer, nicht null)", full.Recipes[1].Steps, full.Recipes[1].Tags)
	}
	if len(full.MealPlanEntries) != 1 || len(full.PlanInfo) != 1 || full.PlanInfo[0].RecipeID != lasagne.ID ||
		full.PlanInfo[0].NextPlanned == nil || *full.PlanInfo[0].NextPlanned != today {
		t.Errorf("Plan: %+v, Termine: %+v", full.MealPlanEntries, full.PlanInfo)
	}
	if len(full.TagGroups) != 5 || len(full.PantryItems) != 5 || full.Settings.FamilySize != 4 {
		t.Errorf("Stammdaten: %d Gruppen, %d Vorrat, Familie %d", len(full.TagGroups), len(full.PantryItems), full.Settings.FamilySize)
	}
	if full.ServerTime == "" || full.PlanFrom >= today || full.ShoppingPeriod.From == "" {
		t.Errorf("server_time %q, plan_from %q, Zeitraum %+v", full.ServerTime, full.PlanFrom, full.ShoppingPeriod)
	}

	// Die Suppe ist "alt": Beim Abgleich seit Juni kommt nur die Lasagne.
	// (Die Zeitstempel echter Änderungen liegen in der Gegenwart.)
	if _, err := f.db.ExecContext(ctx, "UPDATE recipes SET updated_at = '2026-01-01T00:00:00.000Z' WHERE id = ?", soup.ID); err != nil {
		t.Fatal(err)
	}
	_, inc := f.get(t, "?since=2026-06-01T00:00:00.000Z")
	if inc.Full || len(inc.Recipes) != 1 || inc.Recipes[0].ID != lasagne.ID {
		t.Errorf("seit Juni: full %v, %d Rezepte", inc.Full, len(inc.Recipes))
	}

	// Gelöschtes Rezept steht in deletions.
	if _, err := f.recipes.Delete(ctx, soup.ID); err != nil {
		t.Fatal(err)
	}
	_, inc = f.get(t, "?since=2026-06-01T00:00:00.000Z")
	found := false
	for _, d := range inc.Deletions {
		if d.Table == "recipes" && d.ID == soup.ID {
			found = true
		}
	}
	if !found {
		t.Errorf("Löschung fehlt: %+v", inc.Deletions)
	}
}

func TestSyncOverlap(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	if _, err := f.recipes.Create(ctx, recipes.Recipe{Title: "Lasagne", BaseServings: 4}); err != nil {
		t.Fatal(err)
	}
	// Mit dem server_time der eben erhaltenen Antwort kommt das Rezept
	// wegen der Überlappung noch einmal.
	_, full := f.get(t, "")
	_, again := f.get(t, "?since="+full.ServerTime)
	if len(again.Recipes) != 1 {
		t.Errorf("%d Rezepte, erwartet 1 (Überlappung)", len(again.Recipes))
	}
}

func TestSyncRejectsBadSince(t *testing.T) {
	f := newFixture(t)
	if code, _ := f.get(t, "?since=gestern"); code != http.StatusBadRequest {
		t.Errorf("status %d, erwartet 400", code)
	}
}
