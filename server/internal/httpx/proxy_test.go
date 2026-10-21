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
	handler := AllowOnlyFrom(netip.MustParseAddr("172.30.32.2"), ok)

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
