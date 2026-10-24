// Package appsync stellt den Abgleich für die Android-App bereit: alle
// Änderungen seit einem Zeitpunkt in einer Antwort. Siehe docs/PLANUNG.md,
// Abschnitte 7 und 8.
//
// Ablauf in der App: Beim ersten Mal ohne since abrufen (alles), danach
// immer mit dem server_time der vorigen Antwort. Rezepte und Planeinträge
// kommen nur, wenn sie sich geändert haben; was gelöscht wurde, steht in
// deletions. Kleine Listen (Vorrat, Kategorien, Einstellungen, Termine)
// kommen jedes Mal vollständig und ersetzen den Stand in der App.
package appsync

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/database"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/httpx"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/mealplan"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/pantry"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/recipes"
	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/settings"
)

// overlap: So weit vor since wird erneut geliefert. Ein Schreibvorgang
// setzt updated_at, bevor er die Datenbank bekommt; wartet er, während ein
// Abgleich läuft, wird er erst nach dessen server_time sichtbar, trägt aber
// eine frühere Zeit. Die Überlappung fängt das ab. Doppelt gelieferte
// Datensätze schaden nicht – die App ersetzt sie einfach.
const overlap = 5 * time.Minute

// planHistoryDays: So viele Tage zurück enthält die App den Essensplan.
const planHistoryDays = 14

// Response ist die Antwort von GET /api/v1/sync.
type Response struct {
	// ServerTime ist beim nächsten Abgleich als since mitzuschicken.
	ServerTime string `json:"server_time"`
	// Full ist true, wenn ohne since abgerufen wurde: Die App ersetzt dann
	// ihren gesamten Stand.
	Full bool `json:"full"`
	// PlanFrom ist der erste Tag des Plans in der App; ältere Einträge
	// darf sie löschen.
	PlanFrom       string          `json:"plan_from"`
	Today          string          `json:"today"`
	ShoppingPeriod mealplan.Period `json:"shopping_period"`
	// NextShoppingPeriod: der nächste Einkauf (folgende Woche).
	NextShoppingPeriod mealplan.Period          `json:"next_shopping_period"`
	Recipes            []recipes.Recipe         `json:"recipes"`
	PlanInfo           []recipes.RecipePlanInfo `json:"plan_info"`
	MealPlanEntries    []mealplan.Entry         `json:"meal_plan_entries"`
	// Menus: geänderte Menüs ab plan_from (Schritt 9.4); gelöschte stehen
	// in deletions mit table "meal_plan_menus".
	Menus       []mealplan.Menu    `json:"menus"`
	PantryItems []pantry.Item      `json:"pantry_items"`
	TagGroups   []recipes.TagGroup `json:"tag_groups"`
	Settings    settings.Settings  `json:"settings"`
	Deletions   []Deletion         `json:"deletions"`
}

// Deletion ist ein gelöschter Datensatz, z. B. {"table": "recipes", "id": 12}.
type Deletion struct {
	Table     string `json:"table"`
	ID        int64  `json:"id"`
	DeletedAt string `json:"deleted_at"`
}

// Handler stellt GET /api/v1/sync bereit.
type Handler struct {
	db       *sql.DB
	recipes  *recipes.Store
	plan     *mealplan.Store
	pantry   *pantry.Store
	settings *settings.Store
	now      func() time.Time
}

// NewHandler erzeugt den Endpunkt.
func NewHandler(db *sql.DB, r *recipes.Store, p *mealplan.Store, pa *pantry.Store, s *settings.Store) *Handler {
	return &Handler{db: db, recipes: r, plan: p, pantry: pa, settings: s, now: time.Now}
}

// Register meldet den Endpunkt beim Router an.
func (h *Handler) Register(mux httpx.Router) {
	mux.HandleFunc("GET /api/v1/sync", h.sync)
}

func (h *Handler) sync(w http.ResponseWriter, r *http.Request) {
	since := ""
	if raw := r.URL.Query().Get("since"); raw != "" {
		t, err := database.ParseTime(raw)
		if err != nil {
			httpx.WriteJSON(w, http.StatusBadRequest, httpx.ErrorResponse{
				Error: "since muss ein zeitstempel wie 2026-10-23T18:30:00.000Z sein (server_time der letzten antwort)",
				Field: "since"})
			return
		}
		since = database.FormatTime(t.Add(-overlap))
	}
	resp, err := h.collect(r.Context(), since)
	if err != nil {
		httpx.WriteInternalError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}

// collect stellt die Antwort zusammen; since leer = alles.
func (h *Handler) collect(ctx context.Context, since string) (Response, error) {
	// Vor dem Lesen festhalten: Was danach geändert wird, kommt beim
	// nächsten Mal.
	resp := Response{ServerTime: database.FormatTime(h.now()), Full: since == ""}

	today := h.plan.Today()
	resp.Today = today.Format(time.DateOnly)
	resp.PlanFrom = today.AddDate(0, 0, -planHistoryDays).Format(time.DateOnly)

	var err error
	if resp.ShoppingPeriod, err = h.plan.CurrentPeriod(ctx); err != nil {
		return Response{}, err
	}
	resp.NextShoppingPeriod = resp.ShoppingPeriod.Next()
	if resp.Recipes, err = h.recipes.ChangedSince(ctx, since); err != nil {
		return Response{}, err
	}
	if resp.PlanInfo, err = h.recipes.PlanInfos(ctx); err != nil {
		return Response{}, err
	}
	if resp.MealPlanEntries, err = h.plan.ChangedSince(ctx, resp.PlanFrom, since); err != nil {
		return Response{}, err
	}
	if resp.Menus, err = h.plan.MenusChangedSince(ctx, resp.PlanFrom, since); err != nil {
		return Response{}, err
	}
	if resp.PantryItems, err = h.pantry.List(ctx); err != nil {
		return Response{}, err
	}
	if resp.TagGroups, err = h.recipes.ListTags(ctx); err != nil {
		return Response{}, err
	}
	if resp.Settings, err = h.settings.Get(ctx); err != nil {
		return Response{}, err
	}
	resp.Deletions = []Deletion{}
	if since != "" { // beim vollständigen Abgleich ersetzt die App ohnehin alles
		if resp.Deletions, err = h.deletions(ctx, since); err != nil {
			return Response{}, err
		}
	}
	return resp, nil
}

func (h *Handler) deletions(ctx context.Context, since string) ([]Deletion, error) {
	rows, err := h.db.QueryContext(ctx, `
		SELECT table_name, record_id, deleted_at FROM deletions
		WHERE deleted_at >= ? ORDER BY id`, since)
	if err != nil {
		return nil, fmt.Errorf("löschungen lesen: %w", err)
	}
	defer rows.Close()

	list := []Deletion{}
	for rows.Next() {
		var d Deletion
		if err := rows.Scan(&d.Table, &d.ID, &d.DeletedAt); err != nil {
			return nil, fmt.Errorf("löschungen lesen: %w", err)
		}
		list = append(list, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("löschungen lesen: %w", err)
	}
	return list, nil
}
