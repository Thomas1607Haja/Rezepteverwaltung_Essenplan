package shopping

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/database"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/homeassistant"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/mealplan"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/recipes"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/settings"
)

func ptr[T any](v T) *T { return &v }

// ing ist eine Kurzform für Zutaten im Test; q = 0 heißt ohne Menge.
func ing(q float64, unit, name string, relevant bool) recipes.Ingredient {
	i := recipes.Ingredient{Unit: unit, Name: name, ShoppingRelevant: relevant}
	if q != 0 {
		i.Quantity = ptr(q)
	}
	return i
}

// setup legt drei Rezepte und einen Essensplan an. Heute ist Mittwoch,
// 21.10.2026; mit Einkaufstag Samstag läuft der Zeitraum vom 17. bis 23.10.
func setup(t *testing.T) (*Store, *http.ServeMux) {
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

	rs := recipes.NewStore(db)
	create := func(title string, base int, ings ...recipes.Ingredient) int64 {
		r, err := rs.Create(ctx, recipes.Recipe{Title: title, BaseServings: base, Ingredients: ings})
		if err != nil {
			t.Fatal(err)
		}
		return r.ID
	}
	gulasch := create("Gulasch", 4,
		ing(500, "g", "Rindfleisch", true),
		ing(2, "", "Zwiebeln", true),
		ing(1, "EL", "Paprikapulver", true),
		ing(1, "Zehen", "Knoblauch", true),
		ing(0, "", "Salz", false))
	omelett := create("Omelett", 2,
		ing(3, "", "Eier", true),
		ing(1, "", "Zwiebel(n)", true),
		ing(100, "ml", "Milch", true),
		ing(0, "", "Petersilie", true),
		ing(1, "Prise", "Salz", false))
	pfannkuchen := create("Pfannkuchen", 4,
		ing(2, "", "Ei", true),
		ing(250, "ml", "Milch", true))

	plan := mealplan.NewStore(db, settings.NewStore(db))
	plan.SetClock(func() time.Time { return time.Date(2026, 10, 21, 18, 0, 0, 0, time.Local) })
	add := func(date string, recipeID int64, servings int, note string) int64 {
		in := mealplan.NewEntry{Date: date, Servings: ptr(servings), Note: note}
		if recipeID != 0 {
			in.RecipeID = ptr(recipeID)
		}
		e, err := plan.Create(ctx, in)
		if err != nil {
			t.Fatal(err)
		}
		return e.ID
	}
	add("2026-10-19", gulasch, 4, "") // schon vorbei
	add("2026-10-21", gulasch, 6, "") // heute, 1,5-fach
	add("2026-10-22", omelett, 3, "") // 1,5-fach
	have := add("2026-10-22", pfannkuchen, 4, "")
	add("2026-10-23", 0, 4, "Essen gehen")
	add("2026-10-24", pfannkuchen, 4, "") // nächster Zeitraum
	if _, err := plan.Update(ctx, have, mealplan.Changes{RecipeID: ptr(pfannkuchen), Servings: 4, Status: mealplan.StatusHaveIngredients}); err != nil {
		t.Fatal(err)
	}

	store := NewStore(db, plan, settings.NewStore(db), homeassistant.NewTestClient())
	mux := http.NewServeMux()
	NewHandler(store).Register(mux)
	return store, mux
}

// summary macht die Liste im Test gut vergleichbar: "Name: Menge" je Zeile.
func summary(items []Item) string {
	var lines []string
	for _, it := range items {
		lines = append(lines, it.Name+": "+it.Spec)
	}
	return strings.Join(lines, "\n")
}

func TestPreviewCurrentPeriod(t *testing.T) {
	store, _ := setup(t)
	p, err := store.Preview(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if p.Period.From != "2026-10-17" || p.Period.To != "2026-10-23" || p.Today != "2026-10-21" {
		t.Errorf("Zeitraum = %+v", p.Period)
	}

	// Gulasch ×1,5 und Omelett ×1,5; Stück-Mengen aufgerundet; Salz ist
	// nicht einkaufsrelevant; Zwiebel(n) und Zwiebeln, Ei und Eier sind
	// jeweils ein Artikel.
	want := strings.Join([]string{
		"Eier: 5",
		"Knoblauch: 2 Zehen",
		"Milch: 150 ml",
		"Paprikapulver: 1 ½ EL",
		"Petersilie: ",
		"Rindfleisch: 750 g",
		"Zwiebeln: 5",
	}, "\n")
	if got := summary(p.Items); got != want {
		t.Errorf("Liste:\n%s\nerwartet:\n%s", got, want)
	}
	for _, it := range p.Items {
		if it.Name == "Zwiebeln" && strings.Join(it.Recipes, ",") != "Gulasch,Omelett" {
			t.Errorf("Zwiebeln aus %v", it.Recipes)
		}
	}

	var included []string
	for _, e := range p.Entries {
		if e.Included {
			included = append(included, e.Date+" "+e.Title)
		}
	}
	if len(p.Entries) != 5 || strings.Join(included, ", ") != "2026-10-21 Gulasch, 2026-10-22 Omelett" {
		t.Errorf("%d Einträge, eingerechnet: %v", len(p.Entries), included)
	}
}

func TestPreviewNextPeriod(t *testing.T) {
	_, mux := setup(t)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/shopping/preview?period=next", nil))
	body := rec.Body.String()
	if rec.Code != http.StatusOK ||
		!strings.Contains(body, `"period":{"from":"2026-10-24","to":"2026-10-30"}`) ||
		!strings.Contains(body, `"name":"Ei","amounts":[{"quantity":2,"unit":""}],"spec":"2"`) ||
		!strings.Contains(body, `"spec":"250 ml"`) {
		t.Errorf("Status %d: %s", rec.Code, body)
	}

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/shopping/preview?period=gestern", nil))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("ungültiger Zeitraum: Status %d", rec.Code)
	}
}

func TestFormatQuantity(t *testing.T) {
	for q, want := range map[float64]string{
		333.33: "333", 12.46: "12,5", 10: "10", 1.5: "1 ½", 0.33: "⅓",
		2.2: "2,2", 2: "2", 0.75: "¾", 1.99: "2", 0.1: "0,1", 1.125: "1,13",
	} {
		if got := FormatQuantity(q); got != want {
			t.Errorf("FormatQuantity(%v) = %q, erwartet %q", q, got, want)
		}
	}
}

func TestRoundForShopping(t *testing.T) {
	for _, tt := range []struct {
		q    float64
		unit string
		want float64
	}{
		{1.33, "", 2}, {2.0000001, "", 2}, {0.25, "Dose", 1}, {1.5, "Pck.", 2},
		{1.5, "EL", 1.5}, {333.3, "g", 333.3},
	} {
		if got := roundForShopping(tt.q, tt.unit); got != tt.want {
			t.Errorf("roundForShopping(%v, %q) = %v, erwartet %v", tt.q, tt.unit, got, tt.want)
		}
	}
}

func TestNameKey(t *testing.T) {
	same := [][2]string{
		{"Zwiebel", "Zwiebeln"}, {"Ei", "Eier"}, {"Tomate", "Tomaten"},
		{"Kartoffel", "Kartoffeln"}, {"Apfel", "Äpfel"}, {"rote Paprikaschote", "Rote Paprikaschoten"},
	}
	for _, p := range same {
		if nameKey(p[0]) != nameKey(p[1]) {
			t.Errorf("%q und %q: %q ≠ %q", p[0], p[1], nameKey(p[0]), nameKey(p[1]))
		}
	}
	if nameKey("Tomaten") == nameKey("Tomatenmark") || nameKey("Milch") == nameKey("Mehl") {
		t.Error("verschiedene Zutaten fallen zusammen")
	}
}

// chooseList wählt die Testliste als Bring-Liste.
func chooseList(t *testing.T, store *Store) {
	t.Helper()
	if _, err := store.settings.Update(context.Background(), settings.Settings{FamilySize: 4, ShoppingDay: "saturday", BringEntity: homeassistant.TestListID}); err != nil {
		t.Fatal(err)
	}
}

func TestSend(t *testing.T) {
	ctx := context.Background()
	store, _ := setup(t)
	ha := store.ha.(*homeassistant.TestClient) // Typzusicherung: der Testmodus von setup

	var verr *ValidationError
	if _, err := store.Send(ctx, SendOptions{}); !errors.As(err, &verr) || verr.Field != "bring_entity" {
		t.Fatalf("ohne Bring-Liste: err = %v", err)
	}
	chooseList(t, store)
	// Milch steht schon offen auf der Liste: Die Menge wird ergänzt.
	ha.AddItem(ctx, homeassistant.TestListID, "Milch", "1 l")

	result, err := store.Send(ctx, SendOptions{Deselected: []string{"rindfleisch"}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Entries != 2 || len(result.Sent) != 6 || len(result.Failed) != 0 {
		t.Errorf("Ergebnis = %+v", result)
	}

	items, _ := ha.OpenItems(ctx, homeassistant.TestListID)
	var lines []string
	for _, it := range items {
		lines = append(lines, it.Summary+": "+it.Description)
	}
	want := "Milch: 1 l + 150 ml, Eier: 5, Knoblauch: 2 Zehen, Paprikapulver: 1 ½ EL, Petersilie: , Zwiebeln: 5"
	if got := strings.Join(lines, ", "); got != want {
		t.Errorf("Bring-Liste:\n%s\nerwartet:\n%s", got, want)
	}

	// Protokoll: 7 gesendete Zutaten (je Gericht), Rindfleisch abgewählt.
	var sent, skipped int
	store.db.QueryRowContext(ctx, `SELECT count(*) FROM transferred_items WHERE sent = 1 AND period_from = '2026-10-17'`).Scan(&sent)
	store.db.QueryRowContext(ctx, `SELECT count(*) FROM transferred_items WHERE sent = 0 AND item_key = 'rindfleisch'`).Scan(&skipped)
	if sent != 7 || skipped != 1 {
		t.Errorf("Protokoll: %d gesendet, %d abgewählt", sent, skipped)
	}

	// Danach ist die Liste leer und die Gerichte sind übertragen.
	p, _ := store.Preview(ctx, false)
	if len(p.Items) != 0 {
		t.Errorf("Liste nach dem Senden: %s", summary(p.Items))
	}
	for _, e := range p.Entries {
		if (e.Date == "2026-10-21" || e.Date == "2026-10-22" && e.Title == "Omelett") && e.Status != mealplan.StatusTransferred {
			t.Errorf("%s %s: Status %s", e.Date, e.Title, e.Status)
		}
	}
	if _, err := store.Send(ctx, SendOptions{}); !errors.As(err, &verr) || verr.Field != "period" {
		t.Errorf("zweites Senden: err = %v", err)
	}
}

// failingClient lehnt einen bestimmten Artikel ab, sonst wie der Testmodus.
// Durch das eingebettete *TestClient hat er alle anderen Methoden geerbt.
type failingClient struct {
	*homeassistant.TestClient
	fail string
}

func (c failingClient) AddItem(ctx context.Context, list, summary, description string) error {
	if summary == c.fail {
		return fmt.Errorf("%w: abgelehnt", homeassistant.ErrUnavailable)
	}
	return c.TestClient.AddItem(ctx, list, summary, description)
}

func TestSendPartialFailure(t *testing.T) {
	ctx := context.Background()
	store, _ := setup(t)
	store.ha = failingClient{homeassistant.NewTestClient(), "Eier"}
	chooseList(t, store)

	result, err := store.Send(ctx, SendOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Failed) != 1 || result.Failed[0].Name != "Eier" || len(result.Sent) != 6 {
		t.Errorf("Ergebnis = %+v", result)
	}
	// Eier sind nicht protokolliert; die Gerichte gelten trotzdem als übertragen.
	var n int
	store.db.QueryRowContext(ctx, `SELECT count(*) FROM transferred_items WHERE item_key = 'ei'`).Scan(&n)
	if n != 0 {
		t.Errorf("%d Protokollzeilen für Eier", n)
	}
}

func TestSendHomeAssistantDown(t *testing.T) {
	store, mux := setup(t)
	store.ha = homeassistant.New("", "http://127.0.0.1:1", "x")
	chooseList(t, store)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("POST", "/api/v1/shopping/send", strings.NewReader(`{"period": "current"}`)))
	if rec.Code != http.StatusBadGateway {
		t.Errorf("Status %d: %s", rec.Code, rec.Body)
	}
	// Nichts übertragen, nichts geändert.
	p, _ := store.Preview(context.Background(), false)
	if len(p.Items) != 7 {
		t.Errorf("%d Artikel übrig", len(p.Items))
	}
}

func TestDifferences(t *testing.T) {
	ctx := context.Background()
	store, _ := setup(t)
	ha := store.ha.(*homeassistant.TestClient)
	chooseList(t, store)
	if _, err := store.Send(ctx, SendOptions{Deselected: []string{"rindfleisch"}}); err != nil {
		t.Fatal(err)
	}

	// Gerichte des Tests per Titel finden
	p, _ := store.Preview(ctx, false)
	if len(p.Missing) != 0 || len(p.Surplus) != 0 {
		t.Fatalf("direkt nach dem Senden: fehlt %s / zu viel %s", summary(p.Missing), summary(p.Surplus))
	}
	entryID := func(date, title string) int64 {
		for _, e := range p.Entries {
			if e.Date == date && e.Title == title {
				return e.ID
			}
		}
		t.Fatalf("kein Eintrag %s %s", date, title)
		return 0
	}
	gulasch, omelett := entryID("2026-10-21", "Gulasch"), entryID("2026-10-22", "Omelett")
	setServings := func(id int64, servings int) {
		e, err := store.plan.Get(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := store.plan.Update(ctx, id, mealplan.Changes{RecipeID: e.RecipeID, Servings: servings, Status: e.Status}); err != nil {
			t.Fatal(err)
		}
	}

	// Omelett für 4 statt 3, Gulasch für 4 statt 6
	setServings(omelett, 4)
	setServings(gulasch, 4)
	p, _ = store.Preview(ctx, false)
	// Eier: 6 statt 5 gesendet; Milch 200 statt 150 ml. Zwiebeln: 4 statt 5.
	// Rindfleisch wurde abgewählt, ist also nicht "zu viel".
	if got, want := summary(p.Missing), "Eier: 1\nMilch: 50 ml"; got != want {
		t.Errorf("fehlt:\n%s\nerwartet:\n%s", got, want)
	}
	if got, want := summary(p.Surplus), "Knoblauch: 1 Zehen\nPaprikapulver: ½ EL\nZwiebeln: 1"; got != want {
		t.Errorf("zu viel:\n%s\nerwartet:\n%s", got, want)
	}

	// Differenz nachsenden: Die offenen Einträge werden ergänzt.
	result, err := store.Send(ctx, SendOptions{Missing: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Sent) != 2 || !result.Sent[0].Merged || result.Entries != 0 {
		t.Errorf("Nachsenden = %+v", result)
	}
	items, _ := ha.OpenItems(ctx, homeassistant.TestListID)
	for _, it := range items {
		if it.Summary == "Eier" && it.Description != "5 + 1" || it.Summary == "Milch" && it.Description != "150 ml + 50 ml" {
			t.Errorf("%s: %q", it.Summary, it.Description)
		}
	}
	if p, _ = store.Preview(ctx, false); len(p.Missing) != 0 {
		t.Errorf("nach dem Nachsenden fehlt noch: %s", summary(p.Missing))
	}

	// Omelett gelöscht: Alles dafür Gesendete ist jetzt zu viel.
	if err := store.plan.Delete(ctx, omelett); err != nil {
		t.Fatal(err)
	}
	p, _ = store.Preview(ctx, false)
	if got, want := summary(p.Surplus), "Eier: 6\nKnoblauch: 1 Zehen\nMilch: 200 ml\nPaprikapulver: ½ EL\nPetersilie: \nZwiebeln: 3"; got != want {
		t.Errorf("zu viel nach dem Löschen:\n%s\nerwartet:\n%s", got, want)
	}
}
