package settings

import (
	"errors"
	"net/http"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/httpx"
)

// Handler stellt GET und PUT /api/v1/settings bereit.
type Handler struct {
	store *Store
}

// NewHandler erzeugt die Endpunkte.
func NewHandler(store *Store) *Handler {
	return &Handler{store: store}
}

// Register meldet die Endpunkte beim Router an.
func (h *Handler) Register(mux httpx.Router) {
	mux.HandleFunc("GET /api/v1/settings", h.get)
	mux.HandleFunc("PUT /api/v1/settings", h.update)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	s, err := h.store.Get(r.Context())
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s)
}

// update ändert die Einstellungen, die in der Anfrage stehen, z. B.
// {"family_size": 4, "shopping_day": "saturday"}; fehlende behalten ihren
// Wert. So überschreibt das Speichern des Haushalts nicht die Adresse für
// die App, die an anderer Stelle bearbeitet wird.
func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	in, err := h.store.Get(r.Context())
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}
	// json.Unmarshal setzt nur die Felder, die im JSON vorkommen; die
	// übrigen behalten den eben gelesenen Stand.
	if err := httpx.ReadJSON(w, r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	s, err := h.store.Update(r.Context(), in)
	var verr *ValidationError
	switch {
	case errors.As(err, &verr):
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{Error: verr.Message, Field: verr.Field})
	case err != nil:
		httpx.WriteInternalError(w, r, err)
	default:
		httpx.WriteJSON(w, http.StatusOK, s)
	}
}
