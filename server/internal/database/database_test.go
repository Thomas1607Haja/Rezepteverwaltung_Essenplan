package database

import (
	"context"
	"database/sql"
	"testing"
)

// openTestDB öffnet eine leere Datenbank im Arbeitsspeicher, die nach dem
// Test automatisch geschlossen wird.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := Open(context.Background(), ":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestMigrateCreatesSchemaAndTags(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)

	version, err := Migrate(ctx, db)
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if version != 10 {
		t.Errorf("version = %d, erwartet 10", version)
	}

	// 8 Hauptzutat + 2 Geschmack + 8 Art + 5 Eigenschaften
	var tagCount int
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM tags").Scan(&tagCount); err != nil {
		t.Fatalf("tags zählen: %v", err)
	}
	if tagCount != 23 {
		t.Errorf("tags = %d, erwartet 23", tagCount)
	}
}

func TestMigrateTwiceChangesNothing(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)

	if _, err := Migrate(ctx, db); err != nil {
		t.Fatalf("erste Migration: %v", err)
	}
	// Zweiter Lauf darf nichts erneut einspielen (sonst gäbe es doppelte Tags
	// bzw. "table already exists").
	version, err := Migrate(ctx, db)
	if err != nil {
		t.Fatalf("zweite Migration: %v", err)
	}
	if version != 10 {
		t.Errorf("version = %d, erwartet 10", version)
	}
}

// Zeitstempel, die SQL selbst erzeugt (strftime), müssen dasselbe Format
// haben wie die aus Go (FormatTime) – sonst stimmt der Textvergleich nicht.
func TestSQLTimestampsMatchTimeLayout(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	if _, err := Migrate(ctx, db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	var updatedAt string
	if err := db.QueryRowContext(ctx, "SELECT updated_at FROM tags LIMIT 1").Scan(&updatedAt); err != nil {
		t.Fatalf("tag lesen: %v", err)
	}
	parsed, err := ParseTime(updatedAt)
	if err != nil {
		t.Fatalf("ParseTime(%q): %v", updatedAt, err)
	}
	if FormatTime(parsed) != updatedAt {
		t.Errorf("FormatTime = %q, aus SQL %q", FormatTime(parsed), updatedAt)
	}
}

func TestForeignKeysAreEnforced(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	if _, err := Migrate(ctx, db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	// Zutat zu einem Rezept, das es nicht gibt, muss abgelehnt werden.
	_, err := db.ExecContext(ctx, `INSERT INTO recipe_ingredients
		(recipe_id, position, name, updated_at) VALUES (999, 1, 'Mehl', '2026-01-01T00:00:00Z')`)
	if err == nil {
		t.Error("Zutat ohne Rezept wurde gespeichert, Fremdschlüssel sind nicht aktiv")
	}
}

func TestFoldInSQL(t *testing.T) {
	db := openTestDB(t)

	tests := []struct{ in, want string }{
		{"Äpfel", "apfel"},
		{"GRÜNKOHL", "grunkohl"},
		{"Weißwurst", "weisswurst"},
		{"Crème fraîche", "crème fraîche"}, // nur deutsche Sonderzeichen werden ersetzt
	}
	for _, tt := range tests {
		var got string
		if err := db.QueryRow("SELECT fold(?)", tt.in).Scan(&got); err != nil {
			t.Fatalf("fold(%q): %v", tt.in, err)
		}
		if got != tt.want {
			t.Errorf("fold(%q) = %q, erwartet %q", tt.in, got, tt.want)
		}
	}

	// NULL bleibt NULL.
	var null sql.NullString
	if err := db.QueryRow("SELECT fold(NULL)").Scan(&null); err != nil {
		t.Fatal(err)
	}
	if null.Valid {
		t.Errorf("fold(NULL) = %q, erwartet NULL", null.String)
	}
}

// Wird ein Rezept gelöscht, bleiben seine Planeinträge mit dem Titel als
// Notiz erhalten (Trigger meal_plan_keep_title in 003_meal_plan.sql).
func TestDeletingRecipeKeepsPlanEntries(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	if _, err := Migrate(ctx, db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	mustExec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.ExecContext(ctx, query, args...); err != nil {
			t.Fatalf("%s: %v", query, err)
		}
	}
	now := "2026-10-21T10:00:00.000Z"
	mustExec(`INSERT INTO recipes (id, title, base_servings, created_at, updated_at) VALUES (1, 'Lasagne', 4, ?, ?)`, now, now)
	mustExec(`INSERT INTO meal_plan_entries (id, date, recipe_id, servings, created_at, updated_at)
	          VALUES (1, '2026-10-24', 1, 4, ?, ?)`, now, now)
	mustExec(`INSERT INTO meal_plan_entries (id, date, recipe_id, servings, note, created_at, updated_at)
	          VALUES (2, '2026-10-25', 1, 4, 'mit Salat', ?, ?)`, now, now)
	mustExec(`DELETE FROM recipes WHERE id = 1`)

	rows, err := db.QueryContext(ctx, "SELECT recipe_id, note, updated_at FROM meal_plan_entries ORDER BY id")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	wantNotes := []string{"Lasagne", "mit Salat"}
	i := 0
	for rows.Next() {
		var recipeID sql.NullInt64
		var note, updatedAt string
		if err := rows.Scan(&recipeID, &note, &updatedAt); err != nil {
			t.Fatal(err)
		}
		if recipeID.Valid || note != wantNotes[i] || updatedAt == now {
			t.Errorf("Eintrag %d: recipe_id=%v note=%q updated_at=%s", i+1, recipeID, note, updatedAt)
		}
		i++
	}
	if i != 2 {
		t.Errorf("%d Einträge übrig, erwartet 2", i)
	}

	// Ein Eintrag ganz ohne Rezept und Notiz ist nicht erlaubt.
	if _, err := db.ExecContext(ctx, `INSERT INTO meal_plan_entries (date, servings, created_at, updated_at)
	                                   VALUES ('2026-10-26', 4, ?, ?)`, now, now); err == nil {
		t.Error("Eintrag ohne Rezept und Notiz wurde angenommen")
	}
}

func TestAutoincrementMigrationKeepsDataAndIDs(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	if _, err := migrateTo(ctx, db, 7); err != nil {
		t.Fatalf("Migration bis 7: %v", err)
	}
	mustExec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.ExecContext(ctx, query, args...); err != nil {
			t.Fatalf("%s: %v", query, err)
		}
	}
	now := "2026-10-23T10:00:00.000Z"
	mustExec(`INSERT INTO recipes (id, title, base_servings, created_at, updated_at) VALUES (1, 'Lasagne', 4, ?, ?)`, now, now)
	mustExec(`INSERT INTO recipes (id, title, base_servings, created_at, updated_at, parent_recipe_id)
	          VALUES (2, 'Lasagne (Variante)', 4, ?, ?, 1)`, now, now)
	mustExec(`INSERT INTO recipe_ingredients (recipe_id, position, name, updated_at) VALUES (1, 1, 'Nudelplatten', ?)`, now)
	mustExec(`INSERT INTO recipe_tags (recipe_id, tag_id, updated_at) VALUES (1, 3, ?)`, now)
	mustExec(`INSERT INTO meal_plan_entries (id, date, recipe_id, servings, created_at, updated_at)
	          VALUES (1, '2026-10-24', 1, 4, ?, ?)`, now, now)
	mustExec(`INSERT INTO transferred_items (entry_id, period_from, item_key, ingredient_name, sent, transferred_at)
	          VALUES (1, '2026-10-24', 'nudelplatten', 'Nudelplatten', 1, ?)`, now)
	// Rezept 3 war das neueste und ist gelöscht; ohne AUTOINCREMENT käme
	// seine ID beim nächsten Rezept wieder.
	mustExec(`INSERT INTO deletions (table_name, record_id, deleted_at) VALUES ('recipes', 3, ?)`, now)

	version, err := Migrate(ctx, db)
	if err != nil {
		t.Fatalf("Migration 8: %v", err)
	}
	if version < 9 {
		t.Fatalf("version = %d, erwartet mindestens 9", version)
	}

	count := func(query string) int {
		t.Helper()
		var n int
		if err := db.QueryRowContext(ctx, query).Scan(&n); err != nil {
			t.Fatalf("%s: %v", query, err)
		}
		return n
	}
	// Nichts darf per ON DELETE CASCADE / SET NULL verloren gehen.
	checks := map[string]int{
		"SELECT count(*) FROM recipes":                                       2,
		"SELECT count(*) FROM recipes WHERE id = 2 AND parent_recipe_id = 1": 1,
		"SELECT count(*) FROM recipe_ingredients WHERE recipe_id = 1":        1,
		"SELECT count(*) FROM recipe_tags WHERE recipe_id = 1":               1,
		"SELECT count(*) FROM meal_plan_entries WHERE recipe_id = 1":         1,
		"SELECT count(*) FROM transferred_items WHERE entry_id = 1":          1,
		"SELECT count(*) FROM tags":                                          23,
		"SELECT count(*) FROM pantry_items":                                  5,
	}
	for query, want := range checks {
		if got := count(query); got != want {
			t.Errorf("%s = %d, erwartet %d", query, got, want)
		}
	}

	// Neues Rezept bekommt nicht die gelöschte ID 3.
	res, err := db.ExecContext(ctx, `INSERT INTO recipes (title, base_servings, created_at, updated_at) VALUES ('Neu', 2, ?, ?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}
	if id, _ := res.LastInsertId(); id != 4 {
		t.Errorf("neue Rezept-ID = %d, erwartet 4", id)
	}
	// Auch nach dem Löschen des neuesten Planeintrags keine Wiederverwendung.
	mustExec(`DELETE FROM meal_plan_entries WHERE id = 1`)
	res, err = db.ExecContext(ctx, `INSERT INTO meal_plan_entries (date, note, servings, created_at, updated_at)
	                                VALUES ('2026-10-25', 'Essen gehen', 4, ?, ?)`, now, now)
	if err != nil {
		t.Fatal(err)
	}
	if id, _ := res.LastInsertId(); id != 2 {
		t.Errorf("neue Planeintrag-ID = %d, erwartet 2", id)
	}

	// Trigger und Fremdschlüssel wirken nach der Migration wieder.
	mustExec(`INSERT INTO meal_plan_entries (date, recipe_id, servings, created_at, updated_at) VALUES ('2026-10-26', 1, 4, ?, ?)`, now, now)
	mustExec(`DELETE FROM recipes WHERE id = 1`)
	if got := count("SELECT count(*) FROM meal_plan_entries WHERE recipe_id IS NULL AND note = 'Lasagne'"); got != 1 {
		t.Errorf("Trigger meal_plan_keep_title: %d Einträge mit Titel als Notiz, erwartet 1", got)
	}
	if got := count("SELECT count(*) FROM recipe_ingredients"); got != 0 {
		t.Errorf("ON DELETE CASCADE wirkt nicht: %d Zutaten übrig", got)
	}
	if got := count("SELECT count(*) FROM recipes WHERE id = 2 AND parent_recipe_id IS NULL"); got != 1 {
		t.Error("ON DELETE SET NULL für parent_recipe_id wirkt nicht")
	}
}
