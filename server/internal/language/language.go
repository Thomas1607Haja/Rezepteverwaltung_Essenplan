// Package language ermittelt die Sprache des Haushalts: die Systemsprache
// von Home Assistant, ersatzweise die Umgebungsvariable
// REZEPTPLANER_LANGUAGE, sonst Englisch.
//
// Gebraucht wird sie an zwei Stellen: Die Weboberfläche fragt sie ab
// (GET /api/v1/language), und beim ersten Start einer neuen Installation
// werden die Startdaten (Kategorien, Vorratsliste) in dieser Sprache
// angelegt. Abgefragt wird nur bei Bedarf und dann einige Minuten
// zwischengespeichert – im Leerlauf passiert nichts.
package language

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/httpx"
)

// Supported sind die Sprachen der Oberflächen; die erste ist die Rückfallsprache.
var Supported = []string{"en", "de", "fr", "es", "it"}

// Fallback gilt, wenn nichts anderes bekannt ist.
const Fallback = "en"

// Normalize macht aus einer Sprachangabe wie "de-DE", "de_CH" oder "FR" den
// Sprachcode ("de", "fr"). Nicht unterstützte Sprachen ergeben "".
func Normalize(tag string) string {
	base, _, _ := strings.Cut(strings.ReplaceAll(strings.TrimSpace(tag), "_", "-"), "-")
	base = strings.ToLower(base)
	if slices.Contains(Supported, base) {
		return base
	}
	return ""
}

// Source liefert die Sprache von Home Assistant ("de", "en-GB" …); der
// Client von homeassistant erfüllt das Interface.
type Source interface {
	Language(ctx context.Context) (string, error)
}

// cacheFor: So lange gilt eine abgefragte Sprache. Wer die Sprache in
// Home Assistant umstellt, sieht die Weboberfläche nach spätestens dieser
// Zeit (und einem Neuladen) in der neuen Sprache.
const cacheFor = 5 * time.Minute

// Resolver fragt Home Assistant und merkt sich das Ergebnis kurz.
type Resolver struct {
	source Source
	env    string // Sprache aus REZEPTPLANER_LANGUAGE, schon normalisiert

	mu      sync.Mutex
	cached  string
	fetched time.Time
	now     func() time.Time
}

// NewResolver: source darf nil sein (z. B. im Testmodus ohne Home Assistant).
// env ist der Wert von REZEPTPLANER_LANGUAGE.
func NewResolver(source Source, env string) *Resolver {
	return &Resolver{source: source, env: Normalize(env), now: time.Now}
}

// HomeAssistant liefert die unterstützte Sprache von Home Assistant oder ""
// (nicht erreichbar, nicht unterstützt, kein Home Assistant).
func (r *Resolver) HomeAssistant(ctx context.Context) string {
	if r.source == nil {
		return ""
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.fetched.IsZero() && r.now().Sub(r.fetched) < cacheFor {
		return r.cached
	}
	tag, err := r.source.Language(ctx)
	if err != nil {
		// Nicht merken: Beim nächsten Abruf erneut versuchen.
		return ""
	}
	r.cached, r.fetched = Normalize(tag), r.now()
	return r.cached
}

// Default ist die Sprache für Startdaten und alles, was keine Oberfläche
// hat: Home Assistant, sonst REZEPTPLANER_LANGUAGE, sonst Englisch.
func (r *Resolver) Default(ctx context.Context) string {
	if lang := r.HomeAssistant(ctx); lang != "" {
		return lang
	}
	if r.env != "" {
		return r.env
	}
	return Fallback
}

// AtStartup ist Default für den ersten Start einer neuen Installation:
// Home Assistant ist dann eventuell noch nicht bereit. Deshalb wird
// einige Male im Abstand von wait nachgefragt, bevor der Ersatz gilt.
func (r *Resolver) AtStartup(ctx context.Context, tries int, wait time.Duration) string {
	for i := 0; r.source != nil && i < tries; i++ {
		if lang := r.HomeAssistant(ctx); lang != "" {
			return lang
		}
		select {
		case <-ctx.Done():
			return r.Default(ctx)
		case <-time.After(wait):
		}
	}
	return r.Default(ctx)
}

// SettingReader liefert die Einstellung ui_language ("auto" oder ein
// Sprachcode); settings.Store erfüllt das über eine kleine Hülle in main.
type SettingReader func(ctx context.Context) (string, error)

// Handler stellt GET /api/v1/language bereit.
type Handler struct {
	resolver *Resolver
	setting  SettingReader
}

// NewHandler erzeugt den Endpunkt.
func NewHandler(resolver *Resolver, setting SettingReader) *Handler {
	return &Handler{resolver: resolver, setting: setting}
}

// Register meldet den Endpunkt an.
func (h *Handler) Register(mux httpx.Router) {
	mux.HandleFunc("GET /api/v1/language", h.get)
}

// Response: Die Weboberfläche nimmt Setting, wenn es nicht "auto" ist,
// sonst HomeAssistant, sonst die Sprache des Browsers, sonst Englisch.
// Default ist die Sprache ohne Browser (für Startdaten), zur Anzeige.
type Response struct {
	Setting       string   `json:"setting"`
	HomeAssistant string   `json:"home_assistant"`
	Default       string   `json:"default"`
	Supported     []string `json:"supported"`
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	setting, err := h.setting(r.Context())
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, Response{
		Setting:       setting,
		HomeAssistant: h.resolver.HomeAssistant(r.Context()),
		Default:       h.resolver.Default(r.Context()),
		Supported:     Supported,
	})
}
