package recipes

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/httpx"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/images"
)

// setImage nimmt ein Bild als Anfragekörper entgegen (z. B. eine JPEG-Datei
// mit Content-Type image/jpeg), ersetzt damit das Rezeptbild und antwortet
// mit dem aktualisierten Rezept.
func (h *Handler) setImage(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	// Erst prüfen, ob es das Rezept gibt – sonst würde ein Bild umsonst
	// verarbeitet.
	if _, err := h.store.Get(r.Context(), id); err != nil {
		writeStoreError(w, r, err)
		return
	}

	body := http.MaxBytesReader(w, r.Body, images.MaxUploadBytes)
	large, thumb, err := h.images.Save(body)
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

	oldFiles, err := h.store.SetImage(r.Context(), id, large, thumb)
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
	oldFiles, err := h.store.SetImage(r.Context(), id, "", "")
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
