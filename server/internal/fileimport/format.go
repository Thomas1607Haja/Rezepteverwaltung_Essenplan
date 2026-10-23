// Package fileimport liest Rezepte aus Importdateien (ZIP mit recipes.json
// und Bildern oder eine einzelne .json-Datei), zeigt sie zuerst als
// Vorschau und speichert sie erst nach Bestätigung. Format und Schema:
// docs/importformat.md. Erzeugt werden die Dateien z. B. von der
// Claude-Skill in skills/rezeptplaner-import/.
package fileimport

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/recipes"
)

// Kennung und Versionen des Formats, siehe docs/importformat.schema.json.
// Version 2 ergänzt image_kind; Dateien der Version 1 bleiben gültig.
const (
	formatID         = "rezeptplaner-import"
	formatVersionMin = 1
	formatVersion    = 2
)

// document ist der Inhalt von recipes.json. Die Rezepte bleiben zunächst
// rohes JSON, damit jedes einzeln geprüft wird: Ein Fehler in Rezept 3
// soll die Rezepte 1 und 2 nicht verhindern.
type document struct {
	Format    string            `json:"format"`
	Version   int               `json:"version"`
	Generator string            `json:"generator"`
	Recipes   []json.RawMessage `json:"recipes"`
}

// recipeJSON ist ein Rezept im Importformat. Zeiger unterscheiden
// "fehlt" von 0 bzw. false.
type recipeJSON struct {
	Title        string           `json:"title"`
	Description  string           `json:"description"`
	BaseServings *int             `json:"base_servings"`
	PrepMinutes  *int             `json:"prep_minutes"`
	TotalMinutes *int             `json:"total_minutes"`
	Source       *sourceJSON      `json:"source"`
	Notes        string           `json:"notes"`
	Categories   []string         `json:"categories"`
	Image        *string          `json:"image"`
	ImageKind    string           `json:"image_kind"` // ab Version 2: "photo" oder "illustration"
	Ingredients  []ingredientJSON `json:"ingredients"`
	Steps        []string         `json:"steps"`
}

type sourceJSON struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type ingredientJSON struct {
	Group            string   `json:"group"`
	Quantity         *float64 `json:"quantity"`
	Unit             string   `json:"unit"`
	Name             string   `json:"name"`
	Note             string   `json:"note"`
	ShoppingRelevant *bool    `json:"shopping_relevant"`
}

// imagePath ist der erlaubte Bildpfad in der ZIP-Datei. Er kann weder
// ".." noch einen absoluten Pfad enthalten.
var imagePath = regexp.MustCompile(`^images/[A-Za-z0-9][A-Za-z0-9._-]{0,99}\.(jpg|jpeg|png|webp)$`)

// Grenzen wie im Schema.
const (
	maxRecipesPerFile = 100
	maxIngredients    = 150
	maxSteps          = 100
	maxCategories     = 30
	maxMinutes        = 10080 // eine Woche
)

// Meldungen dieses Pakets erscheinen als Sätze in der Vorschau und
// beginnen daher – anders als sonst bei Go-Fehlern üblich – groß.

// errNotImportFile: Die Datei ist kein Rezeptplaner-Import.
var errNotImportFile = errors.New("Keine Rezeptplaner-Importdatei (Formatkennung fehlt)")

// parseDocument liest recipes.json. Ein Fehler betrifft die ganze Datei.
func parseDocument(data []byte) (document, error) {
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf")) // BOM mancher Editoren
	if !utf8.Valid(data) {
		return document{}, errors.New("Die Datei ist nicht UTF-8-kodiert")
	}
	// Erst nur die Kennung lesen: Bei fremden JSON-Dateien soll die Meldung
	// "kein Rezeptplaner-Import" lauten, nicht "unbekanntes Feld".
	var head struct {
		Format  string `json:"format"`
		Version int    `json:"version"`
	}
	if err := json.Unmarshal(data, &head); err != nil {
		return document{}, fmt.Errorf("Ungültiges JSON: %s", jsonMessage(err))
	}
	if head.Format != formatID {
		return document{}, errNotImportFile
	}
	if head.Version < formatVersionMin || head.Version > formatVersion {
		return document{}, fmt.Errorf("Formatversion %d wird nicht unterstützt (erwartet %d bis %d) – ist der Rezeptplaner aktuell?",
			head.Version, formatVersionMin, formatVersion)
	}

	var doc document
	if err := decodeStrict(data, &doc); err != nil {
		return document{}, fmt.Errorf("Ungültiges JSON: %s", jsonMessage(err))
	}
	if len(doc.Recipes) == 0 {
		return document{}, errors.New("Die Datei enthält keine Rezepte")
	}
	if len(doc.Recipes) > maxRecipesPerFile {
		return document{}, fmt.Errorf("Höchstens %d Rezepte je Datei", maxRecipesPerFile)
	}
	return doc, nil
}

// decodeStrict liest JSON und lehnt unbekannte Felder ab: Ein Tippfehler
// wie "ingredient" statt "ingredients" soll auffallen, statt dass die
// Zutaten stillschweigend fehlen.
func decodeStrict(data []byte, dst any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("nach dem Objekt folgen weitere Daten")
	}
	return nil
}

// jsonMessage übersetzt die häufigsten Fehler von encoding/json.
func jsonMessage(err error) string {
	var typeErr *json.UnmarshalTypeError
	var syntaxErr *json.SyntaxError
	switch {
	case errors.As(err, &typeErr):
		field := typeErr.Field
		if field == "" {
			field = "(oberste Ebene)"
		}
		return fmt.Sprintf("Feld „%s“ hat den falschen Typ (%s statt %s)", field, typeErr.Value, typeErr.Type)
	case errors.As(err, &syntaxErr):
		return fmt.Sprintf("Syntaxfehler an Position %d", syntaxErr.Offset)
	}
	msg := err.Error()
	if name, ok := strings.CutPrefix(msg, "json: unknown field "); ok {
		return "unbekanntes Feld „" + strings.Trim(name, `"`) + "“"
	}
	return strings.TrimPrefix(msg, "json: ")
}

// parsedRecipe ist ein Rezept aus der Datei, umgewandelt ins Datenmodell.
// Kategorien sind hier noch Namen; sie werden erst mit der Datenbank
// aufgelöst (resolve).
type parsedRecipe struct {
	recipe     recipes.Recipe
	categories []string
	image      string // Pfad in der ZIP-Datei, leer = kein Bild
	errors     []string
	warnings   []string
}

// parseRecipe prüft ein Rezept und wandelt es um. Fehler verhindern den
// Import dieses Rezepts; Hinweise nicht.
func parseRecipe(raw json.RawMessage) parsedRecipe {
	var in recipeJSON
	var p parsedRecipe
	if err := decodeStrict(raw, &in); err != nil {
		p.errors = append(p.errors, jsonMessage(err))
		// Den Titel trotzdem versuchen, damit die Vorschau das Rezept
		// benennen kann.
		var t struct {
			Title string `json:"title"`
		}
		_ = json.Unmarshal(raw, &t)
		p.recipe.Title = strings.TrimSpace(t.Title)
		return p
	}
	fail := func(format string, args ...any) { p.errors = append(p.errors, fmt.Sprintf(format, args...)) }
	warn := func(format string, args ...any) { p.warnings = append(p.warnings, fmt.Sprintf(format, args...)) }

	r := &p.recipe
	r.Title = strings.TrimSpace(in.Title)
	r.Description = strings.TrimSpace(in.Description)
	r.Notes = strings.TrimSpace(in.Notes)
	r.PrepMinutes, r.TotalMinutes = in.PrepMinutes, in.TotalMinutes

	switch {
	case r.Title == "":
		fail("Titel fehlt")
	case utf8.RuneCountInString(r.Title) > 200:
		fail("Titel ist länger als 200 Zeichen")
	}
	if utf8.RuneCountInString(r.Description) > 2000 {
		fail("Beschreibung ist länger als 2000 Zeichen")
	}
	if utf8.RuneCountInString(r.Notes) > 5000 {
		fail("Notizen sind länger als 5000 Zeichen")
	}
	if in.BaseServings == nil || *in.BaseServings < 1 || *in.BaseServings > 100 {
		fail("Grundportionen fehlen oder liegen nicht zwischen 1 und 100")
	} else {
		r.BaseServings = *in.BaseServings
	}
	for _, m := range []struct {
		name  string
		value *int
	}{{"Vorbereitungszeit", in.PrepMinutes}, {"Gesamtzeit", in.TotalMinutes}} {
		if m.value != nil && (*m.value < 0 || *m.value > maxMinutes) {
			fail("%s muss zwischen 0 und %d Minuten liegen", m.name, maxMinutes)
		}
	}

	if in.Source != nil {
		r.SourceName = strings.TrimSpace(in.Source.Name)
		if utf8.RuneCountInString(r.SourceName) > 200 {
			fail("Quelle ist länger als 200 Zeichen")
		}
		if u := strings.TrimSpace(in.Source.URL); u != "" {
			if parsed, err := url.Parse(u); err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
				warn("Quell-Adresse „%s“ ist keine Webadresse und wird weggelassen", u)
			} else {
				r.SourceURL = u
			}
		}
	}

	if len(in.Categories) > maxCategories {
		fail("Höchstens %d Kategorien", maxCategories)
	}
	for _, c := range in.Categories {
		if c = strings.TrimSpace(c); c != "" {
			p.categories = append(p.categories, c)
		}
	}

	if in.Image != nil && *in.Image != "" {
		if imagePath.MatchString(*in.Image) {
			p.image = *in.Image
		} else {
			warn("Bildpfad „%s“ ist ungültig (erwartet images/name.jpg), das Bild wird weggelassen", *in.Image)
		}
	}
	if p.image != "" {
		switch in.ImageKind {
		case "", recipes.ImagePhoto:
			r.ImageKind = recipes.ImagePhoto
		case recipes.ImageIllustration:
			r.ImageKind = recipes.ImageIllustration
		default:
			warn("Bildart „%s“ ist unbekannt (erwartet photo oder illustration), das Bild gilt als Foto", in.ImageKind)
			r.ImageKind = recipes.ImagePhoto
		}
	}

	if len(in.Ingredients) > maxIngredients {
		fail("Höchstens %d Zutaten", maxIngredients)
	}
	r.Ingredients = []recipes.Ingredient{}
	for i, ing := range in.Ingredients {
		n := i + 1
		out := recipes.Ingredient{
			GroupName:        strings.TrimSpace(ing.Group),
			Quantity:         ing.Quantity,
			Unit:             strings.TrimSpace(ing.Unit),
			Name:             strings.TrimSpace(ing.Name),
			Note:             strings.TrimSpace(ing.Note),
			ShoppingRelevant: ing.ShoppingRelevant == nil || *ing.ShoppingRelevant,
		}
		if out.Name == "" {
			fail("Zutat %d: Name fehlt", n)
		}
		if out.Quantity != nil && *out.Quantity < 0 {
			fail("Zutat %d (%s): Menge darf nicht negativ sein", n, out.Name)
		}
		if utf8.RuneCountInString(out.Name) > 200 || utf8.RuneCountInString(out.Note) > 200 ||
			utf8.RuneCountInString(out.Unit) > 30 || utf8.RuneCountInString(out.GroupName) > 100 {
			fail("Zutat %d (%s): ein Feld ist zu lang", n, out.Name)
		}
		r.Ingredients = append(r.Ingredients, out)
	}
	if len(r.Ingredients) == 0 {
		warn("Keine Zutaten")
	}

	if len(in.Steps) > maxSteps {
		fail("Höchstens %d Schritte", maxSteps)
	}
	r.Steps = []recipes.Step{}
	for i, text := range in.Steps {
		text = strings.TrimSpace(text)
		switch {
		case text == "":
			fail("Schritt %d ist leer", i+1)
		case utf8.RuneCountInString(text) > 5000:
			fail("Schritt %d ist länger als 5000 Zeichen", i+1)
		}
		r.Steps = append(r.Steps, recipes.Step{Text: text})
	}
	if len(r.Steps) == 0 {
		warn("Keine Zubereitungsschritte")
	}
	return p
}
