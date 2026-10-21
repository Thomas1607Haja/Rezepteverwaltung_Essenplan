package shopping

import (
	"math"
	"strconv"
	"strings"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/database"
)

// countUnits sind Einheiten, die man nur ganz kaufen kann (gefaltet, ohne
// Punkt). Eine leere Einheit zählt auch dazu: "2 Eier", "1 Zwiebel".
// Mengen in diesen Einheiten werden aufgerundet – niemand kauft 1,33 Eier.
var countUnits = map[string]bool{
	"": true, "stuck": true, "stk": true,
	"zehe": true, "zehen": true, "dose": true, "dosen": true,
	"packung": true, "packungen": true, "pck": true, "pkt": true, "packchen": true,
	"bund": true, "becher": true, "glas": true, "glaser": true,
	"kopf": true, "kopfe": true, "knolle": true, "knollen": true,
	"stange": true, "stangen": true, "flasche": true, "flaschen": true,
	"beutel": true, "tafel": true, "tafeln": true, "rolle": true, "rollen": true,
	"paar": true, "wurfel": true,
}

// isCountUnit meldet, ob unit eine Stück-Einheit ist.
func isCountUnit(unit string) bool {
	return countUnits[strings.TrimSuffix(database.Fold(strings.TrimSpace(unit)), ".")]
}

// roundForShopping rundet Stück-Einheiten auf ganze Zahlen auf. Kleine
// Rechenungenauigkeiten (2,0000001) zählen nicht als angefangenes Stück.
func roundForShopping(q float64, unit string) float64 {
	if !isCountUnit(unit) {
		return q
	}
	return math.Max(1, math.Ceil(q-0.05))
}

// fractions sind die beim Kochen üblichen Brüche, wie in der Kochansicht
// (web/src/lib/format.js).
var fractions = []struct {
	value  float64
	symbol string
}{
	{1.0 / 4, "¼"}, {1.0 / 3, "⅓"}, {1.0 / 2, "½"}, {2.0 / 3, "⅔"}, {3.0 / 4, "¾"},
}

// FormatQuantity schreibt eine Menge so, wie man sie in der Küche liest,
// genau wie formatQuantity in der Weboberfläche:
// 333.33 → "333", 12.46 → "12,5", 1.5 → "1 ½", 0.33 → "⅓", 2.2 → "2,2".
func FormatQuantity(q float64) string {
	switch {
	case q >= 100:
		return strconv.Itoa(int(math.Round(q)))
	case q >= 10:
		return formatDecimal(q, 1)
	}
	whole := math.Floor(q)
	part := q - whole
	if part < 0.04 {
		return strconv.Itoa(int(whole))
	}
	if part > 0.96 {
		return strconv.Itoa(int(whole) + 1)
	}
	for _, f := range fractions {
		if math.Abs(part-f.value) < 0.04 {
			if whole == 0 {
				return f.symbol
			}
			return strconv.Itoa(int(whole)) + " " + f.symbol
		}
	}
	return formatDecimal(q, 2)
}

// formatDecimal schreibt q mit höchstens digits Nachkommastellen und
// deutschem Komma; Nullen am Ende fallen weg (2,50 → 2,5).
func formatDecimal(q float64, digits int) string {
	// Erst kaufmännisch runden (1,125 → 1,13) wie die Weboberfläche;
	// FormatFloat allein würde zur geraden Ziffer runden (1,12).
	pow := math.Pow10(digits)
	s := strconv.FormatFloat(math.Round(q*pow)/pow, 'f', digits, 64)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	}
	return strings.Replace(s, ".", ",", 1)
}
