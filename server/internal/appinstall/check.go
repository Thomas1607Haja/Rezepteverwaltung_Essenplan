package appinstall

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/apierr"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/fetch"
)

// CheckResult sagt, ob unter der Adresse für die App der App-Zugang
// antwortet. Code (mit Params) übersetzt die Weboberfläche; Message ist
// der englische Text dazu.
type CheckResult struct {
	OK      bool           `json:"ok"`
	Code    string         `json:"code"`
	Params  map[string]any `json:"params,omitempty"`
	Message string         `json:"message"`
}

// result baut das Ergebnis zu einem Code aus dem Katalog (apierr).
func result(ok bool, code string, params ...any) CheckResult {
	e := apierr.New(code, params...)
	return CheckResult{OK: ok, Code: code, Params: e.Params, Message: e.Message()}
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
		return result(false, "app_check_no_url")
	}
	resp, err := p.Probe(ctx, appURL+"/api/v1/health", 64<<10)
	switch {
	case errors.Is(err, fetch.ErrForbiddenAddress):
		return result(false, "app_check_home_network")
	case err != nil:
		return result(false, "app_check_unreachable")
	}

	body := bytes.TrimSpace(resp.Body)
	isJSON := strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json")
	fromCloudflare := resp.Header.Get("Cf-Ray") != ""
	location := resp.Header.Get("Location")

	switch {
	case resp.Status == http.StatusUnauthorized && (bytes.Contains(body, []byte(`"code":"api_key_required"`)) ||
		bytes.Contains(body, []byte("api-schlüssel"))) /* Text bis Version 0.19 */ :
		return result(true, "app_check_ok")
	case resp.Status >= 300 && resp.Status < 400 && strings.Contains(location, "cloudflareaccess.com"),
		resp.Status == http.StatusForbidden && fromCloudflare && !isJSON:
		return result(true, "app_check_cloudflare")
	case resp.Status == http.StatusForbidden && isJSON && (bytes.Contains(body, []byte(`"code":"home_assistant_only"`)) ||
		bytes.Contains(body, []byte("home assistant"))) /* Text bis Version 0.19 */ :
		return result(false, "app_check_web_port")
	case resp.Status == http.StatusNotFound && bytes.HasPrefix(body, []byte("404: Not Found")):
		// So antwortet Home Assistant (aiohttp) auf unbekannte Pfade.
		return result(false, "app_check_home_assistant")
	case resp.Status == http.StatusNotFound:
		return result(false, "app_check_not_found")
	case resp.Status == 502 || resp.Status == 503 || resp.Status == 504 || resp.Status == 530:
		return result(false, "app_check_tunnel", "status", resp.Status)
	case resp.Status == http.StatusOK:
		return result(false, "app_check_no_key")
	}
	return result(false, "app_check_unexpected", "status", resp.Status)
}
