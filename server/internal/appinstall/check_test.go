package appinstall

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/fetch"
)

func TestCheck(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		ok      bool
		want    string
	}{
		{"App-Zugang", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(401)
			w.Write([]byte(`{"error":"gültiger api-schlüssel erforderlich"}`))
		}, true, "antwortet"},
		{"Home Assistant", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(404)
			w.Write([]byte("404: Not Found"))
		}, false, "Home Assistant"},
		{"Ingress-Port", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(403)
			w.Write([]byte(`{"error":"zugriff nur über home assistant"}`))
		}, false, "8099"},
		{"Cloudflare Access", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Location", "https://team.cloudflareaccess.com/cdn-cgi/access/login")
			w.WriteHeader(302)
		}, true, "Cloudflare Access"},
		{"Tunnel ohne Regel", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(404)
		}, false, "Tunnel-Konfiguration"},
		{"Bad Gateway", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(502)
		}, false, "8100"},
	}
	client := fetch.New(5*time.Second, true) // Testserver laufen auf 127.0.0.1
	for _, tt := range tests {
		srv := httptest.NewServer(tt.handler)
		got := Check(context.Background(), client, srv.URL+"/")
		srv.Close()
		if got.OK != tt.ok || !strings.Contains(got.Message, tt.want) {
			t.Errorf("%s: %+v", tt.name, got)
		}
	}

	// Ohne Freigabe für das Heimnetz (wie auf dem Green) wird 127.0.0.1 abgelehnt.
	got := Check(context.Background(), fetch.New(5*time.Second, false), "http://127.0.0.1:1")
	if got.OK || !strings.Contains(got.Message, "Heimnetz") {
		t.Errorf("Heimnetz: %+v", got)
	}
}
