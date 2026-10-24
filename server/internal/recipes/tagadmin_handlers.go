package recipes

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/httpx"
)

// registerTagAdmin meldet die Verwaltung der Kategorien an (tagadmin.go).
// Über den App-Zugang sind die Pfade gesperrt (apikeys.Require).
func (h *Handler) registerTagAdmin(mux httpx.Router) {
	mux.HandleFunc("GET /api/v1/tag-groups", h.adminTags)
	mux.HandleFunc("POST /api/v1/tag-groups", h.createTagGroup)
	mux.HandleFunc("PUT /api/v1/tag-groups/order", h.orderTagGroups)
	mux.HandleFunc("PUT /api/v1/tag-groups/{id}", h.renameTagGroup)
	mux.HandleFunc("DELETE /api/v1/tag-groups/{id}", h.deleteTagGroup)
	mux.HandleFunc("PUT /api/v1/tag-groups/{id}/course", h.setCourseGroup)
	mux.HandleFunc("GET /api/v1/courses/suggestions", h.courseSuggestions)
	mux.HandleFunc("POST /api/v1/courses/assign", h.assignCourses)
	mux.HandleFunc("PUT /api/v1/tag-groups/{id}/tags", h.orderTags)
	mux.HandleFunc("POST /api/v1/tags", h.createTag)
	mux.HandleFunc("PUT /api/v1/tags/{id}", h.updateTag)
	mux.HandleFunc("DELETE /api/v1/tags/{id}", h.deleteTag)
	mux.HandleFunc("POST /api/v1/tags/{id}/merge", h.mergeTag)
	mux.HandleFunc("POST /api/v1/tags/bulk", h.bulkTags)
}

// writeTagError übersetzt Fehler der Verwaltung in HTTP-Antworten.
func writeTagError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrTagNotFound):
		httpx.WriteError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrGroupNotEmpty), errors.Is(err, ErrNoCourseGroup):
		httpx.WriteError(w, http.StatusConflict, err.Error())
	default:
		writeStoreError(w, r, err)
	}
}

// pathID liest {id} aus dem Pfad; bei einem Fehler ist die Antwort schon gesendet.
func pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		httpx.WriteError(w, http.StatusBadRequest, "ungültige id")
		return 0, false
	}
	return id, true
}

// adminTags: {"groups": [{"id", "name", "position", "tags": [{…, "recipe_count"}]}]}
func (h *Handler) adminTags(w http.ResponseWriter, r *http.Request) {
	groups, err := h.store.AdminTags(r.Context())
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, struct {
		Groups []AdminGroup `json:"groups"`
	}{groups})
}

type nameInput struct {
	Name string `json:"name"`
}

type idsInput struct {
	IDs []int64 `json:"ids"`
}

// createTagGroup: {"name": "Anlass"} → 201 {"id": 5}
func (h *Handler) createTagGroup(w http.ResponseWriter, r *http.Request) {
	var in nameInput
	if err := httpx.ReadJSON(w, r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	id, err := h.store.CreateTagGroup(r.Context(), in.Name)
	if err != nil {
		writeTagError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, struct {
		ID int64 `json:"id"`
	}{id})
}

// renameTagGroup: {"name": "…"}
func (h *Handler) renameTagGroup(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in nameInput
	if err := httpx.ReadJSON(w, r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.store.RenameTagGroup(r.Context(), id, in.Name); err != nil {
		writeTagError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// setCourseGroup: {"course": true} macht die Gruppe zur Gang-Gruppe.
func (h *Handler) setCourseGroup(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in struct {
		Course bool `json:"course"`
	}
	if err := httpx.ReadJSON(w, r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.store.SetCourseGroup(r.Context(), id, in.Course); err != nil {
		writeTagError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// courseSuggestions: Vorschläge für Rezepte ohne Gang.
func (h *Handler) courseSuggestions(w http.ResponseWriter, r *http.Request) {
	list, err := h.store.CourseSuggestions(r.Context())
	if err != nil {
		writeTagError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, struct {
		Suggestions []CourseSuggestion `json:"suggestions"`
	}{list})
}

// assignCourses: {"assignments": [{"recipe_id": 3, "tag_id": 41}]}
func (h *Handler) assignCourses(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Assignments []CourseAssignment `json:"assignments"`
	}
	if err := httpx.ReadJSON(w, r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	n, err := h.store.AssignCourses(r.Context(), in.Assignments)
	if err != nil {
		writeTagError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, struct {
		Assigned int `json:"assigned"`
	}{n})
}

// orderTagGroups: {"ids": [3, 1, 2]} – alle Gruppen in neuer Reihenfolge.
func (h *Handler) orderTagGroups(w http.ResponseWriter, r *http.Request) {
	var in idsInput
	if err := httpx.ReadJSON(w, r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.store.OrderTagGroups(r.Context(), in.IDs); err != nil {
		writeTagError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// deleteTagGroup: ?move_to=2 verschiebt die Kategorien, ?with_tags=true
// löscht sie mit; ohne beides 409, wenn die Gruppe nicht leer ist.
func (h *Handler) deleteTagGroup(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var moveTo int64
	if v := r.URL.Query().Get("move_to"); v != "" {
		var err error
		if moveTo, err = strconv.ParseInt(v, 10, 64); err != nil || moveTo <= 0 {
			httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{Error: "ungültige gruppe", Field: "move_to"})
			return
		}
	}
	withTags := r.URL.Query().Get("with_tags") == "true"
	if err := h.store.DeleteTagGroup(r.Context(), id, moveTo, withTags); err != nil {
		writeTagError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// orderTags: {"ids": [7, 3, 12]} – Inhalt und Reihenfolge einer Gruppe.
func (h *Handler) orderTags(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in idsInput
	if err := httpx.ReadJSON(w, r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.store.OrderTags(r.Context(), id, in.IDs); err != nil {
		writeTagError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type tagInput struct {
	Name    string `json:"name"`
	GroupID int64  `json:"group_id"`
}

// createTag: {"name": "Ofengericht", "group_id": 3} → 201 mit der Kategorie.
func (h *Handler) createTag(w http.ResponseWriter, r *http.Request) {
	var in tagInput
	if err := httpx.ReadJSON(w, r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	tag, err := h.store.CreateTag(r.Context(), in.Name, in.GroupID)
	if err != nil {
		writeTagError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, tag)
}

// updateTag: {"name": "…", "group_id": 3} – umbenennen und/oder verschieben.
func (h *Handler) updateTag(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in tagInput
	if err := httpx.ReadJSON(w, r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.store.UpdateTag(r.Context(), id, in.Name, in.GroupID); err != nil {
		writeTagError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) deleteTag(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := h.store.DeleteTag(r.Context(), id); err != nil {
		writeTagError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// mergeTag: {"into": 5} – Rezepte der Kategorie {id} bekommen 5, {id} wird gelöscht.
func (h *Handler) mergeTag(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in struct {
		Into int64 `json:"into"`
	}
	if err := httpx.ReadJSON(w, r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.store.MergeTag(r.Context(), id, in.Into); err != nil {
		writeTagError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
