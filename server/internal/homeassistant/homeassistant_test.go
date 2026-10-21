package homeassistant

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeHA spielt die Teile der Home-Assistant-API nach, die der Client nutzt,
// und merkt sich die Dienstaufrufe.
func fakeHA(t *testing.T) (*httptest.Server, *[]string) {
	t.Helper()
	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer geheim" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		body, _ := io.ReadAll(r.Body)
		calls = append(calls, r.Method+" "+r.URL.String()+" "+string(body))
		switch r.URL.Path {
		case "/api/states":
			w.Write([]byte(`[
				{"entity_id": "sensor.temp", "attributes": {"friendly_name": "Temperatur"}},
				{"entity_id": "todo.zuhause", "attributes": {"friendly_name": "Zuhause"}},
				{"entity_id": "todo.einkauf", "attributes": {}}]`))
		case "/api/services/todo/get_items":
			w.Write([]byte(`{"changed_states": [], "service_response": {"todo.zuhause": {"items": [
				{"uid": "u1", "summary": "Milch", "description": "1 l", "status": "needs_action"}]}}}`))
		case "/api/services/todo/add_item", "/api/services/todo/update_item":
			w.Write([]byte(`[]`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &calls
}

func TestREST(t *testing.T) {
	ctx := context.Background()
	srv, calls := fakeHA(t)
	c := New("", srv.URL+"/", "geheim")
	if c.Mode() != ModeDirect {
		t.Fatalf("Mode = %s", c.Mode())
	}

	lists, err := c.TodoLists(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// Sortiert nach Name; ohne friendly_name gilt die Entitäts-ID.
	if len(lists) != 2 || lists[0] != (TodoList{"todo.zuhause", "Zuhause"}) || lists[1].Name != "todo.einkauf" {
		t.Errorf("Listen = %+v", lists)
	}

	items, err := c.OpenItems(ctx, "todo.zuhause")
	if err != nil || len(items) != 1 || items[0].Summary != "Milch" || items[0].Description != "1 l" {
		t.Errorf("Einträge = %+v, %v", items, err)
	}
	if err := c.AddItem(ctx, "todo.zuhause", "Mehl", "500 g"); err != nil {
		t.Fatal(err)
	}
	if err := c.UpdateDescription(ctx, "todo.zuhause", "u1", "1 l + 250 ml"); err != nil {
		t.Fatal(err)
	}

	got := strings.Join(*calls, "\n")
	for _, want := range []string{
		`POST /api/services/todo/get_items?return_response {"entity_id":"todo.zuhause","status":["needs_action"]}`,
		`POST /api/services/todo/add_item {"description":"500 g","entity_id":"todo.zuhause","item":"Mehl"}`,
		`POST /api/services/todo/update_item {"description":"1 l + 250 ml","entity_id":"todo.zuhause","item":"u1"}`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("Aufruf fehlt: %s\nbisher:\n%s", want, got)
		}
	}
}

func TestRESTErrors(t *testing.T) {
	srv, _ := fakeHA(t)
	c := New("", srv.URL, "falsch")
	if _, err := c.TodoLists(context.Background()); !errors.Is(err, ErrUnavailable) {
		t.Errorf("falscher Token: err = %v", err)
	}
	c = New("", "http://127.0.0.1:1", "geheim")
	if _, err := c.TodoLists(context.Background()); !errors.Is(err, ErrUnavailable) {
		t.Errorf("nicht erreichbar: err = %v", err)
	}
}

func TestModes(t *testing.T) {
	if m := New("tok", "", "").Mode(); m != ModeSupervisor {
		t.Errorf("mit SUPERVISOR_TOKEN: %s", m)
	}
	if m := New("", "http://ha:8123", "").Mode(); m != ModeTest {
		t.Errorf("Adresse ohne Token: %s", m)
	}
}

func TestTestClientAndHandler(t *testing.T) {
	ctx := context.Background()
	c := NewTestClient()
	c.AddItem(ctx, TestListID, "Mehl", "500 g")
	items, _ := c.OpenItems(ctx, TestListID)
	if err := c.UpdateDescription(ctx, TestListID, items[0].UID, "1 kg"); err != nil {
		t.Fatal(err)
	}
	if items, _ = c.OpenItems(ctx, TestListID); items[0].Description != "1 kg" {
		t.Errorf("Einträge = %+v", items)
	}
	if err := c.AddItem(ctx, "todo.anders", "x", ""); err == nil {
		t.Error("unbekannte Liste ohne Fehler")
	}

	mux := http.NewServeMux()
	NewHandler(c).Register(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/homeassistant/status", nil))
	var resp statusResponse
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Mode != ModeTest || len(resp.Lists) != 1 || resp.Error != "" {
		t.Errorf("Status = %+v", resp)
	}

	// Nicht erreichbar: trotzdem 200, mit Fehlertext.
	mux = http.NewServeMux()
	NewHandler(New("", "http://127.0.0.1:1", "x")).Register(mux)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/homeassistant/status", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"error":"home assistant nicht erreichbar`) {
		t.Errorf("%d %s", rec.Code, rec.Body)
	}
}
