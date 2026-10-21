package shopping

import (
	"errors"
	"net/http"

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
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/shopping/preview", h.preview)
	mux.HandleFunc("POST /api/v1/shopping/send", h.send)
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
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{Error: "period muss current oder next sein", Field: "period"})
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
// {"period": "current", "deselected": ["mehl", "zwiebel"]}.
func (h *Handler) send(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Period     string   `json:"period"`
		Deselected []string `json:"deselected"`
	}
	if err := httpx.ReadJSON(w, r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	next, ok := parsePeriod(in.Period)
	if !ok {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{Error: "period muss current oder next sein", Field: "period"})
		return
	}
	result, err := h.store.Send(r.Context(), next, in.Deselected)
	var verr *ValidationError
	switch {
	case errors.As(err, &verr):
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{Error: verr.Message, Field: verr.Field})
	case errors.Is(err, homeassistant.ErrUnavailable):
		// 502: Nicht unser Fehler, sondern der von Home Assistant.
		httpx.WriteError(w, http.StatusBadGateway, err.Error())
	case err != nil:
		httpx.WriteInternalError(w, r, err)
	default:
		httpx.WriteJSON(w, http.StatusOK, result)
	}
}
