package mealplan

import (
	"net/http"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/httpx"
)

// registerMenus meldet die Endpunkte für Menüs an (Schritt 9.4).
func (h *Handler) registerMenus(mux httpx.Router) {
	mux.HandleFunc("POST /api/v1/menus", h.createMenu)
	mux.HandleFunc("GET /api/v1/menus/{id}", h.getMenu)
	mux.HandleFunc("PUT /api/v1/menus/{id}", h.updateMenu)
	mux.HandleFunc("DELETE /api/v1/menus/{id}", h.deleteMenu)
	mux.HandleFunc("POST /api/v1/menus/{id}/move", h.moveMenu)
	mux.HandleFunc("POST /api/v1/menus/{id}/all-available", h.menuAllAvailable)
	mux.HandleFunc("POST /api/v1/meal-plan/{id}/to-menu", h.toMenu)
	mux.HandleFunc("POST /api/v1/meal-plan/{id}/detach", h.detach)
}

// menuResponse: ein Menü mit seinen Gängen (in Reihenfolge).
type menuResponse struct {
	Menu
	Entries []Entry `json:"entries"`
}

func (h *Handler) writeMenu(w http.ResponseWriter, r *http.Request, status int, m Menu) {
	entries, err := h.store.MenuEntries(r.Context(), m.ID)
	if err != nil {
		writeStoreError(w, r, err)
		return
	}
	httpx.WriteJSON(w, status, menuResponse{Menu: m, Entries: entries})
}

// createMenu: {"date", "slot", "title", "note", "servings"} → 201 mit dem (leeren) Menü.
func (h *Handler) createMenu(w http.ResponseWriter, r *http.Request) {
	var in MenuInput
	if err := httpx.ReadJSON(w, r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	m, err := h.store.CreateMenu(r.Context(), in)
	if err != nil {
		writeStoreError(w, r, err)
		return
	}
	h.writeMenu(w, r, http.StatusCreated, m)
}

func (h *Handler) getMenu(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	m, err := h.store.GetMenu(r.Context(), id)
	if err != nil {
		writeStoreError(w, r, err)
		return
	}
	h.writeMenu(w, r, http.StatusOK, m)
}

// updateMenu: {"title", "note", "servings", "apply_servings"}
func (h *Handler) updateMenu(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var in MenuInput
	if err := httpx.ReadJSON(w, r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	m, err := h.store.UpdateMenu(r.Context(), id, in)
	if err != nil {
		writeStoreError(w, r, err)
		return
	}
	h.writeMenu(w, r, http.StatusOK, m)
}

// deleteMenu: ?with_entries=true löscht die Gänge mit, sonst bleiben sie
// als einzelne Einträge („Menü auflösen“).
func (h *Handler) deleteMenu(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := h.store.DeleteMenu(r.Context(), id, r.URL.Query().Get("with_entries") == "true"); err != nil {
		writeStoreError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// moveMenu: wie POST /meal-plan/{id}/move, für das ganze Menü.
func (h *Handler) moveMenu(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var m Move
	if err := httpx.ReadJSON(w, r, &m); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.store.MoveMenu(r.Context(), id, m)
	if err != nil {
		writeStoreError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) menuAllAvailable(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := h.store.MenuAllAvailable(r.Context(), id); err != nil {
		writeStoreError(w, r, err)
		return
	}
	m, err := h.store.GetMenu(r.Context(), id)
	if err != nil {
		writeStoreError(w, r, err)
		return
	}
	h.writeMenu(w, r, http.StatusOK, m)
}

// toMenu: {"title", "note", "course_tag_id"} – „Zu einem Menü erweitern“.
func (h *Handler) toMenu(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var in struct {
		Title       string `json:"title"`
		Note        string `json:"note"`
		CourseTagID int64  `json:"course_tag_id"`
	}
	if err := httpx.ReadJSON(w, r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	m, err := h.store.ToMenu(r.Context(), id, in.Title, in.Note, in.CourseTagID)
	if err != nil {
		writeStoreError(w, r, err)
		return
	}
	h.writeMenu(w, r, http.StatusCreated, m)
}

// detach: „Aus Menü lösen“.
func (h *Handler) detach(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	e, err := h.store.Detach(r.Context(), id)
	if err != nil {
		writeStoreError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, e)
}
