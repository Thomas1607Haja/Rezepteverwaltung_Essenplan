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
	// angehängt. Gehört er zu einem Menü, tauscht das ganze Menü.
	SwapWith *int64 `json:"swap_with"`
	// SwapWithMenu: Menü, mit dem getauscht wird.
	SwapWithMenu *int64 `json:"swap_with_menu"`
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
	Menus    []Menu    `json:"menus"` // verschobene Menüs (Schritt 9.4)
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

// block ist, was zusammen verschoben wird: ein einzelner Eintrag oder ein
// Menü mit allen Gängen (Schritt 9.4).
type block struct {
	menu     *int64
	date     string
	slot     string
	position int // Platz des Blocks im Slot (Menü bzw. Eintrag)
	entries  []*movingEntry
}

// Move verschiebt den Eintrag id oder tauscht ihn mit m.SwapWith bzw.
// m.SwapWithMenu. Gehört der Eintrag zu einem Menü, wandert das ganze Menü
// (Block). Gibt es Warnungen und ist m.Confirm nicht gesetzt, wird nichts
// gespeichert.
func (s *Store) Move(ctx context.Context, id int64, m Move) (MoveResult, error) {
	return s.moveBlock(ctx, func(tx *sql.Tx) (*block, error) { return loadEntryBlock(ctx, tx, id) }, m)
}

// MoveMenu verschiebt das Menü id mit allen Gängen (oder tauscht es).
func (s *Store) MoveMenu(ctx context.Context, id int64, m Move) (MoveResult, error) {
	return s.moveBlock(ctx, func(tx *sql.Tx) (*block, error) { return loadMenuBlock(ctx, tx, id) }, m)
}

func (s *Store) moveBlock(ctx context.Context, load func(*sql.Tx) (*block, error), m Move) (MoveResult, error) {
	if _, err := parseDate("date", m.Date); err != nil {
		return MoveResult{}, err
	}
	if m.Slot == "" {
		m.Slot = SlotDinner
	}
	if !validSlot(m.Slot) {
		return MoveResult{}, &ValidationError{"slot", slotError}
	}
	period, err := s.CurrentPeriod(ctx)
	if err != nil {
		return MoveResult{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return MoveResult{}, fmt.Errorf("verschieben: %w", err)
	}
	defer tx.Rollback()

	src, err := load(tx)
	if err != nil {
		return MoveResult{}, err
	}
	blocks := []*block{src}

	var other *block
	switch {
	case m.SwapWithMenu != nil:
		other, err = loadMenuBlock(ctx, tx, *m.SwapWithMenu)
	case m.SwapWith != nil:
		other, err = loadEntryBlock(ctx, tx, *m.SwapWith)
	}
	if errors.Is(err, ErrNotFound) || errors.Is(err, ErrMenuNotFound) {
		return MoveResult{}, &ValidationError{"swap_with", "diesen eintrag gibt es nicht"}
	}
	if err != nil {
		return MoveResult{}, err
	}
	if other != nil {
		if sameBlock(src, other) {
			return MoveResult{}, &ValidationError{"swap_with", "ein eintrag kann nicht mit sich selbst tauschen"}
		}
		// Der Tauschpartner muss dort stehen, wohin gezogen wurde – sonst
		// hat sich der Plan inzwischen geändert (z. B. auf einem anderen Gerät).
		if other.date != m.Date || other.slot != m.Slot {
			return MoveResult{}, &ValidationError{"swap_with", "der plan hat sich inzwischen geändert, bitte neu laden"}
		}
		// Erst den alten Platz merken: place überschreibt ihn.
		srcDate, srcSlot, srcPos := src.date, src.slot, src.position
		place(src, other.date, other.slot, other.position)
		place(other, srcDate, srcSlot, srcPos)
		blocks = append(blocks, other)
	} else {
		pos, err := nextPosition(ctx, tx, m.Date, m.Slot)
		if err != nil {
			return MoveResult{}, err
		}
		place(src, m.Date, m.Slot, pos)
	}

	result := MoveResult{Warnings: []Warning{}, Entries: []Entry{}, Menus: []Menu{}}
	for _, b := range blocks {
		for _, e := range b.entries {
			if w, ok := checkMove(e, period); ok {
				result.Warnings = append(result.Warnings, w)
			}
		}
	}
	if len(result.Warnings) > 0 && !m.Confirm {
		return result, nil // nichts speichern, erst bestätigen lassen
	}

	now := database.FormatTime(s.now())
	for _, b := range blocks {
		for _, e := range b.entries {
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
		if b.menu != nil {
			if _, err := tx.ExecContext(ctx, "UPDATE meal_plan_menus SET date = ?, slot = ?, position = ?, updated_at = ? WHERE id = ?",
				b.date, b.slot, b.position, now, *b.menu); err != nil {
				return MoveResult{}, fmt.Errorf("menü %d verschieben: %w", *b.menu, err)
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return MoveResult{}, fmt.Errorf("verschieben: %w", err)
	}

	result.Applied = true
	for _, b := range blocks {
		for _, e := range b.entries {
			entry, err := s.Get(ctx, e.id)
			if err != nil {
				return MoveResult{}, err
			}
			result.Entries = append(result.Entries, entry)
		}
		if b.menu != nil {
			menu, err := s.GetMenu(ctx, *b.menu)
			if err != nil {
				return MoveResult{}, err
			}
			result.Menus = append(result.Menus, menu)
		}
	}
	return result, nil
}

// place setzt das Ziel eines Blocks: Tag, Slot und Plätze ab pos in der
// bisherigen Reihenfolge. Danach beschreibt der Block seinen neuen Platz.
func place(b *block, date, slot string, pos int) {
	for i, e := range b.entries {
		e.to = placement{date, slot, pos + i}
	}
	b.date, b.slot, b.position = date, slot, pos
}

func sameBlock(a, b *block) bool {
	if a.menu != nil || b.menu != nil {
		return a.menu != nil && b.menu != nil && *a.menu == *b.menu
	}
	return a.entries[0].id == b.entries[0].id
}

// loadEntryBlock: der Eintrag id – bzw. sein ganzes Menü, falls er zu
// einem gehört.
func loadEntryBlock(ctx context.Context, tx *sql.Tx, id int64) (*block, error) {
	var menuID *int64
	err := tx.QueryRowContext(ctx, "SELECT menu_id FROM meal_plan_entries WHERE id = ?", id).Scan(&menuID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("eintrag %d lesen: %w", id, err)
	}
	if menuID != nil {
		return loadMenuBlock(ctx, tx, *menuID)
	}
	e, err := loadMoving(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	return &block{date: e.from.date, slot: e.from.slot, position: e.from.position, entries: []*movingEntry{e}}, nil
}

// loadMenuBlock: das Menü id mit seinen Gängen in Reihenfolge.
func loadMenuBlock(ctx context.Context, tx *sql.Tx, id int64) (*block, error) {
	b := &block{menu: &id}
	err := tx.QueryRowContext(ctx, "SELECT date, slot, position FROM meal_plan_menus WHERE id = ?", id).
		Scan(&b.date, &b.slot, &b.position)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrMenuNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("menü %d lesen: %w", id, err)
	}
	rows, err := tx.QueryContext(ctx, "SELECT id FROM meal_plan_entries WHERE menu_id = ? ORDER BY position, id", id)
	if err != nil {
		return nil, fmt.Errorf("menü %d lesen: %w", id, err)
	}
	var ids []int64
	for rows.Next() {
		var e int64
		if err := rows.Scan(&e); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, e)
	}
	rows.Close()
	for _, e := range ids {
		m, err := loadMoving(ctx, tx, e)
		if err != nil {
			return nil, err
		}
		b.entries = append(b.entries, m)
	}
	return b, nil
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
			fmt.Sprintf("„%s“ rutscht in die eingekaufte Woche, die Zutaten sind aber noch nicht eingekauft.", e.label)}, true
	case e.status == StatusTransferred && wasIn && !isIn:
		return Warning{WarnTransferredOutOfPeriod, e.id,
			fmt.Sprintf("Die Zutaten für „%s“ sind schon eingekauft und halten sich vielleicht nicht so lange.", e.label)}, true
	}
	return Warning{}, false
}
