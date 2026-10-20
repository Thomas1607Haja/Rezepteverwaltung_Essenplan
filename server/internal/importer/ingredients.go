// Package importer liest Rezepte aus Webseiten (schema.org-Daten) und
// zerlegt Zutatenzeilen in Menge, Einheit, Name und Hinweis.
package importer

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/recipes"
)

// unicodeFractions sind Brüche als einzelne Zeichen.
var unicodeFractions = map[rune]float64{'¼': 0.25, '½': 0.5, '¾': 0.75, '⅓': 1.0 / 3, '⅔': 2.0 / 3}

// quantityPattern erkennt eine Menge am Zeilenanfang, wahlweise als Spanne:
// "500", "1,5", "1.5", "1/2", "1 1/2", "½", "1½", "1 ½", "1.000", "2-3",
// "2 bis 3". Direkt angehängte Einheiten wie "500g" bleiben übrig.
var quantityPattern = regexp.MustCompile(
	`^(\d+\s+\d+\s*/\s*\d+|\d+\s*/\s*\d+|\d{1,3}(?:\.\d{3})+(?:,\d+)?|\d+(?:[.,]\d+)?(?:\s*[¼½¾⅓⅔])?|[¼½¾⅓⅔])` +
		`(?:\s*(?:-|–|bis)\s*(\d+(?:[.,]\d+)?))?`)

// units ordnet bekannte Einheiten (klein geschrieben, ohne Punkt und
// Pluralendung) der gewünschten Schreibweise zu. Ein leerer Wert heißt:
// so übernehmen, wie es in der Zeile steht (z. B. "Dosen").
var units = map[string]string{
	"g": "g", "gr": "g", "gramm": "g",
	"kg": "kg", "kilo": "kg", "kilogramm": "kg",
	"mg": "mg", "ml": "ml", "milliliter": "ml", "cl": "cl", "dl": "dl",
	"l": "l", "liter": "l",
	"el": "EL", "essl": "EL", "esslöffel": "EL",
	"tl": "TL", "teel": "TL", "teelöffel": "TL",
	"msp": "Msp.", "messerspitze": "Msp.",
	"pck": "Pck.", "pkt": "Pck.", "päckchen": "Pck.",
	"stk": "Stück", "st": "Stück",
	"cm":    "cm",
	"tasse": "", "tassen": "", "prise": "", "prisen": "", "packung": "", "packungen": "",
	"dose": "", "dosen": "", "becher": "", "bund": "", "stück": "", "scheibe": "", "scheiben": "",
	"zehe": "", "zehen": "", "glas": "", "gläser": "", "tropfen": "", "zweig": "", "zweige": "",
	"blatt": "", "blätter": "", "handvoll": "", "stange": "", "stangen": "", "würfel": "",
	"spritzer": "", "schuss": "", "kopf": "", "köpfe": "", "knolle": "", "knollen": "",
	"beutel": "", "flasche": "", "flaschen": "", "tafel": "", "tafeln": "", "rolle": "", "rollen": "",
	"paar": "", "portion": "", "portionen": "",
	"cup": "", "cups": "", "tbsp": "EL", "tsp": "TL",
}

// pluralSuffix erkennt Chefkochs Pluralangaben wie "Zwiebel(n)" oder
// "Zehe/n" direkt hinter einem Wort.
var pluralSuffix = regexp.MustCompile(`(\p{L})(?:\((n|en|e|s|er|nen)\)|/(n|en|e|s|er|nen))`)

// vagueAmounts sind Mengenwörter ohne Zahl; sie wandern in den Hinweis.
var vagueAmounts = []string{"nach belieben", "n. b.", "n.b.", "etwas", "evtl.", "eventuell", "einige", "ein paar", "reichlich", "wenig"}

// thousands erkennt Zahlen mit Tausenderpunkt wie "1.000" oder "2.500,5".
var thousands = regexp.MustCompile(`^\d{1,3}(\.\d{3})+(,\d+)?$`)

// ParseIngredient zerlegt eine Zutatenzeile. Was sich nicht als Menge oder
// Einheit erkennen lässt, bleibt im Namen – es geht also nichts verloren.
// Neue Zutaten gelten als einkaufsrelevant (Vorratsliste folgt später).
func ParseIngredient(line string) recipes.Ingredient {
	s := cleanText(line)
	s = strings.TrimLeft(s, "-•*· ")
	ing := recipes.Ingredient{ShoppingRelevant: true}

	var quantity *float64
	var rangeNote string
	if m := quantityPattern.FindStringSubmatchIndex(s); m != nil {
		q := parseQuantity(s[m[2]:m[3]])
		if q != nil {
			quantity = q
			if m[4] >= 0 {
				rangeNote = strings.TrimSpace(s[m[2]:m[3]]) + "–" + s[m[4]:m[5]]
			}
			s = strings.TrimSpace(s[m[1]:])
		}
	}
	ing.Quantity = quantity
	plural := quantity == nil || *quantity > 1

	// Einheit: das nächste Wort, falls es eine bekannte Einheit ist.
	if word, rest, ok := strings.Cut(s+" ", " "); ok && word != "" {
		if unit, isUnit := lookupUnit(word, plural); isUnit {
			ing.Unit = unit
			s = strings.TrimSpace(rest)
		}
	}

	// Mengenwörter ohne Zahl ("etwas Petersilie") in den Hinweis schieben.
	var notes []string
	if rangeNote != "" {
		notes = append(notes, rangeNote)
	}
	if quantity == nil {
		lower := strings.ToLower(s)
		for _, vague := range vagueAmounts {
			if strings.HasPrefix(lower, vague+" ") {
				notes = append(notes, s[:len(vague)])
				s = strings.TrimSpace(s[len(vague):])
				break
			}
		}
	}

	// Nach einer Einheit steht der Name in der Mehrzahl, auch bei einer
	// Menge von 1: "1 kg Kartoffel(n)" -> "Kartoffeln".
	s = applyPlural(s, plural || ing.Unit != "")

	// Hinweis nach dem ersten Komma außerhalb von Klammern
	// ("Kasseler (oder Schnitzel, Schwein)" bleibt zusammen) …
	if i := topLevelComma(s); i >= 0 {
		notes = append(notes, strings.TrimSpace(s[i+1:]))
		s = strings.TrimSpace(s[:i])
	}
	// … oder in Klammern am Ende.
	if strings.HasSuffix(s, ")") {
		if open := matchingOpen(s); open > 0 {
			notes = append(notes, strings.TrimSpace(s[open+1:len(s)-1]))
			s = strings.TrimSpace(s[:open])
		}
	}

	ing.Name = s
	ing.Note = strings.Join(nonEmpty(notes), ", ")
	if ing.Name == "" && ing.Unit != "" {
		// z. B. "1 Prise": die Einheit ist dann eigentlich der Name
		ing.Name, ing.Unit = ing.Unit, ""
	}
	return ing
}

// topLevelComma liefert die Position des ersten Kommas, das nicht in
// Klammern steht, oder -1.
func topLevelComma(s string) int {
	depth := 0
	for i, r := range s {
		switch r {
		case '(':
			depth++
		case ')':
			depth = max(depth-1, 0)
		case ',':
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// matchingOpen liefert die Position der Klammer, die zur schließenden
// Klammer am Ende von s gehört – auch bei Verschachtelung wie
// "Kasseler (oder Schnitzel (Schwein))". -1, wenn es keine gibt.
func matchingOpen(s string) int {
	depth := 0
	for i := len(s) - 1; i >= 0; i-- {
		switch s[i] {
		case ')':
			depth++
		case '(':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// lookupUnit prüft, ob word eine Einheit ist, und liefert die Schreibweise.
func lookupUnit(word string, plural bool) (string, bool) {
	display := applyPlural(word, plural)
	key := strings.ToLower(strings.TrimSuffix(applyPlural(word, false), "."))
	canonical, ok := units[key]
	if !ok {
		return "", false
	}
	if canonical == "" {
		return strings.TrimSuffix(display, "."), true
	}
	return canonical, true
}

// applyPlural löst "Zwiebel(n)" je nach Menge zu "Zwiebeln" oder "Zwiebel" auf.
func applyPlural(s string, plural bool) string {
	return pluralSuffix.ReplaceAllStringFunc(s, func(m string) string {
		sub := pluralSuffix.FindStringSubmatch(m)
		if !plural {
			return sub[1]
		}
		return sub[1] + sub[2] + sub[3] // nur eine der beiden Gruppen ist gefüllt
	})
}

// parseQuantity liest eine Menge wie "1 1/2", "1,5", "½" oder "1.000".
func parseQuantity(text string) *float64 {
	s := strings.TrimSpace(text)
	value := 0.0

	// Bruchzeichen am Ende, z. B. "1½" oder "1 ½"
	if r := []rune(s); len(r) > 0 {
		if f, ok := unicodeFractions[r[len(r)-1]]; ok {
			value = f
			s = strings.TrimSpace(string(r[:len(r)-1]))
			if s == "" {
				return &value
			}
		}
	}

	if whole, frac, ok := strings.Cut(s, "/"); ok {
		// "1 1/2" oder "1/2"
		parts := strings.Fields(whole)
		num, err1 := strconv.Atoi(parts[len(parts)-1])
		den, err2 := strconv.Atoi(strings.TrimSpace(frac))
		if err1 != nil || err2 != nil || den == 0 {
			return nil
		}
		value += float64(num) / float64(den)
		if len(parts) == 2 {
			w, err := strconv.Atoi(parts[0])
			if err != nil {
				return nil
			}
			value += float64(w)
		}
		return &value
	}

	// "1.000" ist Tausend, "1.5" und "1,5" sind anderthalb.
	if thousands.MatchString(s) {
		s = strings.ReplaceAll(s, ".", "")
	}
	f, err := strconv.ParseFloat(strings.Replace(s, ",", ".", 1), 64)
	if err != nil {
		return nil
	}
	value += f
	return &value
}

func nonEmpty(list []string) []string {
	out := list[:0]
	for _, s := range list {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}
