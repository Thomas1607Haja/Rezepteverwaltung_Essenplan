package apikeys

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/database"
)

func newStore(t *testing.T) *Store {
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
	return NewStore(db)
}

func TestCreateStoresOnlyHash(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	key, secret, err := s.Create(ctx, "  Handy   Thomas ")
	if err != nil {
		t.Fatal(err)
	}
	if key.Name != "Handy Thomas" {
		t.Errorf("Name %q", key.Name)
	}
	// "rp_" + 43 Zeichen Base64 = 32 Byte Zufall
	if !strings.HasPrefix(secret, "rp_") || len(secret) != 46 {
		t.Errorf("Schlüssel %q hat unerwartetes Format", secret)
	}

	var stored string
	if err := s.db.QueryRowContext(ctx, `SELECT key_hash FROM api_keys WHERE id = ?`, key.ID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored == secret || strings.Contains(stored, secret[3:]) || stored != hash(secret) {
		t.Errorf("gespeichert ist %q – erwartet nur den Hash", stored)
	}

	ok, err := s.Check(ctx, secret)
	if err != nil || !ok {
		t.Errorf("Check(gültig) = %v, %v", ok, err)
	}
	for _, wrong := range []string{"", "rp_", secret + "x", strings.ToUpper(secret), stored} {
		if ok, _ := s.Check(ctx, wrong); ok {
			t.Errorf("Check(%q) = true", wrong)
		}
	}

	list, err := s.List(ctx)
	if err != nil || len(list) != 1 || list[0].LastUsedAt == nil {
		t.Errorf("List nach Benutzung: %+v, %v", list, err)
	}
}

func TestCreateRejectsBadNames(t *testing.T) {
	s := newStore(t)
	for _, name := range []string{"", "   ", strings.Repeat("x", 61)} {
		if _, _, err := s.Create(context.Background(), name); err != ErrInvalidName {
			t.Errorf("Create(%q): %v", name, err)
		}
	}
}

func TestDeleteUnknown(t *testing.T) {
	if err := newStore(t).Delete(context.Background(), 99); err != ErrNotFound {
		t.Errorf("Delete: %v", err)
	}
}
