// Package importer liest Rezepte aus Webseiten (schema.org-Daten) und
// zerlegt Zutatenzeilen in Menge, Einheit, Name und Hinweis.
package importer

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/database"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/recipes"
)

// unicodeFractions sind Brüche als einzelne Zeichen.
var unicodeFractions = map[rune]float64{'¼': 0.25, '½': 0.5, '¾': 0.75, '⅓': 1.0 / 3, '⅔': 2.0 / 3}

// quantityPattern erkennt eine Menge am Zeilenanfang, wahlweise als Spanne:
// "500", "1,5", "1.5", "1/2", "1 1/2", "½", "1½", "1 ½", "1.000", "2-3",
// "2 bis 3" (auch "2 to 3", "2 à 3", "2 a 3"). Direkt angehängte Einheiten
// wie "500g" bleiben übrig.
var quantityPattern = regexp.MustCompile(
	`^(\d+\s+\d+\s*/\s*\d+|\d+\s*/\s*\d+|\d{1,3}(?:\.\d{3})+(?:,\d+)?|\d+(?:[.,]\d+)?(?:\s*[¼½¾⅓⅔])?|[¼½¾⅓⅔])` +
		`(?:\s*(?:-|–|\s(?:bis|to|à|a)\s)\s*(\d+(?:[.,]\d+)?))?`)

// units ordnet bekannte Einheiten (klein geschrieben, ohne Punkt und
// Pluralendung) der gewünschten Schreibweise zu. Ein leerer Wert heißt:
// so übernehmen, wie es in der Zeile steht (z. B. "Dosen", "cups").
//
// Einheiten bleiben in der Sprache der Vorlage (docs/glossar.md): Aus
// "tablespoon" wird "tbsp", aus "Esslöffel" "EL" – übersetzt wird nicht,
// und Tassen oder Unzen werden nicht in Gramm umgerechnet.
var units = map[string]string{
	// Maße, in allen Sprachen gleich
	"g": "g", "gr": "g", "mg": "mg", "kg": "kg", "ml": "ml", "cl": "cl", "dl": "dl", "l": "l", "cm": "cm",
	// Deutsch
	"gramm": "g", "kilo": "kg", "kilogramm": "kg", "milliliter": "ml", "liter": "l",
	"el": "EL", "essl": "EL", "esslöffel": "EL",
	"tl": "TL", "teel": "TL", "teelöffel": "TL",
	"msp": "Msp.", "messerspitze": "Msp.",
	"pck": "Pck.", "pkt": "Pck.", "päckchen": "Pck.",
	"stk": "Stück", "st": "Stück",
	"tasse": "", "tassen": "", "prise": "", "prisen": "", "packung": "", "packungen": "",
	"dose": "", "dosen": "", "becher": "", "bund": "", "stück": "", "scheibe": "", "scheiben": "",
	"zehe": "", "zehen": "", "glas": "", "gläser": "", "tropfen": "", "zweig": "", "zweige": "",
	"blatt": "", "blätter": "", "handvoll": "", "stange": "", "stangen": "", "würfel": "",
	"spritzer": "", "schuss": "", "kopf": "", "köpfe": "", "knolle": "", "knollen": "",
	"beutel": "", "flasche": "", "flaschen": "", "tafel": "", "tafeln": "", "rolle": "", "rollen": "",
	"paar": "", "portion": "", "portionen": "",
	// Englisch
	"gram": "g", "grams": "g", "kilogram": "kg", "kilograms": "kg", "millilitre": "ml",
	"milliliters": "ml", "millilitres": "ml", "liters": "l", "litre": "l", "litres": "l",
	"tbsp": "tbsp", "tbs": "tbsp", "tablespoon": "tbsp", "tablespoons": "tbsp",
	"tsp": "tsp", "teaspoon": "tsp", "teaspoons": "tsp",
	"oz": "oz", "ounce": "oz", "ounces": "oz", "lb": "lb", "lbs": "lb", "pound": "lb", "pounds": "lb",
	"cup": "", "cups": "", "pint": "", "pints": "", "quart": "", "quarts": "",
	"pinch": "", "pinches": "", "dash": "", "dashes": "", "clove": "", "cloves": "", "can": "", "cans": "",
	"tin": "", "tins": "", "jar": "", "jars": "", "piece": "", "pieces": "", "slice": "", "slices": "",
	"bunch": "", "bunches": "", "handful": "", "handfuls": "", "sprig": "", "sprigs": "", "stick": "", "sticks": "",
	"stalk": "", "stalks": "", "head": "", "heads": "", "leaf": "", "leaves": "", "package": "", "packages": "",
	"pack": "", "packs": "", "packet": "", "packets": "", "bottle": "", "bottles": "", "drop": "", "drops": "",
	// Französisch
	"gramme": "g", "grammes": "g", "kilogramme": "kg", "kilos": "kg",
	"cs": "c. à s.", "cas": "c. à s.", "càs": "c. à s.", "cc": "c. à c.", "cac": "c. à c.", "càc": "c. à c.",
	"pincée": "", "pincées": "", "gousse": "", "gousses": "", "boîte": "", "boîtes": "", "sachet": "", "sachets": "",
	"tranche": "", "tranches": "", "botte": "", "bottes": "", "brin": "", "brins": "", "feuille": "", "feuilles": "",
	"pièce": "", "pièces": "", "verre": "", "verres": "", "bouquet": "", "bouquets": "", "noix": "", "pot": "", "pots": "",
	"tasses": "", "zeste": "",
	// Spanisch
	"gramo": "g", "gramos": "g", "kilogramo": "kg", "kilogramos": "kg", "mililitro": "ml", "mililitros": "ml",
	"litro": "l", "litros": "l",
	"cucharada": "cda", "cucharadas": "cda", "cda": "cda", "cdas": "cda",
	"cucharadita": "cdta", "cucharaditas": "cdta", "cdta": "cdta", "cdtas": "cdta", "cdita": "cdta", "cditas": "cdta",
	"pizca": "", "pizcas": "", "diente": "", "dientes": "", "lata": "", "latas": "", "pieza": "", "piezas": "",
	"rebanada": "", "rebanadas": "", "loncha": "", "lonchas": "", "rodaja": "", "rodajas": "", "manojo": "", "manojos": "",
	"ramita": "", "ramitas": "", "hoja": "", "hojas": "", "taza": "", "tazas": "", "vaso": "", "vasos": "",
	"sobre": "", "sobres": "", "chorrito": "", "chorro": "", "puñado": "", "puñados": "", "bote": "", "botes": "",
	// Italienisch
	"grammo": "g", "grammi": "g", "chilo": "kg", "chilogrammo": "kg", "chilogrammi": "kg",
	"millilitro": "ml", "millilitri": "ml", "litri": "l",
	"cucchiaio": "", "cucchiai": "", "cucchiaino": "", "cucchiaini": "",
	"pizzico": "", "pizzichi": "", "spicchio": "", "spicchi": "", "lattina": "", "lattine": "",
	"scatola": "", "scatole": "", "scatoletta": "", "pezzo": "", "pezzi": "", "fetta": "", "fette": "",
	"mazzo": "", "mazzi": "", "mazzetto": "", "rametto": "", "rametti": "", "foglia": "", "foglie": "",
	"tazza": "", "tazze": "", "bicchiere": "", "bicchieri": "", "bustina": "", "bustine": "", "vasetto": "", "vasetti": "",
	"ciuffo": "", "ciuffi": "", "manciata": "",
}

// phraseUnits sind Einheiten aus mehreren Wörtern. Schlüssel in der Form
// von database.Plain (klein, ohne Akzente), Wörter durch ein Leerzeichen
// getrennt; Wert ist die Schreibweise.
var phraseUnits = map[string]string{
	"c. a s.": "c. à s.", "c.a.s.": "c. à s.", "c. a. s.": "c. à s.",
	"cuillere a soupe": "c. à s.", "cuilleres a soupe": "c. à s.",
	"cuilleree a soupe": "c. à s.", "cuillerees a soupe": "c. à s.",
	"c. a c.": "c. à c.", "c.a.c.": "c. à c.", "c. a. c.": "c. à c.",
	"cuillere a cafe": "c. à c.", "cuilleres a cafe": "c. à c.",
	"cuilleree a cafe": "c. à c.", "cuillerees a cafe": "c. à c.",
	"fl oz": "fl oz", "fl. oz.": "fl oz", "fluid ounce": "fl oz", "fluid ounces": "fl oz",
	"cucharada sopera": "cda", "cucharadas soperas": "cda",
	"cucchiaio da tavola": "cucchiaio",
}

// connectors stehen zwischen Einheit und Zutat: "2 cups of flour",
// "200 g de farine", "1 c. à s. d'huile", "100 g di burro".
var connectors = []string{"of ", "de ", "di ", "d'", "d’"}

// pluralSuffix erkennt Chefkochs Pluralangaben wie "Zwiebel(n)" oder
// "Zehe/n" direkt hinter einem Wort.
var pluralSuffix = regexp.MustCompile(`(\p{L})(?:\((n|en|e|s|er|nen)\)|/(n|en|e|s|er|nen))`)

// vagueAmounts sind Mengenwörter ohne Zahl am Anfang; sie wandern in den Hinweis.
var vagueAmounts = []string{
	"nach belieben", "n. b.", "n.b.", "etwas", "evtl.", "eventuell", "einige", "ein paar", "reichlich", "wenig",
	"some", "a little", "a few", "optional",
	"un peu de", "un peu d'", "quelques", "un filet de", "un filet d'",
	"un poco de", "unas", "unos", "un chorrito de",
	"un po' di", "un po’ di", "qualche", "un filo di",
}

// vagueSuffixes stehen am Ende ("Salz nach Geschmack", "salt to taste",
// "sale q.b."); sie wandern ebenfalls in den Hinweis.
var vagueSuffixes = []string{
	"nach geschmack", "nach belieben", "to taste", "as needed", "(optional)", "optional",
	"au goût", "selon le goût", "selon votre goût",
	"al gusto", "a gusto",
	"q.b.", "q. b.", "qb", "quanto basta", "a piacere",
}

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

	// Einheit: erst Wortfolgen ("cuillère à soupe"), dann das nächste Wort.
	if unit, rest, ok := lookupPhraseUnit(s); ok {
		ing.Unit, s = unit, rest
	} else if word, rest, ok := strings.Cut(s+" ", " "); ok && word != "" {
		if unit, isUnit := lookupUnit(word, plural); isUnit {
			ing.Unit = unit
			s = strings.TrimSpace(rest)
		}
	}
	// "of", "de", "d'", "di" hinter der Einheit gehören nicht zum Namen.
	if ing.Unit != "" {
		lower := strings.ToLower(s)
		for _, c := range connectors {
			if strings.HasPrefix(lower, c) && len(s) > len(c) {
				s = strings.TrimSpace(s[len(c):])
				break
			}
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

	// "… nach Geschmack", "… to taste", "… q.b." am Ende in den Hinweis.
	lowerName := strings.ToLower(s)
	for _, vague := range vagueSuffixes {
		if strings.HasSuffix(lowerName, " "+vague) || strings.HasSuffix(lowerName, ", "+vague) {
			notes = append(notes, strings.Trim(s[len(s)-len(vague):], "()"))
			s = strings.TrimRight(strings.TrimSpace(s[:len(s)-len(vague)]), ",")
			break
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

// lookupPhraseUnit erkennt eine Einheit aus zwei bis vier Wörtern am Anfang
// von s und liefert ihre Schreibweise und den Rest der Zeile.
func lookupPhraseUnit(s string) (string, string, bool) {
	words := strings.Fields(s)
	for n := min(4, len(words)-1); n >= 1; n-- {
		phrase := strings.Join(words[:n], " ")
		// "c.à.s." ist ein Wort, "c. à s." sind drei – beide Formen prüfen.
		if unit, ok := phraseUnits[database.Plain(phrase)]; ok {
			return unit, strings.Join(words[n:], " "), true
		}
	}
	return "", "", false
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
