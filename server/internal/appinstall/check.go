package appinstall

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/fetch"
)

// CheckResult sagt, ob unter der Adresse für die App der App-Zugang
// antwortet. Message ist ein Satz für die Weboberfläche.
type CheckResult struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

// prober ruft eine Adresse ab, ohne Weiterleitungen zu folgen
// (fetch.Client.Probe; in Tests ein Client, der auch 127.0.0.1 erlaubt).
type prober interface {
	Probe(ctx context.Context, rawURL string, maxBytes int64) (fetch.Response, error)
}

// Check ruft appURL/api/v1/health ohne API-Schlüssel ab. Der App-Zugang
// antwortet darauf mit 401 und seiner eigenen Fehlermeldung – daran ist er
// zu erkennen. Alle anderen Antworten deuten auf eine falsch eingerichtete
// Adresse; die häufigsten Fälle werden benannt.
func Check(ctx context.Context, p prober, appURL string) CheckResult {
	appURL = strings.TrimRight(strings.TrimSpace(appURL), "/")
	if appURL == "" {
		return CheckResult{Message: "Keine Adresse eingetragen."}
	}
	resp, err := p.Probe(ctx, appURL+"/api/v1/health", 64<<10)
	switch {
	case errors.Is(err, fetch.ErrForbiddenAddress):
		return CheckResult{Message: "Die Adresse zeigt ins Heimnetz. Die App braucht die Adresse aus dem Internet (Hostname im Cloudflare-Tunnel)."}
	case err != nil:
		return CheckResult{Message: "Nicht erreichbar – Hostname und Tunnel prüfen."}
	}

	body := bytes.TrimSpace(resp.Body)
	isJSON := strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json")
	fromCloudflare := resp.Header.Get("Cf-Ray") != ""
	location := resp.Header.Get("Location")

	switch {
	case resp.Status == http.StatusUnauthorized && bytes.Contains(body, []byte("api-schlüssel")):
		return CheckResult{OK: true, Message: "Der App-Zugang antwortet. Die App kann sich mit dieser Adresse verbinden."}
	case resp.Status >= 300 && resp.Status < 400 && strings.Contains(location, "cloudflareaccess.com"),
		resp.Status == http.StatusForbidden && fromCloudflare && !isJSON:
		return CheckResult{OK: true, Message: "Cloudflare Access schützt diese Adresse. Beim Verbinden der App unter „Cloudflare Access“ Client ID und Secret angeben."}
	case resp.Status == http.StatusForbidden && isJSON && bytes.Contains(body, []byte("home assistant")):
		return CheckResult{Message: "Die Adresse führt zur Weboberfläche (Port 8099). Im Tunnel muss der Hostname auf Port 8100 zeigen."}
	case resp.Status == http.StatusNotFound && bytes.HasPrefix(body, []byte("404: Not Found")):
		// So antwortet Home Assistant (aiohttp) auf unbekannte Pfade.
		return CheckResult{Message: "Diese Adresse führt zu Home Assistant, nicht zum App-Zugang. Im Tunnel einen eigenen Hostnamen anlegen, der auf Port 8100 des Rezeptplaners zeigt (docs/APP-ZUGANG.md)."}
	case resp.Status == http.StatusNotFound:
		return CheckResult{Message: "Unter dieser Adresse antwortet kein Rezeptplaner. Ist der Hostname in der Tunnel-Konfiguration eingetragen?"}
	case resp.Status == 502 || resp.Status == 503 || resp.Status == 504 || resp.Status == 530:
		return CheckResult{Message: fmt.Sprintf("Der Tunnel erreicht den Rezeptplaner nicht (Fehler %d). Hostname und Port 8100 in der Tunnel-Konfiguration prüfen.", resp.Status)}
	case resp.Status == http.StatusOK:
		return CheckResult{Message: "Die Adresse antwortet ohne Schlüssel – das ist nicht der App-Zugang (Port 8100)."}
	}
	return CheckResult{Message: fmt.Sprintf("Unerwartete Antwort (Status %d) – das ist vermutlich nicht der App-Zugang.", resp.Status)}
}
