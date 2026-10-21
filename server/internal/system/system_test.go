package system

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestReadMemory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "status")
	content := "Name:\trezeptplaner\nVmHWM:\t   71000 kB\nVmRSS:\t   23000 kB\nThreads:\t9\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	current, peak := readMemory(path)
	if current == nil || *current != 23000*1024 {
		t.Errorf("RSS = %v, erwartet %d", current, 23000*1024)
	}
	if peak == nil || *peak != 71000*1024 {
		t.Errorf("Spitze = %v, erwartet %d", peak, 71000*1024)
	}

	if c, p := readMemory(filepath.Join(t.TempDir(), "fehlt")); c != nil || p != nil {
		t.Error("ohne Datei sollten beide Werte nil sein")
	}
}

func TestInfo(t *testing.T) {
	dir := t.TempDir()
	files := map[string]int{
		"rezeptplaner.db":     1000,
		"rezeptplaner.db-wal": 200,
		"images/a.jpg":        300,
		"images/a-thumb.jpg":  40,
	}
	if err := os.Mkdir(filepath.Join(dir, "images"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, size := range files {
		if err := os.WriteFile(filepath.Join(dir, name), make([]byte, size), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	mux := http.NewServeMux()
	NewHandler("1.2.3", dir).Register(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/system", nil))

	var info Info
	if err := json.NewDecoder(rec.Body).Decode(&info); err != nil {
		t.Fatal(err)
	}
	if info.Version != "1.2.3" || info.DatabaseBytes != 1200 || info.ImagesBytes != 340 || info.ImagesCount != 2 {
		t.Errorf("unerwartete Antwort: %+v", info)
	}
}
