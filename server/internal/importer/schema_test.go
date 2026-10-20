package importer

import (
	"errors"
	"net/url"
	"testing"
)

// chefkochLike ist einer Chefkoch-Seite nachempfunden: Rezept als
// einzelnes Objekt, Zubereitung als ein langer Text mit Zeilenumbrüchen,
// HTML-Entities, Zeiten mit Tagesangabe.
const chefkochLike = `<!doctype html><html><head>
<script type="application/ld+json">{"@context":"http://schema.org","@type":"BreadcrumbList","itemListElement":[]}</script>
<script type="application/ld+json">
{
  "@context": "http://schema.org",
  "@type": "Recipe",
  "name": "Omas Gulasch",
  "description": "Ein deftiges Gulasch &amp; ganz einfach.",
  "image": "https://img.example.org/gulasch.jpg",
  "recipeYield": "4 Portion(en)",
  "prepTime": "P0DT0H30M",
  "cookTime": "P0DT1H30M",
  "totalTime": "P0DT2H0M",
  "recipeIngredient": ["1 kg Rindergulasch", "3 Zwiebel(n)", "2 EL Paprikapulver, edelsüß", "", "  Salz und Pfeffer"],
  "recipeInstructions": "Zwiebeln würfeln und anbraten.\nFleisch zugeben und kräftig anbraten.\n\nMit Wasser ablöschen und 90 Minuten schmoren."
}
</script></head><body></body></html>`

// graphLike entspricht vielen WordPress-Seiten: alles in "@graph",
// "@type" als Liste, Schritte als HowToSection/HowToStep, Bild als
// ImageObject mit relativer Adresse, Portionen als Zahl.
const graphLike = `<html><head><script type='application/ld+json' class="yoast">
{"@context":"https://schema.org","@graph":[
  {"@type":"WebPage","name":"Seite"},
  {"@type":["Recipe","NewsArticle"],"name":"Kürbissuppe",
   "image":[{"@type":"ImageObject","url":"/bilder/kuerbis.jpg"}],
   "recipeYield":[6,"6 Portionen"],
   "prepTime":"PT15M","cookTime":"PT25M",
   "recipeIngredient":["1 Hokkaido-Kürbis","800 ml Gemüsebrühe"],
   "recipeInstructions":[
     {"@type":"HowToSection","name":"Vorbereitung","itemListElement":[
       {"@type":"HowToStep","text":"Kürbis <b>würfeln</b>."}]},
     {"@type":"HowToSection","name":"Kochen","itemListElement":[
       {"@type":"HowToStep","text":"Mit Brühe aufkochen."},
       {"@type":"HowToStep","text":"P&uuml;rieren."}]}
   ]}
]}
</script></head></html>`

func TestExtractChefkochLike(t *testing.T) {
	page, _ := url.Parse("https://www.chefkoch.de/rezepte/123/Omas-Gulasch.html?utm_source=app#kommentare")
	r, image, err := Extract([]byte(chefkochLike), page)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}

	if r.Title != "Omas Gulasch" || r.Description != "Ein deftiges Gulasch & ganz einfach." {
		t.Errorf("Titel/Beschreibung = %q / %q", r.Title, r.Description)
	}
	if r.BaseServings != 4 {
		t.Errorf("Portionen = %d, erwartet 4", r.BaseServings)
	}
	if r.PrepMinutes == nil || *r.PrepMinutes != 30 || r.TotalMinutes == nil || *r.TotalMinutes != 120 {
		t.Errorf("Zeiten = %v / %v, erwartet 30 / 120", r.PrepMinutes, r.TotalMinutes)
	}
	if r.SourceURL != "https://www.chefkoch.de/rezepte/123/Omas-Gulasch.html" {
		t.Errorf("SourceURL = %q, erwartet ohne Tracking und Anker", r.SourceURL)
	}
	if image != "https://img.example.org/gulasch.jpg" {
		t.Errorf("Bild = %q", image)
	}
	// Die leere Zeile entfällt.
	if len(r.Ingredients) != 4 {
		t.Fatalf("%d Zutaten, erwartet 4: %+v", len(r.Ingredients), r.Ingredients)
	}
	if ing := r.Ingredients[2]; ing.Name != "Paprikapulver" || ing.Unit != "EL" || ing.Note != "edelsüß" {
		t.Errorf("Zutat 3 = %+v", ing)
	}
	if len(r.Steps) != 3 || r.Steps[2].Text != "Mit Wasser ablöschen und 90 Minuten schmoren." {
		t.Errorf("Schritte = %+v", r.Steps)
	}
}

func TestExtractGraphLike(t *testing.T) {
	page, _ := url.Parse("https://blog.example.org/kuerbissuppe/")
	r, image, err := Extract([]byte(graphLike), page)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if r.Title != "Kürbissuppe" || r.BaseServings != 6 {
		t.Errorf("Titel %q, Portionen %d", r.Title, r.BaseServings)
	}
	// Ohne totalTime: Vorbereitung + Kochen
	if r.TotalMinutes == nil || *r.TotalMinutes != 40 {
		t.Errorf("Gesamtzeit = %v, erwartet 40", r.TotalMinutes)
	}
	if image != "https://blog.example.org/bilder/kuerbis.jpg" {
		t.Errorf("Bild = %q, erwartet aufgelöste Adresse", image)
	}
	want := []string{"Kürbis würfeln.", "Mit Brühe aufkochen.", "Pürieren."}
	if len(r.Steps) != len(want) {
		t.Fatalf("Schritte = %+v", r.Steps)
	}
	for i, s := range want {
		if r.Steps[i].Text != s {
			t.Errorf("Schritt %d = %q, erwartet %q", i+1, r.Steps[i].Text, s)
		}
	}
}

func TestExtractWithoutRecipe(t *testing.T) {
	page, _ := url.Parse("https://example.org/")
	pages := []string{
		`<html><body>Kein Rezept</body></html>`,
		`<script type="application/ld+json">{kaputt</script>`,
		`<script type="application/ld+json">{"@type":"Article","name":"Nur ein Artikel"}</script>`,
	}
	for _, p := range pages {
		if _, _, err := Extract([]byte(p), page); !errors.Is(err, ErrNoRecipe) {
			t.Errorf("Fehler = %v, erwartet ErrNoRecipe für %q", err, p)
		}
	}
}

func TestDuration(t *testing.T) {
	tests := map[string]int{"PT20M": 20, "PT1H30M": 90, "P0DT0H45M": 45, "PT90M": 90, "PT1H": 60, "P1DT2H": 1560, "PT1H30M0S": 90}
	for in, want := range tests {
		if got := duration(in); got == nil || *got != want {
			t.Errorf("duration(%q) = %v, erwartet %d", in, got, want)
		}
	}
	for _, in := range []any{"PT0M", "30 Minuten", "", 30.0, nil} {
		if got := duration(in); got != nil {
			t.Errorf("duration(%v) = %d, erwartet nil", in, *got)
		}
	}
}

// referencesLike nutzt Verweise per "@id" wie Chefkoch: Bild und Autor
// stehen als eigene Objekte im @graph, der Titel trägt den Autornamen.
const referencesLike = `<script type="application/ld+json">{"@context":"https://schema.org","@graph":[
 {"@type":"ImageObject","@id":"https://x.example/r#primaryimage","url":"https://img.example/kuchen.jpg"},
 {"@type":"Person","@id":"https://x.example/u#author","name":"esther1107"},
 {"@type":"Recipe","@id":"https://x.example/r#recipe","name":"Schneller Apfelkuchen von esther1107",
  "image":{"@id":"https://x.example/r#primaryimage"},"author":{"@id":"https://x.example/u#author"},
  "recipeIngredient":["125 g Zucker"]}
]}</script>`

func TestExtractResolvesReferences(t *testing.T) {
	page, _ := url.Parse("https://x.example/r")
	r, image, err := Extract([]byte(referencesLike), page)
	if err != nil {
		t.Fatal(err)
	}
	if r.Title != "Schneller Apfelkuchen" {
		t.Errorf("Titel = %q, erwartet ohne Autorname", r.Title)
	}
	if image != "https://img.example/kuchen.jpg" {
		t.Errorf("Bild = %q", image)
	}
}

func TestExtractFallsBackToOGImage(t *testing.T) {
	page, _ := url.Parse("https://x.example/r")
	html := `<meta content="/og/bild.jpg" property="og:image">` +
		`<script type="application/ld+json">{"@type":"Recipe","name":"Kuchen von Oma"}</script>`
	r, image, err := Extract([]byte(html), page)
	if err != nil {
		t.Fatal(err)
	}
	if image != "https://x.example/og/bild.jpg" {
		t.Errorf("Bild = %q, erwartet og:image", image)
	}
	if r.Title != "Kuchen von Oma" {
		t.Errorf("Titel = %q, ohne Autorangabe darf nichts abgeschnitten werden", r.Title)
	}
}
