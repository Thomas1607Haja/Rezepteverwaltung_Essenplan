package seed

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/database"
)

func freshDB(t *testing.T) *sql.DB {
	t.Helper()
	ctx := context.Background()
	db, err := database.Open(ctx, filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := database.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	return db
}

func column(t *testing.T, db *sql.DB, query string) []string {
	t.Helper()
	rows, err := db.Query(query)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		out = append(out, s)
	}
	return out
}

// Jede Kategorie und Gruppe der Startdaten braucht eine Übersetzung – auch
// wenn eine künftige Migration neue Startdaten anlegt.
func TestEveryStartNameIsTranslated(t *testing.T) {
	db := freshDB(t)
	for _, name := range column(t, db, "SELECT name FROM tag_groups") {
		if _, ok := groups[name]; !ok {
			t.Errorf("Gruppe %q fehlt in seed.groups", name)
		}
	}
	for _, name := range column(t, db, "SELECT name FROM tags") {
		if _, ok := tags[name]; !ok {
			t.Errorf("Kategorie %q fehlt in seed.tags", name)
		}
	}
}

func TestLocalize(t *testing.T) {
	ctx := context.Background()
	db := freshDB(t)
	before := column(t, db, "SELECT course_key || '|' || group_id FROM tags ORDER BY id")

	if err := Localize(ctx, db, "fr"); err != nil {
		t.Fatal(err)
	}
	names := column(t, db, "SELECT name FROM tags WHERE course_key = 'main'")
	if len(names) != 1 || names[0] != "Plat principal" {
		t.Errorf("Hauptgang: %v", names)
	}
	if got := column(t, db, "SELECT name FROM tag_groups WHERE role = 'course'"); len(got) != 1 || got[0] != "Plat" {
		t.Errorf("Gang-Gruppe: %v", got)
	}
	// Schlüssel, Gruppen und Anzahl bleiben.
	after := column(t, db, "SELECT course_key || '|' || group_id FROM tags ORDER BY id")
	if len(after) != len(before) {
		t.Fatalf("Anzahl %d → %d", len(before), len(after))
	}
	for i := range before {
		if before[i] != after[i] {
			t.Errorf("Kategorie %d: %s → %s", i, before[i], after[i])
		}
	}
	if got := column(t, db, "SELECT name || ':' || synonyms FROM pantry_items ORDER BY id"); len(got) != 5 || got[0] != "Sel:sel fin, gros sel, sel de mer, fleur de sel" {
		t.Errorf("Vorrat: %v", got)
	}
}

func TestLocalizeGermanKeepsEverything(t *testing.T) {
	db := freshDB(t)
	want := column(t, db, "SELECT name FROM tags ORDER BY id")
	for _, lang := range []string{"de", "nl", ""} {
		if err := Localize(context.Background(), db, lang); err != nil {
			t.Fatal(err)
		}
	}
	got := column(t, db, "SELECT name FROM tags ORDER BY id")
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%q → %q", want[i], got[i])
		}
	}
	if p := column(t, db, "SELECT name FROM pantry_items ORDER BY id"); len(p) != 5 || p[0] != "Salz" {
		t.Errorf("Vorrat: %v", p)
	}
}
