package apikeys

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/httpx"
)

// Handler stellt die Verwaltung unter /api/v1/api-keys bereit. Sie ist nur
// über die Weboberfläche (Ingress) erreichbar, nicht über den App-Zugang:
// Ein Gerät soll sich keine weiteren Schlüssel anlegen können.
type Handler struct {
	store *Store
}

// NewHandler erzeugt die Endpunkte.
func NewHandler(store *Store) *Handler {
	return &Handler{store: store}
}

// Register meldet die Endpunkte beim Router an.
func (h *Handler) Register(mux httpx.Router) {
	mux.HandleFunc("GET /api/v1/api-keys", h.list)
	mux.HandleFunc("POST /api/v1/api-keys", h.create)
	mux.HandleFunc("DELETE /api/v1/api-keys/{id}", h.delete)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	keys, err := h.store.List(r.Context())
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, keys)
}

// create legt einen Schlüssel an, Anfrage {"name": "Handy Thomas"}.
// Die Antwort enthält den Schlüssel unter "key" – das einzige Mal.
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name string `json:"name"`
	}
	if err := httpx.ReadJSON(w, r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	key, secret, err := h.store.Create(r.Context(), in.Name)
	if errors.Is(err, ErrInvalidName) {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{Error: err.Error(), Field: "name"})
		return
	}
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}
	slog.Info("API-Schlüssel angelegt", "id", key.ID, "name", key.Name)
	// Eingebettetes Key: Die Felder von Key stehen in der Antwort auf
	// oberster Ebene, dazu kommt "key".
	httpx.WriteJSON(w, http.StatusCreated, struct {
		Key
		Secret string `json:"key"`
	}{key, secret})
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		httpx.WriteError(w, http.StatusBadRequest, "ungültige id")
		return
	}
	switch err := h.store.Delete(r.Context(), id); {
	case errors.Is(err, ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, err.Error())
		return
	case err != nil:
		httpx.WriteInternalError(w, r, err)
		return
	}
	slog.Info("API-Schlüssel gesperrt", "id", id)
	w.WriteHeader(http.StatusNoContent)
}

// Require ist die Middleware des App-Zugangs: Jede Anfrage braucht die
// Kopfzeile "Authorization: Bearer <Schlüssel>" mit einem gültigen
// Schlüssel, sonst 401. Ausnahmen gibt es keine – auch nicht für
// /api/v1/health oder Bilder.
//
// Danach gilt: nur Pfade unter /api/v1/ und nicht die Schlüsselverwaltung.
// Die Weboberfläche gibt es nur über Ingress.
func Require(store *Store, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		valid := false
		if key, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer "); ok {
			var err error
			if valid, err = store.Check(r.Context(), key); err != nil {
				httpx.WriteInternalError(w, r, err)
				return
			}
		}
		if !valid {
			// Nur Adresse und Pfad protokollieren, nie den Schlüssel.
			slog.Warn("App-Zugang abgelehnt: kein gültiger Schlüssel", "remote", r.RemoteAddr, "path", r.URL.Path)
			w.Header().Set("WWW-Authenticate", `Bearer realm="rezeptplaner"`)
			httpx.WriteError(w, http.StatusUnauthorized, "gültiger api-schlüssel erforderlich")
			return
		}
		path := r.URL.Path
		if !strings.HasPrefix(path, "/api/v1/") {
			httpx.WriteError(w, http.StatusNotFound, "nur api unter /api/v1/")
			return
		}
		if path == "/api/v1/api-keys" || strings.HasPrefix(path, "/api/v1/api-keys/") {
			httpx.WriteError(w, http.StatusForbidden, "schlüssel nur über die weboberfläche verwalten")
			return
		}
		// Kategorien verwalten nur in der Weboberfläche (Schritt 8.C); lesen
		// (GET /api/v1/tags) darf die App.
		if path == "/api/v1/tag-groups" || strings.HasPrefix(path, "/api/v1/tag-groups/") ||
			(r.Method != http.MethodGet && (path == "/api/v1/tags" || strings.HasPrefix(path, "/api/v1/tags/"))) {
			httpx.WriteError(w, http.StatusForbidden, "kategorien nur über die weboberfläche verwalten")
			return
		}
		next.ServeHTTP(w, r)
	})
}
