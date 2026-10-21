// Package shopping berechnet die Einkaufsliste für den Einkaufszeitraum
// aus dem Essensplan. Die Liste wird nicht gespeichert, sondern bei jedem
// Abruf neu berechnet.
package shopping

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/database"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/mealplan"
)

// dateLayout ist das Format eines Kalendertags, z. B. 2026-10-24.
const dateLayout = "2006-01-02"

// Preview ist die Einkaufsliste mit den Gerichten, aus denen sie entsteht.
type Preview struct {
	Period mealplan.Period `json:"period"`
	Today  string          `json:"today"` // aus Sicht des Servers
	// Entries sind alle Planeinträge im Zeitraum, auch solche, die nicht
	// in die Liste eingehen (siehe Entry.Included).
	Entries []Entry `json:"entries"`
	Items   []Item  `json:"items"` // alphabetisch
}

// Entry ist ein Planeintrag, wie ihn die Einkaufsseite anzeigt.
type Entry struct {
	ID       int64  `json:"id"`
	Date     string `json:"date"`
	Slot     string `json:"slot"`
	RecipeID *int64 `json:"recipe_id"` // nil = Eintrag ohne Rezept
	Title    string `json:"title"`     // Rezepttitel oder Notiz
	Servings int    `json:"servings"`
	Status   string `json:"status"`
	// Included: Die Zutaten dieses Eintrags stehen auf der Liste. Nein bei
	// Einträgen ohne Rezept, nicht mehr "planned" oder vor heute.
	Included bool `json:"included"`
}

// Item ist ein Artikel der Einkaufsliste: eine Zutat, zusammengefasst über
// alle Gerichte.
type Item struct {
	// Key bleibt für dieselbe Zutat gleich, auch wenn sie in Rezepten
	// unterschiedlich geschrieben ist ("Zwiebel", "Zwiebeln"). Damit lassen
	// sich Artikel abwählen und später mit Übertragenem vergleichen.
	Key     string   `json:"key"`
	Name    string   `json:"name"`
	Amounts []Amount `json:"amounts"` // je Einheit eine Summe; leer = ohne Mengenangabe
	// Spec ist die Mengenangabe als Text, z. B. "500 g + 2 EL". So geht sie
	// auch an Bring.
	Spec    string   `json:"spec"`
	Recipes []string `json:"recipes"` // Rezepte, aus denen der Artikel kommt
}

// Amount ist die Summe einer Zutat in einer Einheit.
type Amount struct {
	Quantity float64 `json:"quantity"`
	Unit     string  `json:"unit"`
}

// Store berechnet Einkaufslisten.
type Store struct {
	db   *sql.DB
	plan *mealplan.Store
}

// NewStore erzeugt einen Store. plan liefert heute und den Einkaufszeitraum.
func NewStore(db *sql.DB, plan *mealplan.Store) *Store {
	return &Store{db: db, plan: plan}
}

// Preview berechnet die Einkaufsliste für den aktuellen Einkaufszeitraum
// bzw. mit next = true für den folgenden (z. B. am Freitagabend für den
// Einkauf am Samstag). Tage vor heute zählen nicht mit: Was schon gegessen
// ist, muss man nicht mehr kaufen.
func (s *Store) Preview(ctx context.Context, next bool) (Preview, error) {
	period, err := s.plan.CurrentPeriod(ctx)
	if err != nil {
		return Preview{}, err
	}
	if next {
		period = shiftWeek(period)
	}
	today := s.plan.Today().Format(dateLayout)

	entries, err := s.loadEntries(ctx, period, today)
	if err != nil {
		return Preview{}, err
	}
	items, err := s.loadItems(ctx, period, max(today, period.From))
	if err != nil {
		return Preview{}, err
	}
	return Preview{Period: period, Today: today, Entries: entries, Items: items}, nil
}

// shiftWeek verschiebt einen Zeitraum um eine Woche nach hinten.
func shiftWeek(p mealplan.Period) mealplan.Period {
	shift := func(date string) string {
		t, _ := time.Parse(dateLayout, date)
		return t.AddDate(0, 0, 7).Format(dateLayout)
	}
	return mealplan.Period{From: shift(p.From), To: shift(p.To)}
}

// slotOrder sortiert Mittag- vor Abendessen (alphabetisch wäre es umgekehrt).
const slotOrder = `CASE e.slot WHEN 'lunch' THEN 0 ELSE 1 END`

func (s *Store) loadEntries(ctx context.Context, period mealplan.Period, today string) ([]Entry, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT e.id, e.date, e.slot, e.recipe_id, coalesce(r.title, e.note), e.servings, e.status
		FROM meal_plan_entries e LEFT JOIN recipes r ON r.id = e.recipe_id
		WHERE e.date BETWEEN ? AND ?
		ORDER BY e.date, `+slotOrder+`, e.position, e.id`,
		period.From, period.To)
	if err != nil {
		return nil, fmt.Errorf("einkaufsliste: gerichte lesen: %w", err)
	}
	defer rows.Close()

	entries := []Entry{}
	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.ID, &e.Date, &e.Slot, &e.RecipeID, &e.Title, &e.Servings, &e.Status); err != nil {
			return nil, fmt.Errorf("einkaufsliste: gerichte lesen: %w", err)
		}
		e.Included = e.RecipeID != nil && e.Status == mealplan.StatusPlanned && e.Date >= today
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("einkaufsliste: gerichte lesen: %w", err)
	}
	return entries, nil
}

// loadItems liest die einkaufsrelevanten Zutaten aller geplanten Gerichte
// von from bis period.To, rechnet sie auf die Portionen des Eintrags um und
// fasst gleiche Zutaten zusammen.
func (s *Store) loadItems(ctx context.Context, period mealplan.Period, from string) ([]Item, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT r.title, e.servings, r.base_servings, i.quantity, i.unit, i.name
		FROM meal_plan_entries e
		JOIN recipes r ON r.id = e.recipe_id
		JOIN recipe_ingredients i ON i.recipe_id = r.id AND i.shopping_relevant = 1
		WHERE e.date BETWEEN ? AND ? AND e.status = ?
		ORDER BY e.date, `+slotOrder+`, e.position, e.id, i.position`,
		from, period.To, mealplan.StatusPlanned)
	if err != nil {
		return nil, fmt.Errorf("einkaufsliste: zutaten lesen: %w", err)
	}
	defer rows.Close()

	list := newCollector()
	for rows.Next() {
		var title, unit, name string
		var servings, baseServings int
		var quantity *float64
		if err := rows.Scan(&title, &servings, &baseServings, &quantity, &unit, &name); err != nil {
			return nil, fmt.Errorf("einkaufsliste: zutaten lesen: %w", err)
		}
		if quantity != nil {
			scaled := *quantity * float64(servings) / float64(baseServings)
			quantity = &scaled
		}
		list.add(title, name, quantity, unit)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("einkaufsliste: zutaten lesen: %w", err)
	}
	return list.items(), nil
}

// collector fasst Zutaten zu Artikeln zusammen.
type collector struct {
	byKey map[string]*Item
	// unitIndex merkt sich je Artikel, an welcher Stelle von Amounts eine
	// Einheit steht (Schlüssel: Artikel-Key + "|" + Einheiten-Key).
	unitIndex map[string]int
}

func newCollector() *collector {
	return &collector{byKey: map[string]*Item{}, unitIndex: map[string]int{}}
}

func (c *collector) add(recipe, name string, quantity *float64, unit string) {
	display := cleanName(name)
	key := nameKey(display)
	if key == "" {
		return
	}
	it, ok := c.byKey[key]
	if !ok {
		it = &Item{Key: key, Name: display, Amounts: []Amount{}, Recipes: []string{}}
		c.byKey[key] = it
	}
	// Bei verschiedenen Schreibweisen gewinnt die längere – meist der
	// Plural, der auf einer Einkaufsliste natürlicher klingt ("Zwiebeln").
	if len(display) > len(it.Name) {
		it.Name = display
	}
	if !slices.Contains(it.Recipes, recipe) {
		it.Recipes = append(it.Recipes, recipe)
	}
	// Ohne Menge ("Petersilie", "1 Prise" ohne Zahl) bleibt es beim Namen.
	if quantity == nil {
		return
	}

	unit = strings.TrimSpace(unit)
	ukey := key + "|" + stem(database.Fold(strings.TrimSuffix(unit, ".")))
	if i, ok := c.unitIndex[ukey]; ok {
		it.Amounts[i].Quantity += *quantity
		if len(unit) > len(it.Amounts[i].Unit) {
			it.Amounts[i].Unit = unit // "3 Zehen" statt "3 Zehe"
		}
		return
	}
	c.unitIndex[ukey] = len(it.Amounts)
	it.Amounts = append(it.Amounts, Amount{Quantity: *quantity, Unit: unit})
}

// items liefert die fertige Liste: Stück-Einheiten aufgerundet, Mengen als
// Text, alphabetisch sortiert.
func (c *collector) items() []Item {
	result := make([]Item, 0, len(c.byKey))
	for _, it := range c.byKey {
		parts := make([]string, 0, len(it.Amounts))
		for i := range it.Amounts {
			a := &it.Amounts[i]
			a.Quantity = roundForShopping(a.Quantity, a.Unit)
			parts = append(parts, strings.TrimSpace(FormatQuantity(a.Quantity)+" "+a.Unit))
		}
		it.Spec = strings.Join(parts, " + ")
		result = append(result, *it)
	}
	// Die Reihenfolge einer Map ist in Go zufällig, daher hier sortieren.
	slices.SortFunc(result, func(a, b Item) int {
		return strings.Compare(database.Fold(a.Name), database.Fold(b.Name))
	})
	return result
}

// parenthesized trifft Klammerzusätze wie in "Zwiebel(n)" oder
// "Tomaten (aus der Dose)".
var parenthesized = regexp.MustCompile(`\s*\([^)]*\)`)

// cleanName entfernt Klammerzusätze und doppelte Leerzeichen.
func cleanName(name string) string {
	return strings.Join(strings.Fields(parenthesized.ReplaceAllString(name, "")), " ")
}

// nameKey bildet den Vergleichsschlüssel eines Zutatennamens: gefaltet und
// das letzte Wort ohne Pluralendung, damit "Zwiebel" und "Zwiebeln" oder
// "Ei" und "Eier" zusammenfallen.
func nameKey(name string) string {
	words := strings.Fields(database.Fold(name))
	if len(words) == 0 {
		return ""
	}
	words[len(words)-1] = stem(words[len(words)-1])
	return strings.Join(words, " ")
}

// pluralEndings in der Reihenfolge, in der sie geprüft werden.
var pluralEndings = []string{"en", "er", "n", "e", "s"}

// stem entfernt eine typische Pluralendung, wenn danach noch mindestens
// zwei Buchstaben übrig bleiben ("eier" → "ei", "tomaten" → "tomat",
// "tomate" → "tomat"). Das ist bewusst grob: Es geht nur darum, dass
// Singular und Plural denselben Schlüssel bekommen.
func stem(word string) string {
	for _, end := range pluralEndings {
		if rest, ok := strings.CutSuffix(word, end); ok && len([]rune(rest)) >= 2 {
			return rest
		}
	}
	return word
}
