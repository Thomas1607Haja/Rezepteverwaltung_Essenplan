// Package pantry verwaltet die Vorratsliste: Zutaten, die immer im Haus
// sind (z. B. Salz, Öl). Beim Anlegen und Importieren von Rezepten werden
// sie als nicht einkaufsrelevant vorgeschlagen; die Einkaufsliste selbst
// richtet sich nur nach dem Häkchen an der Zutat.
package pantry

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/database"
)

// Item ist ein Vorratsartikel mit seinen Synonymen.
type Item struct {
	ID       int64    `json:"id"`
	Name     string   `json:"name"`
	Synonyms []string `json:"synonyms"` // z. B. ["Olivenöl", "Rapsöl"] zu "Öl"
}

// ErrNotFound wird zurückgegeben, wenn es einen Vorratsartikel nicht gibt.
var ErrNotFound = errors.New("vorratsartikel nicht gefunden")

// ValidationError beschreibt eine ungültige Eingabe.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

// normalize entfernt Leerzeichen, leere und doppelte Synonyme.
func (it *Item) normalize() {
	it.Name = strings.Join(strings.Fields(it.Name), " ")
	seen := map[string]bool{database.Fold(it.Name): true}
	synonyms := []string{}
	for _, syn := range it.Synonyms {
		syn = strings.Join(strings.Fields(syn), " ")
		key := database.Fold(syn)
		if syn == "" || seen[key] {
			continue
		}
		seen[key] = true
		synonyms = append(synonyms, syn)
	}
	it.Synonyms = synonyms
}

func (it *Item) validate() error {
	if it.Name == "" {
		return &ValidationError{"name", "darf nicht leer sein"}
	}
	// Synonyme werden kommagetrennt gespeichert, ein Komma im Namen würde
	// sie beim Lesen zerteilen.
	if strings.Contains(it.Name, ",") {
		return &ValidationError{"name", "darf kein komma enthalten"}
	}
	for i, syn := range it.Synonyms {
		if strings.Contains(syn, ",") {
			return &ValidationError{fmt.Sprintf("synonyms[%d]", i), "darf kein komma enthalten"}
		}
	}
	return nil
}

// splitSynonyms zerlegt die gespeicherte, kommagetrennte Form.
func splitSynonyms(s string) []string {
	result := []string{}
	for _, syn := range strings.Split(s, ",") {
		if syn = strings.TrimSpace(syn); syn != "" {
			result = append(result, syn)
		}
	}
	return result
}

// Store liest und schreibt die Vorratsliste.
type Store struct {
	db *sql.DB
}

// NewStore erzeugt einen Store für die geöffnete Datenbank db.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// List liefert alle Vorratsartikel alphabetisch (Umlaute wie Grundbuchstaben).
func (s *Store) List(ctx context.Context) ([]Item, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, synonyms FROM pantry_items ORDER BY fold(name), id`)
	if err != nil {
		return nil, fmt.Errorf("vorratsliste laden: %w", err)
	}
	defer rows.Close()

	items := []Item{}
	for rows.Next() {
		var it Item
		var synonyms string
		if err := rows.Scan(&it.ID, &it.Name, &synonyms); err != nil {
			return nil, fmt.Errorf("vorratsliste laden: %w", err)
		}
		it.Synonyms = splitSynonyms(synonyms)
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("vorratsliste laden: %w", err)
	}
	return items, nil
}

// Get lädt einen Vorratsartikel. Gibt es ihn nicht, ist der Fehler ErrNotFound.
func (s *Store) Get(ctx context.Context, id int64) (Item, error) {
	it := Item{ID: id}
	var synonyms string
	err := s.db.QueryRowContext(ctx, "SELECT name, synonyms FROM pantry_items WHERE id = ?", id).
		Scan(&it.Name, &synonyms)
	if errors.Is(err, sql.ErrNoRows) {
		return Item{}, ErrNotFound
	}
	if err != nil {
		return Item{}, fmt.Errorf("vorratsartikel %d laden: %w", id, err)
	}
	it.Synonyms = splitSynonyms(synonyms)
	return it, nil
}

// Create legt einen Vorratsartikel an.
func (s *Store) Create(ctx context.Context, it Item) (Item, error) {
	it.normalize()
	if err := it.validate(); err != nil {
		return Item{}, err
	}
	if err := s.checkUnique(ctx, 0, it.Name); err != nil {
		return Item{}, err
	}
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO pantry_items (name, synonyms, updated_at) VALUES (?, ?, ?)`,
		it.Name, strings.Join(it.Synonyms, ", "), database.FormatTime(time.Now()))
	if err != nil {
		return Item{}, fmt.Errorf("vorratsartikel anlegen: %w", err)
	}
	if it.ID, err = res.LastInsertId(); err != nil {
		return Item{}, fmt.Errorf("vorratsartikel anlegen: %w", err)
	}
	return it, nil
}

// Update ersetzt Name und Synonyme des Vorratsartikels id. Bestehende
// Rezepte bleiben unverändert.
func (s *Store) Update(ctx context.Context, id int64, it Item) (Item, error) {
	it.normalize()
	if err := it.validate(); err != nil {
		return Item{}, err
	}
	if err := s.checkUnique(ctx, id, it.Name); err != nil {
		return Item{}, err
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE pantry_items SET name = ?, synonyms = ?, updated_at = ? WHERE id = ?`,
		it.Name, strings.Join(it.Synonyms, ", "), database.FormatTime(time.Now()), id)
	if err != nil {
		return Item{}, fmt.Errorf("vorratsartikel %d ändern: %w", id, err)
	}
	if err := requireRow(res); err != nil {
		return Item{}, err
	}
	it.ID = id
	return it, nil
}

// Delete entfernt den Vorratsartikel id und vermerkt das für den Abgleich
// mit der App.
func (s *Store) Delete(ctx context.Context, id int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("vorratsartikel %d löschen: %w", id, err)
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, "DELETE FROM pantry_items WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("vorratsartikel %d löschen: %w", id, err)
	}
	if err := requireRow(res); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO deletions (table_name, record_id, deleted_at) VALUES ('pantry_items', ?, ?)`,
		id, database.FormatTime(time.Now()))
	if err != nil {
		return fmt.Errorf("vorratsartikel %d löschen: %w", id, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("vorratsartikel %d löschen: %w", id, err)
	}
	return nil
}

// checkUnique verhindert zwei Artikel mit gleichem Namen (ohne Rücksicht
// auf Groß-/Kleinschreibung und Umlaute). exceptID ist der Artikel selbst
// beim Ändern, 0 beim Anlegen.
func (s *Store) checkUnique(ctx context.Context, exceptID int64, name string) error {
	var n int
	err := s.db.QueryRowContext(ctx, `
		SELECT count(*) FROM pantry_items WHERE fold(name) = ? AND id <> ?`,
		database.Fold(name), exceptID).Scan(&n)
	if err != nil {
		return fmt.Errorf("vorratsliste prüfen: %w", err)
	}
	if n > 0 {
		return &ValidationError{"name", "steht schon auf der vorratsliste"}
	}
	return nil
}

// requireRow liefert ErrNotFound, wenn eine Anweisung keine Zeile traf.
func requireRow(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("betroffene zeilen ermitteln: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
