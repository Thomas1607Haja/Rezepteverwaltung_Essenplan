package mealplan

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/httpx"
)

// Handler stellt die Endpunkte unter /api/v1/meal-plan bereit.
type Handler struct {
	store *Store
}

// NewHandler erzeugt die Endpunkte.
func NewHandler(store *Store) *Handler {
	return &Handler{store: store}
}

// Register meldet die Endpunkte beim Router an.
func (h *Handler) Register(mux httpx.Router) {
	mux.HandleFunc("GET /api/v1/meal-plan", h.list)
	mux.HandleFunc("POST /api/v1/meal-plan", h.create)
	mux.HandleFunc("GET /api/v1/meal-plan/{id}", h.get)
	mux.HandleFunc("PUT /api/v1/meal-plan/{id}", h.update)
	mux.HandleFunc("DELETE /api/v1/meal-plan/{id}", h.delete)
	mux.HandleFunc("POST /api/v1/meal-plan/{id}/move", h.move)
}

// listResponse enthält neben den Einträgen den heutigen Tag, die
// eingekaufte Woche und den nächsten Einkauf, alles aus Sicht des Servers.
type listResponse struct {
	Entries            []Entry `json:"entries"`
	Today              string  `json:"today"`
	ShoppingPeriod     Period  `json:"shopping_period"`
	NextShoppingPeriod Period  `json:"next_shopping_period"`
	// SlotTimes: Uhrzeiten der Slots ("lunch": "12:00", …) für den
	// Kalender der Home-Assistant-Integration (Schritt 9.3).
	SlotTimes map[string]string `json:"slot_times"`
}

// list liefert die Einträge eines Zeitraums: ?from=2026-10-17&to=2026-10-23.
// Ohne Angaben gilt der aktuelle Einkaufszeitraum.
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	period, err := h.store.CurrentPeriod(r.Context())
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}
	from, to := r.URL.Query().Get("from"), r.URL.Query().Get("to")
	if from == "" && to == "" {
		from, to = period.From, period.To
	}

	entries, err := h.store.List(r.Context(), from, to)
	if err != nil {
		writeStoreError(w, r, err)
		return
	}
	st, err := h.store.settings.Get(r.Context())
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, listResponse{
		Entries:            entries,
		Today:              h.store.Today().Format(dateLayout),
		ShoppingPeriod:     period,
		NextShoppingPeriod: period.Next(),
		SlotTimes:          map[string]string{SlotLunch: st.LunchTime, SlotCoffee: st.CoffeeTime, SlotDinner: st.DinnerTime},
	})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	e, err := h.store.Get(r.Context(), id)
	if err != nil {
		writeStoreError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, e)
}

// create legt einen Eintrag an, Anfrage z. B.
// {"date": "2026-10-24", "recipe_id": 12} oder {"date": "…", "note": "Essen gehen"}.
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var in NewEntry
	if err := httpx.ReadJSON(w, r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	e, err := h.store.Create(r.Context(), in)
	if err != nil {
		writeStoreError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, e)
}

// update ändert einen Eintrag, Anfrage z. B.
// {"recipe_id": 12, "servings": 6, "note": "", "status": "planned"}.
func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var c Changes
	if err := httpx.ReadJSON(w, r, &c); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	e, err := h.store.Update(r.Context(), id, c)
	if err != nil {
		writeStoreError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, e)
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

// move verschiebt oder tauscht einen Eintrag, Anfrage z. B.
// {"date": "2026-10-24", "slot": "dinner", "swap_with": 7}. Gibt es
// Warnungen, ist "applied" false und nichts gespeichert; mit
// "confirm": true (und ggf. "have_ingredients": [3]) wird trotzdem
// gespeichert.
func (h *Handler) move(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var m Move
	if err := httpx.ReadJSON(w, r, &m); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.store.Move(r.Context(), id, m)
	if err != nil {
		writeStoreError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}

// parseID liest die ID aus dem Pfad. Ist sie ungültig, ist die
// Fehlerantwort schon gesendet und ok false.
func parseID(w http.ResponseWriter, r *http.Request) (id int64, ok bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		httpx.WriteError(w, http.StatusBadRequest, "ungültige eintrags-id")
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
