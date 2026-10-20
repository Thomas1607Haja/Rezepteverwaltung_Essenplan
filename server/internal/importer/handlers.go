package importer

import (
	"errors"
	"net/http"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/fetch"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/httpx"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/recipes"
)

// maxPageBytes begrenzt die Größe einer Rezeptseite (5 MB). Chefkoch-Seiten
// liegen bei einigen hundert KB.
const maxPageBytes = 5 << 20

// maxLines begrenzt die Zeilen, die auf einmal zerlegt werden.
const maxLines = 200

// Handler stellt die Import-Endpunkte unter /api/v1/import bereit.
type Handler struct {
	store   *recipes.Store
	fetcher *fetch.Client
}

// NewHandler erzeugt die Endpunkte. store dient der Dublettenprüfung.
func NewHandler(store *recipes.Store, fetcher *fetch.Client) *Handler {
	return &Handler{store: store, fetcher: fetcher}
}

// Register meldet die Endpunkte beim Router an.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/import/url", h.importURL)
	mux.HandleFunc("POST /api/v1/import/ingredients", h.parseIngredients)
}

// draftResponse ist das Ergebnis des URL-Imports: ein noch nicht
// gespeicherter Entwurf, die Bildadresse der Quellseite und vorhandene
// Rezepte mit derselben Quell-URL.
type draftResponse struct {
	Recipe     recipes.Recipe    `json:"recipe"`
	ImageURL   string            `json:"image_url"`
	Duplicates []recipes.Summary `json:"duplicates"`
}

// importURL lädt eine Rezeptseite, Anfrage {"url": "https://…"}.
func (h *Handler) importURL(w http.ResponseWriter, r *http.Request) {
	var input struct {
		URL string `json:"url"`
	}
	if err := httpx.ReadJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	pageURL, err := ParseURL(input.URL)
	if err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{Error: err.Error(), Field: "url"})
		return
	}

	page, _, err := h.fetcher.Get(r.Context(), pageURL.String(), maxPageBytes)
	switch {
	case errors.Is(err, fetch.ErrForbiddenAddress):
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{Error: err.Error(), Field: "url"})
		return
	case err != nil:
		// 502: Nicht unser Fehler, sondern der der fremden Seite.
		httpx.WriteJSON(w, http.StatusBadGateway, httpx.ErrorResponse{Error: err.Error(), Field: "url"})
		return
	}

	recipe, imageURL, err := Extract(page, pageURL)
	if errors.Is(err, ErrNoRecipe) {
		httpx.WriteJSON(w, http.StatusUnprocessableEntity, httpx.ErrorResponse{Error: err.Error(), Field: "url"})
		return
	}
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	duplicates, err := h.store.FindBySourceURL(r.Context(), recipe.SourceURL)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, draftResponse{Recipe: recipe, ImageURL: imageURL, Duplicates: duplicates})
}

// parseIngredients zerlegt Zutatenzeilen, Anfrage {"lines": ["500 g Mehl", …]}.
// Gedacht für den Foto-Import der App und zum Einfügen mehrerer Zeilen.
func (h *Handler) parseIngredients(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Lines []string `json:"lines"`
	}
	if err := httpx.ReadJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(input.Lines) > maxLines {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{Error: "zu viele zeilen (höchstens 200)", Field: "lines"})
		return
	}
	result := []recipes.Ingredient{}
	for _, line := range input.Lines {
		if ing := ParseIngredient(line); ing.Name != "" {
			result = append(result, ing)
		}
	}
	httpx.WriteJSON(w, http.StatusOK, struct {
		Ingredients []recipes.Ingredient `json:"ingredients"`
	}{result})
}
