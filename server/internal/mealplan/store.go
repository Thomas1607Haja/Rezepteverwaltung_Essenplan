package mealplan

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/database"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/settings"
)

// maxRangeDays begrenzt den Zeitraum einer Abfrage (gut ein Jahr).
const maxRangeDays = 400

// Store liest und schreibt den Essensplan.
type Store struct {
	db       *sql.DB
	settings *settings.Store
	// now liefert die aktuelle Zeit. Tests setzen hier einen festen
	// Zeitpunkt ein, damit "heute" nicht vom Testtag abhängt.
	now func() time.Time
}

// NewStore erzeugt einen Store. settings liefert Familiengröße und
// Einkaufstag.
func NewStore(db *sql.DB, settings *settings.Store) *Store {
	return &Store{db: db, settings: settings, now: time.Now}
}

// SetClock ersetzt die Uhr. Gedacht für Tests anderer Pakete, die einen
// festen Tag als "heute" brauchen.
func (s *Store) SetClock(now func() time.Time) {
	s.now = now
}

// Today ist der heutige Kalendertag in der Zeitzone des Servers (in Home
// Assistant über die Umgebungsvariable TZ eingestellt).
func (s *Store) Today() time.Time {
	y, m, d := s.now().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// CurrentPeriod ist der Einkaufszeitraum, in dem heute liegt.
func (s *Store) CurrentPeriod(ctx context.Context) (Period, error) {
	st, err := s.settings.Get(ctx)
	if err != nil {
		return Period{}, err
	}
	return ShoppingPeriod(s.Today(), st.ShoppingWeekday()), nil
}

// selectEntry ist der gemeinsame Anfang aller Abfragen; scanEntry liest
// eine Zeile davon. Der LEFT JOIN liefert auch Einträge ohne Rezept (die
// Rezeptspalten sind dann NULL).
const selectEntry = `
	SELECT e.id, e.date, e.slot, e.position, e.recipe_id, e.servings, e.status,
	       e.transferred_at, e.note, e.created_at, e.updated_at,
	       r.title, r.thumbnail, r.is_favorite
	FROM meal_plan_entries e
	LEFT JOIN recipes r ON r.id = e.recipe_id`

// scanner passt sowohl auf *sql.Row als auch auf *sql.Rows: Beide haben
// eine Methode Scan mit dieser Signatur.
type scanner interface {
	Scan(dest ...any) error
}

func scanEntry(row scanner) (Entry, error) {
	var e Entry
	var transferredAt, title, thumbnail sql.NullString
	var isFavorite sql.NullBool
	var createdAt, updatedAt string
	err := row.Scan(&e.ID, &e.Date, &e.Slot, &e.Position, &e.RecipeID, &e.Servings, &e.Status,
		&transferredAt, &e.Note, &createdAt, &updatedAt,
		&title, &thumbnail, &isFavorite)
	if err != nil {
		return Entry{}, err
	}
	if e.CreatedAt, err = database.ParseTime(createdAt); err != nil {
		return Entry{}, err
	}
	if e.UpdatedAt, err = database.ParseTime(updatedAt); err != nil {
		return Entry{}, err
	}
	if transferredAt.Valid {
		t, err := database.ParseTime(transferredAt.String)
		if err != nil {
			return Entry{}, err
		}
		e.TransferredAt = &t
	}
	if title.Valid {
		e.Recipe = &RecipeInfo{Title: title.String, Thumbnail: thumbnail.String, IsFavorite: isFavorite.Bool}
	}
	return e, nil
}

// List liefert alle Einträge von from bis to (beide eingeschlossen),
// sortiert nach Datum, Slot (Mittag vor Abend) und Reihenfolge.
func (s *Store) List(ctx context.Context, from, to string) ([]Entry, error) {
	fromDate, err := parseDate("from", from)
	if err != nil {
		return nil, err
	}
	toDate, err := parseDate("to", to)
	if err != nil {
		return nil, err
	}
	if toDate.Before(fromDate) {
		return nil, &ValidationError{"to", "darf nicht vor from liegen"}
	}
	if toDate.Sub(fromDate) > maxRangeDays*24*time.Hour {
		return nil, &ValidationError{"to", fmt.Sprintf("zeitraum höchstens %d tage", maxRangeDays)}
	}

	// "lunch" < "dinner" gilt alphabetisch nicht, daher die CASE-Sortierung.
	rows, err := s.db.QueryContext(ctx, selectEntry+`
		WHERE e.date BETWEEN ? AND ?
		ORDER BY e.date, CASE e.slot WHEN 'lunch' THEN 0 ELSE 1 END, e.position, e.id`,
		from, to)
	if err != nil {
		return nil, fmt.Errorf("essensplan lesen: %w", err)
	}
	defer rows.Close()

	entries := []Entry{} // leer statt nil, damit JSON [] statt null zeigt
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return nil, fmt.Errorf("essensplan lesen: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("essensplan lesen: %w", err)
	}
	return entries, nil
}

// ChangedSince liefert für den Abgleich mit der App alle Einträge ab dem
// Tag from (ohne Ende), deren updated_at nicht vor since liegt
// (Datenbankformat, leer = alle).
func (s *Store) ChangedSince(ctx context.Context, from, since string) ([]Entry, error) {
	rows, err := s.db.QueryContext(ctx, selectEntry+`
		WHERE e.date >= ? AND e.updated_at >= ?
		ORDER BY e.date, CASE e.slot WHEN 'lunch' THEN 0 ELSE 1 END, e.position, e.id`,
		from, since)
	if err != nil {
		return nil, fmt.Errorf("essensplan für den abgleich lesen: %w", err)
	}
	defer rows.Close()

	entries := []Entry{}
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return nil, fmt.Errorf("essensplan für den abgleich lesen: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("essensplan für den abgleich lesen: %w", err)
	}
	return entries, nil
}

// Get liefert den Eintrag id oder ErrNotFound.
func (s *Store) Get(ctx context.Context, id int64) (Entry, error) {
	e, err := scanEntry(s.db.QueryRowContext(ctx, selectEntry+" WHERE e.id = ?", id))
	if errors.Is(err, sql.ErrNoRows) {
		return Entry{}, ErrNotFound
	}
	if err != nil {
		return Entry{}, fmt.Errorf("eintrag %d lesen: %w", id, err)
	}
	return e, nil
}

// querier passt auf *sql.DB und *sql.Tx; so lassen sich die Prüfungen
// innerhalb und außerhalb einer Transaktion verwenden.
type querier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// checkRecipe prüft, ob es das Rezept recipeID gibt (nil ist erlaubt).
func checkRecipe(ctx context.Context, q querier, recipeID *int64) error {
	if recipeID == nil {
		return nil
	}
	var exists bool
	err := q.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM recipes WHERE id = ?)", *recipeID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("rezept prüfen: %w", err)
	}
	if !exists {
		return &ValidationError{"recipe_id", "dieses rezept gibt es nicht"}
	}
	return nil
}

// nextPosition liefert die Position hinter dem letzten Eintrag in date/slot.
func nextPosition(ctx context.Context, q querier, date, slot string) (int, error) {
	var pos int
	err := q.QueryRowContext(ctx,
		"SELECT COALESCE(MAX(position) + 1, 0) FROM meal_plan_entries WHERE date = ? AND slot = ?",
		date, slot).Scan(&pos)
	if err != nil {
		return 0, fmt.Errorf("position bestimmen: %w", err)
	}
	return pos, nil
}

// Create legt einen Eintrag am Ende von Datum und Slot an.
func (s *Store) Create(ctx context.Context, in NewEntry) (Entry, error) {
	if _, err := parseDate("date", in.Date); err != nil {
		return Entry{}, err
	}
	if in.Slot == "" {
		in.Slot = SlotDinner
	}
	if !validSlot(in.Slot) {
		return Entry{}, &ValidationError{"slot", "muss dinner oder lunch sein"}
	}
	note, err := normalizeNote(in.Note)
	if err != nil {
		return Entry{}, err
	}
	if in.RecipeID == nil && note == "" {
		return Entry{}, &ValidationError{"note", "ohne rezept ist eine notiz nötig, z. b. „essen gehen“"}
	}

	servings := 0
	if in.Servings != nil {
		servings = *in.Servings
	} else {
		st, err := s.settings.Get(ctx)
		if err != nil {
			return Entry{}, err
		}
		servings = st.FamilySize
	}
	if servings < 1 || servings > 50 {
		return Entry{}, &ValidationError{"servings", "muss zwischen 1 und 50 liegen"}
	}

	now := database.FormatTime(s.now())
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Entry{}, fmt.Errorf("eintrag anlegen: %w", err)
	}
	defer tx.Rollback()

	if err := checkRecipe(ctx, tx, in.RecipeID); err != nil {
		return Entry{}, err
	}
	pos, err := nextPosition(ctx, tx, in.Date, in.Slot)
	if err != nil {
		return Entry{}, err
	}
	res, err := tx.ExecContext(ctx, `
		INSERT INTO meal_plan_entries (date, slot, position, recipe_id, servings, note, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		in.Date, in.Slot, pos, in.RecipeID, servings, note, now, now)
	if err != nil {
		return Entry{}, fmt.Errorf("eintrag anlegen: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Entry{}, fmt.Errorf("eintrag anlegen: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Entry{}, fmt.Errorf("eintrag anlegen: %w", err)
	}
	return s.Get(ctx, id)
}

// Update ändert Rezept, Portionen, Notiz und Status des Eintrags id.
func (s *Store) Update(ctx context.Context, id int64, c Changes) (Entry, error) {
	note, err := normalizeNote(c.Note)
	if err != nil {
		return Entry{}, err
	}
	if c.RecipeID == nil && note == "" {
		return Entry{}, &ValidationError{"note", "ohne rezept ist eine notiz nötig, z. b. „essen gehen“"}
	}
	if c.Servings < 1 || c.Servings > 50 {
		return Entry{}, &ValidationError{"servings", "muss zwischen 1 und 50 liegen"}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Entry{}, fmt.Errorf("eintrag %d ändern: %w", id, err)
	}
	defer tx.Rollback()

	var oldStatus string
	err = tx.QueryRowContext(ctx, "SELECT status FROM meal_plan_entries WHERE id = ?", id).Scan(&oldStatus)
	if errors.Is(err, sql.ErrNoRows) {
		return Entry{}, ErrNotFound
	}
	if err != nil {
		return Entry{}, fmt.Errorf("eintrag %d ändern: %w", id, err)
	}
	// "transferred" entsteht nur durch die Übertragung an Bring; ein schon
	// übertragener Eintrag darf den Status aber behalten.
	switch c.Status {
	case StatusPlanned, StatusHaveIngredients:
	case StatusTransferred:
		if oldStatus != StatusTransferred {
			return Entry{}, &ValidationError{"status", "„transferred“ setzt nur die übertragung an bring"}
		}
	default:
		return Entry{}, &ValidationError{"status", "muss planned oder have_ingredients sein"}
	}
	if err := checkRecipe(ctx, tx, c.RecipeID); err != nil {
		return Entry{}, err
	}

	// Wer den Status zurücksetzt, löscht auch den Zeitpunkt der Übertragung.
	_, err = tx.ExecContext(ctx, `
		UPDATE meal_plan_entries
		SET recipe_id = ?, servings = ?, note = ?, status = ?,
		    transferred_at = CASE WHEN ? = 'transferred' THEN transferred_at END,
		    updated_at = ?
		WHERE id = ?`,
		c.RecipeID, c.Servings, note, c.Status, c.Status, database.FormatTime(s.now()), id)
	if err != nil {
		return Entry{}, fmt.Errorf("eintrag %d ändern: %w", id, err)
	}
	if err := tx.Commit(); err != nil {
		return Entry{}, fmt.Errorf("eintrag %d ändern: %w", id, err)
	}
	return s.Get(ctx, id)
}

// Delete löscht den Eintrag id und vermerkt das für den Abgleich mit der
// App in deletions.
func (s *Store) Delete(ctx context.Context, id int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("eintrag %d löschen: %w", id, err)
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, "DELETE FROM meal_plan_entries WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("eintrag %d löschen: %w", id, err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO deletions (table_name, record_id, deleted_at)
		VALUES ('meal_plan_entries', ?, ?)`, id, database.FormatTime(s.now()))
	if err != nil {
		return fmt.Errorf("eintrag %d löschen: %w", id, err)
	}
	return tx.Commit()
}
