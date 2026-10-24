package recipes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestBulkTagsAddRemoveUndo(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	clock := time.Now().Add(time.Hour).Truncate(time.Second)
	s.now = func() time.Time { return clock }
	main, soup := tagID(t, s, "Hauptgericht"), tagID(t, s, "Suppe/Eintopf")
	a, _ := s.Create(ctx, Recipe{Title: "Gulasch", BaseServings: 4})
	b, _ := s.Create(ctx, Recipe{Title: "Linsensuppe", BaseServings: 4, Tags: []Tag{{ID: soup}}})
	c, _ := s.Create(ctx, Recipe{Title: "Brot", BaseServings: 1})

	clock = clock.Add(time.Minute)
	// b hat schon Suppe/Eintopf und bekommt Hauptgericht dazu (hinzufügen, nicht ersetzen).
	res, err := s.BulkTags(ctx, BulkTags{Add: []TagChange{{a.ID, main}, {b.ID, main}, {b.ID, soup}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Added) != 2 || res.Recipes != 2 {
		t.Fatalf("hinzugefügt: %+v", res)
	}
	got, _ := s.Get(ctx, b.ID)
	if len(got.Tags) != 2 {
		t.Errorf("b: %+v", got.Tags)
	}
	if !got.UpdatedAt.Equal(clock) {
		t.Errorf("updated_at nicht gesetzt: %v", got.UpdatedAt)
	}
	if unchanged, _ := s.Get(ctx, c.ID); unchanged.UpdatedAt.Equal(clock) {
		t.Error("c ohne Änderung bekam neues updated_at")
	}

	// Rückgängig: Listen vertauscht zurückschicken.
	undo, err := s.BulkTags(ctx, BulkTags{Add: res.Removed, Remove: res.Added})
	if err != nil || len(undo.Removed) != 2 {
		t.Fatalf("rückgängig: %+v %v", undo, err)
	}
	if got, _ := s.Get(ctx, a.ID); len(got.Tags) != 0 {
		t.Errorf("a nach Rückgängig: %+v", got.Tags)
	}
	if got, _ := s.Get(ctx, b.ID); len(got.Tags) != 1 || got.Tags[0].ID != soup {
		t.Errorf("b nach Rückgängig: %+v", got.Tags)
	}

	// Unbekannte IDs: nichts ändert sich.
	var verr *ValidationError
	if _, err := s.BulkTags(ctx, BulkTags{Add: []TagChange{{a.ID, main}, {9999, main}}}); !errors.As(err, &verr) {
		t.Errorf("unbekanntes Rezept: %v", err)
	}
	if _, err := s.BulkTags(ctx, BulkTags{Add: []TagChange{{a.ID, 9999}}}); !errors.As(err, &verr) {
		t.Errorf("unbekannte Kategorie: %v", err)
	}
	if got, _ := s.Get(ctx, a.ID); len(got.Tags) != 0 {
		t.Errorf("trotz Fehler geändert: %+v", got.Tags)
	}
	big := make([]TagChange, MaxBulkTagChanges+1)
	if _, err := s.BulkTags(ctx, BulkTags{Add: big}); !errors.As(err, &verr) {
		t.Errorf("zu viele Änderungen: %v", err)
	}
}

func TestListWithoutCourse(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	main := tagID(t, s, "Hauptgericht")
	rind := tagID(t, s, "Rind")
	s.Create(ctx, Recipe{Title: "Gulasch", BaseServings: 4, Tags: []Tag{{ID: main}, {ID: rind}}})
	s.Create(ctx, Recipe{Title: "Rinderbrühe", BaseServings: 4, Tags: []Tag{{ID: rind}}})
	s.Create(ctx, Recipe{Title: "Apfelkuchen", BaseServings: 1, IsFavorite: true})

	titles := func(f ListFilter) string {
		list, err := s.List(ctx, f)
		if err != nil {
			t.Fatal(err)
		}
		out := ""
		for _, r := range list {
			out += r.Title + ";"
		}
		return out
	}
	if got := titles(ListFilter{WithoutCourse: true}); got != "Apfelkuchen;Rinderbrühe;" {
		t.Errorf("ohne Gang: %s", got)
	}
	// Kombinierbar mit Suche, Kategorien und Favoriten.
	if got := titles(ListFilter{WithoutCourse: true, Query: "kuchen"}); got != "Apfelkuchen;" {
		t.Errorf("mit Suche: %s", got)
	}
	if got := titles(ListFilter{WithoutCourse: true, TagIDs: []int64{rind}}); got != "Rinderbrühe;" {
		t.Errorf("mit Kategorie: %s", got)
	}
	if got := titles(ListFilter{WithoutCourse: true, FavoritesOnly: true}); got != "Apfelkuchen;" {
		t.Errorf("mit Favoriten: %s", got)
	}
	// Die Kacheln kennen ihre Kategorien.
	list, _ := s.List(ctx, ListFilter{Query: "gulasch"})
	if len(list) != 1 || fmt.Sprint(list[0].TagIDs) != fmt.Sprint(sortedIDs(main, rind)) {
		t.Errorf("tag_ids: %+v", list)
	}
}

func sortedIDs(a, b int64) []int64 {
	if a > b {
		return []int64{b, a}
	}
	return []int64{a, b}
}

func TestBulkTagsAndWithoutCourseOverHTTP(t *testing.T) {
	mux := newTestMux(t)
	rec := do(mux, "POST", "/api/v1/recipes", `{"title": "Pfannkuchen", "base_servings": 2}`)
	var r Recipe
	json.Unmarshal(rec.Body.Bytes(), &r)

	rec = do(mux, "GET", "/api/v1/recipes?course=none&q=pfann", "")
	var list listResponse
	json.Unmarshal(rec.Body.Bytes(), &list)
	if rec.Code != 200 || len(list.Recipes) != 1 || list.Recipes[0].TagIDs == nil {
		t.Fatalf("ohne Gang: %d %s", rec.Code, rec.Body)
	}

	rec = do(mux, "GET", "/api/v1/tags", "")
	var tags struct{ Groups []TagGroup }
	json.Unmarshal(rec.Body.Bytes(), &tags)
	var course int64
	for _, g := range tags.Groups {
		if g.Role == RoleCourse {
			course = g.Tags[0].ID
		}
	}
	rec = do(mux, "POST", "/api/v1/tags/bulk", fmt.Sprintf(`{"add": [{"recipe_id": %d, "tag_id": %d}]}`, r.ID, course))
	var res BulkTagsResult
	json.Unmarshal(rec.Body.Bytes(), &res)
	if rec.Code != 200 || len(res.Added) != 1 || res.Removed == nil {
		t.Fatalf("bulk: %d %s", rec.Code, rec.Body)
	}
	rec = do(mux, "GET", "/api/v1/recipes?course=none", "")
	json.Unmarshal(rec.Body.Bytes(), &list)
	if len(list.Recipes) != 0 {
		t.Errorf("nach Zuweisen noch ohne Gang: %s", rec.Body)
	}
	if rec := do(mux, "POST", "/api/v1/tags/bulk", `{"add": [{"recipe_id": 999, "tag_id": 1}]}`); rec.Code != 400 {
		t.Errorf("unbekanntes Rezept: %d %s", rec.Code, rec.Body)
	}
}
