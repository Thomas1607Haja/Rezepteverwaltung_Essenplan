package recipes

import (
	"context"
	"errors"
	"slices"
	"testing"
)

// seedList legt eine kleine Rezeptsammlung an und liefert die Kategorie-IDs
// nach Namen. Reihenfolge des Anlegens: Spaghetti, Äpfel, Risotto, Steak.
func seedList(t *testing.T, s *Store) map[string]int64 {
	t.Helper()
	tags := make(map[string]int64)
	for _, name := range []string{"Vegetarisch", "Rind", "Pasta", "Reis", "Süß"} {
		tags[name] = tagID(t, s, name)
	}

	recipes := []Recipe{
		{Title: "Spaghetti Napoli", BaseServings: 4, IsFavorite: true,
			Ingredients: []Ingredient{{Name: "Spaghetti"}, {Name: "Tomaten"}},
			Tags:        []Tag{{ID: tags["Vegetarisch"]}, {ID: tags["Pasta"]}}},
		{Title: "Äpfel im Schlafrock", BaseServings: 4,
			Description: "Mit 100% Butter",
			Ingredients: []Ingredient{{Name: "Äpfel"}, {Name: "Blätterteig"}},
			Tags:        []Tag{{ID: tags["Vegetarisch"]}, {ID: tags["Süß"]}}},
		{Title: "Pilzrisotto", BaseServings: 2,
			Ingredients: []Ingredient{{Name: "Risottoreis"}, {Name: "Champignons"}, {Name: "Parmesan"}},
			Steps:       []Step{{Text: "Unter Rühren köcheln lassen."}},
			Tags:        []Tag{{ID: tags["Vegetarisch"]}, {ID: tags["Reis"]}}},
		{Title: "Steak mit Tomaten", BaseServings: 2, IsFavorite: true,
			Ingredients: []Ingredient{{Name: "Rumpsteak"}, {Name: "Tomaten"}},
			Tags:        []Tag{{ID: tags["Rind"]}}},
	}
	for _, r := range recipes {
		if _, err := s.Create(context.Background(), r); err != nil {
			t.Fatalf("Create %q: %v", r.Title, err)
		}
	}
	return tags
}

// titles holt die Titel aus einer Liste, damit Vergleiche kurz bleiben.
func titles(list []Summary) []string {
	out := make([]string, len(list))
	for i, sum := range list {
		out[i] = sum.Title
	}
	return out
}

func TestList(t *testing.T) {
	s := newTestStore(t)
	tags := seedList(t, s)

	tests := []struct {
		name   string
		filter ListFilter
		want   []string
	}{
		{"alle nach Name, Ä wie A", ListFilter{},
			[]string{"Äpfel im Schlafrock", "Pilzrisotto", "Spaghetti Napoli", "Steak mit Tomaten"}},
		{"neueste zuerst", ListFilter{Sort: SortNewest},
			[]string{"Steak mit Tomaten", "Pilzrisotto", "Äpfel im Schlafrock", "Spaghetti Napoli"}},

		{"Suche im Titel, Groß/klein egal", ListFilter{Query: "SPAGHETTI"},
			[]string{"Spaghetti Napoli"}},
		{"Suche ohne Umlaut findet Umlaut", ListFilter{Query: "apfel"},
			[]string{"Äpfel im Schlafrock"}},
		{"Suche in Zutaten", ListFilter{Query: "parmesan"},
			[]string{"Pilzrisotto"}},
		{"Suche in Zubereitung, Umlaut egal", ListFilter{Query: "ruhren"},
			[]string{"Pilzrisotto"}},
		{"Suche in Beschreibung", ListFilter{Query: "butter"},
			[]string{"Äpfel im Schlafrock"}},
		{"mehrere Wörter müssen alle passen", ListFilter{Query: "tomaten steak"},
			[]string{"Steak mit Tomaten"}},
		{"Prozentzeichen ist kein Platzhalter", ListFilter{Query: "100%"},
			[]string{"Äpfel im Schlafrock"}},
		{"Unterstrich ist kein Platzhalter", ListFilter{Query: "s_eak"},
			[]string{}},

		{"nur Favoriten", ListFilter{FavoritesOnly: true},
			[]string{"Spaghetti Napoli", "Steak mit Tomaten"}},
		{"eine Kategorie", ListFilter{TagIDs: []int64{tags["Vegetarisch"]}},
			[]string{"Äpfel im Schlafrock", "Pilzrisotto", "Spaghetti Napoli"}},
		{"gleiche Gruppe: oder", ListFilter{TagIDs: []int64{tags["Pasta"], tags["Reis"]}},
			[]string{"Pilzrisotto", "Spaghetti Napoli"}},
		{"verschiedene Gruppen: und", ListFilter{TagIDs: []int64{tags["Vegetarisch"], tags["Süß"]}},
			[]string{"Äpfel im Schlafrock"}},
		{"doppelte Kategorie stört nicht", ListFilter{TagIDs: []int64{tags["Rind"], tags["Rind"]}},
			[]string{"Steak mit Tomaten"}},
		{"alles kombiniert", ListFilter{Query: "tomaten", FavoritesOnly: true,
			TagIDs: []int64{tags["Vegetarisch"]}},
			[]string{"Spaghetti Napoli"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			list, err := s.List(context.Background(), tt.filter)
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			if got := titles(list); !slices.Equal(got, tt.want) {
				t.Errorf("Titel = %q, erwartet %q", got, tt.want)
			}
		})
	}
}

func TestListSummaryFields(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Create(context.Background(), lasagne(t, s)); err != nil {
		t.Fatalf("Create: %v", err)
	}

	list, err := s.List(context.Background(), ListFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("%d Rezepte, erwartet 1", len(list))
	}
	sum := list[0]
	if sum.ID == 0 || sum.Title != "Lasagne" || !sum.IsFavorite ||
		sum.PrepMinutes == nil || *sum.PrepMinutes != 30 || sum.TotalMinutes != nil {
		t.Errorf("Kurzfassung = %+v", sum)
	}
}

func TestListEmptyIsNotNil(t *testing.T) {
	s := newTestStore(t)
	list, err := s.List(context.Background(), ListFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if list == nil {
		t.Error("leere Liste ist nil, erwartet [] für JSON")
	}
}

func TestListRejectsInvalidFilter(t *testing.T) {
	s := newTestStore(t)

	tests := []struct {
		name      string
		filter    ListFilter
		wantField string
	}{
		{"unbekannte Sortierung", ListFilter{Sort: "rating"}, "sort"},
		{"unbekannte Kategorie", ListFilter{TagIDs: []int64{tagID(t, s, "Pasta"), 9999}}, "tag"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := s.List(context.Background(), tt.filter)
			var verr *ValidationError
			if !errors.As(err, &verr) {
				t.Fatalf("Fehler = %v, erwartet ValidationError", err)
			}
			if verr.Field != tt.wantField {
				t.Errorf("Feld = %q, erwartet %q", verr.Field, tt.wantField)
			}
		})
	}
}

func TestFindBySourceURL(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	const u = "https://www.chefkoch.de/rezepte/1/gulasch.html"
	for _, r := range []Recipe{
		{Title: "Gulasch", BaseServings: 4, SourceURL: u},
		{Title: "Gulasch (meine Version)", BaseServings: 4, SourceURL: u},
		{Title: "Ohne Quelle", BaseServings: 2},
	} {
		if _, err := s.Create(ctx, r); err != nil {
			t.Fatal(err)
		}
	}

	found, err := s.FindBySourceURL(ctx, u)
	if err != nil {
		t.Fatal(err)
	}
	if got := titles(found); !slices.Equal(got, []string{"Gulasch", "Gulasch (meine Version)"}) {
		t.Errorf("gefunden = %q", got)
	}
	if found, _ := s.FindBySourceURL(ctx, ""); len(found) != 0 {
		t.Errorf("leere Adresse findet %d Rezepte, erwartet 0", len(found))
	}
}
