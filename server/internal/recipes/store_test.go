package recipes

import (
	"context"
	"errors"
	"testing"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/database"
)

// newTestStore liefert einen Store auf einer frischen, migrierten
// Datenbank im Arbeitsspeicher.
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

// tagID sucht die ID einer Startkategorie über ihren Namen.
func tagID(t *testing.T, s *Store, name string) int64 {
	t.Helper()
	var id int64
	if err := s.db.QueryRow("SELECT id FROM tags WHERE name = ?", name).Scan(&id); err != nil {
		t.Fatalf("Kategorie %q: %v", name, err)
	}
	return id
}

// lasagne ist ein vollständiges Beispielrezept für die Tests.
func lasagne(t *testing.T, s *Store) Recipe {
	return Recipe{
		Title:        "  Lasagne  ",
		BaseServings: 4,
		PrepMinutes:  new(30), // new(Wert) liefert einen Zeiger auf eine Kopie des Werts
		SourceURL:    "https://example.org/lasagne",
		IsFavorite:   true,
		Image:        "../../geheim.jpg", // darf nicht übernommen werden
		Ingredients: []Ingredient{
			{Quantity: new(500.0), Unit: "g", Name: "Hackfleisch", ShoppingRelevant: true},
			{GroupName: "Für die Soße", Quantity: new(1.0), Unit: "Dose", Name: "Tomaten", ShoppingRelevant: true},
			{GroupName: "Für die Soße", Name: "Salz", Note: "nach Geschmack"},
		},
		Steps: []Step{
			{Text: "Hackfleisch anbraten."},
			{Text: "Schichten und 45 Minuten backen."},
		},
		// Doppelte ID absichtlich: darf keinen Fehler auslösen.
		Tags: []Tag{{ID: tagID(t, s, "Hackfleisch")}, {ID: tagID(t, s, "Auflauf/Ofen")}, {ID: tagID(t, s, "Hackfleisch")}},
	}
}

func TestCreateAndGet(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	created, err := s.Create(ctx, lasagne(t, s))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == 0 {
		t.Fatal("ID wurde nicht vergeben")
	}

	got, err := s.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if got.Title != "Lasagne" {
		t.Errorf("Title = %q, erwartet Leerzeichen entfernt", got.Title)
	}
	if got.PrepMinutes == nil || *got.PrepMinutes != 30 {
		t.Errorf("PrepMinutes = %v, erwartet 30", got.PrepMinutes)
	}
	if got.TotalMinutes != nil {
		t.Errorf("TotalMinutes = %v, erwartet nil", *got.TotalMinutes)
	}
	if !got.IsFavorite {
		t.Error("IsFavorite = false, erwartet true")
	}
	if got.Image != "" {
		t.Errorf("Image = %q, darf beim Anlegen nicht gesetzt werden", got.Image)
	}
	if got.CreatedAt.IsZero() || !got.CreatedAt.Equal(got.UpdatedAt) {
		t.Errorf("Zeitstempel: created %v, updated %v", got.CreatedAt, got.UpdatedAt)
	}

	// Zutaten: Reihenfolge, Mengen und "nicht umrechnen" (nil) müssen erhalten bleiben.
	if len(got.Ingredients) != 3 {
		t.Fatalf("%d Zutaten, erwartet 3", len(got.Ingredients))
	}
	wantNames := []string{"Hackfleisch", "Tomaten", "Salz"}
	for i, want := range wantNames {
		if got.Ingredients[i].Name != want {
			t.Errorf("Zutat %d = %q, erwartet %q", i, got.Ingredients[i].Name, want)
		}
	}
	if q := got.Ingredients[0].Quantity; q == nil || *q != 500 {
		t.Errorf("Menge Hackfleisch = %v, erwartet 500", q)
	}
	if got.Ingredients[2].Quantity != nil {
		t.Errorf("Menge Salz = %v, erwartet nil", *got.Ingredients[2].Quantity)
	}
	if got.Ingredients[2].ShoppingRelevant {
		t.Error("Salz ist einkaufsrelevant, erwartet nicht")
	}
	if got.Ingredients[1].GroupName != "Für die Soße" {
		t.Errorf("Gruppe = %q", got.Ingredients[1].GroupName)
	}

	if len(got.Steps) != 2 || got.Steps[0].Text != "Hackfleisch anbraten." {
		t.Errorf("Schritte = %+v", got.Steps)
	}

	if len(got.Tags) != 2 {
		t.Fatalf("%d Kategorien, erwartet 2 (doppelte ID zusammengefasst)", len(got.Tags))
	}
	// Gruppen in Anlege-Reihenfolge: "Hauptzutat" kommt vor "Art".
	if got.Tags[0].Name != "Hackfleisch" || got.Tags[1].Name != "Auflauf/Ofen" {
		t.Errorf("Kategorien = %+v", got.Tags)
	}
}

func TestCreateMinimalRecipe(t *testing.T) {
	s := newTestStore(t)

	got, err := s.Create(context.Background(), Recipe{Title: "Rührei", BaseServings: 2})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	// Leere Listen statt nil, damit die API [] statt null liefert.
	if got.Ingredients == nil || got.Steps == nil || got.Tags == nil {
		t.Errorf("Listen sind nil: %+v", got)
	}
}

// Tabellengesteuerter Test: jede Zeile ist ein eigener Fall mit eigenem Namen.
func TestCreateRejectsInvalidRecipes(t *testing.T) {
	s := newTestStore(t)

	tests := []struct {
		name      string
		recipe    Recipe
		wantField string
	}{
		{"leerer Titel", Recipe{Title: "   ", BaseServings: 2}, "title"},
		{"keine Portionen", Recipe{Title: "X", BaseServings: 0}, "base_servings"},
		{"negative Zeit", Recipe{Title: "X", BaseServings: 2, TotalMinutes: new(-5)}, "total_minutes"},
		{"Zutat ohne Namen", Recipe{Title: "X", BaseServings: 2,
			Ingredients: []Ingredient{{Name: "Mehl"}, {Name: " "}}}, "ingredients[1].name"},
		{"negative Menge", Recipe{Title: "X", BaseServings: 2,
			Ingredients: []Ingredient{{Name: "Mehl", Quantity: new(-1.0)}}}, "ingredients[0].quantity"},
		{"leerer Schritt", Recipe{Title: "X", BaseServings: 2,
			Steps: []Step{{Text: ""}}}, "steps[0].text"},
		{"unbekannte Kategorie", Recipe{Title: "X", BaseServings: 2,
			Tags: []Tag{{ID: 9999}}}, "tags[0].id"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := s.Create(context.Background(), tt.recipe)

			var verr *ValidationError
			if !errors.As(err, &verr) {
				t.Fatalf("Fehler = %v, erwartet ValidationError", err)
			}
			if verr.Field != tt.wantField {
				t.Errorf("Feld = %q, erwartet %q", verr.Field, tt.wantField)
			}
		})
	}

	// Kein ungültiges Rezept darf (auch nicht teilweise) gespeichert sein.
	var count int
	if err := s.db.QueryRow("SELECT count(*) FROM recipes").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Errorf("%d Rezepte gespeichert, erwartet 0", count)
	}
}

func TestGetUnknownRecipe(t *testing.T) {
	s := newTestStore(t)

	_, err := s.Get(context.Background(), 42)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Fehler = %v, erwartet ErrNotFound", err)
	}
}

// countRows zählt die Zeilen von table, die zu recipeID gehören.
func countRows(t *testing.T, s *Store, table string, recipeID int64) int {
	t.Helper()
	var n int
	if err := s.db.QueryRow("SELECT count(*) FROM "+table+" WHERE recipe_id = ?", recipeID).Scan(&n); err != nil {
		t.Fatalf("%s zählen: %v", table, err)
	}
	return n
}

func TestUpdate(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	created, err := s.Create(ctx, lasagne(t, s))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	// Ein Bild gibt es erst mit dem Upload in Phase 2; hier direkt setzen.
	if _, err := s.db.Exec("UPDATE recipes SET image = 'lasagne.jpg' WHERE id = ?", created.ID); err != nil {
		t.Fatal(err)
	}

	changed := Recipe{
		Title:        "Gemüselasagne",
		BaseServings: 6,
		IsFavorite:   false,
		Image:        "anders.jpg", // darf nicht übernommen werden
		Ingredients:  []Ingredient{{Name: "Zucchini"}, {Name: "Tomaten"}},
		Steps:        []Step{{Text: "Gemüse schneiden."}},
		Tags:         []Tag{{ID: tagID(t, s, "Vegetarisch")}},
	}
	got, err := s.Update(ctx, created.ID, changed)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	if got.Title != "Gemüselasagne" || got.BaseServings != 6 || got.IsFavorite {
		t.Errorf("Felder nicht übernommen: %+v", got)
	}
	if got.PrepMinutes != nil || got.SourceURL != "" {
		t.Errorf("weggelassene Felder müssen leer sein: prep %v, url %q", got.PrepMinutes, got.SourceURL)
	}
	if got.Image != "lasagne.jpg" {
		t.Errorf("Image = %q, muss erhalten bleiben", got.Image)
	}
	if !got.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("CreatedAt geändert: %v -> %v", created.CreatedAt, got.CreatedAt)
	}
	if got.UpdatedAt.Before(created.UpdatedAt) {
		t.Errorf("UpdatedAt %v liegt vor %v", got.UpdatedAt, created.UpdatedAt)
	}
	if len(got.Ingredients) != 2 || got.Ingredients[0].Name != "Zucchini" {
		t.Errorf("Zutaten = %+v", got.Ingredients)
	}
	if len(got.Steps) != 1 || len(got.Tags) != 1 || got.Tags[0].Name != "Vegetarisch" {
		t.Errorf("Schritte = %+v, Kategorien = %+v", got.Steps, got.Tags)
	}
	// Die alten Bestandteile dürfen nicht übrig bleiben.
	if n := countRows(t, s, "recipe_ingredients", created.ID); n != 2 {
		t.Errorf("%d Zutatenzeilen, erwartet 2", n)
	}
}

func TestUpdateErrors(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	created, err := s.Create(ctx, lasagne(t, s))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	_, err = s.Update(ctx, 42, Recipe{Title: "X", BaseServings: 2})
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("unbekanntes Rezept: Fehler = %v, erwartet ErrNotFound", err)
	}

	// Unbekannte Kategorie fällt erst nach dem Löschen der alten Teile auf.
	// Die Transaktion muss dann alles zurückrollen.
	_, err = s.Update(ctx, created.ID, Recipe{Title: "X", BaseServings: 2, Tags: []Tag{{ID: 9999}}})
	var verr *ValidationError
	if !errors.As(err, &verr) || verr.Field != "tags[0].id" {
		t.Fatalf("Fehler = %v, erwartet ValidationError für tags[0].id", err)
	}
	got, err := s.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Title != "Lasagne" || len(got.Ingredients) != 3 || len(got.Tags) != 2 {
		t.Errorf("Rezept wurde trotz Fehler verändert: %+v", got)
	}
}

func TestDelete(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	created, err := s.Create(ctx, lasagne(t, s))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := s.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get(ctx, created.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get nach Delete: Fehler = %v, erwartet ErrNotFound", err)
	}
	for _, table := range []string{"recipe_ingredients", "recipe_steps", "recipe_tags"} {
		if n := countRows(t, s, table, created.ID); n != 0 {
			t.Errorf("%s: %d Zeilen übrig, erwartet 0", table, n)
		}
	}

	var n int
	err = s.db.QueryRow("SELECT count(*) FROM deletions WHERE table_name = 'recipes' AND record_id = ?",
		created.ID).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("%d Einträge in deletions, erwartet 1", n)
	}

	// Zweites Löschen: Rezept gibt es nicht mehr.
	if err := s.Delete(ctx, created.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("zweites Delete: Fehler = %v, erwartet ErrNotFound", err)
	}
}

func TestSetFavorite(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	created, err := s.Create(ctx, Recipe{Title: "Rührei", BaseServings: 2})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Zweimal true hintereinander: Setzen ist kein Umschalten.
	for _, want := range []bool{true, true, false} {
		if err := s.SetFavorite(ctx, created.ID, want); err != nil {
			t.Fatalf("SetFavorite(%v): %v", want, err)
		}
		got, err := s.Get(ctx, created.ID)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if got.IsFavorite != want {
			t.Errorf("IsFavorite = %v, erwartet %v", got.IsFavorite, want)
		}
	}

	if err := s.SetFavorite(ctx, 42, true); !errors.Is(err, ErrNotFound) {
		t.Errorf("unbekanntes Rezept: Fehler = %v, erwartet ErrNotFound", err)
	}
}
