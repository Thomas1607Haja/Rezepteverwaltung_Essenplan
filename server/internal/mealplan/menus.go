package mealplan

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/database"
)

// Menüs (Schritt 9.4, docs/PLANUNG.md Phase 9): eine Klammer um normale
// Planeinträge. Die Gerichte bleiben eigene Einträge mit menu_id; das Menü
// hält Titel, Notiz und Portionen und steht an Tag und Slot seiner
// Gerichte. Verschoben und getauscht wird immer der ganze Block
// (move.go); ein einzelner Gang lässt sich „aus dem Menü lösen“.

// ErrMenuNotFound: Das Menü gibt es nicht.
var ErrMenuNotFound = errors.New("menü nicht gefunden")

// Menu ist ein Menü im Essensplan.
type Menu struct {
	ID       int64  `json:"id"`
	Date     string `json:"date"`
	Slot     string `json:"slot"`
	Position int    `json:"position"`
	// Title: vom Nutzer vergeben; leer = automatisch (DisplayTitle).
	Title string `json:"title"`
	// DisplayTitle: Title oder der Titel des Hauptgerichts bzw. des ersten
	// Gerichts; „Menü“, wenn es noch keine Gänge gibt. Nur lesbar.
	DisplayTitle string    `json:"display_title"`
	Note         string    `json:"note"`
	Servings     int       `json:"servings"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// MenuInput sind die Angaben zum Anlegen bzw. Ändern eines Menüs.
type MenuInput struct {
	Date     string `json:"date"`
	Slot     string `json:"slot"`
	Title    string `json:"title"`
	Note     string `json:"note"`
	Servings *int   `json:"servings"`
	// ApplyServings: beim Ändern die Portionen auf alle Gänge übertragen.
	ApplyServings bool `json:"apply_servings"`
}

// nonZero macht aus 0 ein NULL für die Datenbank (z. B. „kein Gang“).
func nonZero(p *int64) any {
	if p == nil || *p == 0 {
		return nil
	}
	return *p
}

// selectMenu liest Menüs samt automatischem Titel. Der Hauptgang ist der
// Gang mit der Kategorie „Hauptgericht“, sonst das erste Gericht.
const selectMenu = `
	SELECT m.id, m.date, m.slot, m.position, m.title, m.note, m.servings, m.created_at, m.updated_at,
	       coalesce(nullif(m.title, ''),
	                (SELECT coalesce(r.title, e.note) FROM meal_plan_entries e
	                 LEFT JOIN recipes r ON r.id = e.recipe_id
	                 LEFT JOIN tags t ON t.id = e.course_tag_id
	                 WHERE e.menu_id = m.id
	                 ORDER BY CASE WHEN lower(t.name) = 'hauptgericht' THEN 0 ELSE 1 END, e.position, e.id
	                 LIMIT 1),
	                'Menü')
	FROM meal_plan_menus m`

func scanMenu(row scanner) (Menu, error) {
	var m Menu
	var createdAt, updatedAt string
	if err := row.Scan(&m.ID, &m.Date, &m.Slot, &m.Position, &m.Title, &m.Note, &m.Servings,
		&createdAt, &updatedAt, &m.DisplayTitle); err != nil {
		return Menu{}, err
	}
	var err error
	if m.CreatedAt, err = database.ParseTime(createdAt); err != nil {
		return Menu{}, err
	}
	if m.UpdatedAt, err = database.ParseTime(updatedAt); err != nil {
		return Menu{}, err
	}
	return m, nil
}

func (s *Store) queryMenus(ctx context.Context, where string, args ...any) ([]Menu, error) {
	rows, err := s.db.QueryContext(ctx, selectMenu+" WHERE "+where+" ORDER BY m.date, m.position, m.id", args...)
	if err != nil {
		return nil, fmt.Errorf("menüs lesen: %w", err)
	}
	defer rows.Close()
	list := []Menu{}
	for rows.Next() {
		m, err := scanMenu(rows)
		if err != nil {
			return nil, fmt.Errorf("menüs lesen: %w", err)
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

// ListMenus liefert die Menüs von from bis to.
func (s *Store) ListMenus(ctx context.Context, from, to string) ([]Menu, error) {
	return s.queryMenus(ctx, "m.date BETWEEN ? AND ?", from, to)
}

// MenusChangedSince liefert für den Abgleich die Menüs ab from, deren
// updated_at nicht vor since liegt. Ein neuer Gang ändert den
// automatischen Titel; deshalb kommen auch Menüs mit geänderten Gängen.
func (s *Store) MenusChangedSince(ctx context.Context, from, since string) ([]Menu, error) {
	return s.queryMenus(ctx, `m.date >= ? AND (m.updated_at >= ?
		OR EXISTS (SELECT 1 FROM meal_plan_entries e WHERE e.menu_id = m.id AND e.updated_at >= ?))`, from, since, since)
}

// GetMenu liefert das Menü id oder ErrMenuNotFound.
func (s *Store) GetMenu(ctx context.Context, id int64) (Menu, error) {
	m, err := scanMenu(s.db.QueryRowContext(ctx, selectMenu+" WHERE m.id = ?", id))
	if errors.Is(err, sql.ErrNoRows) {
		return Menu{}, ErrMenuNotFound
	}
	if err != nil {
		return Menu{}, fmt.Errorf("menü %d lesen: %w", id, err)
	}
	return m, nil
}

// checkMenuText prüft Titel und Notiz.
func checkMenuText(title, note string) (string, string, error) {
	title = strings.TrimSpace(title)
	note = strings.TrimSpace(note)
	if len([]rune(title)) > 200 {
		return "", "", &ValidationError{"title", "höchstens 200 zeichen"}
	}
	if len([]rune(note)) > 2000 {
		return "", "", &ValidationError{"note", "höchstens 2000 zeichen"}
	}
	return title, note, nil
}

// CreateMenu legt ein (noch leeres) Menü am Ende von Tag und Slot an.
// Fehlen die Portionen, gilt die Familiengröße.
func (s *Store) CreateMenu(ctx context.Context, in MenuInput) (Menu, error) {
	if _, err := parseDate("date", in.Date); err != nil {
		return Menu{}, err
	}
	if in.Slot == "" {
		in.Slot = SlotDinner
	}
	if !validSlot(in.Slot) {
		return Menu{}, &ValidationError{"slot", slotError}
	}
	title, note, err := checkMenuText(in.Title, in.Note)
	if err != nil {
		return Menu{}, err
	}
	servings := 0
	if in.Servings != nil {
		servings = *in.Servings
	} else {
		st, err := s.settings.Get(ctx)
		if err != nil {
			return Menu{}, err
		}
		servings = st.FamilySize
	}
	if servings < 1 || servings > 50 {
		return Menu{}, &ValidationError{"servings", "muss zwischen 1 und 50 liegen"}
	}
	now := database.FormatTime(s.now())
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Menu{}, fmt.Errorf("menü anlegen: %w", err)
	}
	defer tx.Rollback()
	pos, err := nextPosition(ctx, tx, in.Date, in.Slot)
	if err != nil {
		return Menu{}, err
	}
	res, err := tx.ExecContext(ctx, `
		INSERT INTO meal_plan_menus (date, slot, position, title, note, servings, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, in.Date, in.Slot, pos, title, note, servings, now, now)
	if err != nil {
		return Menu{}, fmt.Errorf("menü anlegen: %w", err)
	}
	id, _ := res.LastInsertId()
	if err := tx.Commit(); err != nil {
		return Menu{}, fmt.Errorf("menü anlegen: %w", err)
	}
	return s.GetMenu(ctx, id)
}

// UpdateMenu ändert Titel, Notiz und Portionen; mit ApplyServings bekommen
// alle Gänge mit Rezept die neuen Portionen.
func (s *Store) UpdateMenu(ctx context.Context, id int64, in MenuInput) (Menu, error) {
	title, note, err := checkMenuText(in.Title, in.Note)
	if err != nil {
		return Menu{}, err
	}
	if in.Servings == nil || *in.Servings < 1 || *in.Servings > 50 {
		return Menu{}, &ValidationError{"servings", "muss zwischen 1 und 50 liegen"}
	}
	now := database.FormatTime(s.now())
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Menu{}, fmt.Errorf("menü %d ändern: %w", id, err)
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, "UPDATE meal_plan_menus SET title = ?, note = ?, servings = ?, updated_at = ? WHERE id = ?",
		title, note, *in.Servings, now, id)
	if err != nil {
		return Menu{}, fmt.Errorf("menü %d ändern: %w", id, err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return Menu{}, ErrMenuNotFound
	}
	if in.ApplyServings {
		// Nur Gänge mit Portionen: Ein Kuchen bleibt „1 Springform“.
		if _, err := tx.ExecContext(ctx, `UPDATE meal_plan_entries SET servings = ?, updated_at = ?
			WHERE menu_id = ? AND recipe_id IN (SELECT id FROM recipes WHERE yield_unit = '')`, *in.Servings, now, id); err != nil {
			return Menu{}, fmt.Errorf("menü %d ändern: %w", id, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return Menu{}, fmt.Errorf("menü %d ändern: %w", id, err)
	}
	return s.GetMenu(ctx, id)
}

// DeleteMenu löscht das Menü. withEntries: die Gänge mit; sonst bleiben
// sie als einzelne Einträge stehen („Menü auflösen“).
func (s *Store) DeleteMenu(ctx context.Context, id int64, withEntries bool) error {
	now := database.FormatTime(s.now())
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("menü %d löschen: %w", id, err)
	}
	defer tx.Rollback()
	if withEntries {
		rows, err := tx.QueryContext(ctx, "SELECT id FROM meal_plan_entries WHERE menu_id = ?", id)
		if err != nil {
			return fmt.Errorf("menü %d löschen: %w", id, err)
		}
		var ids []int64
		for rows.Next() {
			var e int64
			rows.Scan(&e)
			ids = append(ids, e)
		}
		rows.Close()
		for _, e := range ids {
			if _, err := tx.ExecContext(ctx, "DELETE FROM meal_plan_entries WHERE id = ?", e); err != nil {
				return fmt.Errorf("menü %d löschen: %w", id, err)
			}
			if _, err := tx.ExecContext(ctx, "INSERT INTO deletions (table_name, record_id, deleted_at) VALUES ('meal_plan_entries', ?, ?)", e, now); err != nil {
				return fmt.Errorf("menü %d löschen: %w", id, err)
			}
		}
	} else if _, err := tx.ExecContext(ctx, "UPDATE meal_plan_entries SET menu_id = NULL, updated_at = ? WHERE menu_id = ?", now, id); err != nil {
		return fmt.Errorf("menü %d auflösen: %w", id, err)
	}
	res, err := tx.ExecContext(ctx, "DELETE FROM meal_plan_menus WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("menü %d löschen: %w", id, err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrMenuNotFound
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO deletions (table_name, record_id, deleted_at) VALUES ('meal_plan_menus', ?, ?)", id, now); err != nil {
		return fmt.Errorf("menü %d löschen: %w", id, err)
	}
	return tx.Commit()
}

// ToMenu macht aus dem Eintrag id ein Menü („Zu einem Menü erweitern“): Das
// Menü übernimmt Tag, Slot, Platz und Portionen, der Eintrag wird sein
// erster Gang (Hauptgericht, falls courseTagID > 0).
func (s *Store) ToMenu(ctx context.Context, id int64, title, note string, courseTagID int64) (Menu, error) {
	title, note, err := checkMenuText(title, note)
	if err != nil {
		return Menu{}, err
	}
	now := database.FormatTime(s.now())
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Menu{}, fmt.Errorf("menü anlegen: %w", err)
	}
	defer tx.Rollback()
	var date, slot string
	var pos, servings int
	var menuID *int64
	err = tx.QueryRowContext(ctx, "SELECT date, slot, position, servings, menu_id FROM meal_plan_entries WHERE id = ?", id).
		Scan(&date, &slot, &pos, &servings, &menuID)
	if errors.Is(err, sql.ErrNoRows) {
		return Menu{}, ErrNotFound
	}
	if err != nil {
		return Menu{}, fmt.Errorf("menü anlegen: %w", err)
	}
	if menuID != nil {
		return Menu{}, &ValidationError{"menu_id", "der eintrag gehört schon zu einem menü"}
	}
	res, err := tx.ExecContext(ctx, `
		INSERT INTO meal_plan_menus (date, slot, position, title, note, servings, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, date, slot, pos, title, note, servings, now, now)
	if err != nil {
		return Menu{}, fmt.Errorf("menü anlegen: %w", err)
	}
	newID, _ := res.LastInsertId()
	course := courseTagID
	if _, err := tx.ExecContext(ctx, "UPDATE meal_plan_entries SET menu_id = ?, course_tag_id = coalesce(?, course_tag_id), updated_at = ? WHERE id = ?",
		newID, nonZero(&course), now, id); err != nil {
		return Menu{}, fmt.Errorf("menü anlegen: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Menu{}, fmt.Errorf("menü anlegen: %w", err)
	}
	return s.GetMenu(ctx, newID)
}

// Detach löst den Eintrag id aus seinem Menü; er bleibt an seinem Platz.
func (s *Store) Detach(ctx context.Context, id int64) (Entry, error) {
	res, err := s.db.ExecContext(ctx, "UPDATE meal_plan_entries SET menu_id = NULL, updated_at = ? WHERE id = ?",
		database.FormatTime(s.now()), id)
	if err != nil {
		return Entry{}, fmt.Errorf("eintrag %d lösen: %w", id, err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return Entry{}, ErrNotFound
	}
	return s.Get(ctx, id)
}

// MenuAllAvailable setzt alle geplanten Gänge des Menüs auf „Zutaten
// vorhanden“ (an Bring Gesendetes bleibt, wie es ist).
func (s *Store) MenuAllAvailable(ctx context.Context, id int64) error {
	if _, err := s.GetMenu(ctx, id); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `UPDATE meal_plan_entries SET status = ?, excluded_ingredients = '[]', updated_at = ?
		WHERE menu_id = ? AND recipe_id IS NOT NULL AND status = ?`,
		StatusHaveIngredients, database.FormatTime(s.now()), id, StatusPlanned)
	if err != nil {
		return fmt.Errorf("menü %d: %w", id, err)
	}
	return nil
}

// MenuEntries liefert die Gänge des Menüs id in Reihenfolge.
func (s *Store) MenuEntries(ctx context.Context, id int64) ([]Entry, error) {
	rows, err := s.db.QueryContext(ctx, selectEntry+" WHERE e.menu_id = ? ORDER BY e.position, e.id", id)
	if err != nil {
		return nil, fmt.Errorf("menü %d lesen: %w", id, err)
	}
	defer rows.Close()
	list := []Entry{}
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return nil, fmt.Errorf("menü %d lesen: %w", id, err)
		}
		list = append(list, e)
	}
	return list, rows.Err()
}
