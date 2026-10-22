package pantry

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/httpx"
)

// maxNames begrenzt die Namen, die auf einmal geprüft werden.
const maxNames = 200

// Handler stellt die Endpunkte unter /api/v1/pantry bereit.
type Handler struct {
	store *Store
}

// NewHandler erzeugt die Endpunkte.
func NewHandler(store *Store) *Handler {
	return &Handler{store: store}
}

// Register meldet die Endpunkte beim Router an.
func (h *Handler) Register(mux httpx.Router) {
	mux.HandleFunc("GET /api/v1/pantry", h.list)
	mux.HandleFunc("POST /api/v1/pantry", h.create)
	mux.HandleFunc("PUT /api/v1/pantry/{id}", h.update)
	mux.HandleFunc("DELETE /api/v1/pantry/{id}", h.delete)
	mux.HandleFunc("POST /api/v1/pantry/match", h.match)
	mux.HandleFunc("POST /api/v1/pantry/apply", h.apply)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.store.List(r.Context())
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, items)
}

// create legt einen Artikel an, Anfrage z. B.
// {"name": "Öl", "synonyms": ["Olivenöl", "Rapsöl"]}.
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var in Item
	if err := httpx.ReadJSON(w, r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	it, err := h.store.Create(r.Context(), in)
	if err != nil {
		writeStoreError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, it)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var in Item
	if err := httpx.ReadJSON(w, r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	it, err := h.store.Update(r.Context(), id, in)
	if err != nil {
		writeStoreError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, it)
}

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

// match prüft Zutatennamen gegen den Vorrat, Anfrage
// {"names": ["Olivenöl", "Mehl"]}, Antwort {"in_pantry": [true, false]}.
// Die Weboberfläche nutzt das, wenn eine neue Zutat eingetippt wird.
func (h *Handler) match(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Names []string `json:"names"`
	}
	if err := httpx.ReadJSON(w, r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(in.Names) > maxNames {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{Error: "zu viele namen (höchstens 200)", Field: "names"})
		return
	}
	m, err := h.store.Matcher(r.Context())
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}
	result := make([]bool, len(in.Names))
	for i, name := range in.Names {
		result[i] = m.Matches(name)
	}
	httpx.WriteJSON(w, http.StatusOK, struct {
		InPantry []bool `json:"in_pantry"`
	}{result})
}

// apply überträgt die Vorratsliste auf alle Rezepte. Anfrage {} liefert
// nur die Vorschau ("applied": false), {"confirm": true} speichert.
func (h *Handler) apply(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Confirm bool `json:"confirm"`
	}
	if err := httpx.ReadJSON(w, r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.store.ApplyToRecipes(r.Context(), in.Confirm)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}

// parseID liest die ID aus dem Pfad. Ist sie ungültig, ist die
// Fehlerantwort schon gesendet und ok false.
func parseID(w http.ResponseWriter, r *http.Request) (id int64, ok bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		httpx.WriteError(w, http.StatusBadRequest, "ungültige id")
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
