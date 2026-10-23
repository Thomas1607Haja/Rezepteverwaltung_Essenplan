package fileimport

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/httpx"
)

// Register meldet die Endpunkte beim Router an.
func (im *Importer) Register(mux httpx.Router) {
	mux.HandleFunc("POST /api/v1/import/file", im.handleUpload)
	mux.HandleFunc("GET /api/v1/import/file/{token}/images/{index}", im.handleThumbnail)
	mux.HandleFunc("POST /api/v1/import/file/{token}", im.handleConfirm)
}

// handleUpload nimmt eine oder mehrere Dateien als multipart/form-data
// (Feldname "file") entgegen, speichert sie und liefert die Vorschau.
func (im *Importer) handleUpload(w http.ResponseWriter, r *http.Request) {
	im.mu.Lock()
	defer im.mu.Unlock()
	im.cleanup()

	token, err := newToken()
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}
	dir := filepath.Join(im.dir, token)
	if err := os.Mkdir(dir, 0o700); err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}
	keep := false
	defer func() {
		if !keep {
			os.RemoveAll(dir)
		}
	}()

	// Etwas Luft für die multipart-Kopfzeilen; die Dateien selbst zählt
	// saveParts genau.
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes+1<<20)
	files, err := saveParts(r, dir)
	var tooBig *http.MaxBytesError
	switch {
	case errors.As(err, &tooBig):
		httpx.WriteError(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("Upload zu groß (höchstens %d MB)", maxUploadBytes>>20))
		return
	case err != nil:
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	list, err := json.Marshal(files)
	if err == nil {
		err = os.WriteFile(filepath.Join(dir, "files.json"), list, 0o600)
	}
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}

	preview, err := im.preview(r.Context(), token, dir)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}
	keep = preview.Token != ""
	httpx.WriteJSON(w, http.StatusOK, preview)
}

// errUploadTooLarge: Die Dateien zusammen sind größer als erlaubt.
var errUploadTooLarge = &http.MaxBytesError{Limit: maxUploadBytes}

// saveParts speichert die Dateien der Anfrage im Ordner dir.
func saveParts(r *http.Request, dir string) ([]uploadedFile, error) {
	mr, err := r.MultipartReader()
	if err != nil {
		return nil, errors.New("Erwartet werden Dateien als multipart/form-data (Feld „file“)")
	}
	files := []uploadedFile{}
	var total int64
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			var tooBig *http.MaxBytesError
			if errors.As(err, &tooBig) {
				return nil, err
			}
			return nil, errors.New("Upload unvollständig oder beschädigt")
		}
		if part.FormName() != "file" {
			part.Close()
			continue
		}
		if len(files) == maxFiles {
			return nil, fmt.Errorf("Höchstens %d Dateien auf einmal", maxFiles)
		}
		f := uploadedFile{Name: displayName(part.FileName(), len(files)), Path: fmt.Sprintf("%02d.upload", len(files)+1)}
		out, err := os.OpenFile(filepath.Join(dir, f.Path), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			return nil, err
		}
		n, err := io.Copy(out, io.LimitReader(part, maxUploadBytes-total+1))
		out.Close()
		part.Close()
		if err != nil {
			var tooBig *http.MaxBytesError
			if errors.As(err, &tooBig) {
				return nil, err
			}
			return nil, errors.New("Upload unvollständig oder beschädigt")
		}
		total += n
		if total > maxUploadBytes {
			return nil, errUploadTooLarge
		}
		files = append(files, f)
	}
	if len(files) == 0 {
		return nil, errors.New("Keine Datei im Upload (Feld „file“)")
	}
	return files, nil
}

// displayName macht aus dem mitgeschickten Dateinamen eine kurze Anzeige.
// Der Name wird nur angezeigt, nie als Pfad benutzt.
func displayName(name string, index int) string {
	name = path.Base(strings.ReplaceAll(name, `\`, "/"))
	if name == "." || name == "/" || name == "" || !utf8.ValidString(name) {
		return fmt.Sprintf("Datei %d", index+1)
	}
	if r := []rune(name); len(r) > 100 {
		name = string(r[:100]) + "…"
	}
	return name
}

// handleThumbnail liefert das Vorschaubild eines Rezepts der Vorschau.
func (im *Importer) handleThumbnail(w http.ResponseWriter, r *http.Request) {
	dir := im.pendingDir(r.PathValue("token"))
	index, err := strconv.Atoi(r.PathValue("index"))
	if dir == "" || err != nil || index < 0 {
		httpx.WriteError(w, http.StatusNotFound, "Bild nicht gefunden")
		return
	}
	data, err := os.ReadFile(thumbPath(dir, index))
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "Bild nicht gefunden")
		return
	}
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "private, no-store")
	w.Write(data)
}

// handleConfirm legt die ausgewählten Rezepte an, Anfrage z. B.
// {"selected": [0, 2]} mit den Indizes aus der Vorschau.
func (im *Importer) handleConfirm(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Selected []int `json:"selected"`
	}
	if err := httpx.ReadJSON(w, r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(in.Selected) == 0 {
		httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{Error: "Kein Rezept ausgewählt", Field: "selected"})
		return
	}

	im.mu.Lock()
	defer im.mu.Unlock()
	result, err := im.confirm(r.Context(), r.PathValue("token"), in.Selected)
	switch {
	case errors.Is(err, errExpired):
		httpx.WriteError(w, http.StatusNotFound, err.Error())
	case err != nil:
		httpx.WriteInternalError(w, r, err)
	default:
		httpx.WriteJSON(w, http.StatusOK, result)
	}
}
