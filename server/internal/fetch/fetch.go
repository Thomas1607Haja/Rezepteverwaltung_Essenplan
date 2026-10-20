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
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	if !allowPrivate {
		// Control läuft für jede Verbindung, nachdem der Name zur IP-Adresse
		// aufgelöst wurde – auch nach Weiterleitungen. So hilft auch kein
		// Trick mit einem Namen, der auf 192.168.x.x zeigt.
		dialer.Control = func(network, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}
			if ip := net.ParseIP(host); ip == nil || !isPublic(ip) {
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

// sharedAddressSpace ist 100.64.0.0/10. Go zählt den Bereich nicht zu den
// privaten, er wird aber z. B. von Tailscale für Geräte im eigenen Netz
// genutzt.
var sharedAddressSpace = &net.IPNet{IP: net.IPv4(100, 64, 0, 0), Mask: net.CIDRMask(10, 32)}

// isPublic ist false für den eigenen Rechner, private Netze (192.168.x.x,
// 10.x.x.x, …), link-lokale Adressen und Ähnliches.
func isPublic(ip net.IP) bool {
	return !(ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast() ||
		ip.IsInterfaceLocalMulticast() || sharedAddressSpace.Contains(ip))
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
