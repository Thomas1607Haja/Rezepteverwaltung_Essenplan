// Package httpx enthält Hilfsfunktionen für die JSON-API, die alle
// Fachbereiche gemeinsam nutzen.
package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// maxBodyBytes begrenzt die Größe einer JSON-Anfrage (1 MB). Ein Rezept ist
// weit kleiner; die Grenze schützt vor versehentlich riesigen Anfragen.
const maxBodyBytes = 1 << 20

// ErrorResponse ist der Aufbau jeder Fehlerantwort der API.
type ErrorResponse struct {
	Error string `json:"error"`
	Field string `json:"field,omitempty"` // betroffenes Feld bei Eingabefehlern
}

// WriteJSON sendet v als JSON mit dem HTTP-Status status.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// Status und Teile der Antwort sind dann schon unterwegs; mehr als
		// Protokollieren geht nicht mehr.
		slog.Warn("Antwort konnte nicht gesendet werden", "error", err)
	}
}

// WriteError sendet eine Fehlerantwort {"error": message}.
func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, ErrorResponse{Error: message})
}

// WriteInternalError protokolliert err und sendet eine allgemeine
// Fehlermeldung. Interna (SQL, Dateipfade) gehen so nicht nach außen.
func WriteInternalError(w http.ResponseWriter, r *http.Request, err error) {
	slog.Error("Interner Fehler", "method", r.Method, "path", r.URL.Path, "error", err)
	WriteError(w, http.StatusInternalServerError, "interner fehler")
}

// ReadJSON liest genau ein JSON-Objekt aus dem Anfragekörper nach dst.
// Unbekannte Felder werden abgelehnt, damit Tippfehler wie "titel" statt
// "title" auffallen statt still ignoriert zu werden.
func ReadJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("ungültiges JSON: %w", err)
	}
	// Nach dem Objekt darf nichts mehr folgen.
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("ungültiges JSON: nach dem objekt folgen weitere daten")
	}
	return nil
}

// SecurityHeaders ist eine Middleware, die jeder Antwort zwei Kopfzeilen
// mitgibt: Der Browser soll den Content-Type nicht erraten (sonst könnte er
// z. B. eine Textantwort als HTML ausführen) und beim Folgen von Links
// keine Adresse der Oberfläche weitergeben (die Ingress-Adresse enthält
// ein Sitzungsmerkmal).
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

// statusRecorder merkt sich den gesendeten HTTP-Status für das Protokoll.
// Durch das eingebettete http.ResponseWriter hat er automatisch alle
// Methoden des Originals; nur WriteHeader wird ersetzt.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (rec *statusRecorder) WriteHeader(status int) {
	rec.status = status
	rec.ResponseWriter.WriteHeader(status)
}

// LogRequests ist eine Middleware: Sie umhüllt next und protokolliert jede
// Anfrage mit Status und Dauer.
func LogRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		slog.Info("Anfrage",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"dauer", time.Since(start).Round(time.Microsecond))
	})
}

// Router ist alles, was die Fachbereiche zum Anmelden ihrer Endpunkte
// brauchen. *http.ServeMux erfüllt das Interface; ein Test kann stattdessen
// einen Router übergeben, der alle Muster mitschreibt.
type Router interface {
	Handle(pattern string, handler http.Handler)
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
}
