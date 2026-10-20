package recipes

import (
	"bytes"
	"encoding/json"
	"image"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestImageUploadOverHTTP(t *testing.T) {
	mux := newTestMux(t)
	if rec := do(mux, "POST", "/api/v1/recipes", `{"title": "Pfannkuchen", "base_servings": 2}`); rec.Code != http.StatusCreated {
		t.Fatalf("POST Status = %d", rec.Code)
	}

	var jpg bytes.Buffer
	jpeg.Encode(&jpg, image.NewRGBA(image.Rect(0, 0, 2000, 1000)), nil)

	upload := func(body []byte) *httptest.ResponseRecorder {
		req := httptest.NewRequest("PUT", "/api/v1/recipes/1/image", bytes.NewReader(body))
		req.Header.Set("Content-Type", "image/jpeg")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec
	}

	rec := upload(jpg.Bytes())
	if rec.Code != http.StatusOK {
		t.Fatalf("Upload Status = %d; Antwort: %s", rec.Code, rec.Body)
	}
	var r Recipe
	json.Unmarshal(rec.Body.Bytes(), &r)
	if !strings.HasSuffix(r.Image, ".jpg") || !strings.HasSuffix(r.Thumbnail, "_thumb.jpg") {
		t.Fatalf("Bildfelder = %q / %q", r.Image, r.Thumbnail)
	}

	// Das Bild ist abrufbar.
	if rec := do(mux, "GET", "/api/v1/images/"+r.Thumbnail, ""); rec.Code != http.StatusOK {
		t.Errorf("Vorschaubild abrufen: Status %d", rec.Code)
	}

	// Ersetzen: Das alte Bild verschwindet.
	if rec := upload(jpg.Bytes()); rec.Code != http.StatusOK {
		t.Fatalf("zweiter Upload Status = %d", rec.Code)
	}
	if rec := do(mux, "GET", "/api/v1/images/"+r.Image, ""); rec.Code != http.StatusNotFound {
		t.Errorf("altes Bild nach Ersetzen: Status %d, erwartet 404", rec.Code)
	}

	if rec := upload([]byte("kein bild")); rec.Code != http.StatusUnsupportedMediaType {
		t.Errorf("kein Bild: Status %d, erwartet 415", rec.Code)
	}

	if rec := do(mux, "DELETE", "/api/v1/recipes/1/image", ""); rec.Code != http.StatusNoContent {
		t.Errorf("Bild entfernen: Status %d, erwartet 204", rec.Code)
	}
	json.Unmarshal(do(mux, "GET", "/api/v1/recipes/1", "").Body.Bytes(), &r)
	if r.Image != "" || r.Thumbnail != "" {
		t.Errorf("nach Entfernen: %q / %q", r.Image, r.Thumbnail)
	}

	if rec := do(mux, "PUT", "/api/v1/recipes/42/image", "x"); rec.Code != http.StatusNotFound {
		t.Errorf("unbekanntes Rezept: Status %d, erwartet 404", rec.Code)
	}
}
