// Package fetch lädt Seiten und Bilder aus dem Internet, z. B. für den
// Rezeptimport. Weil die Adresse vom Nutzer kommt, gelten feste Grenzen:
// nur http/https, Zeitlimit, Größenlimit und keine Adressen im Heimnetz.
package fetch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"syscall"
	"time"
)

// userAgent stellt den Abruf als normalen Browser-Zugriff mit Zusatz vor;
// manche Seiten liefern unbekannten Programmen sonst nichts aus.
const userAgent = "Mozilla/5.0 (compatible; Rezeptplaner/1.0; +https://github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan)"

// ErrForbiddenAddress: Die Adresse zeigt ins Heimnetz oder auf den Rechner
// selbst.
var ErrForbiddenAddress = errors.New("adressen im heimnetz sind nicht erlaubt")

// ErrTooLarge: Die Antwort ist größer als erlaubt.
var ErrTooLarge = errors.New("antwort ist zu groß")

// Client lädt Dateien mit den genannten Grenzen.
type Client struct {
	http *http.Client
}

// New erzeugt einen Client. allowPrivate erlaubt Adressen im Heimnetz und
// auf dem eigenen Rechner – nur für Tests mit einem lokalen Testserver.
func New(timeout time.Duration, allowPrivate bool) *Client {
	if allowPrivate {
		return newClient(timeout, nil)
	}
	return newClient(timeout, func(ip net.IP, _ string) bool { return isPublic(ip) })
}

// newClient erzeugt einen Client, der nur Verbindungen aufbaut, die allowed
// erlaubt (nil = alle). allowed bekommt die IP-Adresse und den Port.
func newClient(timeout time.Duration, allowed func(ip net.IP, port string) bool) *Client {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	if allowed != nil {
		// Control läuft für jede Verbindung, nachdem der Name zur IP-Adresse
		// aufgelöst wurde – auch nach Weiterleitungen. So hilft auch kein
		// Trick mit einem Namen, der auf 192.168.x.x zeigt.
		dialer.Control = func(network, address string, _ syscall.RawConn) error {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}
			if ip := net.ParseIP(host); ip == nil || !allowed(ip, port) {
				return ErrForbiddenAddress
			}
			return nil
		}
	}
	transport := &http.Transport{
		// Nur IPv4: Geräte im Heimnetz haben oft zusätzlich eine weltweit
		// gültige IPv6-Adresse (z. B. Home Assistant), die sich nicht von
		// einer Adresse im Internet unterscheiden lässt. Im Heimnetz sind
		// IPv4-Adressen dagegen immer privat und damit gesperrt. Jede
		// Rezeptseite ist per IPv4 erreichbar.
		DialContext: func(ctx context.Context, _, address string) (net.Conn, error) {
			return dialer.DialContext(ctx, "tcp4", address)
		},
		TLSHandshakeTimeout: 10 * time.Second,
		// Keine offenen Verbindungen aufheben: Importe sind selten, und im
		// Leerlauf soll nichts Ressourcen belegen.
		DisableKeepAlives: true,
	}
	return &Client{http: &http.Client{Timeout: timeout, Transport: transport}}
}

// reserved sind IPv4-Bereiche, die Go nicht zu den privaten zählt, die
// aber nie zu einer Rezeptseite im Internet gehören.
var reserved = parseCIDRs(
	"0.0.0.0/8",       // „dieses Netz“; 0.x.x.x erreicht unter Linux teils den Rechner selbst
	"100.64.0.0/10",   // Carrier-NAT, z. B. Tailscale für Geräte im eigenen Netz
	"192.0.0.0/24",    // Protokollzuweisungen der IETF
	"192.0.2.0/24",    // Dokumentation
	"198.18.0.0/15",   // Lasttests
	"198.51.100.0/24", // Dokumentation
	"203.0.113.0/24",  // Dokumentation
	"240.0.0.0/4",     // reserviert, inkl. 255.255.255.255
)

func parseCIDRs(cidrs ...string) []*net.IPNet {
	nets := make([]*net.IPNet, len(cidrs))
	for i, c := range cidrs {
		_, n, err := net.ParseCIDR(c)
		if err != nil {
			panic(err)
		}
		nets[i] = n
	}
	return nets
}

// isPublic ist false für den eigenen Rechner, private Netze (192.168.x.x,
// 10.x.x.x, …, auch das interne Netz des Supervisors 172.30.32.0/23),
// link-lokale und reservierte Adressen.
func isPublic(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast() ||
		ip.IsInterfaceLocalMulticast() {
		return false
	}
	for _, n := range reserved {
		if n.Contains(ip) {
			return false
		}
	}
	return true
}

// Get lädt rawURL und liefert den Inhalt (höchstens maxBytes) samt
// Content-Type. Fehler sind deutsch formuliert und für Nutzer gedacht.
func (c *Client) Get(ctx context.Context, rawURL string, maxBytes int64) (body []byte, contentType string, err error) {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, "", errors.New("ungültige adresse (erwartet http:// oder https://)")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept-Language", "de-DE,de;q=0.9")

	resp, err := c.http.Do(req)
	if err != nil {
		if errors.Is(err, ErrForbiddenAddress) {
			return nil, "", ErrForbiddenAddress
		}
		return nil, "", fmt.Errorf("seite nicht erreichbar: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("seite antwortet mit status %d", resp.StatusCode)
	}

	// Ein Byte mehr lesen als erlaubt: Kommt es an, war die Antwort zu groß.
	body, err = io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, "", fmt.Errorf("seite lesen: %w", err)
	}
	if int64(len(body)) > maxBytes {
		return nil, "", ErrTooLarge
	}
	return body, resp.Header.Get("Content-Type"), nil
}

// Response ist eine Antwort von Probe, unabhängig vom Status.
type Response struct {
	Status int
	Header http.Header
	Body   []byte // höchstens maxBytes
}

// Probe ruft rawURL ab wie Get, liefert aber jede Antwort (auch 4xx/5xx)
// und folgt keiner Weiterleitung – etwa um zu prüfen, was unter einer
// Adresse antwortet. Die Sperre für Adressen im Heimnetz gilt genauso.
func (c *Client) Probe(ctx context.Context, rawURL string, maxBytes int64) (Response, error) {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return Response{}, errors.New("ungültige adresse (erwartet http:// oder https://)")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return Response{}, err
	}
	req.Header.Set("User-Agent", userAgent)

	// Kopie des Clients mit gleichem Transport (also gleicher Sperre),
	// aber ohne Weiterleitungen.
	client := *c.http
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := client.Do(req)
	if err != nil {
		if errors.Is(err, ErrForbiddenAddress) {
			return Response{}, ErrForbiddenAddress
		}
		return Response{}, fmt.Errorf("nicht erreichbar: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes))
	if err != nil {
		return Response{}, fmt.Errorf("antwort lesen: %w", err)
	}
	return Response{Status: resp.StatusCode, Header: resp.Header, Body: body}, nil
}
