package shopping

import (
	"net/http"

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
}

// preview liefert die Einkaufsliste des aktuellen Einkaufszeitraums,
// mit ?period=next die des folgenden.
func (h *Handler) preview(w http.ResponseWriter, r *http.Request) {
	var next bool
	switch r.URL.Query().Get("period") {
	case "", "current":
	case "next":
		next = true
	default:
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
