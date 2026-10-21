package shopping

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/database"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/homeassistant"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/mealplan"
)

// ValidationError beschreibt eine Anfrage, die so nicht ausführbar ist.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

// SentItem ist ein Artikel, der jetzt auf der Bring-Liste steht.
type SentItem struct {
	Name string `json:"name"`
	Spec string `json:"spec"`
	// Merged: Der Artikel stand schon offen auf der Liste; seine
	// Mengenangabe wurde ergänzt statt einen zweiten Eintrag anzulegen.
	Merged bool `json:"merged"`
}

// FailedItem ist ein Artikel, den Home Assistant nicht angenommen hat.
type FailedItem struct {
	Name  string `json:"name"`
	Spec  string `json:"spec"`
	Error string `json:"error"`
}

// SendResult beschreibt, was übertragen wurde.
type SendResult struct {
	List    string       `json:"list"` // To-do-Entität, z. B. todo.zuhause
	Sent    []SentItem   `json:"sent"`
	Failed  []FailedItem `json:"failed"`
	Entries int          `json:"entries"` // Gerichte, die jetzt "transferred" sind
}

// SendOptions legen fest, was gesendet wird.
type SendOptions struct {
	Next bool // folgender statt aktueller Einkaufszeitraum
	// Deselected sind die keys abgewählter Artikel: Sie werden nicht
	// gesendet, aber wie gesendete protokolliert und gelten damit als
	// erledigt (z. B. schon im Haus).
	Deselected []string
	// Missing: nur die Differenz der schon übertragenen Gerichte nachsenden.
	Missing bool
	// Entries: nur die Zutaten dieser Gerichte senden (z. B. direkt nach
	// dem Verschieben in den Einkaufszeitraum). Leer = alle geplanten.
	Entries []int64
}

// logRow ist eine künftige Zeile in transferred_items.
type logRow struct {
	EntryID  *int64 // nil bei nachgesendeten Differenzen
	Key      string
	Name     string
	Quantity *float64
	Unit     string
}

// Send überträgt Artikel an die gewählte Bring-Liste. Danach haben die
// eingerechneten Gerichte den Status "transferred", und alles Gesendete
// bzw. Abgewählte steht im Protokoll.
//
// Schlägt ein einzelner Artikel fehl, geht es mit den übrigen weiter. Der
// fehlgeschlagene wird nicht protokolliert und taucht deshalb danach unter
// den nachzusendenden Artikeln auf.
func (s *Store) Send(ctx context.Context, opts SendOptions) (SendResult, error) {
	// Zwei gleichzeitige Übertragungen (z. B. von zwei Handys) würden jeden
	// Artikel doppelt senden; die Sperre lässt die zweite warten.
	s.sendMu.Lock()
	defer s.sendMu.Unlock()

	st, err := s.settings.Get(ctx)
	if err != nil {
		return SendResult{}, err
	}
	if st.BringEntity == "" {
		return SendResult{}, &ValidationError{"bring_entity", "bitte zuerst in den einstellungen eine bring-liste wählen"}
	}
	period, today, err := s.period(ctx, opts.Next)
	if err != nil {
		return SendResult{}, err
	}

	var items []Item
	var rows []logRow
	var entryIDs []int64
	if opts.Missing {
		if items, _, err = s.differences(ctx, period); err != nil {
			return SendResult{}, err
		}
		for _, it := range items {
			if len(it.Amounts) == 0 {
				rows = append(rows, logRow{Key: it.Key, Name: it.Name})
			}
			for _, a := range it.Amounts {
				rows = append(rows, logRow{Key: it.Key, Name: it.Name, Quantity: &a.Quantity, Unit: a.Unit})
			}
		}
	} else {
		entries, err := s.loadEntries(ctx, period, today)
		if err != nil {
			return SendResult{}, err
		}
		for _, e := range entries {
			if e.Included && (len(opts.Entries) == 0 || slices.Contains(opts.Entries, e.ID)) {
				entryIDs = append(entryIDs, e.ID)
			}
		}
		needs, err := s.loadNeeds(ctx, max(today, period.From), period.To, mealplan.StatusPlanned)
		if err != nil {
			return SendResult{}, err
		}
		// Nur der Bedarf der ausgewählten Gerichte. slices.DeleteFunc
		// entfernt alle Elemente, für die die Funktion true liefert.
		needs = slices.DeleteFunc(needs, func(n need) bool { return !slices.Contains(entryIDs, n.EntryID) })
		items = summarize(needs)
		for _, n := range needs {
			rows = append(rows, logRow{EntryID: &n.EntryID, Key: n.key(), Name: cleanName(n.Name), Quantity: n.Quantity, Unit: n.Unit})
		}
	}
	if len(items) == 0 && len(entryIDs) == 0 {
		return SendResult{}, &ValidationError{"period", "in diesem zeitraum ist nichts zu senden"}
	}

	// Ab hier wird nach außen gesendet. Bricht der Browser die Anfrage ab,
	// soll trotzdem alles zu Ende laufen und protokolliert werden – sonst
	// stünden Artikel auf der Liste, von denen die App nichts weiß.
	// context.WithoutCancel behält die Werte von ctx, ignoriert aber dessen
	// Abbruch; ein eigenes Zeitlimit schützt vor endlosem Warten.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Minute)
	defer cancel()

	open, err := s.ha.OpenItems(ctx, st.BringEntity)
	if err != nil {
		return SendResult{}, err
	}

	result := SendResult{List: st.BringEntity, Sent: []SentItem{}, Failed: []FailedItem{}}
	done := map[string]bool{} // gesendete oder abgewählte Artikel
	for _, it := range items {
		if slices.Contains(opts.Deselected, it.Key) {
			done[it.Key] = true
			continue
		}
		merged, err := s.sendItem(ctx, st.BringEntity, open, it)
		if err != nil {
			result.Failed = append(result.Failed, FailedItem{Name: it.Name, Spec: it.Spec, Error: err.Error()})
			continue
		}
		done[it.Key] = true
		result.Sent = append(result.Sent, SentItem{Name: it.Name, Spec: it.Spec, Merged: merged})
	}

	if err := s.record(ctx, period, entryIDs, rows, opts.Deselected, done); err != nil {
		return SendResult{}, err
	}
	result.Entries = len(entryIDs)
	return result, nil
}

// sendItem legt einen Artikel auf der Liste an oder ergänzt einen offenen
// Eintrag mit gleichem Namen um die Mengenangabe.
func (s *Store) sendItem(ctx context.Context, list string, open []homeassistant.TodoItem, it Item) (merged bool, err error) {
	for _, o := range open {
		if nameKey(cleanName(o.Summary)) != it.Key {
			continue
		}
		if it.Spec == "" {
			return true, nil // steht schon drauf, ohne Menge gibt es nichts zu ergänzen
		}
		return true, s.ha.UpdateDescription(ctx, list, o.UID, joinSpec(o.Description, it.Spec))
	}
	return false, s.ha.AddItem(ctx, list, it.Name, it.Spec)
}

// joinSpec hängt eine Mengenangabe an eine vorhandene an.
func joinSpec(old, add string) string {
	if old == "" {
		return add
	}
	return old + " + " + add
}

// record setzt die Gerichte auf "transferred" und protokolliert, was
// gesendet (sent = 1) oder abgewählt (sent = 0) wurde.
func (s *Store) record(ctx context.Context, period mealplan.Period, entryIDs []int64, rows []logRow, deselected []string, done map[string]bool) error {
	now := database.FormatTime(time.Now())
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("übertragung speichern: %w", err)
	}
	defer tx.Rollback()

	for _, id := range entryIDs {
		_, err := tx.ExecContext(ctx, `
			UPDATE meal_plan_entries SET status = ?, transferred_at = ?, updated_at = ?
			WHERE id = ? AND status = ?`,
			mealplan.StatusTransferred, now, now, id, mealplan.StatusPlanned)
		if err != nil {
			return fmt.Errorf("übertragung speichern: %w", err)
		}
	}
	for _, r := range rows {
		if !done[r.Key] {
			continue
		}
		sent := !slices.Contains(deselected, r.Key)
		_, err := tx.ExecContext(ctx, `
			INSERT INTO transferred_items (entry_id, period_from, item_key, ingredient_name,
			                               quantity, unit, sent, transferred_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			r.EntryID, period.From, r.Key, r.Name, r.Quantity, r.Unit, sent, now)
		if err != nil {
			return fmt.Errorf("übertragung speichern: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("übertragung speichern: %w", err)
	}
	return nil
}
