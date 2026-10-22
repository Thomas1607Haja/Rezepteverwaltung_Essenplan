package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/apikeys"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/database"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/fetch"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/homeassistant"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/images"
)

// recordingMux schreibt jedes angemeldete Muster mit und reicht es an den
// echten Router weiter. So kennt der Test alle Endpunkte, auch künftige.
type recordingMux struct {
	*http.ServeMux
	patterns []string
}

func (m *recordingMux) Handle(pattern string, h http.Handler) {
	m.patterns = append(m.patterns, pattern)
	m.ServeMux.Handle(pattern, h)
}

func (m *recordingMux) HandleFunc(pattern string, h func(http.ResponseWriter, *http.Request)) {
	m.patterns = append(m.patterns, pattern)
	m.ServeMux.HandleFunc(pattern, h)
}

// newAppHandler baut den App-Zugang wie in run() und liefert alle Muster.
func newAppHandler(t *testing.T) (http.Handler, *apikeys.Store, []string) {
	t.Helper()
	ctx := context.Background()
	dir := t.TempDir()
	db, err := database.Open(ctx, filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := database.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	imgs, err := images.NewStore(filepath.Join(dir, "images"))
	if err != nil {
		t.Fatal(err)
	}
	keys := apikeys.NewStore(db)
	mux := &recordingMux{ServeMux: http.NewServeMux()}
	registerRoutes(mux, services{
		db:      db,
		images:  imgs,
		fetcher: fetch.New(time.Second, false),
		ha:      homeassistant.New("", "", ""),
		keys:    keys,
		version: "test",
		dataDir: dir,
	})
	return apikeys.Require(keys, mux), keys, mux.patterns
}

// requestFor macht aus einem Muster wie "PUT /api/v1/recipes/{id}" eine
// Anfrage mit eingesetzten Platzhaltern.
func requestFor(pattern string) *http.Request {
	method, path, _ := strings.Cut(pattern, " ")
	path = strings.NewReplacer("{id}", "1", "{name}", "0123456789abcdef0123456789abcdef.jpg").Replace(path)
	req := httptest.NewRequest(method, path, strings.NewReader("{}"))
	req.RemoteAddr = "172.30.32.2:41234" // auch die Ingress-Adresse hilft hier nicht
	return req
}

func TestAppAccessRequiresKeyOnEveryRoute(t *testing.T) {
	handler, _, patterns := newAppHandler(t)
	if len(patterns) < 30 {
		t.Fatalf("nur %d Muster angemeldet – fehlt etwas?", len(patterns))
	}

	cases := []struct {
		name   string
		header map[string]string
	}{
		{"ohne Schlüssel", nil},
		{"falscher Schlüssel", map[string]string{"Authorization": "Bearer rp_falsch"}},
		{"ohne Bearer", map[string]string{"Authorization": "rp_irgendwas"}},
		{"nur Ingress-Kopfzeilen", map[string]string{
			"X-Ingress-Path":  "/api/hassio_ingress/abc",
			"X-Forwarded-For": "172.30.32.2",
			"X-Hass-Source":   "core.ingress",
		}},
	}
	for _, c := range cases {
		for _, pattern := range patterns {
			req := requestFor(pattern)
			for k, v := range c.header {
				req.Header.Set(k, v)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("%s, %s: Status %d, erwartet 401", c.name, pattern, rec.Code)
			}
		}
	}
}

func TestAppAccessWithValidKey(t *testing.T) {
	handler, keys, _ := newAppHandler(t)
	_, secret, err := keys.Create(context.Background(), "Testhandy")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		pattern string
		want    int
	}{
		{"GET /api/v1/health", http.StatusOK},
		{"GET /api/v1/recipes", http.StatusOK},
		{"GET /api/v1/api-keys", http.StatusForbidden}, // Verwaltung nur über Ingress
		{"POST /api/v1/api-keys", http.StatusForbidden},
		{"DELETE /api/v1/api-keys/{id}", http.StatusForbidden},
		{"GET /", http.StatusNotFound}, // keine Weboberfläche
		{"GET /index.html", http.StatusNotFound},
	}
	for _, tt := range tests {
		req := requestFor(tt.pattern)
		req.Header.Set("Authorization", "Bearer "+secret)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != tt.want {
			t.Errorf("%s: Status %d, erwartet %d", tt.pattern, rec.Code, tt.want)
		}
	}

	// Gesperrter Schlüssel gilt sofort nicht mehr.
	list, err := keys.List(context.Background())
	if err != nil || len(list) != 1 {
		t.Fatalf("List: %v, %d Schlüssel", err, len(list))
	}
	if err := keys.Delete(context.Background(), list[0].ID); err != nil {
		t.Fatal(err)
	}
	req := requestFor("GET /api/v1/health")
	req.Header.Set("Authorization", "Bearer "+secret)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("gesperrter Schlüssel: Status %d, erwartet 401", rec.Code)
	}
}
