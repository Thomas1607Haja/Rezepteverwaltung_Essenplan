package settings

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/database"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	ctx := context.Background()
	db, err := database.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := database.Migrate(ctx, db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return NewStore(db)
}

func TestDefaultsAndUpdate(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	got, err := store.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got != Defaults {
		t.Errorf("leere Datenbank: %+v, erwartet Standardwerte %+v", got, Defaults)
	}

	want := Settings{FamilySize: 3, ShoppingDay: "friday", BringEntity: "todo.zuhause"}
	got, err = store.Update(ctx, want)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("nach Update: %+v, erwartet %+v", got, want)
	}
	if got.ShoppingWeekday() != time.Friday {
		t.Errorf("ShoppingWeekday = %v", got.ShoppingWeekday())
	}
}

func TestUpdateKeepsUpdatedAtWhenUnchanged(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)
	if _, err := store.Update(ctx, Settings{FamilySize: 5, ShoppingDay: "saturday"}); err != nil {
		t.Fatal(err)
	}
	updatedAt := func(key string) string {
		var s string
		if err := store.db.QueryRowContext(ctx, "SELECT updated_at FROM settings WHERE key = ?", key).Scan(&s); err != nil {
			t.Fatal(err)
		}
		return s
	}
	before := updatedAt("family_size")
	time.Sleep(5 * time.Millisecond)
	if _, err := store.Update(ctx, Settings{FamilySize: 5, ShoppingDay: "monday"}); err != nil {
		t.Fatal(err)
	}
	if after := updatedAt("family_size"); after != before {
		t.Errorf("updated_at von family_size geändert (%s -> %s), obwohl der Wert gleich blieb", before, after)
	}
}

func TestValidation(t *testing.T) {
	store := newTestStore(t)
	for _, in := range []Settings{
		{FamilySize: 0, ShoppingDay: "saturday"},
		{FamilySize: 4, ShoppingDay: "samstag"},
		{FamilySize: 4, ShoppingDay: ""},
		{FamilySize: 4, ShoppingDay: "saturday", BringEntity: "sensor.bring"},
		{FamilySize: 4, ShoppingDay: "saturday", BringEntity: "todo.a b"},
	} {
		_, err := store.Update(context.Background(), in)
		var verr *ValidationError
		if !errors.As(err, &verr) {
			t.Errorf("%+v: erwartet ValidationError, bekommen %v", in, err)
		}
	}
}

func TestHandlers(t *testing.T) {
	mux := http.NewServeMux()
	NewHandler(newTestStore(t)).Register(mux)

	do := func(method, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, "/api/v1/settings", strings.NewReader(body))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec
	}

	if rec := do(http.MethodGet, ""); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"family_size":4`) {
		t.Errorf("GET: %d %s", rec.Code, rec.Body)
	}
	if rec := do(http.MethodPut, `{"family_size":2,"shopping_day":"monday"}`); rec.Code != http.StatusOK ||
		!strings.Contains(rec.Body.String(), `"shopping_day":"monday"`) {
		t.Errorf("PUT: %d %s", rec.Code, rec.Body)
	}
	if rec := do(http.MethodPut, `{"family_size":2,"shopping_day":"montag"}`); rec.Code != http.StatusBadRequest ||
		!strings.Contains(rec.Body.String(), `"field":"shopping_day"`) {
		t.Errorf("PUT ungültig: %d %s", rec.Code, rec.Body)
	}
}

func TestAppURLAndPartialUpdate(t *testing.T) {
	mux := http.NewServeMux()
	NewHandler(newTestStore(t)).Register(mux)
	do := func(body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPut, "/api/v1/settings", strings.NewReader(body)))
		return rec
	}

	if rec := do(`{"family_size":3}`); rec.Code != http.StatusOK {
		t.Fatalf("PUT family_size: %d %s", rec.Code, rec.Body)
	}
	// Nur app_url: Familiengröße bleibt, Schrägstrich am Ende fällt weg.
	rec := do(`{"app_url":" https://rezepte.example.org/ "}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"app_url":"https://rezepte.example.org"`) ||
		!strings.Contains(rec.Body.String(), `"family_size":3`) {
		t.Errorf("PUT app_url: %d %s", rec.Code, rec.Body)
	}
	for _, bad := range []string{"rezepte.example.org", "javascript:alert(1)", "https://user:pw@example.org", "https://example.org/?x=1"} {
		if rec := do(`{"app_url":"` + bad + `"}`); rec.Code != http.StatusBadRequest {
			t.Errorf("app_url %q: %d, erwartet 400", bad, rec.Code)
		}
	}
}
