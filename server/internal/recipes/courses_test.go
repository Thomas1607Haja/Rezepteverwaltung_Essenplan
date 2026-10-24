package recipes

import (
	"context"
	"errors"
	"testing"
)

func TestSuggestCourse(t *testing.T) {
	tests := []struct {
		title       string
		ingredients []string
		tags        []string
		want        string
	}{
		{"Schneller Apfelkuchen", nil, nil, "Kuchen & Torten"},
		{"Kartoffelsalat mit Gurke", nil, nil, "Salat"},
		{"Linseneintopf", nil, nil, "Suppe/Eintopf"},
		{"Schoko-Muffins", nil, nil, "Gebäck"},
		{"Dinkelbrötchen", nil, nil, "Brot"},
		{"Panna cotta mit Beeren", nil, nil, "Nachspeise"},
		{"Omas Rezept", []string{"Mehl", "Backpulver", "Zucker", "Butter"}, nil, "Kuchen & Torten"},
		{"Fladen vom Blech", []string{"Weizenmehl", "Trockenhefe", "Salz"}, nil, "Brot"},
		{"Grießflammeri", []string{"Grieß", "Milch"}, []string{"Süß"}, "Nachspeise"},
		{"Gulasch", []string{"Rindfleisch", "Zwiebeln"}, []string{"Rind"}, "Hauptgericht"},
		{"Gemüsecurry mit Reis", []string{"Reis"}, nil, "Hauptgericht"},
		{"Eis am Stiel", nil, nil, "Nachspeise"},
		{"Hummus-Dip", nil, nil, "Snack"},
	}
	for _, tt := range tests {
		if got, _ := suggestCourse(tt.title, tt.ingredients, tt.tags); got != tt.want {
			t.Errorf("%s: %q, erwartet %q", tt.title, got, tt.want)
		}
	}
}

func TestCourseSuggestionsAndAssign(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	kuchen, err := s.Create(ctx, Recipe{Title: "Zitronenkuchen", BaseServings: 12})
	if err != nil {
		t.Fatal(err)
	}
	// Hat schon einen Gang: kein Vorschlag.
	if _, err := s.Create(ctx, Recipe{Title: "Tomatensuppe", BaseServings: 4, Tags: []Tag{{ID: tagID(t, s, "Hauptgericht")}}}); err != nil {
		t.Fatal(err)
	}
	list, err := s.CourseSuggestions(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Course != "Kuchen & Torten" || list[0].RecipeID != kuchen.ID {
		t.Fatalf("Vorschläge = %+v", list)
	}
	n, err := s.AssignCourses(ctx, []CourseAssignment{{RecipeID: kuchen.ID, TagID: list[0].TagID}})
	if err != nil || n != 1 {
		t.Fatalf("Zuordnen: %d %v", n, err)
	}
	if list, _ := s.CourseSuggestions(ctx); len(list) != 0 {
		t.Errorf("nach dem Zuordnen noch Vorschläge: %+v", list)
	}
	// Nur Gänge sind erlaubt.
	var verr *ValidationError
	if _, err := s.AssignCourses(ctx, []CourseAssignment{{RecipeID: kuchen.ID, TagID: tagID(t, s, "Rind")}}); !errors.As(err, &verr) {
		t.Errorf("Rind als Gang: %v", err)
	}
	// Gang-Kennzeichen umsetzen: höchstens eine Gruppe.
	groups, _ := s.AdminTags(ctx)
	if err := s.SetCourseGroup(ctx, groups[1].ID, true); err != nil {
		t.Fatal(err)
	}
	groups, _ = s.AdminTags(ctx)
	courses := 0
	for _, g := range groups {
		if g.Role == RoleCourse {
			courses++
		}
	}
	if courses != 1 || groups[1].Role != RoleCourse {
		t.Errorf("Gang-Gruppen: %d, zweite %q", courses, groups[1].Role)
	}
}
