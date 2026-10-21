// Package mealplan verwaltet den Essensplan: Einträge mit Datum und Slot,
// mit oder ohne Rezept, und den daraus berechneten Einkaufszeitraum.
package mealplan

import (
	"errors"
	"strings"
	"time"
)

// dateLayout ist das Format eines Kalendertags, z. B. 2026-10-24.
const dateLayout = "2006-01-02"

// Slots und Status, wie sie in der Datenbank und der API stehen.
const (
	SlotDinner = "dinner"
	SlotLunch  = "lunch"

	StatusPlanned         = "planned"
	StatusTransferred     = "transferred"
	StatusHaveIngredients = "have_ingredients"
)

// Entry ist ein Eintrag im Essensplan.
type Entry struct {
	ID       int64  `json:"id"`
	Date     string `json:"date"` // z. B. "2026-10-24"
	Slot     string `json:"slot"` // "dinner" oder "lunch"
	Position int    `json:"position"`
	RecipeID *int64 `json:"recipe_id"` // nil = Eintrag ohne Rezept
	// Recipe enthält das Nötigste für die Anzeige im Plan; nur lesbar.
	Recipe        *RecipeInfo `json:"recipe"`
	Servings      int         `json:"servings"`
	Status        string      `json:"status"`
	TransferredAt *time.Time  `json:"transferred_at"`
	Note          string      `json:"note"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
}

// RecipeInfo sind die Angaben zum Rezept eines Eintrags.
type RecipeInfo struct {
	Title      string `json:"title"`
	Thumbnail  string `json:"thumbnail"`
	IsFavorite bool   `json:"is_favorite"`
}

// NewEntry sind die Angaben zum Anlegen eines Eintrags. Fehlt Slot, gilt
// Abendessen; fehlen die Portionen, gilt die Familiengröße.
type NewEntry struct {
	Date     string `json:"date"`
	Slot     string `json:"slot"`
	RecipeID *int64 `json:"recipe_id"`
	Servings *int   `json:"servings"`
	Note     string `json:"note"`
}

// Changes sind die Angaben, die sich an einem Eintrag direkt ändern lassen.
// Datum und Slot ändern sich nur übers Verschieben (mit Warnprüfung).
type Changes struct {
	RecipeID *int64 `json:"recipe_id"`
	Servings int    `json:"servings"`
	Note     string `json:"note"`
	// Status: "planned" oder "have_ingredients". "transferred" setzt nur
	// die Übertragung an Bring.
	Status string `json:"status"`
}

// Period ist ein Zeitraum von From bis To, beide Tage eingeschlossen.
type Period struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// ErrNotFound: Den Eintrag gibt es nicht.
var ErrNotFound = errors.New("eintrag nicht gefunden")

// ValidationError beschreibt eine ungültige Eingabe.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

// parseDate prüft einen Kalendertag wie "2026-10-24". time.Parse lehnt
// auch Tage ab, die es nicht gibt (z. B. 2026-02-30).
func parseDate(field, s string) (time.Time, error) {
	t, err := time.Parse(dateLayout, s)
	if err != nil {
		return time.Time{}, &ValidationError{field, "muss ein datum wie 2026-10-24 sein"}
	}
	return t, nil
}

func validSlot(slot string) bool {
	return slot == SlotDinner || slot == SlotLunch
}

// normalizeNote entfernt Leerzeichen am Rand und begrenzt die Länge.
func normalizeNote(note string) (string, error) {
	note = strings.TrimSpace(note)
	if len([]rune(note)) > 200 {
		return "", &ValidationError{"note", "höchstens 200 zeichen"}
	}
	return note, nil
}
