package httpx

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
)

func TestAllowOnlyFrom(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	handler := AllowOnlyFrom(netip.MustParseAddr("172.30.32.2"), "", ok)

	tests := []struct {
		remote string
		want   int
	}{
		{"172.30.32.2:41234", http.StatusNoContent},
		{"[::ffff:172.30.32.2]:41234", http.StatusNoContent}, // IPv4 in IPv6-Schreibweise
		{"172.30.32.1:41234", http.StatusForbidden},          // anderes Gerät im Docker-Netz
		{"192.168.1.20:5000", http.StatusForbidden},          // Rechner im Heimnetz
		{"[fe80::1]:5000", http.StatusForbidden},
		{"kaputt", http.StatusForbidden},
	}
	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/recipes", nil)
		req.RemoteAddr = tt.remote
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != tt.want {
			t.Errorf("RemoteAddr %q: Status %d, erwartet %d", tt.remote, rec.Code, tt.want)
		}
	}
}

func TestAllowOnlyFromWithToken(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	handler := AllowOnlyFrom(netip.MustParseAddr("172.30.32.2"), "geheim", ok)

	tests := []struct {
		method, auth string
		want         int
	}{
		{"GET", "Bearer geheim", http.StatusNoContent},
		{"GET", "Bearer falsch", http.StatusForbidden},
		{"GET", "geheim", http.StatusForbidden},
		{"GET", "", http.StatusForbidden},
		{"POST", "Bearer geheim", http.StatusForbidden}, // nur lesen
		{"DELETE", "Bearer geheim", http.StatusForbidden},
	}
	for _, tt := range tests {
		req := httptest.NewRequest(tt.method, "/api/v1/meal-plan", nil)
		req.RemoteAddr = "172.30.32.1:5000" // Home Assistant Core im internen Netz
		if tt.auth != "" {
			req.Header.Set("Authorization", tt.auth)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != tt.want {
			t.Errorf("%s mit %q: Status %d, erwartet %d", tt.method, tt.auth, rec.Code, tt.want)
		}
	}
}

func TestIntegrationTokenOnlyForItsPaths(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	handler := AllowOnlyFrom(netip.MustParseAddr("172.30.32.2"), "geheim", ok)

	tests := []struct {
		path string
		want int
	}{
		{"/api/v1/meal-plan", http.StatusNoContent},
		{"/api/v1/recipes/12", http.StatusNoContent},
		{"/api/v1/images/abc.jpg", http.StatusNoContent},
		{"/api/v1/recipes", http.StatusForbidden}, // Liste braucht die Integration nicht
		{"/api/v1/settings", http.StatusForbidden},
		{"/api/v1/system", http.StatusForbidden},
		{"/api/v1/homeassistant/status", http.StatusForbidden},
		{"/api/v1/api-keys", http.StatusForbidden},
		{"/", http.StatusForbidden},
	}
	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodGet, tt.path, nil)
		req.RemoteAddr = "172.30.32.1:5000"
		req.Header.Set("Authorization", "Bearer geheim")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != tt.want {
			t.Errorf("%s: Status %d, erwartet %d", tt.path, rec.Code, tt.want)
		}
	}
}

func TestIngressHeaderDoesNotBypass(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	handler := AllowOnlyFrom(netip.MustParseAddr("172.30.32.2"), "geheim", ok)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/recipes", nil)
	req.RemoteAddr = "172.30.33.5:5000" // anderes Add-on im internen Netz
	req.Header.Set("X-Ingress-Path", "/api/hassio_ingress/abc")
	req.Header.Set("X-Forwarded-For", "172.30.32.2")
	req.Header.Set("X-Hass-Source", "core.ingress")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("Status %d, erwartet 403", rec.Code)
	}
}

func TestSecurityHeaders(t *testing.T) {
	h := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" || rec.Header().Get("Referrer-Policy") != "no-referrer" {
		t.Errorf("Kopfzeilen: %v", rec.Header())
	}
}
