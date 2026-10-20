// Package web liefert die eingebettete Weboberfläche aus (Quellcode in web/
// im Hauptverzeichnis, gebaut mit npm run build).
package web

import (
	"embed"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/httpx"
)

// distFiles enthält den Ordner dist/ samt gebauter Oberfläche in dist/app/.
// Das Präfix "all:" bettet auch Dateien ein, die mit . oder _ beginnen.
//
//go:embed all:dist
var distFiles embed.FS

// Handler liefert die Weboberfläche aus. Fehlt der Build, antwortet er mit
// einer Hinweisseite, damit der Server trotzdem startet (z. B. beim
// Arbeiten nur am Go-Code).
func Handler() http.Handler {
	// fs.Sub schneidet das Präfix ab: "dist/app/index.html" heißt danach
	// "index.html". Ein Fehler ist nur bei ungültigem Pfad möglich.
	app, err := fs.Sub(distFiles, "dist/app")
	if err != nil {
		panic(err)
	}
	if _, err := fs.Stat(app, "index.html"); err != nil {
		slog.Warn("Weboberfläche nicht eingebettet – vor go build im Ordner web/ npm run build ausführen")
		return newHandler(nil)
	}
	return newHandler(app)
}

// newHandler liefert die Dateien aus app aus; bei app == nil die
// Hinweisseite. Getrennt von Handler, damit Tests eigene Dateien
// übergeben können.
func newHandler(app fs.FS) http.Handler {
	var files http.Handler
	if app != nil {
		files = http.FileServerFS(app)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Unbekannte API-Pfade landen ebenfalls hier, weil dieser Handler
		// alles Übrige bekommt. Sie erhalten eine JSON-Antwort statt HTML.
		if strings.HasPrefix(path, "/api/") {
			httpx.WriteError(w, http.StatusNotFound, "unbekannter api-pfad")
			return
		}

		if files == nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(missingBuildPage))
			return
		}

		// Keine Ordnerinhalte auflisten (http.FileServer täte das sonst).
		if path != "/" && strings.HasSuffix(path, "/") {
			http.NotFound(w, r)
			return
		}

		if strings.HasPrefix(path, "/assets/") {
			// Vite hängt einen Hash des Inhalts an die Dateinamen
			// (index-CzapRqoI.js). Neue Inhalte haben neue Namen, daher darf
			// der Browser diese Dateien dauerhaft zwischenspeichern.
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			// index.html immer neu holen, damit ein Update sofort ankommt.
			w.Header().Set("Cache-Control", "no-cache")
		}
		files.ServeHTTP(w, r)
	})
}

const missingBuildPage = `<!doctype html>
<html lang="de">
<meta charset="utf-8">
<title>Rezeptplaner</title>
<p>Die Weboberfläche ist nicht eingebettet. Bitte im Ordner <code>web/</code>
<code>npm run build</code> ausführen und den Server neu bauen.</p>
<p>Die API unter <code>api/v1/</code> funktioniert trotzdem.</p>
`
