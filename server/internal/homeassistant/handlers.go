package homeassistant

import (
	"log/slog"
	"net/http"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/apierr"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/httpx"
)

// Handler stellt GET /api/v1/homeassistant/status bereit.
type Handler struct {
	client Client
}

// NewHandler erzeugt den Endpunkt.
func NewHandler(client Client) *Handler {
	return &Handler{client: client}
}

// Register meldet den Endpunkt beim Router an.
func (h *Handler) Register(mux httpx.Router) {
	mux.HandleFunc("GET /api/v1/homeassistant/status", h.status)
}

// statusResponse meldet die Betriebsart und die vorhandenen To-do-Listen.
// Ist Home Assistant nicht erreichbar, steht der Grund in Error und Lists
// ist leer – die Einstellungsseite zeigt das an, statt zu scheitern.
type statusResponse struct {
	Mode  string     `json:"mode"`
	Lists []TodoList `json:"lists"`
	Error string     `json:"error,omitempty"` // englischer Text zu ErrorCode
	// ErrorCode zum Übersetzen, z. B. "home_assistant_down".
	ErrorCode string `json:"error_code,omitempty"`
}

func (h *Handler) status(w http.ResponseWriter, r *http.Request) {
	resp := statusResponse{Mode: h.client.Mode(), Lists: []TodoList{}}
	lists, err := h.client.TodoLists(r.Context())
	if err != nil {
		slog.Warn("To-do-Listen nicht lesbar", "error", err)
		e := apierr.From(err, "home_assistant_down")
		resp.Error, resp.ErrorCode = e.Message(), e.Code
	} else {
		resp.Lists = lists
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}
