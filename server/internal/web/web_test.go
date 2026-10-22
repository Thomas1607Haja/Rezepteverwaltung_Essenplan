package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

// testApp ist eine gebaute Oberfläche im Kleinformat. fstest.MapFS ist ein
// Dateisystem im Arbeitsspeicher, nur für Tests.
var testApp = fstest.MapFS{
	"index.html":           {Data: []byte("<!doctype html><title>Rezeptplaner</title>")},
	"assets/index-abc1.js": {Data: []byte("console.log('hallo');")},
}

func get(h http.Handler, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
	return rec
}

func TestServesApp(t *testing.T) {
	h := newHandler(testApp)

	tests := []struct {
		name        string
		path        string
		wantStatus  int
		wantType    string // Anfang des Content-Type
		wantCache   string
		wantContain string
	}{
		{"Startseite", "/", 200, "text/html", "no-cache", "<title>Rezeptplaner</title>"},
		{"Asset", "/assets/index-abc1.js", 200, "text/javascript", "public, max-age=31536000, immutable", "hallo"},
		{"fehlende Datei", "/gibtsnicht.js", 404, "", "", ""},
		{"keine Ordnerliste", "/assets/", 404, "", "", ""},
		{"unbekannte API", "/api/v1/gibtsnicht", 404, "application/json", "", `"error"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := get(h, tt.path)
			if rec.Code != tt.wantStatus {
				t.Fatalf("Status = %d, erwartet %d", rec.Code, tt.wantStatus)
			}
			if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, tt.wantType) {
				t.Errorf("Content-Type = %q, erwartet Anfang %q", got, tt.wantType)
			}
			if tt.wantCache != "" && rec.Header().Get("Cache-Control") != tt.wantCache {
				t.Errorf("Cache-Control = %q, erwartet %q", rec.Header().Get("Cache-Control"), tt.wantCache)
			}
			if !strings.Contains(rec.Body.String(), tt.wantContain) {
				t.Errorf("Antwort enthält %q nicht: %s", tt.wantContain, rec.Body)
			}
		})
	}
}

func TestMissingBuild(t *testing.T) {
	h := newHandler(nil)

	rec := get(h, "/")
	if rec.Code != http.StatusServiceUnavailable || !strings.Contains(rec.Body.String(), "npm run build") {
		t.Errorf("Status = %d, Antwort: %s", rec.Code, rec.Body)
	}
	// Die API-Antwort für unbekannte Pfade bleibt JSON.
	if rec := get(h, "/api/v1/x"); rec.Code != http.StatusNotFound {
		t.Errorf("API-Status = %d, erwartet 404", rec.Code)
	}
}

func TestContentSecurityPolicy(t *testing.T) {
	rec := get(newHandler(testApp), "/")
	csp := rec.Header().Get("Content-Security-Policy")
	for _, want := range []string{"script-src 'self'", "object-src 'none'", "frame-ancestors 'self'"} {
		if !strings.Contains(csp, want) {
			t.Errorf("CSP %q enthält nicht %q", csp, want)
		}
	}
}
