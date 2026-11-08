// Package homeassistant spricht die REST-API von Home Assistant an, um
// To-do-Listen (z. B. die Bring-Einkaufsliste) zu lesen und zu füllen.
//
// In der App läuft das über den Supervisor (Option homeassistant_api in
// config.yaml), lokal wahlweise direkt gegen eine Home-Assistant-Instanz
// oder im Testmodus mit einer Liste im Arbeitsspeicher.
package homeassistant

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/apierr"
)

// Betriebsarten, wie sie die API meldet.
const (
	ModeSupervisor = "supervisor" // als App im Home Assistant
	ModeDirect     = "direct"     // lokal mit Adresse und Token
	ModeTest       = "test"       // lokal ohne Home Assistant
)

// TodoList ist eine To-do-Entität, z. B. {"todo.zuhause", "Zuhause"}.
type TodoList struct {
	EntityID string `json:"entity_id"`
	Name     string `json:"name"`
}

// TodoItem ist ein Eintrag einer To-do-Liste. Bei Bring ist Summary der
// Artikel und Description die Spezifikation (Mengenangabe).
type TodoItem struct {
	UID         string `json:"uid"`
	Summary     string `json:"summary"`
	Description string `json:"description"`
	Status      string `json:"status"` // "needs_action" oder "completed"
}

// Client ist der Zugang zu den To-do-Listen. Ein Interface, damit Tests
// und der lokale Testmodus ohne Home Assistant auskommen.
type Client interface {
	Mode() string
	TodoLists(ctx context.Context) ([]TodoList, error)
	// OpenItems liefert die noch nicht erledigten Einträge der Liste.
	OpenItems(ctx context.Context, entityID string) ([]TodoItem, error)
	AddItem(ctx context.Context, entityID, summary, description string) error
	// UpdateDescription ändert die Beschreibung des Eintrags uid.
	UpdateDescription(ctx context.Context, entityID, uid, description string) error
	// RemoveItem löscht den Eintrag uid von der Liste.
	RemoveItem(ctx context.Context, entityID, uid string) error
	// Language liefert die Systemsprache von Home Assistant, z. B. "de"
	// oder "en-GB"; leer im Testmodus.
	Language(ctx context.Context) (string, error)
}

// New wählt die Betriebsart: Mit supervisorToken (in der App gesetzt) geht
// es über den Supervisor, sonst mit url und token direkt, und ohne beides
// im Testmodus.
func New(supervisorToken, url, token string) Client {
	switch {
	case supervisorToken != "":
		return newREST(ModeSupervisor, "http://supervisor/core/api", supervisorToken)
	case url != "" && token != "":
		return newREST(ModeDirect, strings.TrimSuffix(url, "/")+"/api", token)
	default:
		return NewTestClient()
	}
}

// rest spricht die REST-API von Home Assistant an.
type rest struct {
	mode   string
	base   string
	token  string
	client *http.Client
}

func newREST(mode, base, token string) *rest {
	return &rest{mode: mode, base: base, token: token, client: &http.Client{Timeout: 15 * time.Second}}
}

func (c *rest) Mode() string { return c.mode }

// ErrUnavailable: Home Assistant ist nicht erreichbar oder lehnt ab.
var ErrUnavailable = apierr.New("home_assistant_down")

// call schickt eine Anfrage; ist out nicht nil, wird die Antwort dorthin
// gelesen.
func (c *rest) call(ctx context.Context, method, path string, body, out any) error {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	// Einzelheiten (Adressen, Antworttexte von Home Assistant) nur ins
	// Protokoll; die Oberfläche bekommt eine kurze Meldung.
	resp, err := c.client.Do(req)
	if err != nil {
		slog.Warn("Home Assistant nicht erreichbar", "method", method, "path", path, "error", err)
		return ErrUnavailable
	}
	defer resp.Body.Close()
	// Antworten von Home Assistant sind klein; 1 MB reicht für jede Liste.
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		slog.Warn("Antwort von Home Assistant nicht lesbar", "method", method, "path", path, "error", err)
		return ErrUnavailable
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		detail := strings.TrimSpace(string(data))
		if len(detail) > 500 {
			detail = detail[:500]
		}
		slog.Warn("Home Assistant lehnt Anfrage ab", "method", method, "path", path, "status", resp.StatusCode, "antwort", detail)
		return fmt.Errorf("%w (status %d)", ErrUnavailable, resp.StatusCode)
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("response from home assistant: %w", err)
	}
	return nil
}

// Language liest die Systemsprache aus GET /api/config (Einstellungen →
// System → Allgemein in Home Assistant).
func (c *rest) Language(ctx context.Context) (string, error) {
	var config struct {
		Language string `json:"language"`
	}
	if err := c.call(ctx, http.MethodGet, "/config", nil, &config); err != nil {
		return "", err
	}
	return config.Language, nil
}

// TodoLists liest alle Entitäten, deren ID mit "todo." beginnt.
func (c *rest) TodoLists(ctx context.Context) ([]TodoList, error) {
	var states []struct {
		EntityID   string `json:"entity_id"`
		Attributes struct {
			FriendlyName string `json:"friendly_name"`
		} `json:"attributes"`
	}
	if err := c.call(ctx, http.MethodGet, "/states", nil, &states); err != nil {
		return nil, err
	}
	lists := []TodoList{}
	for _, s := range states {
		if !strings.HasPrefix(s.EntityID, "todo.") {
			continue
		}
		name := s.Attributes.FriendlyName
		if name == "" {
			name = s.EntityID
		}
		lists = append(lists, TodoList{EntityID: s.EntityID, Name: name})
	}
	slices.SortFunc(lists, func(a, b TodoList) int { return strings.Compare(a.Name, b.Name) })
	return lists, nil
}

// OpenItems ruft den Dienst todo.get_items auf. "?return_response" sorgt
// dafür, dass Home Assistant das Ergebnis des Dienstes mitschickt.
func (c *rest) OpenItems(ctx context.Context, entityID string) ([]TodoItem, error) {
	var resp struct {
		ServiceResponse map[string]struct {
			Items []TodoItem `json:"items"`
		} `json:"service_response"`
	}
	body := map[string]any{"entity_id": entityID, "status": []string{"needs_action"}}
	if err := c.call(ctx, http.MethodPost, "/services/todo/get_items?return_response", body, &resp); err != nil {
		return nil, err
	}
	items := resp.ServiceResponse[entityID].Items
	if items == nil {
		items = []TodoItem{}
	}
	return items, nil
}

// AddItem ruft den Dienst todo.add_item auf.
func (c *rest) AddItem(ctx context.Context, entityID, summary, description string) error {
	body := map[string]string{"entity_id": entityID, "item": summary}
	if description != "" {
		body["description"] = description
	}
	return c.call(ctx, http.MethodPost, "/services/todo/add_item", body, nil)
}

// UpdateDescription ruft den Dienst todo.update_item auf; "item" darf die
// UID oder der Name sein.
func (c *rest) UpdateDescription(ctx context.Context, entityID, uid, description string) error {
	body := map[string]string{"entity_id": entityID, "item": uid, "description": description}
	return c.call(ctx, http.MethodPost, "/services/todo/update_item", body, nil)
}

func (c *rest) RemoveItem(ctx context.Context, entityID, uid string) error {
	body := map[string]string{"entity_id": entityID, "item": uid}
	return c.call(ctx, http.MethodPost, "/services/todo/remove_item", body, nil)
}

// TestClient hält zwei To-do-Listen im Arbeitsspeicher. Er dient zum
// lokalen Ausprobieren und für Tests; nach einem Neustart sind die Listen
// leer. Die zweite Liste erlaubt es, den Wechsel der Bring-Liste (z. B. von
// einer Test- auf die Familienliste) durchzuspielen.
type TestClient struct {
	// mu schützt lists: Mehrere Anfragen können gleichzeitig laufen.
	mu     sync.Mutex
	lists  map[string][]TodoItem
	nextID int
}

// TestListID und TestListID2 sind die Listen des Testmodus.
const (
	TestListID  = "todo.testliste"
	TestListID2 = "todo.testliste_2"
)

// NewTestClient erzeugt einen Testmodus mit leeren Listen.
func NewTestClient() *TestClient {
	return &TestClient{lists: map[string][]TodoItem{TestListID: {}, TestListID2: {}}}
}

func (c *TestClient) Mode() string { return ModeTest }

// Language: Der Testmodus kennt keine Sprache; es gilt REZEPTPLANER_LANGUAGE.
func (c *TestClient) Language(context.Context) (string, error) { return "", nil }

func (c *TestClient) TodoLists(context.Context) ([]TodoList, error) {
	return []TodoList{
		{EntityID: TestListID, Name: "Testliste (nur lokal)"},
		{EntityID: TestListID2, Name: "Testliste 2 (nur lokal)"},
	}, nil
}

// list liefert die Einträge einer Liste; c.mu muss gesperrt sein.
func (c *TestClient) list(entityID string) ([]TodoItem, error) {
	items, ok := c.lists[entityID]
	if !ok {
		return nil, fmt.Errorf("%w: unknown list %s", ErrUnavailable, entityID)
	}
	return items, nil
}

func (c *TestClient) OpenItems(_ context.Context, entityID string) ([]TodoItem, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	items, err := c.list(entityID)
	return slices.Clone(items), err
}

func (c *TestClient) AddItem(_ context.Context, entityID, summary, description string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	items, err := c.list(entityID)
	if err != nil {
		return err
	}
	c.nextID++
	c.lists[entityID] = append(items, TodoItem{UID: fmt.Sprint(c.nextID), Summary: summary, Description: description, Status: "needs_action"})
	return nil
}

func (c *TestClient) UpdateDescription(_ context.Context, entityID, uid, description string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	items, err := c.list(entityID)
	if err != nil {
		return err
	}
	for i := range items {
		if items[i].UID == uid {
			items[i].Description = description
			return nil
		}
	}
	return fmt.Errorf("%w: item %s not found", ErrUnavailable, uid)
}

func (c *TestClient) RemoveItem(_ context.Context, entityID, uid string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	items, err := c.list(entityID)
	if err != nil {
		return err
	}
	for i := range items {
		if items[i].UID == uid {
			c.lists[entityID] = slices.Delete(items, i, i+1)
			return nil
		}
	}
	return fmt.Errorf("%w: item %s not found", ErrUnavailable, uid)
}
