package httpx

import (
	"crypto/subtle"
	"log/slog"
	"net/http"
	"net/netip"
	"strings"
)

// AllowOnlyFrom ist eine Middleware: Sie lässt nur Anfragen durch, deren
// Verbindung von der Adresse allowed kommt, und lehnt alle anderen mit 403
// ab.
//
// In Home Assistant ist das der Ingress-Vermittler des Supervisors
// (172.30.32.2). Er prüft die Anmeldung, bevor er eine Anfrage weiterreicht.
// Nimmt die App nur Verbindungen von ihm an, kann niemand im Heimnetz die
// Anmeldung umgehen, indem er die App direkt anspricht.
//
// Ausnahme: Ist token gesetzt, dürfen lesende Anfragen (GET) mit der
// Kopfzeile "Authorization: Bearer <token>" auch von anderswo kommen – aber
// nur auf die Pfade, die die Home-Assistant-Integration braucht (siehe
// integrationPath). So holt sie sich den Essensplan über das interne Netz
// des Supervisors.
//
// Kopfzeilen wie X-Ingress-Path spielen keine Rolle: Entscheidend ist
// allein die Gegenstelle der Verbindung.
func AllowOnlyFrom(allowed netip.Addr, token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// RemoteAddr ist die Gegenstelle der TCP-Verbindung, z. B.
		// "172.30.32.2:41234". Anders als Kopfzeilen wie X-Forwarded-For
		// kann der Absender sie nicht frei wählen.
		remote, err := netip.ParseAddrPort(r.RemoteAddr)
		// Unmap macht aus der IPv6-Schreibweise einer IPv4-Adresse
		// ("::ffff:172.30.32.2") wieder die IPv4-Adresse.
		if err == nil && remote.Addr().Unmap() == allowed {
			next.ServeHTTP(w, r)
			return
		}
		if token != "" && r.Method == http.MethodGet && integrationPath(r.URL.Path) && validBearer(r, token) {
			next.ServeHTTP(w, r)
			return
		}
		slog.Warn("Anfrage abgelehnt: nicht über Ingress", "remote", r.RemoteAddr, "path", r.URL.Path)
		WriteError(w, http.StatusForbidden, "zugriff nur über home assistant")
	})
}

// validBearer prüft die Kopfzeile "Authorization: Bearer <token>".
// subtle.ConstantTimeCompare braucht für jeden Vergleich gleich lange, egal
// wie viele Zeichen stimmen – sonst ließe sich der Schlüssel über die
// Antwortzeit Zeichen für Zeichen erraten.
func validBearer(r *http.Request, token string) bool {
	got, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	return ok && subtle.ConstantTimeCompare([]byte(got), []byte(token)) == 1
}

// integrationPath meldet, ob die Home-Assistant-Integration path lesen
// darf: Essensplan, einzelne Rezepte und Bilder – mehr braucht sie nicht
// (siehe custom_components/rezeptplaner/api.py).
func integrationPath(path string) bool {
	return path == "/api/v1/meal-plan" ||
		strings.HasPrefix(path, "/api/v1/recipes/") ||
		strings.HasPrefix(path, "/api/v1/images/")
}
