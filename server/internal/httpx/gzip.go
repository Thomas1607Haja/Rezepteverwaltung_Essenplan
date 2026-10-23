package httpx

import (
	"compress/gzip"
	"net/http"
	"strings"
	"sync"
)

// gzipPool hält Kompressoren zur Wiederverwendung: Jeder belegt einige
// hundert KB, neu anlegen je Antwort wäre unnötige Last auf dem Green.
var gzipPool = sync.Pool{New: func() any {
	w, _ := gzip.NewWriterLevel(nil, gzip.BestSpeed)
	return w
}}

// Gzip komprimiert JSON-Antworten, wenn der Client gzip versteht (die App
// tut das über OkHttp von selbst). Der volle Abgleich mit 500 Rezepten
// schrumpft so von 1,4 MB auf unter 100 KB. Bilder (schon komprimiert) und
// alles andere bleiben unverändert.
func Gzip(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Accept-Encoding")
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		gw := &gzipResponse{ResponseWriter: w}
		defer gw.close()
		next.ServeHTTP(gw, r)
	})
}

// gzipResponse entscheidet beim Schreiben der Kopfzeilen, ob komprimiert
// wird: nur JSON und nur, wenn noch keine Kodierung gesetzt ist.
type gzipResponse struct {
	http.ResponseWriter
	zw      *gzip.Writer
	decided bool
}

func (g *gzipResponse) WriteHeader(status int) {
	if !g.decided {
		g.decided = true
		h := g.Header()
		if strings.HasPrefix(h.Get("Content-Type"), "application/json") && h.Get("Content-Encoding") == "" &&
			status != http.StatusNoContent && status != http.StatusNotModified {
			h.Set("Content-Encoding", "gzip")
			h.Del("Content-Length") // stimmt nach dem Komprimieren nicht mehr
			g.zw = gzipPool.Get().(*gzip.Writer)
			g.zw.Reset(g.ResponseWriter)
		}
	}
	g.ResponseWriter.WriteHeader(status)
}

func (g *gzipResponse) Write(b []byte) (int, error) {
	if !g.decided {
		g.WriteHeader(http.StatusOK)
	}
	if g.zw != nil {
		return g.zw.Write(b)
	}
	return g.ResponseWriter.Write(b)
}

func (g *gzipResponse) close() {
	if g.zw != nil {
		g.zw.Close()
		gzipPool.Put(g.zw)
		g.zw = nil
	}
}
