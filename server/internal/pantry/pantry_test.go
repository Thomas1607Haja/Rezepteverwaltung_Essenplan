package pantry

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/database"
)

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
	return NewStore(db)
}

func TestStartItems(t *testing.T) {
	items, err := newTestStore(t).List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, it := range items {
		names = append(names, it.Name)
	}
	// "Öl" sortiert wie "Ol", also zwischen Pfeffer und Salz.
	if got := strings.Join(names, ","); got != "Öl,Pfeffer,Salz,Wasser,Zucker" {
		t.Errorf("Startbestand = %s", got)
	}
	if len(items[0].Synonyms) != 5 || items[0].Synonyms[0] != "Olivenöl" {
		t.Errorf("Synonyme von Öl = %q", items[0].Synonyms)
	}
}

func TestCreateUpdateDelete(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	it, err := store.Create(ctx, Item{Name: "  Mehl ", Synonyms: []string{"Weizenmehl", "", "weizenmehl", "mehl", " Dinkelmehl "}})
	if err != nil {
		t.Fatal(err)
	}
	if it.Name != "Mehl" || strings.Join(it.Synonyms, "|") != "Weizenmehl|Dinkelmehl" {
		t.Errorf("nach Create: %+v", it)
	}
	got, err := store.Get(ctx, it.ID)
	if err != nil || strings.Join(got.Synonyms, "|") != "Weizenmehl|Dinkelmehl" {
		t.Errorf("Get = %+v, %v", got, err)
	}

	// Gleicher Name in anderer Schreibweise ist eine Dublette.
	var verr *ValidationError
	if _, err := store.Create(ctx, Item{Name: "SALZ"}); !errors.As(err, &verr) {
		t.Errorf("Dublette: err = %v", err)
	}
	if _, err := store.Create(ctx, Item{Name: "Salz, grob"}); !errors.As(err, &verr) {
		t.Errorf("Komma im Namen: err = %v", err)
	}
	// Den eigenen Namen behalten ist keine Dublette.
	if _, err := store.Update(ctx, it.ID, Item{Name: "mehl"}); err != nil {
		t.Errorf("Update mit eigenem Namen: %v", err)
	}
	if _, err := store.Update(ctx, 9999, Item{Name: "Hefe"}); !errors.Is(err, ErrNotFound) {
		t.Errorf("Update unbekannt: err = %v", err)
	}

	if err := store.Delete(ctx, it.ID); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(ctx, it.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("zweites Delete: err = %v", err)
	}
	var n int
	store.db.QueryRowContext(ctx, `SELECT count(*) FROM deletions WHERE table_name = 'pantry_items' AND record_id = ?`, it.ID).Scan(&n)
	if n != 1 {
		t.Errorf("deletions: %d Einträge", n)
	}
}

func TestMatches(t *testing.T) {
	m, err := newTestStore(t).Matcher(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]bool{
		"Salz":                    true,
		"salz":                    true,
		"OLIVENÖL":                true,
		"Olivenöl, extra vergine": true,
		"Öl (zum Braten)":         true,
		"Salz und Pfeffer":        true,
		"Salz & Pfeffer":          true,
		"Salz/Pfeffer":            true,
		"Pfeffer aus der Mühle":   true,
		"schwarzer  Pfeffer":      true,
		"Salzbrezel":              false,
		"Salz und Mehl":           false,
		"Mehl":                    false,
		"Zuckerschoten":           false,
		"":                        false,
		"(optional)":              false,
	} {
		if got := m.Matches(name); got != want {
			t.Errorf("Matches(%q) = %v, erwartet %v", name, got, want)
		}
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

	rec := do("POST", "/api/v1/pantry", `{"name": "Butter", "synonyms": ["Margarine"]}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST: %d %s", rec.Code, rec.Body)
	}
	var created Item
	json.Unmarshal(rec.Body.Bytes(), &created)

	if rec := do("POST", "/api/v1/pantry", `{"name": ""}`); rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), `"field":"name"`) {
		t.Errorf("leerer Name: %d %s", rec.Code, rec.Body)
	}
	if rec := do("PUT", "/api/v1/pantry/abc", `{"name": "x"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("PUT mit ungültiger ID: %d", rec.Code)
	}

	rec = do("POST", "/api/v1/pantry/match", `{"names": ["Margarine", "Mehl", "Salz und Pfeffer"]}`)
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != `{"in_pantry":[true,false,true]}` {
		t.Errorf("match: %d %s", rec.Code, rec.Body)
	}

	if rec := do("DELETE", "/api/v1/pantry/"+jsonID(created.ID), ""); rec.Code != http.StatusNoContent {
		t.Errorf("DELETE: %d", rec.Code)
	}
	if rec := do("GET", "/api/v1/pantry", ""); strings.Contains(rec.Body.String(), "Butter") {
		t.Errorf("Butter nach dem Löschen noch da: %s", rec.Body)
	}
}

func jsonID(id int64) string {
	b, _ := json.Marshal(id)
	return string(b)
}

func TestApplyToRecipes(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	// Rezepte direkt per SQL anlegen: Das Paket recipes hier einzubinden
	// lohnt nicht für zwei Zeilen.
	_, err := store.db.ExecContext(ctx, `
		INSERT INTO recipes (id, title, base_servings, created_at, updated_at) VALUES
			(1, 'Zwiebelkuchen', 4, '2026-01-01T00:00:00.000Z', '2026-01-01T00:00:00.000Z'),
			(2, 'Apfelkuchen', 4, '2026-01-01T00:00:00.000Z', '2026-01-01T00:00:00.000Z');
		INSERT INTO recipe_ingredients (id, recipe_id, position, name, shopping_relevant, updated_at) VALUES
			(10, 1, 1, 'Zwiebeln', 1, '2026-01-01T00:00:00.000Z'),
			(11, 1, 2, 'Salz', 1, '2026-01-01T00:00:00.000Z'),
			(20, 2, 1, 'Äpfel', 1, '2026-01-01T00:00:00.000Z'),
			(21, 2, 2, 'Mehl', 0, '2026-01-01T00:00:00.000Z'),
			(22, 2, 3, 'Zucker', 0, '2026-01-01T00:00:00.000Z')`)
	if err != nil {
		t.Fatal(err)
	}

	// Vorschau: Salz kommt in den Vorrat, Mehl (nicht im Vorrat) wird
	// wieder eingekauft; Zucker passt schon.
	preview, err := store.ApplyToRecipes(ctx, false)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Applied || preview.Recipes != 2 || len(preview.Changes) != 2 {
		t.Fatalf("Vorschau = %+v", preview)
	}
	// Sortiert nach Rezepttitel: Apfelkuchen vor Zwiebelkuchen.
	if c := preview.Changes[0]; c.IngredientID != 21 || !c.ShoppingRelevant || c.RecipeTitle != "Apfelkuchen" {
		t.Errorf("erste Änderung = %+v", c)
	}
	if c := preview.Changes[1]; c.IngredientID != 11 || c.ShoppingRelevant {
		t.Errorf("zweite Änderung = %+v", c)
	}
	relevant := func(id int) bool {
		var v bool
		store.db.QueryRowContext(ctx, "SELECT shopping_relevant FROM recipe_ingredients WHERE id = ?", id).Scan(&v)
		return v
	}
	if !relevant(11) {
		t.Error("Vorschau hat schon gespeichert")
	}

	done, err := store.ApplyToRecipes(ctx, true)
	if err != nil {
		t.Fatal(err)
	}
	if !done.Applied || len(done.Changes) != 2 || relevant(11) || !relevant(21) {
		t.Errorf("nach Bestätigung: %+v, Salz %v, Mehl %v", done, relevant(11), relevant(21))
	}
	var updated string
	store.db.QueryRowContext(ctx, "SELECT updated_at FROM recipes WHERE id = 1").Scan(&updated)
	if updated == "2026-01-01T00:00:00.000Z" {
		t.Error("updated_at des Rezepts nicht gesetzt")
	}

	// Danach ist nichts mehr zu tun.
	again, _ := store.ApplyToRecipes(ctx, false)
	if len(again.Changes) != 0 || again.Recipes != 0 {
		t.Errorf("zweiter Lauf = %+v", again)
	}
}
