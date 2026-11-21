package shopping

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/apierr"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/homeassistant"
)

// recipeID sucht ein Rezept aus setup über seinen Titel.
func recipeID(t *testing.T, store *Store, title string) int64 {
	t.Helper()
	var id int64
	if err := store.db.QueryRow("SELECT id FROM recipes WHERE title = ?", title).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// counts zählt Planeinträge und Protokollzeilen – „Jetzt einkaufen“ darf
// an beidem nichts ändern.
func counts(t *testing.T, store *Store) (entries, logged int) {
	t.Helper()
	store.db.QueryRow("SELECT count(*) FROM meal_plan_entries").Scan(&entries)
	store.db.QueryRow("SELECT count(*) FROM transferred_items").Scan(&logged)
	return entries, logged
}

func TestSendRecipe(t *testing.T) {
	ctx := context.Background()
	store, _ := setup(t)
	ha := store.ha.(*homeassistant.TestClient)
	gulasch := recipeID(t, store, "Gulasch")

	var verr *apierr.Error
	if _, err := store.SendRecipe(ctx, RecipeSendOptions{RecipeID: gulasch, Servings: 4}); !errors.As(err, &verr) || verr.Field != "bring_entity" {
		t.Fatalf("ohne Bring-Liste: err = %v", err)
	}
	chooseList(t, store)
	beforeEntries, beforeLogged := counts(t, store)
	previewBefore, _ := store.Preview(ctx, false)

	// Zwiebeln stehen schon offen auf der Liste: Menge wird ergänzt wie
	// bei der normalen Übertragung.
	ha.AddItem(ctx, homeassistant.TestListID, "Zwiebeln", "1")

	// 6 statt 4 Portionen (1,5-fach), Knoblauch ist noch da. Salz ist
	// Vorrat (nicht einkaufsrelevant) und kommt nie mit.
	result, err := store.SendRecipe(ctx, RecipeSendOptions{RecipeID: gulasch, Servings: 6, Excluded: []string{nameKey("Knoblauch")}})
	if err != nil {
		t.Fatal(err)
	}
	if result.List != homeassistant.TestListID || len(result.Sent) != 3 || len(result.Failed) != 0 || result.Entries != 0 {
		t.Errorf("Ergebnis = %+v", result)
	}
	items, _ := ha.OpenItems(ctx, homeassistant.TestListID)
	var lines []string
	for _, it := range items {
		lines = append(lines, it.Summary+": "+it.Description)
	}
	want := "Zwiebeln: 1 + 3, Paprikapulver: 1 ½ EL, Rindfleisch: 750 g"
	if got := strings.Join(lines, ", "); got != want {
		t.Errorf("Bring-Liste:\n%s\nerwartet:\n%s", got, want)
	}
	for _, s := range result.Sent {
		if s.Name == "Zwiebeln" && !s.Merged {
			t.Errorf("Zwiebeln: Merged = false, erwartet ergänzt")
		}
	}

	// Kein Planeintrag, kein Protokoll, Einkaufsliste unverändert.
	afterEntries, afterLogged := counts(t, store)
	if afterEntries != beforeEntries || afterLogged != beforeLogged {
		t.Errorf("Einträge %d→%d, Protokoll %d→%d", beforeEntries, afterEntries, beforeLogged, afterLogged)
	}
	previewAfter, _ := store.Preview(ctx, false)
	if summary(previewAfter.Items) != summary(previewBefore.Items) || len(previewAfter.Missing) != 0 {
		t.Errorf("Einkaufsliste geändert:\n%s\nvorher:\n%s", summary(previewAfter.Items), summary(previewBefore.Items))
	}
}

func TestSendRecipeValidation(t *testing.T) {
	ctx := context.Background()
	store, _ := setup(t)
	chooseList(t, store)
	gulasch := recipeID(t, store, "Gulasch")
	all := []string{nameKey("Rindfleisch"), nameKey("Zwiebeln"), nameKey("Paprikapulver"), nameKey("Knoblauch")}

	for name, tc := range map[string]struct {
		opts  RecipeSendOptions
		field string
	}{
		"0 Portionen":     {RecipeSendOptions{RecipeID: gulasch, Servings: 0}, "servings"},
		"51 Portionen":    {RecipeSendOptions{RecipeID: gulasch, Servings: 51}, "servings"},
		"unbekannt":       {RecipeSendOptions{RecipeID: 9999, Servings: 4}, "recipe_id"},
		"alles abgewählt": {RecipeSendOptions{RecipeID: gulasch, Servings: 4, Excluded: all}, "excluded"},
	} {
		var verr *apierr.Error
		if _, err := store.SendRecipe(ctx, tc.opts); !errors.As(err, &verr) || verr.Field != tc.field {
			t.Errorf("%s: err = %v, erwartet Feld %s", name, err, tc.field)
		}
	}
	items, _ := store.ha.OpenItems(ctx, homeassistant.TestListID)
	if len(items) != 0 {
		t.Errorf("nach Fehlern %d Artikel auf der Liste", len(items))
	}
}

func TestSendRecipeHandlers(t *testing.T) {
	store, mux := setup(t)
	chooseList(t, store)
	do := func(method, path, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
		return rec
	}

	rec := do("GET", "/api/v1/shopping/list", "")
	var ref ListRef
	json.Unmarshal(rec.Body.Bytes(), &ref)
	if rec.Code != http.StatusOK || ref.EntityID != homeassistant.TestListID || ref.Name == "" {
		t.Errorf("list: %d %s", rec.Code, rec.Body)
	}

	body := `{"recipe_id": ` + jsonInt(recipeID(t, store, "Pfannkuchen")) + `, "servings": 8, "excluded": ["` + nameKey("Ei") + `"]}`
	rec = do("POST", "/api/v1/shopping/recipe", body)
	var result SendResult
	json.Unmarshal(rec.Body.Bytes(), &result)
	if rec.Code != http.StatusOK || len(result.Sent) != 1 || result.Sent[0].Name != "Milch" || result.Sent[0].Spec != "500 ml" {
		t.Errorf("recipe: %d %s", rec.Code, rec.Body)
	}
	if rec := do("POST", "/api/v1/shopping/recipe", `{"recipe_id": 1, "servings": 0}`); rec.Code != http.StatusBadRequest {
		t.Errorf("0 Portionen: %d %s", rec.Code, rec.Body)
	}
}

func jsonInt(n int64) string {
	b, _ := json.Marshal(n)
	return string(b)
}
