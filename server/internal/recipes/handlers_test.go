package recipes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/httpx"
)

// newTestMux liefert einen Router mit den Rezept-Endpunkten auf einer
// frischen Testdatenbank.
func newTestMux(t *testing.T) *http.ServeMux {
	t.Helper()
	mux := http.NewServeMux()
	NewHandler(newTestStore(t)).Register(mux)
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
