package httpx

import (
	"log/slog"
	"net/http"
	"net/netip"
)

// AllowOnlyFrom ist eine Middleware: Sie lässt nur Anfragen durch, deren
// Verbindung von der Adresse allowed kommt, und lehnt alle anderen mit 403
// ab.
//
// In Home Assistant ist das der Ingress-Vermittler des Supervisors
// (172.30.32.2). Er prüft die Anmeldung, bevor er eine Anfrage weiterreicht.
// Nimmt die App nur Verbindungen von ihm an, kann niemand im Heimnetz die
// Anmeldung umgehen, indem er die App direkt anspricht.
func AllowOnlyFrom(allowed netip.Addr, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// RemoteAddr ist die Gegenstelle der TCP-Verbindung, z. B.
		// "172.30.32.2:41234". Anders als Kopfzeilen wie X-Forwarded-For
		// kann der Absender sie nicht frei wählen.
		remote, err := netip.ParseAddrPort(r.RemoteAddr)
		// Unmap macht aus der IPv6-Schreibweise einer IPv4-Adresse
		// ("::ffff:172.30.32.2") wieder die IPv4-Adresse.
		if err != nil || remote.Addr().Unmap() != allowed {
			slog.Warn("Anfrage abgelehnt: nicht über Ingress", "remote", r.RemoteAddr, "path", r.URL.Path)
			WriteError(w, http.StatusForbidden, "zugriff nur über home assistant")
			return
		}
		next.ServeHTTP(w, r)
	})
}
