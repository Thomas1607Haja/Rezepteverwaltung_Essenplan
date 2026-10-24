package fileimport

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/database"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/fetch"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/images"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/pantry"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/recipes"
)

type fixture struct {
	mux     *http.ServeMux
	recipes *recipes.Store
	images  *images.Store
}

func newFixture(t *testing.T) fixture { return newFixtureWith(t, nil) }

// newFixtureWith: mit fetcher, der Fotos von Quellseiten lädt.
func newFixtureWith(t *testing.T, fetcher *fetch.Client) fixture {
	t.Helper()
	ctx := context.Background()
	db, err := database.Open(ctx, ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := database.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	img, err := images.NewStore(dir + "/images")
	if err != nil {
		t.Fatal(err)
	}
	rs := recipes.NewStore(db)
	im, err := NewImporter(dir+"/import", rs, pantry.NewStore(db), img, fetcher)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	im.Register(mux)
	return fixture{mux: mux, recipes: rs, images: img}
}

// upload schickt die Dateien (Name → Inhalt) und liefert Status und Vorschau.
func (f fixture) upload(t *testing.T, files ...namedFile) (int, Preview, string) {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	for _, file := range files {
		w, err := mw.CreateFormFile("file", file.name)
		if err != nil {
			t.Fatal(err)
		}
		w.Write(file.data)
	}
	mw.Close()
	req := httptest.NewRequest("POST", "/api/v1/import/file", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, req)
	var p Preview
	if rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
			t.Fatal(err)
		}
	}
	return rec.Code, p, rec.Body.String()
}

func (f fixture) confirm(t *testing.T, token string, selected ...int) (int, Result) {
	t.Helper()
	body, _ := json.Marshal(map[string][]int{"selected": selected})
	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, httptest.NewRequest("POST", "/api/v1/import/file/"+token, bytes.NewReader(body)))
	var r Result
	if rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), &r); err != nil {
			t.Fatal(err)
		}
	}
	return rec.Code, r
}

type namedFile struct {
	name string
	data []byte
}

// buildZip packt die Einträge (Pfad → Inhalt) in eine ZIP-Datei.
func buildZip(t *testing.T, entries map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, data := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		w.Write(data)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func testPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 64, 48))
	for x := range 64 {
		for y := range 48 {
			img.Set(x, y, color.RGBA{uint8(x * 4), uint8(y * 5), 100, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

const twoRecipes = `{
  "format": "rezeptplaner-import",
  "version": 1,
  "generator": "test",
  "recipes": [
    {
      "title": "Gnocchi-Pfanne",
      "base_servings": 2,
      "total_minutes": 25,
      "source": {"name": "HelloFresh-Karte"},
      "categories": ["Pfanne", "schnell", "Lieblingsessen"],
      "image": "images/gnocchi.png",
      "ingredients": [
        {"quantity": 400, "unit": "g", "name": "Gnocchi"},
        {"name": "Salz", "note": "nach Geschmack"},
        {"quantity": 1, "unit": "EL", "name": "Olivenöl", "shopping_relevant": false}
      ],
      "steps": ["Gnocchi anbraten.", "Würzen."]
    },
    {
      "title": "",
      "base_servings": 0,
      "ingredients": [{"name": ""}]
    }
  ]
}`

func TestZipPreviewAndConfirm(t *testing.T) {
	f := newFixture(t)
	// Alles in einem Ordner, wie beim Packen eines Ordners.
	data := buildZip(t, map[string][]byte{
		"rezepte/recipes.json":        []byte(twoRecipes),
		"rezepte/images/gnocchi.png":  testPNG(t),
		"__MACOSX/rezepte/._x":        []byte("x"),
		"rezepte/images/unbenutzt.jg": []byte("egal"),
	})
	code, p, body := f.upload(t, namedFile{"rezepte.zip", data})
	if code != http.StatusOK || p.Token == "" {
		t.Fatalf("Upload: %d %s", code, body)
	}
	if len(p.Recipes) != 2 || len(p.FileErrors) != 0 {
		t.Fatalf("Vorschau: %+v", p)
	}
	g := p.Recipes[0]
	if len(g.Errors) != 0 || !g.Thumbnail || g.Recipe.BaseServings != 2 || g.Recipe.SourceName != "HelloFresh-Karte" {
		t.Errorf("Gnocchi: %+v", g)
	}
	// "schnell" passt trotz Kleinschreibung, "Lieblingsessen" gibt es nicht.
	if len(g.Recipe.Tags) != 2 || !strings.Contains(strings.Join(g.Warnings, "|"), "Lieblingsessen") {
		t.Errorf("Kategorien %+v, Hinweise %v", g.Recipe.Tags, g.Warnings)
	}
	// Salz steht auf der Vorratsliste.
	if !g.Recipe.Ingredients[0].ShoppingRelevant || g.Recipe.Ingredients[1].ShoppingRelevant || g.Recipe.Ingredients[2].ShoppingRelevant {
		t.Errorf("einkaufsrelevant: %+v", g.Recipe.Ingredients)
	}
	bad := p.Recipes[1]
	if len(bad.Errors) < 3 {
		t.Errorf("fehlerhaftes Rezept: %v", bad.Errors)
	}

	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/import/file/"+p.Token+"/images/0", nil))
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "image/jpeg" {
		t.Errorf("Vorschaubild: %d", rec.Code)
	}

	code, result := f.confirm(t, p.Token, 0, 1)
	if code != http.StatusOK || len(result.Created) != 1 || len(result.Failed) != 1 {
		t.Fatalf("Bestätigen: %d %+v", code, result)
	}
	saved, err := f.recipes.Get(context.Background(), result.Created[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Image == "" || saved.Thumbnail == "" || len(saved.Ingredients) != 3 || len(saved.Steps) != 2 || saved.SourceName != "HelloFresh-Karte" {
		t.Errorf("gespeichert: %+v", saved)
	}
	// Version 1 kennt keine Bildart: Das Bild gilt als Foto.
	if saved.ImageKind != recipes.ImagePhoto {
		t.Errorf("Bildart = %q, erwartet photo", saved.ImageKind)
	}

	// Zweites Bestätigen: Upload ist weg, keine doppelten Rezepte.
	if code, _ := f.confirm(t, p.Token, 0); code != http.StatusNotFound {
		t.Errorf("zweites Bestätigen: %d, erwartet 404", code)
	}

	// Gleicher Titel wird beim nächsten Mal als Dublette gemeldet.
	_, p2, _ := f.upload(t, namedFile{"nochmal.zip", data})
	if len(p2.Recipes[0].Duplicates) != 1 {
		t.Errorf("Dublette nicht erkannt: %+v", p2.Recipes[0].Duplicates)
	}
}

func TestSingleJSONAndContentDetection(t *testing.T) {
	f := newFixture(t)
	one := `{"format":"rezeptplaner-import","version":1,"recipes":[{"title":"Linsensuppe","base_servings":4,"image":"images/x.jpg"}]}`
	// Name ohne Endung, wie manche Apps beim Teilen schicken.
	code, p, body := f.upload(t,
		namedFile{"content://irgendwas/1234", []byte("\xef\xbb\xbf  " + one)},
		namedFile{"fremd.json", []byte(`{"name": "kein Rezept"}`)},
		namedFile{"notiz.txt", []byte("Hallo")},
		namedFile{"neu.json", []byte(`{"format":"rezeptplaner-import","version":3,"recipes":[]}`)},
	)
	if code != http.StatusOK || len(p.Recipes) != 1 || len(p.FileErrors) != 3 {
		t.Fatalf("%d %s", code, body)
	}
	if p.Recipes[0].File != "1234" {
		t.Errorf("Dateiname %q", p.Recipes[0].File)
	}
	if !strings.Contains(strings.Join(p.Recipes[0].Warnings, "|"), "nur in ZIP") {
		t.Errorf("Hinweis zum Bild fehlt: %v", p.Recipes[0].Warnings)
	}
	want := []string{"Formatkennung", "Weder ZIP", "Formatversion 3"}
	for i, fe := range p.FileErrors {
		if !strings.Contains(fe.Error, want[i]) {
			t.Errorf("Dateifehler %d: %q, erwartet %q", i, fe.Error, want[i])
		}
	}
}

// Version 2: image_kind wird übernommen; ohne Bild bleibt es wirkungslos,
// unbekannte Werte gelten mit Hinweis als Foto.
func TestImageKindVersion2(t *testing.T) {
	f := newFixture(t)
	doc := `{"format":"rezeptplaner-import","version":2,"recipes":[
		{"title":"Kürbissuppe","base_servings":4,"image":"images/a.png","image_kind":"illustration"},
		{"title":"Linsen","base_servings":4,"image":"images/b.png","image_kind":"gemalt"},
		{"title":"Brot","base_servings":4,"image_kind":"illustration"}]}`
	data := buildZip(t, map[string][]byte{
		"recipes.json": []byte(doc), "images/a.png": testPNG(t), "images/b.png": testPNG(t),
	})
	code, p, body := f.upload(t, namedFile{"r.zip", data})
	if code != http.StatusOK || len(p.Recipes) != 3 {
		t.Fatalf("%d %s", code, body)
	}
	if k := p.Recipes[0].Recipe.ImageKind; k != recipes.ImageIllustration {
		t.Errorf("Vorschau Bildart = %q", k)
	}
	if !strings.Contains(strings.Join(p.Recipes[1].Warnings, "|"), "Bildart „gemalt“") {
		t.Errorf("Hinweis fehlt: %v", p.Recipes[1].Warnings)
	}
	_, result := f.confirm(t, p.Token, 0, 1, 2)
	if len(result.Created) != 3 {
		t.Fatalf("Bestätigen: %+v", result)
	}
	want := []string{recipes.ImageIllustration, recipes.ImagePhoto, ""}
	for i, c := range result.Created {
		saved, _ := f.recipes.Get(context.Background(), c.ID)
		if saved.ImageKind != want[i] {
			t.Errorf("%s: Bildart %q, erwartet %q", saved.Title, saved.ImageKind, want[i])
		}
	}
	list, _ := f.recipes.List(context.Background(), recipes.ListFilter{IllustrationsOnly: true})
	if len(list) != 1 || list[0].Title != "Kürbissuppe" {
		t.Errorf("Filter Illustrationen: %+v", list)
	}
}

func TestUnknownFieldIsReported(t *testing.T) {
	f := newFixture(t)
	doc := `{"format":"rezeptplaner-import","version":1,"recipes":[
		{"title":"A","base_servings":2,"ingredient":[]},
		{"title":"B","base_servings":"zwei"}]}`
	_, p, body := f.upload(t, namedFile{"r.json", []byte(doc)})
	if len(p.Recipes) != 2 || p.Token != "" {
		t.Fatalf("%s", body)
	}
	if !strings.Contains(p.Recipes[0].Errors[0], "unbekanntes Feld „ingredient“") ||
		!strings.Contains(p.Recipes[1].Errors[0], "falschen Typ") || p.Recipes[1].Recipe.Title != "B" {
		t.Errorf("Meldungen: %v / %v", p.Recipes[0].Errors, p.Recipes[1].Errors)
	}
}

func TestUnsafeAndOversizedZips(t *testing.T) {
	f := newFixture(t)
	doc := []byte(`{"format":"rezeptplaner-import","version":1,"recipes":[{"title":"A","base_servings":2,"image":"images/a.jpg"}]}`)

	tests := []struct {
		name    string
		entries map[string][]byte
		want    string
	}{
		{"zip slip", map[string][]byte{"recipes.json": doc, "../../etc/passwd": []byte("x")}, "Unzulässiger Pfad"},
		{"absolut", map[string][]byte{"recipes.json": doc, "/etc/passwd": []byte("x")}, "Unzulässiger Pfad"},
		{"backslash", map[string][]byte{"recipes.json": doc, `..\x`: []byte("x")}, "Unzulässiger Pfad"},
		{"ohne recipes.json", map[string][]byte{"rezepte.json": doc}, "recipes.json fehlt"},
		{"zu tief", map[string][]byte{"a/b/recipes.json": doc}, "recipes.json fehlt"},
	}
	for _, tt := range tests {
		_, p, body := f.upload(t, namedFile{tt.name + ".zip", buildZip(t, tt.entries)})
		if len(p.FileErrors) != 1 || !strings.Contains(p.FileErrors[0].Error, tt.want) {
			t.Errorf("%s: %s", tt.name, body)
		}
	}

	// ZIP-Bombe im Kleinen: 25 MB Nullen sind gepackt nur ein paar
	// Kilobyte, entpackt aber größer als die Grenze für ein Bild.
	bomb := buildZip(t, map[string][]byte{"recipes.json": doc, "images/a.jpg": make([]byte, 25<<20)})
	if len(bomb) > 1<<20 {
		t.Fatalf("Testdatei unerwartet groß: %d", len(bomb))
	}
	_, p, body := f.upload(t, namedFile{"bombe.zip", bomb})
	if len(p.Recipes) != 1 || !strings.Contains(strings.Join(p.Recipes[0].Warnings, "|"), "zu groß") {
		t.Errorf("Bombe: %s", body)
	}

	// Ein "Bild", das keines ist: Rezept kommt ohne Bild.
	fake := buildZip(t, map[string][]byte{"recipes.json": doc, "images/a.jpg": []byte("<script>alert(1)</script>")})
	_, p, body = f.upload(t, namedFile{"fake.zip", fake})
	if len(p.Recipes) != 1 || p.Recipes[0].Thumbnail || !strings.Contains(strings.Join(p.Recipes[0].Warnings, "|"), "unbrauchbar") {
		t.Errorf("falsches Bild: %s", body)
	}
}

func TestTooManyEntriesAndFiles(t *testing.T) {
	f := newFixture(t)
	entries := map[string][]byte{"recipes.json": []byte(`{}`)}
	for i := range maxEntries {
		entries[fmt.Sprintf("images/%d.jpg", i)] = []byte("x")
	}
	_, p, body := f.upload(t, namedFile{"viele.zip", buildZip(t, entries)})
	if len(p.FileErrors) != 1 || !strings.Contains(p.FileErrors[0].Error, "Zu viele Einträge") {
		t.Errorf("Einträge: %s", body)
	}

	var files []namedFile
	for range maxFiles + 1 {
		files = append(files, namedFile{"a.json", []byte("{}")})
	}
	if code, _, body := f.upload(t, files...); code != http.StatusBadRequest {
		t.Errorf("zu viele Dateien: %d %s", code, body)
	}
}

func TestUploadTooLarge(t *testing.T) {
	f := newFixture(t)
	code, _, body := f.upload(t, namedFile{"gross.zip", make([]byte, maxUploadBytes+10)})
	if code != http.StatusRequestEntityTooLarge {
		t.Errorf("%d %s", code, body)
	}
}
