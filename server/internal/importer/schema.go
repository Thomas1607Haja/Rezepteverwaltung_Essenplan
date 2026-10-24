package importer

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/recipes"
)

// ErrNoRecipe: Die Seite enthält keine lesbaren schema.org-Rezeptdaten.
var ErrNoRecipe = errors.New("auf der seite wurden keine rezeptdaten gefunden")

// defaultServings gilt, wenn die Seite keine Portionenzahl nennt.
const defaultServings = 4

// ldJSON findet die Blöcke <script type="application/ld+json">…</script>.
// Ein vollständiger HTML-Parser wäre gründlicher, ist hier aber nicht nötig:
// Gesucht wird nur dieses eine, klar abgegrenzte Element.
var ldJSON = regexp.MustCompile(`(?is)<script[^>]*type\s*=\s*["']?application/ld\+json["']?[^>]*>(.*?)</script>`)

// Extract liest das Rezept aus einer Webseite. pageURL dient dazu,
// relative Bildadressen aufzulösen. Ergebnis ist ein Entwurf, der noch
// nicht gespeichert ist, und die Adresse des Rezeptbildes (oder leer).
func Extract(page []byte, pageURL *url.URL) (recipes.Recipe, string, error) {
	for _, m := range ldJSON.FindAllSubmatch(page, -1) {
		var data any
		if err := json.Unmarshal(m[1], &data); err != nil {
			continue // fehlerhafte Blöcke überspringen, vielleicht passt ein anderer
		}
		if node := findRecipe(data, 0); node != nil {
			ids := make(map[string]map[string]any)
			collectIDs(data, ids, 0)
			r, imageURL := toRecipe(node, ids, pageURL)
			if r.Title == "" {
				continue
			}
			// Manche Seiten nennen das Bild nur im Vorschaubild für soziale
			// Netze (og:image), nicht in den Rezeptdaten.
			if imageURL == "" {
				imageURL = ogImage(page, pageURL)
			}
			return r, imageURL, nil
		}
	}
	return recipes.Recipe{}, "", ErrNoRecipe
}

// collectIDs merkt sich alle Objekte mit "@id". Viele Seiten (z. B.
// Chefkoch) schreiben Bild und Autor nicht ins Rezept, sondern verweisen
// mit {"@id": "…#primaryimage"} auf ein Objekt an anderer Stelle.
func collectIDs(v any, ids map[string]map[string]any, depth int) {
	if depth > 8 {
		return
	}
	switch v := v.(type) {
	case map[string]any:
		if id, ok := v["@id"].(string); ok && len(v) > 1 {
			ids[id] = v
		}
		for _, child := range v {
			collectIDs(child, ids, depth+1)
		}
	case []any:
		for _, child := range v {
			collectIDs(child, ids, depth+1)
		}
	}
}

// resolve ersetzt einen reinen Verweis {"@id": "…"} durch das Objekt, auf
// das er zeigt. Alles andere bleibt, wie es ist.
func resolve(v any, ids map[string]map[string]any) any {
	if m, ok := v.(map[string]any); ok && len(m) == 1 {
		if id, ok := m["@id"].(string); ok {
			if target, found := ids[id]; found {
				return target
			}
		}
	}
	return v
}

// findRecipe durchsucht die JSON-Daten nach einem Objekt mit
// "@type": "Recipe". Es kann ganz oben stehen, in einer Liste, in
// "@graph" oder verschachtelt (z. B. "mainEntity").
func findRecipe(v any, depth int) map[string]any {
	if depth > 8 {
		return nil
	}
	switch v := v.(type) {
	case map[string]any:
		if hasType(v["@type"], "Recipe") {
			return v
		}
		for _, child := range v {
			if found := findRecipe(child, depth+1); found != nil {
				return found
			}
		}
	case []any:
		for _, child := range v {
			if found := findRecipe(child, depth+1); found != nil {
				return found
			}
		}
	}
	return nil
}

// hasType prüft "@type", das ein Text oder eine Liste von Texten sein kann.
func hasType(t any, want string) bool {
	switch t := t.(type) {
	case string:
		return t == want
	case []any:
		for _, x := range t {
			if s, ok := x.(string); ok && s == want {
				return true
			}
		}
	}
	return false
}

func toRecipe(node map[string]any, ids map[string]map[string]any, pageURL *url.URL) (recipes.Recipe, string) {
	r := recipes.Recipe{
		Title:        stripAuthor(text(node["name"]), authorName(resolve(node["author"], ids))),
		Description:  text(node["description"]),
		BaseServings: servings(node["recipeYield"]),
		SourceURL:    NormalizeURL(pageURL),
		Tags:         []recipes.Tag{},
	}

	prep := duration(node["prepTime"])
	cook := duration(node["cookTime"])
	total := duration(node["totalTime"])
	r.PrepMinutes = prep
	if total == nil && (prep != nil || cook != nil) {
		sum := deref(prep) + deref(cook)
		total = &sum
	}
	r.TotalMinutes = total

	lines := stringList(node["recipeIngredient"])
	if len(lines) == 0 {
		lines = stringList(node["ingredients"]) // ältere Schreibweise
	}
	r.Ingredients = []recipes.Ingredient{}
	for _, line := range lines {
		if ing := ParseIngredient(line); ing.Name != "" {
			r.Ingredients = append(r.Ingredients, ing)
		}
	}

	r.Steps = []recipes.Step{}
	for _, s := range instructions(node["recipeInstructions"]) {
		r.Steps = append(r.Steps, recipes.Step{Text: s})
	}

	return r, imageURL(node["image"], ids, pageURL)
}

// authorName liest den Namen aus "author": Text, Person-Objekt oder Liste.
func authorName(v any) string {
	switch v := v.(type) {
	case string:
		return cleanText(v)
	case map[string]any:
		return text(v["name"])
	case []any:
		if len(v) > 0 {
			return authorName(v[0])
		}
	}
	return ""
}

// stripAuthor entfernt einen angehängten Autorennamen wie bei Chefkoch
// ("Schneller Apfelkuchen von esther1107"). Nur wenn er genau dem Autor
// entspricht – "Kuchen von Oma" bleibt so stehen.
func stripAuthor(title, author string) string {
	if author == "" {
		return title
	}
	if trimmed, found := strings.CutSuffix(title, " von "+author); found && trimmed != "" {
		return trimmed
	}
	return title
}

// ogImage* finden <meta property="og:image" content="…"> in beiden
// Reihenfolgen der Attribute.
var (
	ogImageA = regexp.MustCompile(`(?is)<meta[^>]+property\s*=\s*["']og:image["'][^>]+content\s*=\s*["']([^"']+)["']`)
	ogImageB = regexp.MustCompile(`(?is)<meta[^>]+content\s*=\s*["']([^"']+)["'][^>]+property\s*=\s*["']og:image["']`)
)

// ogImage liefert die Adresse aus og:image oder leer.
func ogImage(page []byte, base *url.URL) string {
	for _, re := range []*regexp.Regexp{ogImageA, ogImageB} {
		if m := re.FindSubmatch(page); m != nil {
			return imageURL(html.UnescapeString(string(m[1])), nil, base)
		}
	}
	return ""
}

// instructions macht aus den verschiedenen Formen der Zubereitung eine
// Liste von Schritten: ein langer Text, eine Liste von Texten, HowToStep-
// Objekte oder HowToSection-Abschnitte mit eigenen Schritten.
func instructions(v any) []string {
	var steps []string
	switch v := v.(type) {
	case string:
		// Ein einziger Text: an Zeilenumbrüchen bzw. <br>/<p> aufteilen.
		for _, part := range lineBreaks.Split(v, -1) {
			if t := text(part); t != "" {
				steps = append(steps, t)
			}
		}
	case []any:
		for _, item := range v {
			steps = append(steps, instructions(item)...)
		}
	case map[string]any:
		if hasType(v["@type"], "HowToSection") {
			steps = append(steps, instructions(v["itemListElement"])...)
		} else if t := text(v["text"]); t != "" {
			steps = append(steps, t)
		} else if t := text(v["name"]); t != "" {
			steps = append(steps, t)
		}
	}
	return steps
}

var lineBreaks = regexp.MustCompile(`(?i)\r?\n|<br\s*/?>|</p>`)

// servings liest die Portionen aus "4", 4, "4 Portionen" oder ["4", "4 servings"].
func servings(v any) int {
	switch v := v.(type) {
	case float64:
		if v >= 1 {
			return int(v)
		}
	case string:
		if m := firstNumber.FindString(v); m != "" {
			if n, err := strconv.Atoi(m); err == nil && n >= 1 && n <= 100 {
				return n
			}
		}
	case []any:
		for _, x := range v {
			if n := servings(x); n != defaultServings {
				return n
			}
		}
	}
	return defaultServings
}

var firstNumber = regexp.MustCompile(`\d+`)

// isoDuration erkennt ISO-8601-Dauern wie "PT1H30M", "P0DT0H20M" oder "PT90M".
var isoDuration = regexp.MustCompile(`(?i)^P(?:(\d+)D)?(?:T(?:(\d+)H)?(?:(\d+)M)?(?:(\d+(?:\.\d+)?)S)?)?$`)

// duration liefert die Minuten einer ISO-8601-Dauer oder nil.
func duration(v any) *int {
	s, ok := v.(string)
	if !ok {
		return nil
	}
	m := isoDuration.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return nil
	}
	days, _ := strconv.Atoi(m[1])
	hours, _ := strconv.Atoi(m[2])
	minutes, _ := strconv.Atoi(m[3])
	total := days*24*60 + hours*60 + minutes
	if total == 0 {
		return nil // "PT0M" heißt bei vielen Seiten "keine Angabe"
	}
	return &total
}

// imageURL nimmt die erste Bildadresse aus "image": Text, Liste oder
// ImageObject mit "url" bzw. "contentUrl", auch als Verweis per "@id".
// Relative Adressen werden aufgelöst.
func imageURL(v any, ids map[string]map[string]any, base *url.URL) string {
	var raw string
	switch v := resolve(v, ids).(type) {
	case string:
		raw = v
	case []any:
		for _, x := range v {
			if u := imageURL(x, ids, base); u != "" {
				return u
			}
		}
	case map[string]any:
		raw, _ = v["url"].(string)
		if raw == "" {
			raw, _ = v["contentUrl"].(string)
		}
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := base.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return ""
	}
	return u.String()
}

// stringList liefert einen Text oder eine Liste von Texten als Liste.
func stringList(v any) []string {
	switch v := v.(type) {
	case string:
		return []string{v}
	case []any:
		var out []string
		for _, x := range v {
			if s, ok := x.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

var htmlTags = regexp.MustCompile(`<[^>]*>`)

// spaceBeforePunct findet Leerraum vor Satzzeichen, der beim Entfernen
// von Tags entsteht ("<b>würfeln</b>." -> "würfeln .").
var spaceBeforePunct = regexp.MustCompile(`\s+([.,;:!?])`)

// text liest einen Textwert und bereinigt ihn (siehe cleanText).
func text(v any) string {
	s, _ := v.(string)
	return cleanText(s)
}

// cleanText entfernt HTML-Tags, löst Entities wie &amp; auf (manche Seiten
// kodieren doppelt) und fasst Leerraum zusammen.
func cleanText(s string) string {
	// Tags durch Leerzeichen ersetzen, damit "Zeile<br>Zeile" nicht
	// zusammenklebt; danach überflüssigen Leerraum wieder entfernen.
	s = htmlTags.ReplaceAllString(s, " ")
	s = html.UnescapeString(html.UnescapeString(s))
	s = strings.Join(strings.Fields(s), " ")
	return strings.TrimSpace(spaceBeforePunct.ReplaceAllString(s, "$1"))
}

func deref(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

// trackingParams sind Zusätze, die nur der Auswertung dienen. Ohne sie
// wird dieselbe Seite als Dublette erkannt, egal woher der Link kommt.
var trackingParams = []string{"utm_", "fbclid", "gclid", "mc_", "ref", "portionen"}

// NormalizeURL vereinheitlicht eine Adresse für die Dublettenprüfung:
// ohne #Anker, ohne Tracking-Parameter, Host klein geschrieben.
func NormalizeURL(u *url.URL) string {
	n := *u // Kopie, das Original bleibt unverändert
	n.Fragment = ""
	n.Host = strings.ToLower(n.Host)
	q := n.Query()
	for key := range q {
		lower := strings.ToLower(key)
		for _, prefix := range trackingParams {
			if strings.HasPrefix(lower, prefix) {
				q.Del(key)
			}
		}
	}
	n.RawQuery = q.Encode()
	return n.String()
}

// ParseURL prüft eine vom Nutzer eingegebene Adresse.
func ParseURL(raw string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("ungültige adresse: %q (erwartet http:// oder https://)", raw)
	}
	return u, nil
}

// PageImage liefert die Adresse des Rezeptbildes einer Seite: aus den
// Rezeptdaten, sonst og:image; leer, wenn die Seite keins nennt. Gedacht
// für den Datei-Import, der von der Quellseite nur das Bild übernimmt.
func PageImage(page []byte, pageURL *url.URL) string {
	if _, imageURL, err := Extract(page, pageURL); err == nil && imageURL != "" {
		return imageURL
	}
	return ogImage(page, pageURL)
}

// MaxPageBytes begrenzt die Größe einer Rezeptseite auch für andere Pakete.
const MaxPageBytes = maxPageBytes
