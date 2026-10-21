package shopping

import (
	"context"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/database"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/mealplan"
)

// logged ist eine Zeile aus transferred_items.
type logged struct {
	Key      string
	Name     string
	Quantity *float64
	Unit     string
	Sent     bool
}

func (s *Store) loadLogged(ctx context.Context, period mealplan.Period) ([]logged, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT item_key, ingredient_name, quantity, unit, sent
		FROM transferred_items WHERE period_from = ? ORDER BY id`, period.From)
	if err != nil {
		return nil, fmt.Errorf("übertragene artikel lesen: %w", err)
	}
	defer rows.Close()
	var result []logged
	for rows.Next() {
		var l logged
		if err := rows.Scan(&l.Key, &l.Name, &l.Quantity, &l.Unit, &l.Sent); err != nil {
			return nil, fmt.Errorf("übertragene artikel lesen: %w", err)
		}
		result = append(result, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("übertragene artikel lesen: %w", err)
	}
	return result, nil
}

// differences vergleicht, was die schon übertragenen Gerichte des Zeitraums
// jetzt brauchen, mit dem, was übertragen (oder abgewählt) wurde.
//
// missing: Das fehlt noch, z. B. weil Portionen erhöht oder ein Rezept
// ersetzt wurde oder ein Artikel beim Senden nicht ankam.
// surplus: Das wurde gesendet, wird aber nicht mehr gebraucht. Es wird nur
// angezeigt und nicht aus Bring gelöscht – vielleicht ist es schon gekauft.
//
// Anders als bei der Einkaufsliste zählen hier auch Tage vor heute: Was
// gegessen ist, wurde ja gebraucht und ist nicht "zu viel".
func (s *Store) differences(ctx context.Context, period mealplan.Period) (missing, surplus []Item, err error) {
	needs, err := s.loadNeeds(ctx, period.From, period.To, mealplan.StatusTransferred)
	if err != nil {
		return nil, nil, err
	}
	logs, err := s.loadLogged(ctx, period)
	if err != nil {
		return nil, nil, err
	}

	// Je Artikel und Einheit summieren; amount merkt sich dazu, ob es
	// Einträge ohne Menge gab.
	type amount struct {
		qty, sent float64
		unit      string
		sentCount int // gesendete Einträge ohne Menge
	}
	type side struct {
		name    string
		recipes []string
		units   map[string]*amount
	}
	add := func(m map[string]*side, key, name, recipe string, q *float64, unit string, sent bool) {
		sd, ok := m[key]
		if !ok {
			sd = &side{name: name, recipes: []string{}, units: map[string]*amount{}}
			m[key] = sd
		}
		if len(name) > len(sd.name) {
			sd.name = name
		}
		if recipe != "" && !slices.Contains(sd.recipes, recipe) {
			sd.recipes = append(sd.recipes, recipe)
		}
		ukey := "" // Einträge ohne Menge zählen unter "keiner Einheit"
		if q != nil {
			ukey = "=" + unitKey(unit)
		}
		a, ok := sd.units[ukey]
		if !ok {
			a = &amount{unit: strings.TrimSpace(unit)}
			sd.units[ukey] = a
		}
		if q == nil {
			if sent {
				a.sentCount++
			}
			return
		}
		a.qty += *q
		if sent {
			a.sent += *q
		}
		if len(strings.TrimSpace(unit)) > len(a.unit) {
			a.unit = strings.TrimSpace(unit)
		}
	}

	need := map[string]*side{}
	for _, n := range needs {
		add(need, n.key(), cleanName(n.Name), n.Recipe, n.Quantity, n.Unit, true)
	}
	have := map[string]*side{}
	for _, l := range logs {
		add(have, l.Key, l.Name, "", l.Quantity, l.Unit, l.Sent)
	}

	// rounded vergleicht Stück-Einheiten so, wie sie gesendet wurden.
	rounded := func(q float64, unit string) float64 {
		if q <= 0 {
			return 0
		}
		return roundForShopping(q, unit)
	}
	// Kleine Rechenreste (z. B. 0,0001 g) sind keine Differenz.
	const epsilon = 0.01

	for key, n := range need {
		h := have[key]
		it := Item{Key: key, Name: n.name, Amounts: []Amount{}, Recipes: n.recipes}
		// Ohne Menge fehlt ein Artikel nur, wenn er gar nicht übertragen wurde.
		nameOnly := false
		for ukey, a := range n.units {
			if ukey == "" {
				nameOnly = nameOnly || h == nil
				continue
			}
			var got float64
			if h != nil && h.units[ukey] != nil {
				got = h.units[ukey].qty
			}
			if d := rounded(a.qty, a.unit) - rounded(got, a.unit); d > epsilon {
				it.Amounts = append(it.Amounts, Amount{Quantity: d, Unit: a.unit})
			}
		}
		if len(it.Amounts) > 0 || nameOnly {
			missing = append(missing, it)
		}
	}

	for key, h := range have {
		n := need[key]
		it := Item{Key: key, Name: h.name, Amounts: []Amount{}, Recipes: []string{}}
		if n != nil {
			it.Name = n.name
		}
		nameOnly := false
		for ukey, a := range h.units {
			if ukey == "" {
				nameOnly = nameOnly || (n == nil && a.sentCount > 0)
				continue
			}
			var needed float64
			if n != nil && n.units[ukey] != nil {
				needed = n.units[ukey].qty
			}
			// Zu viel ist nur, was wirklich gesendet wurde; Abgewähltes
			// wurde ja nicht gekauft.
			excess := rounded(a.qty, a.unit) - rounded(needed, a.unit)
			if e := math.Min(excess, rounded(a.sent, a.unit)); e > epsilon {
				it.Amounts = append(it.Amounts, Amount{Quantity: e, Unit: a.unit})
			}
		}
		if len(it.Amounts) > 0 || nameOnly {
			surplus = append(surplus, it)
		}
	}
	return finish(missing), finish(surplus), nil
}

// finish schreibt die Mengenangaben und sortiert wie die Einkaufsliste.
// Die Mengen sind hier schon gerundet.
func finish(items []Item) []Item {
	if items == nil {
		return []Item{}
	}
	for i := range items {
		it := &items[i]
		// Feste Reihenfolge der Einheiten, da sie aus einer Map stammen.
		slices.SortFunc(it.Amounts, func(a, b Amount) int { return strings.Compare(a.Unit, b.Unit) })
		parts := make([]string, 0, len(it.Amounts))
		for _, a := range it.Amounts {
			parts = append(parts, strings.TrimSpace(FormatQuantity(a.Quantity)+" "+a.Unit))
		}
		it.Spec = strings.Join(parts, " + ")
	}
	slices.SortFunc(items, func(a, b Item) int {
		return strings.Compare(database.Fold(a.Name), database.Fold(b.Name))
	})
	return items
}
