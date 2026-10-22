package fetch

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write([]byte("<html>hallo</html>"))
		case "/big":
			w.Write([]byte(strings.Repeat("x", 2000)))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	ctx := context.Background()

	// Der Testserver läuft auf 127.0.0.1, daher hier ausnahmsweise erlaubt.
	c := New(5*time.Second, true)

	body, ct, err := c.Get(ctx, srv.URL+"/ok", 1000)
	if err != nil || string(body) != "<html>hallo</html>" || !strings.HasPrefix(ct, "text/html") {
		t.Errorf("ok: %q %q %v", body, ct, err)
	}
	if _, _, err := c.Get(ctx, srv.URL+"/big", 1000); !errors.Is(err, ErrTooLarge) {
		t.Errorf("big: Fehler = %v, erwartet ErrTooLarge", err)
	}
	if _, _, err := c.Get(ctx, srv.URL+"/fehlt", 1000); err == nil || !strings.Contains(err.Error(), "404") {
		t.Errorf("fehlt: Fehler = %v", err)
	}
	for _, bad := range []string{"ftp://example.org/x", "file:///etc/passwd", "keine adresse", "http://"} {
		if _, _, err := c.Get(ctx, bad, 1000); err == nil {
			t.Errorf("%q: kein Fehler", bad)
		}
	}
}

func TestBlocksPrivateAddresses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("geheim"))
	}))
	defer srv.Close()

	c := New(5*time.Second, false)
	_, _, err := c.Get(context.Background(), srv.URL, 1000)
	if !errors.Is(err, ErrForbiddenAddress) {
		t.Errorf("Fehler = %v, erwartet ErrForbiddenAddress", err)
	}
}

func TestIsPublic(t *testing.T) {
	tests := map[string]bool{
		"8.8.8.8":            true,
		"2a00:1450::1":       true,
		"127.0.0.1":          false,
		"::1":                false,
		"192.168.1.10":       false,
		"10.0.0.5":           false,
		"172.16.3.4":         false,
		"169.254.1.1":        false,
		"fe80::1":            false,
		"0.0.0.0":            false,
		"fd00::1234":         false,
		"100.64.0.1":         false, // z. B. Tailscale
		"100.128.0.1":        true,  // knapp außerhalb von 100.64.0.0/10
		"172.30.32.2":        false, // Supervisor (Ingress, API)
		"0.1.2.3":            false,
		"198.18.0.1":         false,
		"192.0.0.8":          false,
		"240.0.0.1":          false,
		"255.255.255.255":    false,
		"::ffff:192.168.1.1": false, // IPv4 in IPv6-Schreibweise
	}
	for addr, want := range tests {
		if got := isPublic(net.ParseIP(addr)); got != want {
			t.Errorf("isPublic(%s) = %v, erwartet %v", addr, got, want)
		}
	}
}

// Weltweit gültige IPv6-Adressen von Geräten im Heimnetz lassen sich nicht
// von Adressen im Internet unterscheiden; deshalb wird nur IPv4 genutzt.
func TestRefusesIPv6(t *testing.T) {
	c := New(2*time.Second, false)
	_, _, err := c.Get(context.Background(), "http://[2001:db8::1]:8123/", 1000)
	if err == nil {
		t.Fatal("IPv6-Adresse wurde abgerufen, erwartet Fehler")
	}
}

// Eine erlaubte Seite leitet auf eine Adresse im Heimnetz weiter: Die
// Prüfung muss auch beim zweiten Verbindungsaufbau greifen.
func TestBlocksRedirectToPrivateAddress(t *testing.T) {
	internal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("geheim"))
	}))
	defer internal.Close()
	redirected := false
	public := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirected = true
		http.Redirect(w, r, internal.URL+"/admin", http.StatusFound)
	}))
	defer public.Close()

	// Beide Testserver laufen auf 127.0.0.1. Als „öffentlich“ gilt hier nur
	// der Port des ersten; der zweite steht für ein Gerät im Heimnetz.
	_, publicPort, _ := net.SplitHostPort(public.Listener.Addr().String())
	c := newClient(5*time.Second, func(_ net.IP, port string) bool { return port == publicPort })

	body, _, err := c.Get(context.Background(), public.URL, 1000)
	if !errors.Is(err, ErrForbiddenAddress) {
		t.Errorf("Fehler = %v (Inhalt %q), erwartet ErrForbiddenAddress", err, body)
	}
	if !redirected {
		t.Error("die erlaubte Seite wurde nicht abgerufen – der Test prüft dann keine Weiterleitung")
	}
}
