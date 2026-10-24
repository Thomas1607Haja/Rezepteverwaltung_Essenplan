package shopping

import (
	"context"
	"slices"
	"testing"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/itemkey"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/mealplan"
)

// Abgewählte Zutaten eines Planeintrags („habe ich noch“, Schritt 8.E):
// Sie fehlen auf der Einkaufsliste, wandern beim Tauschen mit, verfallen
// beim Ersetzen und erscheinen nach dem Senden als Differenz.
func TestExcludedIngredients(t *testing.T) {
	ctx := context.Background()
	store, _ := setup(t)
	p, _ := store.Preview(ctx, false)
	find := func(date, title string) mealplan.Entry {
		t.Helper()
		for _, e := range p.Entries {
			if e.Date == date && e.Title == title {
				got, err := store.plan.Get(ctx, e.ID)
				if err != nil {
					t.Fatal(err)
				}
				return got
			}
		}
		t.Fatalf("kein Eintrag %s %s", date, title)
		return mealplan.Entry{}
	}
	gulasch, omelett := find("2026-10-21", "Gulasch"), find("2026-10-22", "Omelett")

	// Beim Gulasch Zwiebeln und Knoblauch abwählen (Schlüssel wie item_key).
	keys := []string{itemkey.Of("Zwiebeln"), itemkey.Of("Knoblauch")}
	if _, err := store.plan.Update(ctx, gulasch.ID, mealplan.Changes{
		RecipeID: gulasch.RecipeID, Servings: gulasch.Servings, Status: gulasch.Status, ExcludedIngredients: &keys,
	}); err != nil {
		t.Fatal(err)
	}
	p, _ = store.Preview(ctx, false)
	// Zwiebeln nur noch vom Omelett (1,5 → 2), Knoblauch fehlt ganz.
	if got, want := summary(p.Items), "Eier: 5\nMilch: 150 ml\nPaprikapulver: 1 ½ EL\nPetersilie: \nRindfleisch: 750 g\nZwiebel: 2"; got != want {
		t.Errorf("Liste mit Abwahl:\n%s\nerwartet:\n%s", got, want)
	}

	// Ändern ohne das Feld (ältere App): Abwahl bleibt.
	if _, err := store.plan.Update(ctx, gulasch.ID, mealplan.Changes{
		RecipeID: gulasch.RecipeID, Servings: 4, Status: gulasch.Status,
	}); err != nil {
		t.Fatal(err)
	}
	if e, _ := store.plan.Get(ctx, gulasch.ID); !slices.Equal(e.ExcludedIngredients, []string{"knoblauch", "zwiebel"}) {
		t.Errorf("nach Ändern: %v", e.ExcludedIngredients)
	}

	// Tauschen: Die Abwahl wandert mit dem Eintrag.
	if _, err := store.plan.Move(ctx, gulasch.ID, mealplan.Move{Date: omelett.Date, Slot: omelett.Slot, SwapWith: &omelett.ID, Confirm: true}); err != nil {
		t.Fatal(err)
	}
	moved, _ := store.plan.Get(ctx, gulasch.ID)
	other, _ := store.plan.Get(ctx, omelett.ID)
	if moved.Date != "2026-10-22" || len(moved.ExcludedIngredients) != 2 || len(other.ExcludedIngredients) != 0 {
		t.Errorf("nach Tauschen: %s %v / %v", moved.Date, moved.ExcludedIngredients, other.ExcludedIngredients)
	}

	// Senden, danach Zwiebeln wieder einschließen: fehlen jetzt (nachsenden).
	chooseList(t, store)
	if _, err := store.Send(ctx, SendOptions{}); err != nil {
		t.Fatal(err)
	}
	moved, _ = store.plan.Get(ctx, gulasch.ID)
	onlyGarlic := []string{itemkey.Of("Knoblauch")}
	if _, err := store.plan.Update(ctx, moved.ID, mealplan.Changes{
		RecipeID: moved.RecipeID, Servings: moved.Servings, Status: moved.Status, ExcludedIngredients: &onlyGarlic,
	}); err != nil {
		t.Fatal(err)
	}
	p, _ = store.Preview(ctx, false)
	if got := summary(p.Missing); got != "Zwiebeln: 2" {
		t.Errorf("fehlt nach Einschließen: %q", got)
	}
	// Rindfleisch nachträglich abwählen: war gesendet, ist jetzt zu viel.
	more := []string{itemkey.Of("Knoblauch"), itemkey.Of("Rindfleisch")}
	if _, err := store.plan.Update(ctx, moved.ID, mealplan.Changes{
		RecipeID: moved.RecipeID, Servings: moved.Servings, Status: moved.Status, ExcludedIngredients: &more,
	}); err != nil {
		t.Fatal(err)
	}
	p, _ = store.Preview(ctx, false)
	if got := summary(p.Surplus); got != "Rindfleisch: 500 g" {
		t.Errorf("zu viel nach Abwahl: %q", got)
	}

	// Ersetzen durch ein anderes Rezept: Abwahl verfällt.
	pfannkuchen := find("2026-10-22", "Pfannkuchen").RecipeID
	if _, err := store.plan.Update(ctx, moved.ID, mealplan.Changes{
		RecipeID: pfannkuchen, Servings: moved.Servings, Status: mealplan.StatusPlanned,
	}); err != nil {
		t.Fatal(err)
	}
	if e, _ := store.plan.Get(ctx, moved.ID); len(e.ExcludedIngredients) != 0 {
		t.Errorf("nach Ersetzen: %v", e.ExcludedIngredients)
	}
}
