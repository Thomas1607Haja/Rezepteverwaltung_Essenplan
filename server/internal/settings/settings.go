// Package settings verwaltet die Einstellungen des Haushalts, z. B.
// Familiengröße und Einkaufstag. Sie liegen als Schlüssel-Wert-Paare in der
// Tabelle settings; fehlt ein Schlüssel, gilt der Standardwert.
package settings

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/database"
)

// Settings sind alle Einstellungen, wie sie die API liefert und annimmt.
type Settings struct {
	// FamilySize ist die Standard-Portionenzahl neuer Planeinträge.
	FamilySize int `json:"family_size"`
	// ShoppingDay ist der Wochentag des Einkaufs auf Englisch in
	// Kleinbuchstaben, z. B. "saturday". Mit ihm beginnt der
	// Einkaufszeitraum.
	ShoppingDay string `json:"shopping_day"`
	// BringEntity ist die To-do-Entität in Home Assistant, an die die
	// Einkaufsliste geht, z. B. "todo.zuhause"; leer = noch nicht gewählt.
	BringEntity string `json:"bring_entity"`
}

// Defaults gelten, solange nichts gespeichert ist.
var Defaults = Settings{FamilySize: 4, ShoppingDay: "saturday"}

// weekdays ordnet die Namen den Wochentagen von Go zu (Sonntag = 0).
var weekdays = map[string]time.Weekday{
	"sunday": time.Sunday, "monday": time.Monday, "tuesday": time.Tuesday,
	"wednesday": time.Wednesday, "thursday": time.Thursday,
	"friday": time.Friday, "saturday": time.Saturday,
}

// ShoppingWeekday liefert den Einkaufstag als time.Weekday.
func (s Settings) ShoppingWeekday() time.Weekday {
	return weekdays[s.ShoppingDay]
}

// ValidationError beschreibt eine ungültige Einstellung.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

func (s Settings) validate() error {
	if s.FamilySize < 1 || s.FamilySize > 50 {
		return &ValidationError{"family_size", "muss zwischen 1 und 50 liegen"}
	}
	// Bei einer Map liefert der zweite Rückgabewert, ob es den Schlüssel gibt.
	if _, ok := weekdays[s.ShoppingDay]; !ok {
		return &ValidationError{"shopping_day", "muss ein englischer wochentag sein, z. b. saturday"}
	}
	if s.BringEntity != "" && (!strings.HasPrefix(s.BringEntity, "todo.") || strings.ContainsAny(s.BringEntity, " ,")) {
		return &ValidationError{"bring_entity", "muss eine to-do-entität sein, z. b. todo.zuhause"}
	}
	return nil
}

// Store liest und schreibt die Einstellungen.
type Store struct {
	db *sql.DB
}

// NewStore erzeugt einen Store für die geöffnete Datenbank db.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Get liefert die Einstellungen; Fehlendes kommt aus Defaults.
func (s *Store) Get(ctx context.Context) (Settings, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT key, value FROM settings")
	if err != nil {
		return Settings{}, fmt.Errorf("einstellungen lesen: %w", err)
	}
	defer rows.Close()

	result := Defaults
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return Settings{}, fmt.Errorf("einstellungen lesen: %w", err)
		}
		switch key {
		case "family_size":
			// Ein unlesbarer Wert (z. B. von Hand geändert) fällt auf den
			// Standard zurück, statt die ganze Anwendung zu blockieren.
			if n, err := strconv.Atoi(value); err == nil && n >= 1 {
				result.FamilySize = n
			}
		case "shopping_day":
			if _, ok := weekdays[value]; ok {
				result.ShoppingDay = value
			}
		case "bring_entity":
			result.BringEntity = value
		}
	}
	if err := rows.Err(); err != nil {
		return Settings{}, fmt.Errorf("einstellungen lesen: %w", err)
	}
	return result, nil
}

// Update speichert alle Einstellungen und gibt den neuen Stand zurück.
func (s *Store) Update(ctx context.Context, in Settings) (Settings, error) {
	if err := in.validate(); err != nil {
		return Settings{}, err
	}
	now := database.FormatTime(time.Now())

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Settings{}, fmt.Errorf("einstellungen speichern: %w", err)
	}
	defer tx.Rollback()

	values := map[string]string{
		"family_size":  strconv.Itoa(in.FamilySize),
		"shopping_day": in.ShoppingDay,
		"bring_entity": in.BringEntity,
	}
	for key, value := range values {
		// "Upsert": neu anlegen oder, falls der Schlüssel schon existiert,
		// ändern. updated_at nur anfassen, wenn sich der Wert ändert –
		// sonst holt sich die App beim Abgleich unnötig Daten.
		_, err := tx.ExecContext(ctx, `
			INSERT INTO settings (key, value, updated_at) VALUES (?, ?, ?)
			ON CONFLICT (key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at
			WHERE value <> excluded.value`,
			key, value, now)
		if err != nil {
			return Settings{}, fmt.Errorf("einstellungen speichern: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return Settings{}, fmt.Errorf("einstellungen speichern: %w", err)
	}
	return s.Get(ctx)
}
