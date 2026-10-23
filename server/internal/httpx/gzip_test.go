package httpx

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGzipOnlyForJSON(t *testing.T) {
	body := `{"recipes":[` + strings.Repeat(`{"title":"Linsensuppe"},`, 200) + `{}]}`
	h := Gzip(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/json":
			WriteJSON(w, http.StatusOK, map[string]string{"text": body})
		case "/image":
			w.Header().Set("Content-Type", "image/jpeg")
			w.Write([]byte("jpeg-daten"))
		case "/empty":
			w.WriteHeader(http.StatusNoContent)
		}
	}))

	get := func(path, accept string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", path, nil)
		if accept != "" {
			req.Header.Set("Accept-Encoding", accept)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	rec := get("/json", "gzip, br")
	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("JSON nicht komprimiert: %v", rec.Header())
	}
	zr, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatal(err)
	}
	plain, _ := io.ReadAll(zr)
	if !strings.Contains(string(plain), "Linsensuppe") || rec.Body.Len() >= len(plain) {
		t.Errorf("entpackt %d B, komprimiert %d B", len(plain), rec.Body.Len())
	}

	if rec := get("/json", ""); rec.Header().Get("Content-Encoding") != "" || !strings.Contains(rec.Body.String(), "Linsensuppe") {
		t.Errorf("ohne Accept-Encoding komprimiert: %v", rec.Header())
	}
	if rec := get("/image", "gzip"); rec.Header().Get("Content-Encoding") != "" || rec.Body.String() != "jpeg-daten" {
		t.Errorf("Bild verändert: %v %q", rec.Header(), rec.Body.String())
	}
	if rec := get("/empty", "gzip"); rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
		t.Errorf("204: %d, %d B", rec.Code, rec.Body.Len())
	}
}
