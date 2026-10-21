package importer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/database"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/fetch"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/pantry"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/recipes"
)

func newTestMux(t *testing.T) (*http.ServeMux, *recipes.Store) {
	t.Helper()
	ctx := context.Background()
	db, err := database.Open(ctx, ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := database.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	store := recipes.NewStore(db)
	mux := http.NewServeMux()
	// Testserver laufen auf 127.0.0.1, daher Heimnetz-Sperre aus.
	NewHandler(store, pantry.NewStore(db), fetch.New(5*time.Second, true)).Register(mux)
	return mux, store
}

func post(mux *http.ServeMux, path, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("POST", path, strings.NewReader(body)))
	return rec
}

func TestImportURL(t *testing.T) {
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rezepte/123/gulasch.html":
			w.Write([]byte(chefkochLike))
		case "/blog":
			w.Write([]byte("<html>kein Rezept</html>"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer site.Close()
	mux, store := newTestMux(t)
	pageURL := site.URL + "/rezepte/123/gulasch.html"

	rec := post(mux, "/api/v1/import/url", `{"url": "`+pageURL+`?utm_campaign=x"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("Status = %d; %s", rec.Code, rec.Body)
	}
	var draft draftResponse
	json.Unmarshal(rec.Body.Bytes(), &draft)
	if draft.Recipe.Title != "Omas Gulasch" || draft.Recipe.SourceURL != pageURL || len(draft.Duplicates) != 0 {
		t.Errorf("Entwurf = %+v", draft)
	}
	if draft.ImageURL != "https://img.example.org/gulasch.jpg" {
		t.Errorf("Bild = %q", draft.ImageURL)
	}

	// Speichern und erneut importieren: Jetzt gibt es eine Dublette.
	if _, err := store.Create(context.Background(), draft.Recipe); err != nil {
		t.Fatal(err)
	}
	json.Unmarshal(post(mux, "/api/v1/import/url", `{"url": "`+pageURL+`#kommentare"}`).Body.Bytes(), &draft)
	if len(draft.Duplicates) != 1 || draft.Duplicates[0].Title != "Omas Gulasch" {
		t.Errorf("Dubletten = %+v", draft.Duplicates)
	}

	tests := []struct {
		name, body string
		want       int
	}{
		{"kein Rezept auf der Seite", `{"url": "` + site.URL + `/blog"}`, http.StatusUnprocessableEntity},
		{"Seite fehlt", `{"url": "` + site.URL + `/weg"}`, http.StatusBadGateway},
		{"keine Adresse", `{"url": "chefkoch"}`, http.StatusBadRequest},
		{"falsches JSON", `{"adresse": "x"}`, http.StatusBadRequest},
	}
	for _, tt := range tests {
		if rec := post(mux, "/api/v1/import/url", tt.body); rec.Code != tt.want {
			t.Errorf("%s: Status %d, erwartet %d; %s", tt.name, rec.Code, tt.want, rec.Body)
		}
	}
}

func TestImportBlocksHomeNetwork(t *testing.T) {
	ctx := context.Background()
	db, _ := database.Open(ctx, ":memory:")
	defer db.Close()
	database.Migrate(ctx, db)
	mux := http.NewServeMux()
	NewHandler(recipes.NewStore(db), pantry.NewStore(db), fetch.New(5*time.Second, false)).Register(mux)

	rec := post(mux, "/api/v1/import/url", `{"url": "http://192.168.1.1/admin"}`)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "heimnetz") {
		t.Errorf("Status %d; %s", rec.Code, rec.Body)
	}
}

func TestParseIngredientsOverHTTP(t *testing.T) {
	mux, _ := newTestMux(t)
	rec := post(mux, "/api/v1/import/ingredients", `{"lines": ["500 g Mehl", "", "2 Eier", "1 Prise Salz"]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("Status = %d; %s", rec.Code, rec.Body)
	}
	var resp struct{ Ingredients []recipes.Ingredient }
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if len(resp.Ingredients) != 3 || resp.Ingredients[0].Name != "Mehl" || resp.Ingredients[1].Name != "Eier" {
		t.Fatalf("Zutaten = %+v", resp.Ingredients)
	}
	// Salz steht im Startbestand der Vorratsliste.
	if !resp.Ingredients[0].ShoppingRelevant || resp.Ingredients[2].ShoppingRelevant {
		t.Errorf("einkaufsrelevant: Mehl %v, Salz %v", resp.Ingredients[0].ShoppingRelevant, resp.Ingredients[2].ShoppingRelevant)
	}
}
