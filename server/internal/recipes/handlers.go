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
	mux.HandleFunc("PUT /api/v1/recipes/{id}", h.update)
	mux.HandleFunc("DELETE /api/v1/recipes/{id}", h.delete)
	mux.HandleFunc("PUT /api/v1/recipes/{id}/favorite", h.setFavorite)
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

// update ersetzt ein Rezept vollständig. Antwort: 200 mit dem neuen Stand.
func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var input Recipe
	if err := httpx.ReadJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	recipe, err := h.store.Update(r.Context(), id, input)
	if err != nil {
		writeStoreError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, recipe)
}

// delete löscht ein Rezept. Antwort: 204 ohne Inhalt.
func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := h.store.Delete(r.Context(), id); err != nil {
		writeStoreError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// setFavorite setzt die Favoriten-Markierung, Anfrage {"is_favorite": true}.
// Antwort: 204 ohne Inhalt.
func (h *Handler) setFavorite(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	// Ein anonymer Struct-Typ genügt, wenn er nur hier gebraucht wird.
	// *bool statt bool: So lässt sich ein fehlendes Feld (nil) von false
	// unterscheiden.
	var input struct {
		IsFavorite *bool `json:"is_favorite"`
	}
	if err := httpx.ReadJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if input.IsFavorite == nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{Error: "fehlt", Field: "is_favorite"})
		return
	}

	if err := h.store.SetFavorite(r.Context(), id, *input.IsFavorite); err != nil {
		writeStoreError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
