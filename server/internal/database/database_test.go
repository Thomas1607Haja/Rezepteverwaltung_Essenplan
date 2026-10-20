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
	if version != 2 {
		t.Errorf("version = %d, erwartet 2", version)
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
	if version != 2 {
		t.Errorf("version = %d, erwartet 2", version)
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
