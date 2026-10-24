package fileimport

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/fetch"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/recipes"
)

// TestPhotoFromSourcePage: Foto von der Quellseite statt Illustration oder
// fehlendem Bild, nie statt eines echten Fotos; Fehler bleiben folgenlos.
func TestPhotoFromSourcePage(t *testing.T) {
	png := testPNG(t)
	var pageHits atomic.Int32
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rezept-og", "/rezept-foto":
			pageHits.Add(1)
			fmt.Fprint(w, `<html><head><meta property="og:image" content="/bild.png"></head><body>Kein Rezept</body></html>`)
		case "/rezept-ld":
			pageHits.Add(1)
			fmt.Fprint(w, `<script type="application/ld+json">{"@type":"Recipe","name":"Linsen","image":"/bild.png"}</script>`)
		case "/bild.png":
			w.Write(png)
		default:
			http.NotFound(w, r)
		}
	}))
	defer site.Close()

	f := newFixtureWith(t, fetch.New(5*time.Second, true))
	doc := fmt.Sprintf(`{"format":"rezeptplaner-import","version":2,"recipes":[
		{"title":"Kürbissuppe","base_servings":4,"image":"images/a.png","image_kind":"illustration","source":{"url":"%[1]s/rezept-og"}},
		{"title":"Linsen","base_servings":4,"source":{"url":"%[1]s/rezept-ld"}},
		{"title":"Brot","base_servings":4,"image":"images/b.png","source":{"url":"%[1]s/rezept-foto"}},
		{"title":"Suppe","base_servings":4,"source":{"url":"%[1]s/gibt-es-nicht"}}]}`, site.URL)
	data := buildZip(t, map[string][]byte{
		"recipes.json": []byte(doc), "images/a.png": png, "images/b.png": png,
	})
	code, p, body := f.upload(t, namedFile{"r.zip", data})
	if code != http.StatusOK || len(p.Recipes) != 4 {
		t.Fatalf("%d %s", code, body)
	}
	if pageHits.Load() != 2 {
		t.Errorf("%d Seiten geladen, erwartet 2 (nie für ein echtes Foto)", pageHits.Load())
	}
	host := strings.TrimPrefix(site.URL, "http://")
	host = host[:strings.LastIndex(host, ":")]
	for i, want := range []string{host, host, "", ""} {
		r := p.Recipes[i]
		if r.WebImage != want {
			t.Errorf("%s: web_image %q, erwartet %q", r.Recipe.Title, r.WebImage, want)
		}
	}
	if w := strings.Join(p.Recipes[0].Warnings, "|"); !strings.Contains(w, "ersetzt die Illustration") {
		t.Errorf("Hinweis fehlt: %v", p.Recipes[0].Warnings)
	}
	if k := p.Recipes[0].Recipe.ImageKind; k != recipes.ImagePhoto {
		t.Errorf("Vorschau Bildart = %q", k)
	}
	if !p.Recipes[1].Thumbnail || p.Recipes[3].Thumbnail {
		t.Errorf("Vorschaubilder: %v %v", p.Recipes[1].Thumbnail, p.Recipes[3].Thumbnail)
	}
	if w := strings.Join(p.Recipes[3].Warnings, "|"); !strings.HasSuffix(w, "Kein Bild") || strings.Contains(w, "Foto von") {
		t.Errorf("Suppe: Hinweise %q", w)
	}

	// Beim Bestätigen wird nichts erneut geladen.
	site.Close()
	_, result := f.confirm(t, p.Token, 0, 1, 2, 3)
	if len(result.Created) != 4 {
		t.Fatalf("Bestätigen: %+v", result)
	}
	want := []string{recipes.ImagePhoto, recipes.ImagePhoto, recipes.ImagePhoto, ""}
	for i, c := range result.Created {
		saved, _ := f.recipes.Get(context.Background(), c.ID)
		if saved.ImageKind != want[i] || (want[i] != "") != (saved.Image != "") {
			t.Errorf("%s: Bildart %q, Bild %q", saved.Title, saved.ImageKind, saved.Image)
		}
		if len(c.Warnings) > 0 {
			t.Errorf("%s: Hinweise %v", saved.Title, c.Warnings)
		}
	}
}
