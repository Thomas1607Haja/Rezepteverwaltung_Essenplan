package recipes

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/fetch"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/httpx"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/images"
)

// Handler stellt die Rezept-Endpunkte unter /api/v1/recipes, die Bilder
// unter /api/v1/images und die Kategorieliste unter /api/v1/tags bereit.
type Handler struct {
	store   *Store
	images  *images.Store
	fetcher *fetch.Client // lädt Bilder von fremden Seiten (URL-Import)
}

// NewHandler erzeugt die Endpunkte für store; Rezeptbilder liegen in imgs,
// fetcher lädt Bilder von fremden Seiten.
func NewHandler(store *Store, imgs *images.Store, fetcher *fetch.Client) *Handler {
	return &Handler{store: store, images: imgs, fetcher: fetcher}
}

// Register meldet alle Rezept-Endpunkte beim Router mux an.
func (h *Handler) Register(mux httpx.Router) {
	mux.HandleFunc("GET /api/v1/recipes", h.list)
	mux.HandleFunc("POST /api/v1/recipes", h.create)
	mux.HandleFunc("GET /api/v1/recipes/{id}", h.get)
	mux.HandleFunc("PUT /api/v1/recipes/{id}", h.update)
	mux.HandleFunc("DELETE /api/v1/recipes/{id}", h.delete)
	mux.HandleFunc("POST /api/v1/recipes/{id}/variants", h.createVariant)
	mux.HandleFunc("PUT /api/v1/recipes/{id}/favorite", h.setFavorite)
	mux.HandleFunc("PUT /api/v1/recipes/{id}/image", h.setImage)
	mux.HandleFunc("DELETE /api/v1/recipes/{id}/image", h.deleteImage)
	mux.HandleFunc("POST /api/v1/recipes/{id}/image/fetch", h.fetchImage)
	mux.Handle("GET /api/v1/images/{name}", h.images)
	mux.HandleFunc("GET /api/v1/tags", h.listTags)
}

// listResponse ist die Antwort der Rezeptliste. Das umschließende Objekt
// lässt Platz für spätere Angaben (z. B. Gesamtzahl) neben der Liste.
type listResponse struct {
	Recipes []Summary `json:"recipes"`
}

// list liefert die Rezeptliste. Parameter in der Adresse, alle optional:
//
//	q=Nudeln%20Tomate   Suchtext
//	tag=3&tag=11        Kategorie-IDs (mehrfach möglich)
//	favorite=true       nur Favoriten
//	sort=name|newest    Sortierung (Standard: name)
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	filter := ListFilter{
		Query: params.Get("q"),
		Sort:  params.Get("sort"),
	}

	// params["tag"] enthält alle Werte des mehrfach angegebenen Parameters.
	for _, value := range params["tag"] {
		id, err := strconv.ParseInt(value, 10, 64)
		if err != nil || id <= 0 {
			httpx.WriteJSON(w, http.StatusBadRequest,
				httpx.ErrorResponse{Error: fmt.Sprintf("ungültige kategorie-id %q", value), Field: "tag"})
			return
		}
		filter.TagIDs = append(filter.TagIDs, id)
	}

	if value := params.Get("favorite"); value != "" {
		favorite, err := strconv.ParseBool(value)
		if err != nil {
			httpx.WriteJSON(w, http.StatusBadRequest,
				httpx.ErrorResponse{Error: "erwartet true oder false", Field: "favorite"})
			return
		}
		filter.FavoritesOnly = favorite
	}

	// image=illustration: nur Rezepte mit gezeichnetem Bild.
	switch value := params.Get("image"); value {
	case "":
	case ImageIllustration:
		filter.IllustrationsOnly = true
	default:
		httpx.WriteJSON(w, http.StatusBadRequest,
			httpx.ErrorResponse{Error: "erwartet image=illustration", Field: "image"})
		return
	}

	recipes, err := h.store.List(r.Context(), filter)
	if err != nil {
		writeStoreError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, listResponse{Recipes: recipes})
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
	files, err := h.store.Delete(r.Context(), id)
	if err != nil {
		writeStoreError(w, r, err)
		return
	}
	h.removeImageFiles(files)
	w.WriteHeader(http.StatusNoContent)
}

// createVariant legt eine Kopie des Rezepts als Variante an (siehe
// variants.go) und antwortet mit 201 und der neuen Variante. Das Bild wird
// als eigene Datei kopiert.
func (h *Handler) createVariant(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	source, err := h.store.Get(r.Context(), id)
	if err != nil {
		writeStoreError(w, r, err)
		return
	}
	var image, thumbnail string
	if source.Image != "" {
		if image, thumbnail, err = h.images.Copy(source.Image, source.Thumbnail); err != nil {
			httpx.WriteInternalError(w, r, err)
			return
		}
	}
	variant, err := h.store.CreateVariant(r.Context(), id, image, thumbnail)
	if err != nil {
		h.removeImageFiles([]string{image, thumbnail}) // Kopien nicht verwaisen lassen
		writeStoreError(w, r, err)
		return
	}
	w.Header().Set("Location", fmt.Sprintf("/api/v1/recipes/%d", variant.ID))
	httpx.WriteJSON(w, http.StatusCreated, variant)
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

// listTags liefert alle Kategorien als {"groups": [{"name": ..., "tags": [...]}]}.
func (h *Handler) listTags(w http.ResponseWriter, r *http.Request) {
	groups, err := h.store.ListTags(r.Context())
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, struct {
		Groups []TagGroup `json:"groups"`
	}{groups})
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
