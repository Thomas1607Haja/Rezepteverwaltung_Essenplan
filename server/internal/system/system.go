// Package system liefert Angaben zum laufenden Server: Version,
// Arbeitsspeicher und Platzbedarf der Daten. Gedacht für die
// Einstellungsseite und zum Messen auf dem Green.
package system

import (
	"bufio"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/httpx"
)

// APIVersion der Schnittstelle. 1 = erste Version für die Android-App
// (Abgleich, Datei-Import; App 0.6.0).
const APIVersion = 1

// Handler stellt GET /api/v1/system bereit.
type Handler struct {
	version string
	dataDir string
	started time.Time
}

// NewHandler erzeugt den Endpunkt. version wird beim Bauen gesetzt
// (siehe Dockerfile), dataDir ist der Datenordner.
func NewHandler(version, dataDir string) *Handler {
	return &Handler{version: version, dataDir: dataDir, started: time.Now()}
}

// Register meldet den Endpunkt beim Router an.
func (h *Handler) Register(mux httpx.Router) {
	mux.HandleFunc("GET /api/v1/system", h.info)
}

// Info ist die Antwort. Speicherwerte sind nil, wo das Betriebssystem sie
// nicht liefert (nur Linux kennt /proc).
type Info struct {
	Version string `json:"version"`
	// APIVersion steigt, wenn sich die Schnittstelle so ändert, dass die
	// Android-App angepasst werden muss. Die App prüft sie beim Einrichten.
	APIVersion    int    `json:"api_version"`
	GoVersion     string `json:"go_version"`
	UptimeSeconds int64  `json:"uptime_seconds"`
	MemoryBytes   *int64 `json:"memory_bytes"`      // belegter Arbeitsspeicher (RSS)
	MemoryPeak    *int64 `json:"memory_peak_bytes"` // Spitze seit dem Start
	DatabaseBytes int64  `json:"database_bytes"`    // Datenbank inkl. WAL-Datei
	ImagesBytes   int64  `json:"images_bytes"`
	ImagesCount   int    `json:"images_count"`
}

func (h *Handler) info(w http.ResponseWriter, r *http.Request) {
	info := Info{
		Version:       h.version,
		APIVersion:    APIVersion,
		GoVersion:     runtime.Version(),
		UptimeSeconds: int64(time.Since(h.started).Seconds()),
	}
	info.MemoryBytes, info.MemoryPeak = readMemory("/proc/self/status")

	// Wird nur bei Abruf berechnet, nicht im Hintergrund. Bei einigen
	// hundert Bildern dauert das wenige Millisekunden.
	_ = filepath.WalkDir(h.dataDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil // nicht lesbare Einträge überspringen
		}
		fi, err := d.Info()
		if err != nil {
			return nil
		}
		switch {
		case strings.HasPrefix(d.Name(), "rezeptplaner.db"):
			info.DatabaseBytes += fi.Size()
		case filepath.Base(filepath.Dir(path)) == "images":
			info.ImagesBytes += fi.Size()
			info.ImagesCount++
		}
		return nil
	})

	httpx.WriteJSON(w, http.StatusOK, info)
}

// readMemory liest VmRSS (aktuell belegt) und VmHWM (Spitze) aus
// /proc/self/status. Die Werte stehen dort in KiB, z. B. "VmRSS:  12345 kB".
// Gibt es die Datei nicht (Windows), sind beide nil.
func readMemory(path string) (current, peak *int64) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil
	}
	defer f.Close()

	// bufio.Scanner liest die Datei Zeile für Zeile.
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		kib, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			continue
		}
		bytes := kib * 1024
		switch fields[0] {
		case "VmRSS:":
			current = &bytes
		case "VmHWM:":
			peak = &bytes
		}
	}
	return current, peak
}
