package importer

import (
	"errors"
	"net/http"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/apierr"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/fetch"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/httpx"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/pantry"
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
	pantry  *pantry.Store
	fetcher *fetch.Client
}

// NewHandler erzeugt die Endpunkte. store dient der Dublettenprüfung,
// pantryStore dem Abhaken von Vorratsartikeln.
func NewHandler(store *recipes.Store, pantryStore *pantry.Store, fetcher *fetch.Client) *Handler {
	return &Handler{store: store, pantry: pantryStore, fetcher: fetcher}
}

// markPantry setzt bei Zutaten aus dem Vorrat das Häkchen
// "einkaufsrelevant" zurück. Der Nutzer kann es im Entwurf wieder setzen.
func (h *Handler) markPantry(r *http.Request, ingredients []recipes.Ingredient) error {
	m, err := h.pantry.Matcher(r.Context())
	if err != nil {
		return err
	}
	for i := range ingredients {
		if m.Matches(ingredients[i].Name) {
			ingredients[i].ShoppingRelevant = false
		}
	}
	return nil
}

// Register meldet die Endpunkte beim Router an.
func (h *Handler) Register(mux httpx.Router) {
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
		httpx.WriteError(w, http.StatusBadRequest, err)
		return
	}
	pageURL, err := ParseURL(input.URL)
	if err != nil {
		httpx.WriteFieldError(w, http.StatusBadRequest, "url", err)
		return
	}

	page, _, err := h.fetcher.Get(r.Context(), pageURL.String(), maxPageBytes)
	switch {
	case errors.Is(err, fetch.ErrForbiddenAddress):
		httpx.WriteFieldError(w, http.StatusBadRequest, "url", err)
		return
	case err != nil:
		// 502: Nicht unser Fehler, sondern der der fremden Seite.
		httpx.WriteFieldError(w, http.StatusBadGateway, "url", err)
		return
	}

	recipe, imageURL, err := Extract(page, pageURL)
	if errors.Is(err, ErrNoRecipe) {
		httpx.WriteFieldError(w, http.StatusUnprocessableEntity, "url", err)
		return
	}
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	if err := h.markPantry(r, recipe.Ingredients); err != nil {
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
		httpx.WriteError(w, http.StatusBadRequest, err)
		return
	}
	if len(input.Lines) > maxLines {
		httpx.WriteError(w, http.StatusBadRequest, apierr.Field("lines", "too_many", "max", 200))
		return
	}
	result := []recipes.Ingredient{}
	for _, line := range input.Lines {
		if ing := ParseIngredient(line); ing.Name != "" {
			result = append(result, ing)
		}
	}
	if err := h.markPantry(r, result); err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, struct {
		Ingredients []recipes.Ingredient `json:"ingredients"`
	}{result})
}
