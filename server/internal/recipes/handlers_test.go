package recipes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/fetch"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/httpx"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/images"
)

// newTestMux liefert einen Router mit den Rezept-Endpunkten auf einer
// frischen Testdatenbank.
func newTestMux(t *testing.T) *http.ServeMux {
	t.Helper()
	mux := http.NewServeMux()
	imgs, err := images.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	// Testserver laufen auf 127.0.0.1, daher Heimnetz-Sperre aus.
	NewHandler(newTestStore(t), imgs, fetch.New(5*time.Second, true)).Register(mux)
	return mux
}

// do schickt eine Anfrage an mux, ohne einen echten Server zu starten.
// httptest.NewRecorder fängt die Antwort auf.
func do(mux *http.ServeMux, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestCreateThenGetOverHTTP(t *testing.T) {
	mux := newTestMux(t)

	rec := do(mux, "POST", "/api/v1/recipes", `{
		"title": "Pfannkuchen",
		"base_servings": 2,
		"ingredients": [{"quantity": 250, "unit": "g", "name": "Mehl", "shopping_relevant": true}],
		"steps": [{"text": "Alles verrühren."}]
	}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST Status = %d, erwartet 201; Antwort: %s", rec.Code, rec.Body)
	}
	var created Recipe
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("Antwort ist kein Rezept: %v", err)
	}
	if want := "/api/v1/recipes/1"; rec.Header().Get("Location") != want {
		t.Errorf("Location = %q, erwartet %q", rec.Header().Get("Location"), want)
	}

	rec = do(mux, "GET", "/api/v1/recipes/1", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET Status = %d, erwartet 200", rec.Code)
	}
	var got Recipe
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("Antwort ist kein Rezept: %v", err)
	}
	if got.Title != "Pfannkuchen" || len(got.Ingredients) != 1 || len(got.Steps) != 1 {
		t.Errorf("Rezept = %+v", got)
	}
}

func TestHTTPErrors(t *testing.T) {
	mux := newTestMux(t)

	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
		wantField  string
	}{
		{"kaputtes JSON", "POST", "/api/v1/recipes", `{"title": `, 400, ""},
		{"unbekanntes Feld", "POST", "/api/v1/recipes", `{"titel": "X", "base_servings": 2}`, 400, ""},
		{"falscher Typ", "POST", "/api/v1/recipes", `{"title": "X", "base_servings": "zwei"}`, 400, ""},
		{"zwei Objekte", "POST", "/api/v1/recipes", `{"title": "X", "base_servings": 2} {}`, 400, ""},
		{"Prüfregel verletzt", "POST", "/api/v1/recipes", `{"title": "", "base_servings": 2}`, 400, "title"},
		{"ID keine Zahl", "GET", "/api/v1/recipes/abc", "", 400, ""},
		{"ID negativ", "GET", "/api/v1/recipes/-1", "", 400, ""},
		{"Rezept fehlt", "GET", "/api/v1/recipes/42", "", 404, ""},
		{"falsche Methode", "PATCH", "/api/v1/recipes/1", "", 405, ""},
		{"Ändern: Rezept fehlt", "PUT", "/api/v1/recipes/42", `{"title": "X", "base_servings": 2}`, 404, ""},
		{"Ändern: Prüfregel verletzt", "PUT", "/api/v1/recipes/42", `{"title": "X", "base_servings": 0}`, 400, "base_servings"},
		{"Löschen: Rezept fehlt", "DELETE", "/api/v1/recipes/42", "", 404, ""},
		{"Favorit: Feld fehlt", "PUT", "/api/v1/recipes/42/favorite", `{}`, 400, "is_favorite"},
		{"Favorit: Rezept fehlt", "PUT", "/api/v1/recipes/42/favorite", `{"is_favorite": true}`, 404, ""},
		{"Liste: Kategorie keine Zahl", "GET", "/api/v1/recipes?tag=pasta", "", 400, "tag"},
		{"Liste: Kategorie unbekannt", "GET", "/api/v1/recipes?tag=9999", "", 400, "tag"},
		{"Liste: Favorit kein Wahrheitswert", "GET", "/api/v1/recipes?favorite=vielleicht", "", 400, "favorite"},
		{"Liste: Sortierung unbekannt", "GET", "/api/v1/recipes?sort=rating", "", 400, "sort"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(mux, tt.method, tt.path, tt.body)
			if rec.Code != tt.wantStatus {
				t.Fatalf("Status = %d, erwartet %d; Antwort: %s", rec.Code, tt.wantStatus, rec.Body)
			}
			if tt.wantStatus == 405 {
				return // Antwort kommt vom Router selbst, nicht als JSON
			}
			var resp httpx.ErrorResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil || resp.Error == "" {
				t.Fatalf("keine JSON-Fehlerantwort: %s", rec.Body)
			}
			if resp.Field != tt.wantField {
				t.Errorf("field = %q, erwartet %q", resp.Field, tt.wantField)
			}
		})
	}
}

func TestUpdateFavoriteDeleteOverHTTP(t *testing.T) {
	mux := newTestMux(t)
	if rec := do(mux, "POST", "/api/v1/recipes", `{"title": "Pfannkuchen", "base_servings": 2}`); rec.Code != http.StatusCreated {
		t.Fatalf("POST Status = %d; Antwort: %s", rec.Code, rec.Body)
	}

	rec := do(mux, "PUT", "/api/v1/recipes/1", `{"title": "Crêpes", "base_servings": 4}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT Status = %d, erwartet 200; Antwort: %s", rec.Code, rec.Body)
	}
	var updated Recipe
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("Antwort ist kein Rezept: %v", err)
	}
	if updated.Title != "Crêpes" || updated.BaseServings != 4 {
		t.Errorf("Rezept = %+v", updated)
	}

	rec = do(mux, "PUT", "/api/v1/recipes/1/favorite", `{"is_favorite": true}`)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("Favorit Status = %d, erwartet 204; Antwort: %s", rec.Code, rec.Body)
	}
	var got Recipe
	json.Unmarshal(do(mux, "GET", "/api/v1/recipes/1", "").Body.Bytes(), &got)
	if !got.IsFavorite {
		t.Error("IsFavorite = false nach PUT .../favorite")
	}

	if rec := do(mux, "DELETE", "/api/v1/recipes/1", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("DELETE Status = %d, erwartet 204; Antwort: %s", rec.Code, rec.Body)
	}
	if rec := do(mux, "GET", "/api/v1/recipes/1", ""); rec.Code != http.StatusNotFound {
		t.Errorf("GET nach DELETE Status = %d, erwartet 404", rec.Code)
	}
}

func TestListOverHTTP(t *testing.T) {
	mux := newTestMux(t)
	for _, body := range []string{
		`{"title": "Pfannkuchen", "base_servings": 2, "is_favorite": true}`,
		`{"title": "Grünkohl", "base_servings": 4}`,
	} {
		if rec := do(mux, "POST", "/api/v1/recipes", body); rec.Code != http.StatusCreated {
			t.Fatalf("POST Status = %d; Antwort: %s", rec.Code, rec.Body)
		}
	}

	tests := []struct {
		name string
		path string
		want []string
	}{
		{"alle", "/api/v1/recipes", []string{"Grünkohl", "Pfannkuchen"}},
		// %C3%BC ist das ü in der Adresse (URL-kodiert).
		{"Suche", "/api/v1/recipes?q=gr%C3%BCnkohl", []string{"Grünkohl"}},
		{"Favoriten", "/api/v1/recipes?favorite=true", []string{"Pfannkuchen"}},
		{"neueste", "/api/v1/recipes?sort=newest", []string{"Grünkohl", "Pfannkuchen"}},
		{"nichts gefunden", "/api/v1/recipes?q=pizza", []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(mux, "GET", tt.path, "")
			if rec.Code != http.StatusOK {
				t.Fatalf("Status = %d, erwartet 200; Antwort: %s", rec.Code, rec.Body)
			}
			var resp listResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
				t.Fatalf("Antwort ist keine Liste: %v", err)
			}
			if resp.Recipes == nil {
				t.Fatalf("recipes fehlt oder ist null: %s", rec.Body)
			}
			if got := titles(resp.Recipes); !slices.Equal(got, tt.want) {
				t.Errorf("Titel = %q, erwartet %q", got, tt.want)
			}
		})
	}
}
