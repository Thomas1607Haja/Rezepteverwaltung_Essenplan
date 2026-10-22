package recipes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCreateVariantCopiesRecipe(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	orig, err := s.Create(ctx, lasagne(t, s))
	if err != nil {
		t.Fatal(err)
	}
	// Ein Planeintrag des Originals darf nicht mitwandern.
	if _, err := s.db.Exec(`INSERT INTO meal_plan_entries (date, recipe_id, servings, created_at, updated_at)
		VALUES ('2026-10-24', ?, 4, 'x', 'x')`, orig.ID); err != nil {
		t.Fatal(err)
	}

	v, err := s.CreateVariant(ctx, orig.ID, "aaaa.jpg", "aaaa_thumb.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if v.ID == orig.ID || v.Title != "Lasagne (Variante)" {
		t.Errorf("ID %d, Titel %q", v.ID, v.Title)
	}
	if v.ParentRecipeID == nil || *v.ParentRecipeID != orig.ID {
		t.Errorf("parent_recipe_id = %v, erwartet %d", v.ParentRecipeID, orig.ID)
	}
	if v.IsFavorite {
		t.Error("Favoritenstatus wurde mitkopiert")
	}
	if v.PlanCount != 0 {
		t.Errorf("Planeinträge: %d, erwartet 0", v.PlanCount)
	}
	if v.Image != "aaaa.jpg" || v.Thumbnail != "aaaa_thumb.jpg" {
		t.Errorf("Bild %q / %q", v.Image, v.Thumbnail)
	}
	if v.BaseServings != orig.BaseServings || *v.PrepMinutes != *orig.PrepMinutes || v.SourceURL != orig.SourceURL {
		t.Errorf("Felder nicht kopiert: %+v", v)
	}

	// Zutaten mit Reihenfolge, Gruppe und Einkaufsrelevanz; neue IDs.
	if len(v.Ingredients) != len(orig.Ingredients) {
		t.Fatalf("%d Zutaten, erwartet %d", len(v.Ingredients), len(orig.Ingredients))
	}
	for i, ing := range v.Ingredients {
		o := orig.Ingredients[i]
		if ing.ID == o.ID || ing.Name != o.Name || ing.GroupName != o.GroupName || ing.ShoppingRelevant != o.ShoppingRelevant ||
			(ing.Quantity == nil) != (o.Quantity == nil) {
			t.Errorf("Zutat %d: %+v, erwartet wie %+v", i, ing, o)
		}
	}
	if len(v.Steps) != 2 || v.Steps[0].Text != orig.Steps[0].Text || v.Steps[1].Text != orig.Steps[1].Text {
		t.Errorf("Schritte: %+v", v.Steps)
	}
	if len(v.Tags) != len(orig.Tags) {
		t.Errorf("Kategorien: %+v", v.Tags)
	}

	// Änderungen an der Variante lassen das Original unberührt.
	v.Ingredients = v.Ingredients[:1]
	if _, err := s.Update(ctx, v.ID, v); err != nil {
		t.Fatal(err)
	}
	if n := countRows(t, s, "recipe_ingredients", orig.ID); n != 3 {
		t.Errorf("Original hat %d Zutaten, erwartet 3", n)
	}

	if _, err := s.CreateVariant(ctx, 999, "", ""); !errors.Is(err, ErrNotFound) {
		t.Errorf("unbekanntes Rezept: %v", err)
	}
}

func TestVariantFamilyHasOneLevel(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	orig, _ := s.Create(ctx, Recipe{Title: "Linsensuppe", BaseServings: 4})
	v1, _ := s.CreateVariant(ctx, orig.ID, "", "")
	// Variante einer Variante: hängt trotzdem am Ursprungsrezept, Titel ohne
	// doppeltes „(Variante)“.
	v2, err := s.CreateVariant(ctx, v1.ID, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if *v2.ParentRecipeID != orig.ID || v2.Title != "Linsensuppe (Variante)" {
		t.Errorf("v2: parent %d, Titel %q", *v2.ParentRecipeID, v2.Title)
	}

	ids := func(f []FamilyMember) []int64 {
		var r []int64
		for _, m := range f {
			r = append(r, m.ID)
		}
		return r
	}
	o, _ := s.Get(ctx, orig.ID)
	if got := ids(o.Family); len(got) != 2 || got[0] != v1.ID || got[1] != v2.ID {
		t.Errorf("Familie des Ursprungsrezepts: %v", got)
	}
	got, _ := s.Get(ctx, v2.ID)
	if f := got.Family; len(f) != 2 || f[0].ID != orig.ID || !f[0].IsOrigin || f[1].ID != v1.ID || f[1].IsOrigin {
		t.Errorf("Familie von v2: %+v", f)
	}
	single, _ := s.Create(ctx, Recipe{Title: "Allein", BaseServings: 2})
	if single.Family == nil || len(single.Family) != 0 || single.ParentRecipeID != nil {
		t.Errorf("Rezept ohne Familie: %+v / %v", single.Family, single.ParentRecipeID)
	}

	list, _ := s.List(ctx, ListFilter{Query: "Linsensuppe"})
	variants := 0
	for _, sum := range list {
		if sum.ParentRecipeID != nil {
			variants++
		}
	}
	if len(list) != 3 || variants != 2 {
		t.Errorf("Liste: %d Rezepte, davon %d Varianten", len(list), variants)
	}
}

func TestDeleteOriginPromotesOldestVariant(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	orig, _ := s.Create(ctx, Recipe{Title: "Chili", BaseServings: 4})
	v1, _ := s.CreateVariant(ctx, orig.ID, "", "")
	v2, _ := s.CreateVariant(ctx, orig.ID, "", "")
	v3, _ := s.CreateVariant(ctx, orig.ID, "", "")
	time.Sleep(5 * time.Millisecond) // damit updated_at sich messbar ändert

	if _, err := s.Delete(ctx, orig.ID); err != nil {
		t.Fatal(err)
	}
	n1, _ := s.Get(ctx, v1.ID)
	if n1.ParentRecipeID != nil {
		t.Errorf("älteste Variante hat noch parent %d", *n1.ParentRecipeID)
	}
	if !n1.UpdatedAt.After(v1.UpdatedAt) {
		t.Error("updated_at des neuen Ursprungsrezepts unverändert")
	}
	for _, v := range []Recipe{v2, v3} {
		got, _ := s.Get(ctx, v.ID)
		if got.ParentRecipeID == nil || *got.ParentRecipeID != v1.ID {
			t.Errorf("Variante %d hängt an %v, erwartet %d", v.ID, got.ParentRecipeID, v1.ID)
		}
		if !got.UpdatedAt.After(v.UpdatedAt) {
			t.Errorf("updated_at von %d unverändert", v.ID)
		}
	}
	if len(n1.Family) != 2 {
		t.Errorf("Familie des neuen Ursprungsrezepts: %+v", n1.Family)
	}

	// Eine Variante löschen: nur sie verschwindet.
	if _, err := s.Delete(ctx, v3.ID); err != nil {
		t.Fatal(err)
	}
	n1, _ = s.Get(ctx, v1.ID)
	if len(n1.Family) != 1 || n1.Family[0].ID != v2.ID {
		t.Errorf("Familie nach Löschen einer Variante: %+v", n1.Family)
	}
}

func TestCreateVariantOverHTTP(t *testing.T) {
	mux := newTestMux(t)
	do(mux, "POST", "/api/v1/recipes", `{"title": "Pfannkuchen", "base_servings": 2}`)
	var jpg bytes.Buffer
	jpeg.Encode(&jpg, image.NewRGBA(image.Rect(0, 0, 800, 600)), nil)
	req := httptest.NewRequest("PUT", "/api/v1/recipes/1/image", bytes.NewReader(jpg.Bytes()))
	req.Header.Set("Content-Type", "image/jpeg")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	var orig Recipe
	json.Unmarshal(rec.Body.Bytes(), &orig)

	rec = do(mux, "POST", "/api/v1/recipes/1/variants", "")
	if rec.Code != http.StatusCreated || rec.Header().Get("Location") != "/api/v1/recipes/2" {
		t.Fatalf("Status %d, Location %q: %s", rec.Code, rec.Header().Get("Location"), rec.Body)
	}
	var v Recipe
	json.Unmarshal(rec.Body.Bytes(), &v)
	// Eigene Bilddateien, beide abrufbar.
	if v.Image == "" || v.Image == orig.Image || v.Thumbnail == orig.Thumbnail {
		t.Errorf("Bild der Variante %q / %q, Original %q", v.Image, v.Thumbnail, orig.Image)
	}
	for _, name := range []string{v.Image, v.Thumbnail, orig.Image} {
		if rec := do(mux, "GET", "/api/v1/images/"+name, ""); rec.Code != http.StatusOK {
			t.Errorf("Bild %s: Status %d", name, rec.Code)
		}
	}
	// Bild der Variante entfernen: das Original bleibt.
	do(mux, "DELETE", "/api/v1/recipes/2/image", "")
	if rec := do(mux, "GET", "/api/v1/images/"+orig.Image, ""); rec.Code != http.StatusOK {
		t.Errorf("Originalbild nach Entfernen des Variantenbilds: Status %d", rec.Code)
	}

	// Abruf liefert Familie und parent_recipe_id.
	got := do(mux, "GET", "/api/v1/recipes/2", "").Body.String()
	var body map[string]any
	json.Unmarshal([]byte(got), &body)
	if body["parent_recipe_id"] != float64(1) || len(body["family"].([]any)) != 1 {
		t.Errorf("Abruf: parent %v, family %v", body["parent_recipe_id"], body["family"])
	}
	// Das abgerufene Rezept unverändert zurückschicken geht; parent_recipe_id
	// ist nur lesbar und bleibt.
	if rec := do(mux, "PUT", "/api/v1/recipes/2", got); rec.Code != http.StatusOK {
		t.Errorf("PUT: Status %d: %s", rec.Code, rec.Body)
	}
	json.Unmarshal(do(mux, "GET", "/api/v1/recipes/2", "").Body.Bytes(), &v)
	if v.ParentRecipeID == nil || *v.ParentRecipeID != 1 {
		t.Errorf("parent nach PUT: %v", v.ParentRecipeID)
	}

	if rec := do(mux, "POST", "/api/v1/recipes/42/variants", ""); rec.Code != http.StatusNotFound {
		t.Errorf("unbekanntes Rezept: Status %d, erwartet 404", rec.Code)
	}
}
