package shopping

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/apierr"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/homeassistant"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/httpx"
)

// Handler stellt die Endpunkte unter /api/v1/shopping bereit.
type Handler struct {
	store *Store
}

// NewHandler erzeugt die Endpunkte.
func NewHandler(store *Store) *Handler {
	return &Handler{store: store}
}

// Register meldet die Endpunkte beim Router an.
func (h *Handler) Register(mux httpx.Router) {
	mux.HandleFunc("GET /api/v1/shopping/preview", h.preview)
	mux.HandleFunc("POST /api/v1/shopping/send", h.send)
	mux.HandleFunc("GET /api/v1/shopping/reset", h.resetPreview)
	mux.HandleFunc("POST /api/v1/shopping/reset", h.reset)
}

// parsePeriod liest "current" (oder leer) bzw. "next".
func parsePeriod(value string) (next bool, ok bool) {
	switch value {
	case "", "current":
		return false, true
	case "next":
		return true, true
	}
	return false, false
}

// preview liefert die Einkaufsliste des aktuellen Einkaufszeitraums,
// mit ?period=next die des folgenden.
func (h *Handler) preview(w http.ResponseWriter, r *http.Request) {
	next, ok := parsePeriod(r.URL.Query().Get("period"))
	if !ok {
		httpx.WriteError(w, http.StatusBadRequest, apierr.Field("period", "invalid_value", "expected", "current, next"))
		return
	}
	p, err := h.store.Preview(r.Context(), next)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, p)
}

// send überträgt die Einkaufsliste an Bring, Anfrage z. B.
// {"period": "current", "deselected": ["mehl", "zwiebel"]}. Mit
// "missing": true nur die Differenz, mit "entries": [12] nur diese Gerichte.
func (h *Handler) send(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Period     string   `json:"period"`
		Deselected []string `json:"deselected"`
		Missing    bool     `json:"missing"`
		Entries    []int64  `json:"entries"`
	}
	if err := httpx.ReadJSON(w, r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err)
		return
	}
	next, ok := parsePeriod(in.Period)
	if !ok {
		httpx.WriteError(w, http.StatusBadRequest, apierr.Field("period", "invalid_value", "expected", "current, next"))
		return
	}
	result, err := h.store.Send(r.Context(), SendOptions{Next: next, Deselected: in.Deselected, Missing: in.Missing, Entries: in.Entries})
	h.writeResult(w, r, result, err)
}

// resetPreview zeigt, welche übertragenen Gerichte des Einkaufszeitraums
// (?period=current|next) bzw. welches einzelne Gericht (?entry=12) sich
// zurücksetzen lassen und an welche Liste sie gingen.
func (h *Handler) resetPreview(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	next, ok := parsePeriod(q.Get("period"))
	if !ok {
		httpx.WriteError(w, http.StatusBadRequest, apierr.Field("period", "invalid_value", "expected", "current, next"))
		return
	}
	var entryID int64
	if v := q.Get("entry"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil || id <= 0 {
			httpx.WriteError(w, http.StatusBadRequest, apierr.Field("entry", "invalid_id"))
			return
		}
		entryID = id
	}
	p, err := h.store.ResetPreview(r.Context(), next, entryID)
	h.writeResult(w, r, p, err)
}

// reset setzt übertragene Gerichte zurück, Anfrage z. B.
// {"entries": [12, 13], "remove": true, "fallback_list": "todo.testliste"}.
func (h *Handler) reset(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Entries      []int64 `json:"entries"`
		Remove       bool    `json:"remove"`
		FallbackList string  `json:"fallback_list"`
	}
	if err := httpx.ReadJSON(w, r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err)
		return
	}
	result, err := h.store.Reset(r.Context(), ResetOptions{Entries: in.Entries, Remove: in.Remove, FallbackList: in.FallbackList})
	h.writeResult(w, r, result, err)
}

// writeResult schreibt das Ergebnis oder den passenden Fehler.
func (h *Handler) writeResult(w http.ResponseWriter, r *http.Request, result any, err error) {
	switch _, isField := apierr.AsField(err); {
	case isField:
		httpx.WriteError(w, http.StatusBadRequest, err)
	case errors.Is(err, homeassistant.ErrUnavailable):
		// 502: Nicht unser Fehler, sondern der von Home Assistant.
		httpx.WriteError(w, http.StatusBadGateway, err)
	case err != nil:
		httpx.WriteInternalError(w, r, err)
	default:
		httpx.WriteJSON(w, http.StatusOK, result)
	}
}
