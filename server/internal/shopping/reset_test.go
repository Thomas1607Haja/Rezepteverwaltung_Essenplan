package shopping

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/homeassistant"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/itemkey"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/mealplan"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/settings"
)

// Übertragung zurücksetzen (Server 0.18): falsch gesendet, zurücksetzen,
// an die richtige Liste neu senden.

// switchList stellt eine andere Bring-Liste ein.
func switchList(t *testing.T, store *Store, list string) {
	t.Helper()
	if _, err := store.settings.Update(context.Background(), settings.Settings{FamilySize: 4, ShoppingDay: "saturday", BringEntity: list}); err != nil {
		t.Fatal(err)
	}
}

// listText zeigt eine Liste des Testmodus als "Name: Menge, …".
func listText(t *testing.T, store *Store, list string) string {
	t.Helper()
	items, err := store.ha.OpenItems(context.Background(), list)
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	for _, it := range items {
		lines = append(lines, it.Summary+": "+it.Description)
	}
	slices.Sort(lines)
	return strings.Join(lines, ", ")
}

// transferredEntries liefert die IDs der übertragenen Gerichte im Zeitraum.
func transferredEntries(t *testing.T, store *Store) map[string]int64 {
	t.Helper()
	p, err := store.Preview(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]int64{}
	for _, e := range p.Entries {
		if e.Status == mealplan.StatusTransferred {
			ids[e.Title] = e.ID
		}
	}
	return ids
}

func countRows(t *testing.T, store *Store, where string) int {
	t.Helper()
	var n int
	if err := store.db.QueryRowContext(context.Background(), "SELECT count(*) FROM transferred_items WHERE "+where).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// Der Fall aus dem echten Einsatz: an die Testliste gesendet, dann die
// richtige Liste eingestellt, alles zurückgesetzt und neu gesendet.
func TestResetAndResendToNewList(t *testing.T) {
	ctx := context.Background()
	store, _ := setup(t)
	chooseList(t, store)

	// Beim Gulasch ist Knoblauch vorhanden (Kurzvorschau).
	gulasch := transferredEntriesOrPlanned(t, store, "Gulasch")
	keys := []string{itemkey.Of("Knoblauch")}
	if _, err := store.plan.Update(ctx, gulasch.ID, mealplan.Changes{
		RecipeID: gulasch.RecipeID, Servings: gulasch.Servings, Status: gulasch.Status, ExcludedIngredients: &keys,
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := store.Send(ctx, SendOptions{}); err != nil {
		t.Fatal(err)
	}
	if n := countRows(t, store, "list_entity = '"+homeassistant.TestListID+"'"); n == 0 {
		t.Fatal("Ziel-Liste nicht protokolliert")
	}
	// Danach mehr Portionen und die Differenz nachgesendet: Diese Zeilen
	// hängen an keinem Gericht.
	if _, err := store.plan.Update(ctx, gulasch.ID, mealplan.Changes{
		RecipeID: gulasch.RecipeID, Servings: 8, Status: mealplan.StatusTransferred,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Send(ctx, SendOptions{Missing: true}); err != nil {
		t.Fatal(err)
	}
	if countRows(t, store, "entry_id IS NULL") == 0 {
		t.Fatal("keine nachgesendete Differenz protokolliert")
	}

	// Jetzt die richtige Liste einstellen.
	switchList(t, store, homeassistant.TestListID2)
	p, err := store.Preview(ctx, false)
	if err != nil {
		t.Fatal(err)
	}
	if p.List != (ListRef{homeassistant.TestListID2, "Testliste 2 (nur lokal)"}) {
		t.Errorf("Liste in der Vorschau = %+v", p.List)
	}

	// Vorschau: beide Gerichte, damals an die Testliste.
	rp, err := store.ResetPreview(ctx, false, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(rp.Entries) != 2 || rp.Current.EntityID != homeassistant.TestListID2 || len(rp.Lists) != 2 {
		t.Fatalf("Vorschau = %+v", rp)
	}
	var ids []int64
	for _, e := range rp.Entries {
		ids = append(ids, e.ID)
		if len(e.Lists) != 1 || e.Lists[0] != (ListRef{homeassistant.TestListID, "Testliste (nur lokal)"}) || e.UnknownList || e.Items == 0 {
			t.Errorf("%s: Listen %+v, unbekannt %v, %d Artikel", e.Title, e.Lists, e.UnknownList, e.Items)
		}
	}

	result, err := store.Reset(ctx, ResetOptions{Entries: ids, Remove: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.Entries != 2 || len(result.Removed) == 0 || len(result.Reduced)+len(result.NotFound)+len(result.Failed)+len(result.Skipped) != 0 {
		t.Errorf("Ergebnis = %+v", result)
	}
	if got := listText(t, store, homeassistant.TestListID); got != "" {
		t.Errorf("alte Liste nicht leer: %s", got)
	}
	if n := countRows(t, store, "1 = 1"); n != 0 {
		t.Errorf("%d Protokollzeilen übrig", n)
	}

	// Status zurück, Abwahl erhalten, keine Differenzen mehr.
	for _, id := range ids {
		e, _ := store.plan.Get(ctx, id)
		if e.Status != mealplan.StatusPlanned || e.TransferredAt != nil {
			t.Errorf("%d: Status %s, übertragen %v", id, e.Status, e.TransferredAt)
		}
	}
	if e, _ := store.plan.Get(ctx, gulasch.ID); !slices.Equal(e.ExcludedIngredients, []string{"knoblauch"}) {
		t.Errorf("Abwahl verloren: %v", e.ExcludedIngredients)
	}
	p, _ = store.Preview(ctx, false)
	if len(p.Missing)+len(p.Surplus) != 0 || len(p.Items) == 0 {
		t.Errorf("nach dem Zurücksetzen: %d Artikel, fehlt %v, zu viel %v", len(p.Items), p.Missing, p.Surplus)
	}

	// Neu senden: geht an die jetzt eingestellte Liste.
	sent, err := store.Send(ctx, SendOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if sent.List != homeassistant.TestListID2 || sent.Entries != 2 {
		t.Errorf("neu gesendet = %+v", sent)
	}
	if got := listText(t, store, homeassistant.TestListID); got != "" {
		t.Errorf("alte Liste: %s", got)
	}
	got := listText(t, store, homeassistant.TestListID2)
	if !strings.Contains(got, "Rindfleisch: 1000 g") || strings.Contains(got, "Knoblauch") {
		t.Errorf("neue Liste: %s", got)
	}
	if n := countRows(t, store, "list_entity IS NOT '"+homeassistant.TestListID2+"'"); n != 0 {
		t.Errorf("%d Zeilen nicht an Liste 2", n)
	}
}

// transferredEntriesOrPlanned liefert das Gericht title im aktuellen Zeitraum.
func transferredEntriesOrPlanned(t *testing.T, store *Store, title string) mealplan.Entry {
	t.Helper()
	ctx := context.Background()
	p, _ := store.Preview(ctx, false)
	for _, e := range p.Entries {
		if e.Title == title && e.Date >= p.Today {
			got, err := store.plan.Get(ctx, e.ID)
			if err != nil {
				t.Fatal(err)
			}
			return got
		}
	}
	t.Fatalf("kein Gericht %s", title)
	return mealplan.Entry{}
}

// Ein einzelnes Gericht zurücksetzen: Was das andere, weiter übertragene
// Gericht braucht, bleibt mit verringerter Menge stehen.
func TestResetSingleEntry(t *testing.T) {
	ctx := context.Background()
	store, _ := setup(t)
	chooseList(t, store)
	if _, err := store.Send(ctx, SendOptions{}); err != nil {
		t.Fatal(err)
	}
	// Eier hat jemand schon abgehakt bzw. von Hand gelöscht.
	items, _ := store.ha.OpenItems(ctx, homeassistant.TestListID)
	for _, it := range items {
		if it.Summary == "Eier" {
			store.ha.RemoveItem(ctx, homeassistant.TestListID, it.UID)
		}
	}
	ids := transferredEntries(t, store)

	rp, err := store.ResetPreview(ctx, false, ids["Omelett"])
	if err != nil || len(rp.Entries) != 1 || rp.Entries[0].Title != "Omelett" {
		t.Fatalf("Vorschau = %+v, %v", rp, err)
	}
	result, err := store.Reset(ctx, ResetOptions{Entries: []int64{ids["Omelett"]}, Remove: true})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(result.Removed, ","); got != "Milch,Petersilie" {
		t.Errorf("entfernt: %s", got)
	}
	if got := strings.Join(result.Reduced, ","); got != "Zwiebel" {
		t.Errorf("verringert: %s", got)
	}
	if got := strings.Join(result.NotFound, ","); got != "Eier" {
		t.Errorf("nicht gefunden: %s", got)
	}
	// Gulasch ×1,5: 3 Zwiebeln; sein Rest bleibt unverändert.
	want := "Knoblauch: 2 Zehen, Paprikapulver: 1 ½ EL, Rindfleisch: 750 g, Zwiebeln: 3"
	if got := listText(t, store, homeassistant.TestListID); got != want {
		t.Errorf("Liste:\n%s\nerwartet:\n%s", got, want)
	}
	if e, _ := store.plan.Get(ctx, ids["Gulasch"]); e.Status != mealplan.StatusTransferred {
		t.Errorf("Gulasch: %s", e.Status)
	}
	if countRows(t, store, "entry_id = "+itoa(ids["Omelett"])) != 0 || countRows(t, store, "entry_id = "+itoa(ids["Gulasch"])) == 0 {
		t.Error("Protokoll falsch gelöscht")
	}
	// Das Omelett steht wieder auf der Einkaufsliste, ohne Differenz.
	p, _ := store.Preview(ctx, false)
	if got := summary(p.Items); got != "Eier: 5\nMilch: 150 ml\nPetersilie: \nZwiebel(n): 2" && got != "Eier: 5\nMilch: 150 ml\nPetersilie: \nZwiebel: 2" {
		t.Errorf("Liste:\n%s", got)
	}
	if len(p.Missing)+len(p.Surplus) != 0 {
		t.Errorf("Differenzen: %v %v", p.Missing, p.Surplus)
	}

	// Ein nicht übertragenes Gericht lässt sich nicht zurücksetzen.
	var verr *ValidationError
	if _, err := store.ResetPreview(ctx, false, ids["Omelett"]); !errors.As(err, &verr) {
		t.Errorf("Vorschau für geplantes Gericht: %v", err)
	}
	if _, err := store.Reset(ctx, ResetOptions{Entries: []int64{ids["Omelett"]}}); !errors.As(err, &verr) {
		t.Errorf("geplantes Gericht zurückgesetzt: %v", err)
	}
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

// Vor Server 0.18 übertragen: Die Liste ist nicht gespeichert.
func TestResetWithoutStoredList(t *testing.T) {
	ctx := context.Background()
	store, _ := setup(t)
	chooseList(t, store)
	if _, err := store.Send(ctx, SendOptions{}); err != nil {
		t.Fatal(err)
	}
	store.db.ExecContext(ctx, "UPDATE transferred_items SET list_entity = NULL")
	before := listText(t, store, homeassistant.TestListID)
	ids := transferredEntries(t, store)

	rp, _ := store.ResetPreview(ctx, false, 0)
	for _, e := range rp.Entries {
		if !e.UnknownList || len(e.Lists) != 0 {
			t.Errorf("%s: unbekannt %v, Listen %+v", e.Title, e.UnknownList, e.Lists)
		}
	}

	// Ohne gewählte Liste: zurücksetzen, aber nichts entfernen.
	result, err := store.Reset(ctx, ResetOptions{Entries: []int64{ids["Omelett"]}, Remove: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Skipped) != 4 || len(result.Removed) != 0 {
		t.Errorf("Ergebnis = %+v", result)
	}
	if got := listText(t, store, homeassistant.TestListID); got != before {
		t.Errorf("Liste verändert: %s", got)
	}

	// Mit gewählter Liste: Die Artikel des Gulaschs gehen von der Testliste.
	result, err = store.Reset(ctx, ResetOptions{Entries: []int64{ids["Gulasch"]}, Remove: true, FallbackList: homeassistant.TestListID})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(result.Removed, ","); got != "Knoblauch,Paprikapulver,Rindfleisch,Zwiebeln" || len(result.Skipped) != 0 {
		t.Errorf("Ergebnis = %+v", result)
	}
	// Übrig sind die Artikel des Omeletts (oben nicht entfernt) – bis auf
	// die Zwiebeln, die zum Gulasch gehörten.
	if got := listText(t, store, homeassistant.TestListID); got != "Eier: 5, Milch: 150 ml, Petersilie: " {
		t.Errorf("Liste: %s", got)
	}
	if n := countRows(t, store, "1 = 1"); n != 0 {
		t.Errorf("%d Protokollzeilen übrig", n)
	}
}

// Ohne „Artikel entfernen“ bleibt die alte Liste, wie sie ist; ist sie mit
// „entfernen“ nicht erreichbar, ändert sich gar nichts.
func TestResetKeepItemsAndHomeAssistantDown(t *testing.T) {
	ctx := context.Background()
	store, mux := setup(t)
	chooseList(t, store)
	if _, err := store.Send(ctx, SendOptions{}); err != nil {
		t.Fatal(err)
	}
	before := listText(t, store, homeassistant.TestListID)
	ids := transferredEntries(t, store)
	body := `{"entries": [` + itoa(ids["Gulasch"]) + `, ` + itoa(ids["Omelett"]) + `], "remove": true}`

	working := store.ha
	store.ha = homeassistant.New("", "http://127.0.0.1:1", "x")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("POST", "/api/v1/shopping/reset", strings.NewReader(body)))
	if rec.Code != http.StatusBadGateway {
		t.Errorf("Status %d: %s", rec.Code, rec.Body)
	}
	if len(transferredEntries(t, store)) != 2 || countRows(t, store, "1 = 1") == 0 {
		t.Error("trotz Fehler zurückgesetzt")
	}
	// Die Vorschau geht auch ohne Home Assistant, nur ohne Listenauswahl.
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/shopping/reset?period=current", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"lists_error"`) {
		t.Errorf("Vorschau: %d %s", rec.Code, rec.Body)
	}

	store.ha = working
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("POST", "/api/v1/shopping/reset", strings.NewReader(strings.Replace(body, "true", "false", 1))))
	if rec.Code != http.StatusOK {
		t.Fatalf("Status %d: %s", rec.Code, rec.Body)
	}
	if got := listText(t, store, homeassistant.TestListID); got != before {
		t.Errorf("Liste verändert:\n%s\nvorher:\n%s", got, before)
	}
	if len(transferredEntries(t, store)) != 0 {
		t.Error("nicht zurückgesetzt")
	}
}
