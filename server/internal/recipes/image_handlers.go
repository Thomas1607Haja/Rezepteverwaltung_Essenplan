package recipes

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/fetch"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/httpx"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/images"
)

// setImage nimmt ein Bild als Anfragekörper entgegen (z. B. eine JPEG-Datei
// mit Content-Type image/jpeg), ersetzt damit das Rezeptbild und antwortet
// mit dem aktualisierten Rezept.
func (h *Handler) setImage(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok || !h.recipeExists(w, r, id) {
		return
	}
	body := http.MaxBytesReader(w, r.Body, images.MaxUploadBytes)
	h.saveImage(w, r, id, body)
}

// fetchImage lädt das Rezeptbild von einer Adresse, Anfrage
// {"url": "https://…"}. Gedacht für den URL-Import: Das Bild der
// Quellseite wird erst beim Speichern geholt, damit bei einem
// abgebrochenen Import keine verwaisten Dateien entstehen.
func (h *Handler) fetchImage(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var input struct {
		URL string `json:"url"`
	}
	if err := httpx.ReadJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if !h.recipeExists(w, r, id) {
		return
	}

	data, contentType, err := h.fetcher.Get(r.Context(), input.URL, images.MaxUploadBytes)
	switch {
	case errors.Is(err, fetch.ErrTooLarge):
		httpx.WriteError(w, http.StatusRequestEntityTooLarge, "bild ist zu groß (höchstens 20 mb)")
		return
	case err != nil:
		httpx.WriteJSON(w, http.StatusBadGateway, httpx.ErrorResponse{Error: err.Error(), Field: "url"})
		return
	}
	if contentType != "" && !strings.HasPrefix(contentType, "image/") {
		httpx.WriteJSON(w, http.StatusUnsupportedMediaType,
			httpx.ErrorResponse{Error: "die adresse liefert kein bild", Field: "url"})
		return
	}
	h.saveImage(w, r, id, bytes.NewReader(data))
}

// recipeExists prüft vorab, ob es das Rezept gibt – sonst würde ein Bild
// umsonst verarbeitet. Gibt es es nicht, ist die Fehlerantwort schon
// gesendet.
func (h *Handler) recipeExists(w http.ResponseWriter, r *http.Request, id int64) bool {
	if _, err := h.store.Get(r.Context(), id); err != nil {
		writeStoreError(w, r, err)
		return false
	}
	return true
}

// saveImage verarbeitet das Bild aus src, trägt es beim Rezept ein, löscht
// das alte und antwortet mit dem aktualisierten Rezept.
func (h *Handler) saveImage(w http.ResponseWriter, r *http.Request, id int64, src io.Reader) {
	large, thumb, err := h.images.Save(src)
	var tooBig *http.MaxBytesError
	switch {
	case errors.As(err, &tooBig):
		httpx.WriteError(w, http.StatusRequestEntityTooLarge, "datei ist zu groß (höchstens 20 mb)")
		return
	case errors.Is(err, images.ErrUnsupported), errors.Is(err, images.ErrTooLarge):
		httpx.WriteError(w, http.StatusUnsupportedMediaType, err.Error())
		return
	case err != nil:
		httpx.WriteInternalError(w, r, err)
		return
	}

	oldFiles, err := h.store.SetImage(r.Context(), id, large, thumb, ImagePhoto)
	if err != nil {
		h.removeImageFiles([]string{large, thumb}) // neue Dateien nicht verwaisen lassen
		writeStoreError(w, r, err)
		return
	}
	h.removeImageFiles(oldFiles)

	recipe, err := h.store.Get(r.Context(), id)
	if err != nil {
		writeStoreError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, recipe)
}

// deleteImage entfernt das Rezeptbild. Antwort: 204 ohne Inhalt.
func (h *Handler) deleteImage(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	oldFiles, err := h.store.SetImage(r.Context(), id, "", "", "")
	if err != nil {
		writeStoreError(w, r, err)
		return
	}
	h.removeImageFiles(oldFiles)
	w.WriteHeader(http.StatusNoContent)
}

// removeImageFiles löscht nicht mehr benötigte Bilddateien. Schlägt das
// fehl, bleibt nur eine verwaiste Datei zurück – kein Grund, die Anfrage
// scheitern zu lassen. Daher nur protokollieren.
func (h *Handler) removeImageFiles(files []string) {
	if err := h.images.Delete(files...); err != nil {
		slog.Warn("Alte Bilddateien konnten nicht gelöscht werden", "error", err)
	}
}
