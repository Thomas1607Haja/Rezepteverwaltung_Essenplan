package importer

import "testing"

func TestParseIngredient(t *testing.T) {
	tests := []struct {
		line     string
		quantity float64 // -1 = keine Menge
		unit     string
		name     string
		note     string
	}{
		{"500 g Hackfleisch, gemischt", 500, "g", "Hackfleisch", "gemischt"},
		{"500g Mehl", 500, "g", "Mehl", ""},
		{"2 Zwiebel(n)", 2, "", "Zwiebeln", ""},
		{"1 Zwiebel(n), fein gewürfelt", 1, "", "Zwiebel", "fein gewürfelt"},
		{"1 Zehe/n Knoblauch", 1, "Zehe", "Knoblauch", ""},
		{"3 Zehe/n Knoblauch", 3, "Zehen", "Knoblauch", ""},
		{"1 ½ EL Olivenöl", 1.5, "EL", "Olivenöl", ""},
		{"1½ TL Salz", 1.5, "TL", "Salz", ""},
		{"½ Zitrone", 0.5, "", "Zitrone", ""},
		{"1/2 TL Zimt", 0.5, "TL", "Zimt", ""},
		{"1 1/2 Tassen Reis", 1.5, "Tassen", "Reis", ""},
		{"1,5 l Milch", 1.5, "l", "Milch", ""},
		{"1.000 g Kartoffeln", 1000, "g", "Kartoffeln", ""},
		{"2-3 Möhren", 2, "", "Möhren", "2–3"},
		{"2 bis 3 Stangen Lauch", 2, "Stangen", "Lauch", "2–3"},
		{"200 ml Sahne (oder Milch)", 200, "ml", "Sahne", "oder Milch"},
		{"1 Pck. Vanillezucker", 1, "Pck.", "Vanillezucker", ""},
		{"2 EL Essig", 2, "EL", "Essig", ""},
		{"1 el Zucker", 1, "EL", "Zucker", ""},
		{"2 Dose/n Tomaten, stückig", 2, "Dosen", "Tomaten", "stückig"},
		{"1 Prise Salz", 1, "Prise", "Salz", ""},
		{"4 Ei(er)", 4, "", "Eier", ""},
		{"1 Ei(er)", 1, "", "Ei", ""},
		{"etwas Petersilie", -1, "", "Petersilie", "etwas"},
		{"n. B. Chiliflocken", -1, "", "Chiliflocken", "n. B."},
		{"Salz und Pfeffer", -1, "", "Salz und Pfeffer", ""},
		{"  Öl zum Braten ", -1, "", "Öl zum Braten", ""},
		{"- 100 g Butter", 100, "g", "Butter", ""},
		{"2 Eier (Größe M)", 2, "", "Eier", "Größe M"},
		{"1 Bund Schnittlauch", 1, "Bund", "Schnittlauch", ""},
		{"Tomaten &amp; Gurken", -1, "", "Tomaten & Gurken", ""},
		{"2 kg Kasseler (oder Schnitzelfleisch, Schwein)", 2, "kg", "Kasseler", "oder Schnitzelfleisch, Schwein"},
		{"1 kg Kartoffel(n), festkochend", 1, "kg", "Kartoffeln", "festkochend"},
		{"3 Paar Wiener Würstchen", 3, "Paar", "Wiener Würstchen", ""},
		{"2 kg Kasseler (oder Schnitzelfleisch (Schwein))", 2, "kg", "Kasseler", "oder Schnitzelfleisch (Schwein)"},
		{"n. B. Ketchup (zum Andicken)", -1, "", "Ketchup", "n. B., zum Andicken"},
		// Englisch
		{"2 cups all-purpose flour", 2, "cups", "all-purpose flour", ""},
		{"1 cup of milk", 1, "cup", "milk", ""},
		{"2 tablespoons olive oil", 2, "tbsp", "olive oil", ""},
		{"1/2 tsp ground cinnamon", 0.5, "tsp", "ground cinnamon", ""},
		{"8 oz cream cheese, softened", 8, "oz", "cream cheese", "softened"},
		{"1 lb ground beef", 1, "lb", "ground beef", ""},
		{"2 cloves garlic, minced", 2, "cloves", "garlic", "minced"},
		{"1 (14 oz) can diced tomatoes", 1, "", "(14 oz) can diced tomatoes", ""},
		{"Salt to taste", -1, "", "Salt", "to taste"},
		{"2 to 3 carrots", 2, "", "carrots", "2–3"},
		{"4 fl oz cream", 4, "fl oz", "cream", ""},
		{"1 pinch of salt", 1, "pinch", "salt", ""},
		// Französisch
		{"200 g de farine", 200, "g", "farine", ""},
		{"2 cuillères à soupe d'huile d'olive", 2, "c. à s.", "huile d'olive", ""},
		{"1 c. à c. de sel", 1, "c. à c.", "sel", ""},
		{"1 c.à.s. de sucre", 1, "c. à s.", "sucre", ""},
		{"3 gousses d'ail", 3, "gousses", "ail", ""},
		{"1 pincée de sel", 1, "pincée", "sel", ""},
		{"Poivre au goût", -1, "", "Poivre", "au goût"},
		{"2 à 3 carottes", 2, "", "carottes", "2–3"},
		// Spanisch
		{"200 g de harina", 200, "g", "harina", ""},
		{"2 cucharadas de aceite de oliva", 2, "cda", "aceite de oliva", ""},
		{"1 cucharadita de sal", 1, "cdta", "sal", ""},
		{"2 dientes de ajo", 2, "dientes", "ajo", ""},
		{"Sal al gusto", -1, "", "Sal", "al gusto"},
		{"1 pieza de jengibre", 1, "pieza", "jengibre", ""},
		// Italienisch
		{"200 g di farina", 200, "g", "farina", ""},
		{"2 cucchiai di olio extravergine", 2, "cucchiai", "olio extravergine", ""},
		{"1 cucchiaino di sale", 1, "cucchiaino", "sale", ""},
		{"2 spicchi d'aglio", 2, "spicchi", "aglio", ""},
		{"Sale q.b.", -1, "", "Sale", "q.b."},
		{"3 pezzi di zenzero", 3, "pezzi", "zenzero", ""},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			got := ParseIngredient(tt.line)
			if tt.quantity < 0 {
				if got.Quantity != nil {
					t.Errorf("Menge = %v, erwartet keine", *got.Quantity)
				}
			} else if got.Quantity == nil || *got.Quantity != tt.quantity {
				t.Errorf("Menge = %v, erwartet %v", got.Quantity, tt.quantity)
			}
			if got.Unit != tt.unit || got.Name != tt.name || got.Note != tt.note {
				t.Errorf("Einheit %q, Name %q, Hinweis %q; erwartet %q, %q, %q",
					got.Unit, got.Name, got.Note, tt.unit, tt.name, tt.note)
			}
			if !got.ShoppingRelevant {
				t.Error("neue Zutaten sollen einkaufsrelevant sein")
			}
		})
	}
}
