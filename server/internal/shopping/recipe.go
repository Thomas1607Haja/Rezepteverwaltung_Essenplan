package shopping

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/apierr"
)

// RecipeSendOptions: „Jetzt einkaufen“ – die Zutaten eines einzelnen
// Rezepts spontan an Bring senden, ohne es einzuplanen.
type RecipeSendOptions struct {
	RecipeID int64
	// Servings: Portionen (bzw. Menge der Ergebnis-Einheit, z. B. Kuchen),
	// auf die die Mengen umgerechnet werden; 1–50.
	Servings int
	// Excluded: Schlüssel (item_key) der Zutaten, die noch im Haus sind.
	Excluded []string
}

// SendRecipe sendet die einkaufsrelevanten Zutaten eines Rezepts an die
// eingestellte Bring-Liste – wie die normale Übertragung umgerechnet,
// zusammengefasst und bei schon offenen Artikeln ergänzt.
//
// Anders als Send legt SendRecipe nichts an und protokolliert nichts: kein
// Planeintrag, kein Eintrag in transferred_items. Die Einkaufsliste des
// Einkaufszeitraums, „Fehlende nachsenden“ und die Statistik bleiben
// dadurch unberührt; zurücksetzen lässt sich ein solcher Einkauf nicht.
func (s *Store) SendRecipe(ctx context.Context, opts RecipeSendOptions) (SendResult, error) {
	if opts.Servings < 1 || opts.Servings > 50 {
		return SendResult{}, apierr.Field("servings", "out_of_range", "min", 1, "max", 50)
	}
	st, err := s.settings.Get(ctx)
	if err != nil {
		return SendResult{}, err
	}
	if st.BringEntity == "" {
		return SendResult{}, apierr.Field("bring_entity", "no_bring_list")
	}
	needs, err := s.recipeNeeds(ctx, opts.RecipeID, opts.Servings)
	if err != nil {
		return SendResult{}, err
	}
	// Abgewählte Zutaten fallen heraus (wie excluded_ingredients bei
	// Planeinträgen).
	needs = slices.DeleteFunc(needs, func(n need) bool { return slices.Contains(opts.Excluded, n.key()) })
	items := summarize(needs, s.comma(ctx))
	if len(items) == 0 {
		return SendResult{}, apierr.Field("excluded", "no_items_selected")
	}

	// Dieselbe Sperre wie Send: keine zwei Übertragungen gleichzeitig, sonst
	// könnte ein Artikel doppelt angelegt statt ergänzt werden.
	s.sendMu.Lock()
	defer s.sendMu.Unlock()
	// Wie in Send: auch bei abgebrochener Anfrage zu Ende senden.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Minute)
	defer cancel()

	open, err := s.ha.OpenItems(ctx, st.BringEntity)
	if err != nil {
		return SendResult{}, err
	}
	result := SendResult{List: st.BringEntity, Sent: []SentItem{}, Failed: []FailedItem{}}
	for _, it := range items {
		merged, err := s.sendItem(ctx, st.BringEntity, open, it)
		if err != nil {
			result.Failed = append(result.Failed, newFailedItem(it.Name, it.Spec, err))
			continue
		}
		result.Sent = append(result.Sent, SentItem{Name: it.Name, Spec: it.Spec, Merged: merged})
	}
	return result, nil
}

// recipeNeeds liest die einkaufsrelevanten Zutaten eines Rezepts, auf
// servings umgerechnet. Vorratsartikel (shopping_relevant = 0) zählen wie
// beim normalen Einkauf nicht mit.
func (s *Store) recipeNeeds(ctx context.Context, recipeID int64, servings int) ([]need, error) {
	var title string
	var baseServings int
	err := s.db.QueryRowContext(ctx, "SELECT title, base_servings FROM recipes WHERE id = ?", recipeID).Scan(&title, &baseServings)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apierr.Field("recipe_id", "recipe_gone")
	}
	if err != nil {
		return nil, fmt.Errorf("jetzt einkaufen: rezept lesen: %w", err)
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT quantity, unit, name FROM recipe_ingredients
		WHERE recipe_id = ? AND shopping_relevant = 1
		ORDER BY position`, recipeID)
	if err != nil {
		return nil, fmt.Errorf("jetzt einkaufen: zutaten lesen: %w", err)
	}
	defer rows.Close()

	needs := []need{}
	for rows.Next() {
		n := need{Recipe: title}
		if err := rows.Scan(&n.Quantity, &n.Unit, &n.Name); err != nil {
			return nil, fmt.Errorf("jetzt einkaufen: zutaten lesen: %w", err)
		}
		if n.Quantity != nil && baseServings > 0 {
			scaled := *n.Quantity * float64(servings) / float64(baseServings)
			n.Quantity = &scaled
		}
		needs = append(needs, n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("jetzt einkaufen: zutaten lesen: %w", err)
	}
	return needs, nil
}

// List liefert die eingestellte Bring-Liste mit Namen (für den Knopf
// „An Bring senden · <Liste>“); leer, wenn keine gewählt ist.
func (s *Store) List(ctx context.Context) (ListRef, error) {
	st, err := s.settings.Get(ctx)
	if err != nil {
		return ListRef{}, err
	}
	return s.listRef(ctx, st.BringEntity), nil
}
