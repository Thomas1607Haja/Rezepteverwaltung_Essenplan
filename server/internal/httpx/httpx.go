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
	"strings"
	"time"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/apierr"
)

// maxBodyBytes begrenzt die Größe einer JSON-Anfrage (1 MB). Ein Rezept ist
// weit kleiner; die Grenze schützt vor versehentlich riesigen Anfragen.
const maxBodyBytes = 1 << 20

// ErrorResponse ist der Aufbau jeder Fehlerantwort der API. Die Oberflächen
// übersetzen Code und Parameter; Error ist der englische Text dazu (für das
// Protokoll und ältere App-Versionen). Siehe Paket apierr.
type ErrorResponse struct {
	Error  string         `json:"error"`
	Code   string         `json:"code"`
	Field  string         `json:"field,omitempty"` // betroffenes Feld bei Eingabefehlern
	Params map[string]any `json:"params,omitempty"`
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

// WriteError sendet err als Fehlerantwort. Fehler sollten aus apierr
// stammen; andere gehen mit dem Code "error" und ihrem Text hinaus (und
// fallen im Protokoll auf, damit sie einen Code bekommen).
func WriteError(w http.ResponseWriter, status int, err error) {
	var e *apierr.Error
	if !errors.As(err, &e) {
		slog.Warn("Fehler ohne Code", "status", status, "error", err)
		WriteJSON(w, status, ErrorResponse{Error: err.Error(), Code: "error"})
		return
	}
	WriteJSON(w, status, ErrorResponse{Error: e.Message(), Code: e.Code, Field: e.Field, Params: e.Params})
}

// WriteFieldError sendet err als Eingabefehler zum Feld field. Fehler ohne
// Code (z. B. Netzwerkfehler beim Laden einer fremden Seite) gehen als
// "page_unreachable" mit der Meldung als Detail hinaus.
func WriteFieldError(w http.ResponseWriter, status int, field string, err error) {
	var e *apierr.Error
	if !errors.As(err, &e) {
		e = apierr.New("page_unreachable", "detail", err.Error())
	}
	c := *e
	c.Field = field
	WriteError(w, status, &c)
}

// WriteInternalError protokolliert err und sendet eine allgemeine
// Fehlermeldung. Interna (SQL, Dateipfade) gehen so nicht nach außen.
func WriteInternalError(w http.ResponseWriter, r *http.Request, err error) {
	slog.Error("Interner Fehler", "method", r.Method, "path", r.URL.Path, "error", err)
	WriteError(w, http.StatusInternalServerError, apierr.New("internal_error"))
}

// ReadJSON liest genau ein JSON-Objekt aus dem Anfragekörper nach dst.
// Unbekannte Felder werden abgelehnt, damit Tippfehler wie "titel" statt
// "title" auffallen statt still ignoriert zu werden.
func ReadJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return apierr.New("invalid_json", "detail", jsonDetail(err))
	}
	// Nach dem Objekt darf nichts mehr folgen.
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return apierr.New("invalid_json", "detail", "extra data after the object")
	}
	return nil
}

// jsonDetail ist die (englische) Meldung des JSON-Decoders, ohne das
// Präfix "json: ". Bei zu großem Körper meldet MaxBytesReader einen
// eigenen Fehler.
func jsonDetail(err error) string {
	var tooBig *http.MaxBytesError
	if errors.As(err, &tooBig) {
		return fmt.Sprintf("request body larger than %d bytes", tooBig.Limit)
	}
	return strings.TrimPrefix(err.Error(), "json: ")
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
