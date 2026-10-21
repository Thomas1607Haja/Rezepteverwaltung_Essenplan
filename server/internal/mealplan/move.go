package mealplan

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/database"
)

// Move beschreibt das Verschieben eines Eintrags per Drag and Drop.
type Move struct {
	Date string `json:"date"`
	Slot string `json:"slot"`
	// SwapWith: Eintrag, mit dem getauscht wird (er bekommt Datum, Slot und
	// Position des verschobenen). Ohne Angabe wird ans Ende von Date/Slot
	// angehängt.
	SwapWith *int64 `json:"swap_with"`
	// Confirm: Warnungen sind gesehen, trotzdem speichern.
	Confirm bool `json:"confirm"`
	// HaveIngredients: Einträge, deren Zutaten laut Bestätigung vorhanden
	// sind; sie bekommen den Status have_ingredients.
	HaveIngredients []int64 `json:"have_ingredients"`
}

// Warnungsarten, siehe docs/PLANUNG.md, „Verschieben und Tauschen“.
const (
	// Ein geplantes Gericht rutscht in den Einkaufszeitraum, seine Zutaten
	// sind aber noch nicht eingekauft.
	WarnPlannedIntoPeriod = "planned_into_period"
	// Ein übertragenes Gericht wandert aus dem Einkaufszeitraum; seine
	// Zutaten sind schon gekauft und vielleicht verderblich.
	WarnTransferredOutOfPeriod = "transferred_out_of_period"
)

// Warning ist ein Hinweis zu einem Eintrag, der verschoben würde.
type Warning struct {
	Type    string `json:"type"`
	EntryID int64  `json:"entry_id"`
	Message string `json:"message"`
}

// MoveResult: Applied ist false, wenn es Warnungen gab und Confirm fehlte;
// dann ist nichts gespeichert. Entries sind die betroffenen Einträge nach
// dem Speichern.
type MoveResult struct {
	Applied  bool      `json:"applied"`
	Warnings []Warning `json:"warnings"`
	Entries  []Entry   `json:"entries"`
}

// placement ist der Platz eines Eintrags im Plan.
type placement struct {
	date, slot string
	position   int
}

// movingEntry ist ein Eintrag, der seinen Platz wechselt.
type movingEntry struct {
	id     int64
	label  string // Rezepttitel oder Notiz, für die Meldungen
	status string
	from   placement
	to     placement
}

// Move verschiebt den Eintrag id oder tauscht ihn mit m.SwapWith. Gibt es
// Warnungen und ist m.Confirm nicht gesetzt, wird nichts gespeichert.
func (s *Store) Move(ctx context.Context, id int64, m Move) (MoveResult, error) {
	if _, err := parseDate("date", m.Date); err != nil {
		return MoveResult{}, err
	}
	if m.Slot == "" {
		m.Slot = SlotDinner
	}
	if !validSlot(m.Slot) {
		return MoveResult{}, &ValidationError{"slot", "muss dinner oder lunch sein"}
	}
	if m.SwapWith != nil && *m.SwapWith == id {
		return MoveResult{}, &ValidationError{"swap_with", "ein eintrag kann nicht mit sich selbst tauschen"}
	}
	period, err := s.CurrentPeriod(ctx)
	if err != nil {
		return MoveResult{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return MoveResult{}, fmt.Errorf("eintrag %d verschieben: %w", id, err)
	}
	defer tx.Rollback()

	moved, err := loadMoving(ctx, tx, id)
	if err != nil {
		return MoveResult{}, err
	}
	moving := []*movingEntry{moved}

	if m.SwapWith != nil {
		other, err := loadMoving(ctx, tx, *m.SwapWith)
		if errors.Is(err, ErrNotFound) {
			return MoveResult{}, &ValidationError{"swap_with", "diesen eintrag gibt es nicht"}
		}
		if err != nil {
			return MoveResult{}, err
		}
		// Der Tauschpartner muss dort stehen, wohin gezogen wurde – sonst
		// hat sich der Plan inzwischen geändert (z. B. auf einem anderen Gerät).
		if other.from.date != m.Date || other.from.slot != m.Slot {
			return MoveResult{}, &ValidationError{"swap_with", "der plan hat sich inzwischen geändert, bitte neu laden"}
		}
		moved.to, other.to = other.from, moved.from
		moving = append(moving, other)
	} else {
		pos, err := nextPosition(ctx, tx, m.Date, m.Slot)
		if err != nil {
			return MoveResult{}, err
		}
		moved.to = placement{m.Date, m.Slot, pos}
	}

	result := MoveResult{Warnings: []Warning{}, Entries: []Entry{}}
	for _, e := range moving {
		if w, ok := checkMove(e, period); ok {
			result.Warnings = append(result.Warnings, w)
		}
	}
	if len(result.Warnings) > 0 && !m.Confirm {
		return result, nil // nichts speichern, erst bestätigen lassen
	}

	now := database.FormatTime(s.now())
	for _, e := range moving {
		status := e.status
		// slices.Contains prüft, ob die Liste den Wert enthält.
		if slices.Contains(m.HaveIngredients, e.id) && status == StatusPlanned {
			status = StatusHaveIngredients
		}
		_, err := tx.ExecContext(ctx, `
			UPDATE meal_plan_entries SET date = ?, slot = ?, position = ?, status = ?, updated_at = ?
			WHERE id = ?`,
			e.to.date, e.to.slot, e.to.position, status, now, e.id)
		if err != nil {
			return MoveResult{}, fmt.Errorf("eintrag %d verschieben: %w", e.id, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return MoveResult{}, fmt.Errorf("eintrag %d verschieben: %w", id, err)
	}

	result.Applied = true
	for _, e := range moving {
		entry, err := s.Get(ctx, e.id)
		if err != nil {
			return MoveResult{}, err
		}
		result.Entries = append(result.Entries, entry)
	}
	return result, nil
}

// loadMoving liest Platz, Status und Bezeichnung des Eintrags id.
func loadMoving(ctx context.Context, tx *sql.Tx, id int64) (*movingEntry, error) {
	e := &movingEntry{id: id}
	err := tx.QueryRowContext(ctx, `
		SELECT e.date, e.slot, e.position, e.status, COALESCE(r.title, e.note)
		FROM meal_plan_entries e LEFT JOIN recipes r ON r.id = e.recipe_id
		WHERE e.id = ?`, id).
		Scan(&e.from.date, &e.from.slot, &e.from.position, &e.status, &e.label)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("eintrag %d lesen: %w", id, err)
	}
	return e, nil
}

// checkMove wendet die Warnregeln auf einen Eintrag an. Ein Tausch
// innerhalb des Einkaufszeitraums löst keine Regel aus, weil beide
// Einträge im Zeitraum bleiben.
func checkMove(e *movingEntry, period Period) (Warning, bool) {
	wasIn, isIn := period.Contains(e.from.date), period.Contains(e.to.date)
	switch {
	case e.status == StatusPlanned && !wasIn && isIn:
		return Warning{WarnPlannedIntoPeriod, e.id,
			fmt.Sprintf("„%s“ rutscht in den aktuellen Einkaufszeitraum, die Zutaten sind aber noch nicht eingekauft.", e.label)}, true
	case e.status == StatusTransferred && wasIn && !isIn:
		return Warning{WarnTransferredOutOfPeriod, e.id,
			fmt.Sprintf("Die Zutaten für „%s“ sind schon eingekauft und halten sich vielleicht nicht so lange.", e.label)}, true
	}
	return Warning{}, false
}
