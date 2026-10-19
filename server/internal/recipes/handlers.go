package recipes

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/httpx"
)

// Handler stellt die Rezept-Endpunkte unter /api/v1/recipes bereit.
type Handler struct {
	store *Store
}

// NewHandler erzeugt die Endpunkte für store.
func NewHandler(store *Store) *Handler {
	return &Handler{store: store}
}

// Register meldet alle Rezept-Endpunkte beim Router mux an.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/recipes", h.create)
	mux.HandleFunc("GET /api/v1/recipes/{id}", h.get)
}

// create legt ein Rezept an. Antwort: 201 mit dem gespeicherten Rezept.
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var input Recipe
	if err := httpx.ReadJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	recipe, err := h.store.Create(r.Context(), input)
	if err != nil {
		writeStoreError(w, r, err)
		return
	}
	// Location nennt die Adresse des neuen Rezepts (HTTP-Konvention bei 201).
	w.Header().Set("Location", fmt.Sprintf("/api/v1/recipes/%d", recipe.ID))
	httpx.WriteJSON(w, http.StatusCreated, recipe)
}

// get liefert ein Rezept mit allen Bestandteilen.
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	recipe, err := h.store.Get(r.Context(), id)
	if err != nil {
		writeStoreError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, recipe)
}

// parseID liest den Platzhalter {id} aus dem Pfad. Ist er keine positive
// Zahl, sendet parseID selbst die Fehlerantwort und liefert ok = false.
func parseID(w http.ResponseWriter, r *http.Request) (id int64, ok bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		httpx.WriteError(w, http.StatusBadRequest, "ungültige rezept-id")
		return 0, false
	}
	return id, true
}

// writeStoreError übersetzt Fehler des Stores in passende HTTP-Antworten.
func writeStoreError(w http.ResponseWriter, r *http.Request, err error) {
	var verr *ValidationError
	switch {
	case errors.As(err, &verr):
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{Error: verr.Message, Field: verr.Field})
	case errors.Is(err, ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, err.Error())
	default:
		httpx.WriteInternalError(w, r, err)
	}
}
