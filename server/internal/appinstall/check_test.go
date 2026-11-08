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
		}, true, "app_check_ok"},
		{"App-Zugang ab 0.20", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(401)
			w.Write([]byte(`{"error":"A valid API key is required","code":"api_key_required"}`))
		}, true, "app_check_ok"},
		{"Ingress-Port ab 0.20", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(403)
			w.Write([]byte(`{"error":"Access only via Home Assistant","code":"home_assistant_only"}`))
		}, false, "app_check_web_port"},
		{"Home Assistant", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(404)
			w.Write([]byte("404: Not Found"))
		}, false, "app_check_home_assistant"},
		{"Ingress-Port", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(403)
			w.Write([]byte(`{"error":"zugriff nur über home assistant"}`))
		}, false, "app_check_web_port"},
		{"Cloudflare Access", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Location", "https://team.cloudflareaccess.com/cdn-cgi/access/login")
			w.WriteHeader(302)
		}, true, "app_check_cloudflare"},
		{"Tunnel ohne Regel", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(404)
		}, false, "app_check_not_found"},
		{"Bad Gateway", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(502)
		}, false, "app_check_tunnel"},
	}
	client := fetch.New(5*time.Second, true) // Testserver laufen auf 127.0.0.1
	for _, tt := range tests {
		srv := httptest.NewServer(tt.handler)
		got := Check(context.Background(), client, srv.URL+"/")
		srv.Close()
		if got.OK != tt.ok || !strings.Contains(got.Code, tt.want) {
			t.Errorf("%s: %+v", tt.name, got)
		}
	}

	// Ohne Freigabe für das Heimnetz (wie auf dem Green) wird 127.0.0.1 abgelehnt.
	got := Check(context.Background(), fetch.New(5*time.Second, false), "http://127.0.0.1:1")
	if got.OK || got.Code != "app_check_home_network" {
		t.Errorf("Heimnetz: %+v", got)
	}
}
