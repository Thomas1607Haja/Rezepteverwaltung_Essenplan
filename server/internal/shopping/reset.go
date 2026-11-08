package shopping

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/apierr"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/database"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/homeassistant"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/mealplan"
)

// Übertragung zurücksetzen: Gerichte, die schon an Bring gingen, wieder auf
// "planned" setzen – z. B. weil versehentlich an die falsche Liste gesendet
// wurde. Auf Wunsch entfernt der Server die damals gesendeten Artikel aus
// der damaligen Liste. Danach lässt sich ganz normal neu senden, und zwar
// an die Liste, die jetzt in den Einstellungen steht.

// ListRef ist eine To-do-Liste in Home Assistant.
type ListRef struct {
	EntityID string `json:"entity_id"` // leer = keine Liste
	Name     string `json:"name"`      // Anzeigename, sonst die Entität
}

// ResetEntry ist ein übertragenes Gericht in der Vorschau. Entry ist
// eingebettet: Seine Felder erscheinen im JSON auf derselben Ebene wie die
// zusätzlichen (id, title, …, lists, unknown_list, items).
type ResetEntry struct {
	Entry
	// Lists sind die Listen, an die Artikel dieses Gerichts gingen.
	Lists []ListRef `json:"lists"`
	// UnknownList: Artikel wurden vor Server 0.18 gesendet, die Liste ist
	// nicht gespeichert.
	UnknownList bool `json:"unknown_list"`
	Items       int  `json:"items"` // gesendete Artikel (ohne abgewählte)
}

// ResetPreview zeigt, was ein Zurücksetzen beträfe.
type ResetPreview struct {
	Entries []ResetEntry `json:"entries"`
	Current ListRef      `json:"current_list"` // jetzt eingestellte Bring-Liste
	// Lists sind alle To-do-Listen, zur Auswahl für Gerichte ohne
	// gespeicherte Liste. Leer, wenn Home Assistant nicht erreichbar ist.
	Lists      []ListRef `json:"lists"`
	ListsError string    `json:"lists_error,omitempty"`
	// ListsErrorCode zum Übersetzen (ListsError ist der englische Text).
	ListsErrorCode string `json:"lists_error_code,omitempty"`
}

// ResetOptions legen fest, was zurückgesetzt wird.
type ResetOptions struct {
	Entries []int64
	// Remove: die gesendeten Artikel aus der damaligen Liste entfernen.
	Remove bool
	// FallbackList gilt für Artikel ohne gespeicherte Liste; leer = diese
	// nicht entfernen.
	FallbackList string
}

// ResetResult beschreibt, was zurückgesetzt und entfernt wurde.
type ResetResult struct {
	Entries int      `json:"entries"` // Gerichte, die jetzt wieder "planned" sind
	Removed []string `json:"removed"` // von der Liste gelöschte Artikel
	// Reduced: Andere übertragene Gerichte brauchen den Artikel noch; nur
	// seine Mengenangabe wurde verringert.
	Reduced []string `json:"reduced"`
	// NotFound: nicht mehr offen auf der Liste – schon abgehakt oder von
	// Hand gelöscht. Bleibt unverändert.
	NotFound []string     `json:"not_found"`
	Failed   []FailedItem `json:"failed"`
	// Skipped: Artikel ohne gespeicherte Liste, für die keine gewählt war.
	Skipped []string `json:"skipped"`
}

// ResetPreview liefert die übertragenen Gerichte des Einkaufszeitraums
// (next wie bei Preview) bzw. mit entryID != 0 nur dieses eine Gericht.
func (s *Store) ResetPreview(ctx context.Context, next bool, entryID int64) (ResetPreview, error) {
	var entries []Entry
	var err error
	if entryID != 0 {
		entries, err = s.queryEntries(ctx, "e.id = ? AND e.status = ?", "", entryID, mealplan.StatusTransferred)
		if err == nil && len(entries) == 0 {
			return ResetPreview{}, apierr.Field("entry", "entry_not_sent")
		}
	} else {
		var period mealplan.Period
		if period, _, err = s.period(ctx, next); err != nil {
			return ResetPreview{}, err
		}
		entries, err = s.queryEntries(ctx, "e.date BETWEEN ? AND ? AND e.status = ?", "", period.From, period.To, mealplan.StatusTransferred)
	}
	if err != nil {
		return ResetPreview{}, err
	}

	st, err := s.settings.Get(ctx)
	if err != nil {
		return ResetPreview{}, err
	}
	result := ResetPreview{Entries: []ResetEntry{}, Lists: []ListRef{}}
	names := map[string]string{}
	if lists, err := s.ha.TodoLists(ctx); err != nil {
		e := apierr.From(err, "home_assistant_down")
		result.ListsError, result.ListsErrorCode = e.Message(), e.Code
	} else {
		for _, l := range lists {
			names[l.EntityID] = l.Name
			result.Lists = append(result.Lists, ListRef(l))
		}
		s.rememberNames(names)
	}
	ref := func(entity string) ListRef {
		if name := names[entity]; name != "" {
			return ListRef{entity, name}
		}
		return ListRef{entity, entity}
	}
	if st.BringEntity != "" {
		result.Current = ref(st.BringEntity)
	}

	for _, e := range entries {
		re := ResetEntry{Entry: e, Lists: []ListRef{}}
		rows, err := s.db.QueryContext(ctx, `
			SELECT list_entity, sum(sent) FROM transferred_items
			WHERE entry_id = ? GROUP BY list_entity ORDER BY list_entity`, e.ID)
		if err != nil {
			return ResetPreview{}, fmt.Errorf("übertragung lesen: %w", err)
		}
		for rows.Next() {
			var list *string
			var sent int
			if err := rows.Scan(&list, &sent); err != nil {
				rows.Close()
				return ResetPreview{}, fmt.Errorf("übertragung lesen: %w", err)
			}
			re.Items += sent
			switch {
			case list == nil:
				re.UnknownList = re.UnknownList || sent > 0
			default:
				re.Lists = append(re.Lists, ref(*list))
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return ResetPreview{}, fmt.Errorf("übertragung lesen: %w", err)
		}
		result.Entries = append(result.Entries, re)
	}
	return result, nil
}

// resetRow ist eine Zeile aus transferred_items, die zurückgesetzt wird.
type resetRow struct {
	ID       int64
	Period   string
	Key      string
	Name     string
	Quantity *float64
	Unit     string
	Sent     bool
	List     string // leer = nicht gespeichert
}

// Reset setzt die Gerichte opts.Entries auf "planned" zurück und löscht
// ihr Protokoll. Die abgewählten Zutaten („Zutaten vorhanden“) bleiben.
//
// Mit opts.Remove entfernt der Server die gesendeten Artikel vorher aus
// der Liste, an die sie gingen. Brauchen andere, weiter übertragene
// Gerichte denselben Artikel, wird nur seine Mengenangabe verringert. Ist
// die Liste nicht erreichbar, ändert sich nichts.
func (s *Store) Reset(ctx context.Context, opts ResetOptions) (ResetResult, error) {
	s.sendMu.Lock()
	defer s.sendMu.Unlock()

	if opts.FallbackList != "" && !strings.HasPrefix(opts.FallbackList, "todo.") {
		return ResetResult{}, apierr.Field("fallback_list", "invalid_todo_entity")
	}
	ids, err := s.transferredIDs(ctx, opts.Entries)
	if err != nil {
		return ResetResult{}, err
	}
	if len(ids) == 0 {
		return ResetResult{}, apierr.Field("entries", "no_entries_selected")
	}
	rows, err := s.resetRows(ctx, ids)
	if err != nil {
		return ResetResult{}, err
	}

	result := ResetResult{Entries: len(ids), Removed: []string{}, Reduced: []string{}, NotFound: []string{}, Failed: []FailedItem{}, Skipped: []string{}}
	if opts.Remove {
		// Wie beim Senden: einmal begonnen, läuft es zu Ende, auch wenn der
		// Browser die Anfrage abbricht.
		ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Minute)
		defer cancel()
		if err := s.removeFromLists(ctx, rows, opts.FallbackList, &result); err != nil {
			return ResetResult{}, err
		}
	}

	now := database.FormatTime(time.Now())
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ResetResult{}, fmt.Errorf("übertragung zurücksetzen: %w", err)
	}
	defer tx.Rollback()
	for _, r := range rows {
		if _, err := tx.ExecContext(ctx, "DELETE FROM transferred_items WHERE id = ?", r.ID); err != nil {
			return ResetResult{}, fmt.Errorf("übertragung zurücksetzen: %w", err)
		}
	}
	for _, id := range ids {
		// updated_at: Die App holt den neuen Status beim nächsten Abgleich.
		_, err := tx.ExecContext(ctx, `
			UPDATE meal_plan_entries SET status = ?, transferred_at = NULL, updated_at = ?
			WHERE id = ? AND status = ?`,
			mealplan.StatusPlanned, now, id, mealplan.StatusTransferred)
		if err != nil {
			return ResetResult{}, fmt.Errorf("übertragung zurücksetzen: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return ResetResult{}, fmt.Errorf("übertragung zurücksetzen: %w", err)
	}
	return result, nil
}

// transferredIDs liefert von ids die Gerichte, die wirklich übertragen sind.
func (s *Store) transferredIDs(ctx context.Context, ids []int64) ([]int64, error) {
	result := []int64{}
	for _, id := range ids {
		var status string
		err := s.db.QueryRowContext(ctx, "SELECT status FROM meal_plan_entries WHERE id = ?", id).Scan(&status)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("übertragung zurücksetzen: %w", err)
		}
		if status == mealplan.StatusTransferred && !slices.Contains(result, id) {
			result = append(result, id)
		}
	}
	return result, nil
}

const resetColumns = "id, period_from, item_key, ingredient_name, quantity, unit, sent, coalesce(list_entity, '')"

// queryRows liest Protokollzeilen.
func (s *Store) queryRows(ctx context.Context, where string, args ...any) ([]resetRow, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT "+resetColumns+" FROM transferred_items WHERE "+where+" ORDER BY id", args...)
	if err != nil {
		return nil, fmt.Errorf("übertragung lesen: %w", err)
	}
	defer rows.Close()
	var result []resetRow
	for rows.Next() {
		var r resetRow
		if err := rows.Scan(&r.ID, &r.Period, &r.Key, &r.Name, &r.Quantity, &r.Unit, &r.Sent, &r.List); err != nil {
			return nil, fmt.Errorf("übertragung lesen: %w", err)
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// resetRows liefert das Protokoll der Gerichte ids. Bleibt danach in einem
// Einkaufszeitraum kein übertragenes Gericht übrig, gehören auch die
// nachgesendeten Differenzen dieses Zeitraums dazu (sie hängen an keinem
// Gericht) – sonst erschienen sie später als „zu viel gesendet“.
func (s *Store) resetRows(ctx context.Context, ids []int64) ([]resetRow, error) {
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	in := placeholders(len(ids))
	rows, err := s.queryRows(ctx, "entry_id IN ("+in+")", args...)
	if err != nil {
		return nil, err
	}
	var periods []string
	for _, r := range rows {
		if !slices.Contains(periods, r.Period) {
			periods = append(periods, r.Period)
		}
	}
	for _, p := range periods {
		var others int
		err := s.db.QueryRowContext(ctx, `
			SELECT count(*) FROM transferred_items
			WHERE period_from = ? AND entry_id IS NOT NULL AND entry_id NOT IN (`+in+`)`,
			append([]any{p}, args...)...).Scan(&others)
		if err != nil {
			return nil, fmt.Errorf("übertragung lesen: %w", err)
		}
		if others > 0 {
			continue
		}
		orphans, err := s.queryRows(ctx, "period_from = ? AND entry_id IS NULL", p)
		if err != nil {
			return nil, err
		}
		rows = append(rows, orphans...)
	}
	return rows, nil
}

// removeFromLists entfernt die gesendeten Artikel aus ihren Listen.
func (s *Store) removeFromLists(ctx context.Context, rows []resetRow, fallback string, result *ResetResult) error {
	// Je Liste die Artikel (key -> Anzeigename), dazu die Zeiträume.
	type target struct {
		names   map[string]string
		periods []string
	}
	targets := map[string]*target{}
	var lists []string
	ids := map[int64]bool{}
	for _, r := range rows {
		ids[r.ID] = true
		if !r.Sent {
			continue // abgewählt, also nie auf der Liste
		}
		list := r.List
		if list == "" {
			list = fallback
		}
		if list == "" {
			if !slices.Contains(result.Skipped, r.Name) {
				result.Skipped = append(result.Skipped, r.Name)
			}
			continue
		}
		t, ok := targets[list]
		if !ok {
			t = &target{names: map[string]string{}}
			targets[list] = t
			lists = append(lists, list)
		}
		if len(r.Name) > len(t.names[r.Key]) {
			t.names[r.Key] = r.Name
		}
		if !slices.Contains(t.periods, r.Period) {
			t.periods = append(t.periods, r.Period)
		}
	}

	// Erst alle Listen lesen: Ist eine nicht erreichbar, ändert sich nichts.
	open := map[string][]homeassistant.TodoItem{}
	for _, list := range lists {
		items, err := s.ha.OpenItems(ctx, list)
		if err != nil {
			return err
		}
		open[list] = items
	}

	for _, list := range lists {
		t := targets[list]
		// Was andere, weiter übertragene Gerichte derselben Zeiträume auf
		// dieser Liste brauchen. Zeilen ohne gespeicherte Liste zählen zur
		// gewählten Ersatzliste.
		args := []any{list, fallback, list}
		for _, p := range t.periods {
			args = append(args, p)
		}
		remaining, err := s.queryRows(ctx, `sent = 1
			AND (list_entity = ? OR (list_entity IS NULL AND ? = ?))
			AND period_from IN (`+placeholders(len(t.periods))+`)`, args...)
		if err != nil {
			return err
		}
		still := newCollector()
		for _, r := range remaining {
			if !ids[r.ID] {
				still.add("", r.Name, r.Quantity, r.Unit)
			}
		}
		stillByKey := map[string]Item{}
		for _, it := range still.items() {
			stillByKey[it.Key] = it
		}

		keys := make([]string, 0, len(t.names))
		for k := range t.names {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		for _, key := range keys {
			name := t.names[key]
			var matches []homeassistant.TodoItem
			for _, o := range open[list] {
				if nameKey(cleanName(o.Summary)) == key {
					matches = append(matches, o)
				}
			}
			if len(matches) == 0 {
				result.NotFound = append(result.NotFound, name)
				continue
			}
			// Nur ein offener Eintrag ist „unserer“; weitere gleichnamige
			// (z. B. von Hand angelegt) bleiben.
			o := matches[0]
			keep, needed := stillByKey[key]
			var err error
			switch {
			case !needed:
				err = s.ha.RemoveItem(ctx, list, o.UID)
			case keep.Spec != o.Description:
				err = s.ha.UpdateDescription(ctx, list, o.UID, keep.Spec)
			}
			switch {
			case err != nil:
				result.Failed = append(result.Failed, newFailedItem(name, "", err))
			case needed:
				result.Reduced = append(result.Reduced, name)
			default:
				result.Removed = append(result.Removed, name)
			}
		}
	}
	return nil
}

// placeholders liefert "?, ?, ?" für n Werte in einer IN-Liste.
func placeholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?, ", n), ", ")
}
