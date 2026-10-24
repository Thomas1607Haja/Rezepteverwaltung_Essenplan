package fileimport

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

// confirmWith bestätigt mit einer Wahl für unbekannte Kategorien.
func (f fixture) confirmWith(t *testing.T, token string, choices map[string]CategoryChoice, selected ...int) Result {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"selected": selected, "categories": choices})
	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, httptest.NewRequest("POST", "/api/v1/import/file/"+token, bytes.NewReader(body)))
	if rec.Code != http.StatusOK {
		t.Fatalf("Bestätigen: %d %s", rec.Code, rec.Body)
	}
	var r Result
	if err := json.Unmarshal(rec.Body.Bytes(), &r); err != nil {
		t.Fatal(err)
	}
	return r
}

// Unbekannte Kategorien: einmal je Name in der Vorschau, beim Bestätigen
// zuordnen, neu anlegen oder weglassen.
func TestUnknownCategoriesCanBeMappedOrCreated(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	doc := `{"format":"rezeptplaner-import","version":2,"recipes":[
		{"title":"Ofenkartoffeln","base_servings":2,"categories":["Ofengericht","Familienliebling","Low Carb"]},
		{"title":"Auflauf","base_servings":2,"categories":["ofengericht","Kartoffeln"]}]}`
	code, p, body := f.upload(t, namedFile{"r.json", []byte(doc)})
	if code != http.StatusOK {
		t.Fatalf("%d %s", code, body)
	}
	if !slices.Equal(p.UnknownCategories, []string{"Ofengericht", "Familienliebling", "Low Carb"}) {
		t.Errorf("unknown_categories = %v", p.UnknownCategories)
	}
	if !slices.Equal(p.Recipes[1].UnknownCategories, []string{"ofengericht"}) {
		t.Errorf("Rezept 2: %v", p.Recipes[1].UnknownCategories)
	}

	groups, _ := f.recipes.ListTags(ctx)
	var eigenschaften, kinderliebling int64
	for _, g := range groups {
		if g.Name == "Eigenschaften" {
			eigenschaften = g.ID
			for _, tag := range g.Tags {
				if tag.Name == "Kinderliebling" {
					kinderliebling = tag.ID
				}
			}
		}
	}
	result := f.confirmWith(t, p.Token, map[string]CategoryChoice{
		"Ofengericht":      {NewInGroup: eigenschaften},
		"Familienliebling": {TagID: kinderliebling},
		// "Low Carb" fehlt: weglassen
	}, 0, 1)
	if len(result.Created) != 2 || len(result.Warnings) != 0 {
		t.Fatalf("Ergebnis: %+v", result)
	}
	tags := func(id int64) []string {
		r, _ := f.recipes.Get(ctx, id)
		names := []string{}
		for _, tag := range r.Tags {
			names = append(names, tag.Name)
		}
		slices.Sort(names)
		return names
	}
	if got := tags(result.Created[0].ID); !slices.Equal(got, []string{"Kinderliebling", "Ofengericht"}) {
		t.Errorf("Ofenkartoffeln: %v", got)
	}
	if got := tags(result.Created[1].ID); !slices.Equal(got, []string{"Kartoffeln", "Ofengericht"}) {
		t.Errorf("Auflauf: %v", got)
	}
}
