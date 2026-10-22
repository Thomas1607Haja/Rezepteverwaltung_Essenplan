// Package recipes verwaltet Rezepte mit Zutaten, Schritten und Kategorien.
package recipes

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Recipe ist ein Rezept mit allen Bestandteilen. Es wird immer als Ganzes
// gespeichert und geladen.
//
// Die Angaben in `json:"..."` legen fest, wie ein Feld in JSON heißt.
type Recipe struct {
	ID           int64        `json:"id"`
	Title        string       `json:"title"`
	Description  string       `json:"description"`
	BaseServings int          `json:"base_servings"`
	PrepMinutes  *int         `json:"prep_minutes"`  // nil = keine Angabe
	TotalMinutes *int         `json:"total_minutes"` // nil = keine Angabe
	SourceURL    string       `json:"source_url"`
	Image        string       `json:"image"`     // nur lesbar, wird per Upload gesetzt
	Thumbnail    string       `json:"thumbnail"` // nur lesbar, wird per Upload gesetzt
	IsFavorite   bool         `json:"is_favorite"`
	Notes        string       `json:"notes"`
	Ingredients  []Ingredient `json:"ingredients"` // in Anzeigereihenfolge
	Steps        []Step       `json:"steps"`       // in Anzeigereihenfolge
	Tags         []Tag        `json:"tags"`        // beim Speichern zählt nur die ID
	PlanInfo                  // nur lesbar, siehe list.go
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

// Ingredient ist eine Zutatenzeile. Die Reihenfolge ergibt sich aus der
// Position in Recipe.Ingredients.
type Ingredient struct {
	ID               int64    `json:"id"`
	GroupName        string   `json:"group_name"` // z. B. "Für die Soße"
	Quantity         *float64 `json:"quantity"`   // nil = nicht umrechnen
	Unit             string   `json:"unit"`
	Name             string   `json:"name"`
	Note             string   `json:"note"`
	ShoppingRelevant bool     `json:"shopping_relevant"`
}

// Step ist ein Zubereitungsschritt.
type Step struct {
	ID   int64  `json:"id"`
	Text string `json:"text"`
}

// Tag ist eine Kategorie, z. B. "Vegetarisch" in der Gruppe "Hauptzutat".
type Tag struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Group    string `json:"group"`
	Position int    `json:"position"`
}

// ErrNotFound wird zurückgegeben, wenn es ein Rezept nicht gibt.
var ErrNotFound = errors.New("rezept nicht gefunden")

// ValidationError beschreibt eine ungültige Eingabe. Field nennt das
// betroffene Feld im JSON-Format, z. B. "ingredients[2].name".
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

// normalize entfernt überflüssige Leerzeichen am Anfang und Ende aller Texte.
func (r *Recipe) normalize() {
	r.Title = strings.TrimSpace(r.Title)
	r.Description = strings.TrimSpace(r.Description)
	r.SourceURL = strings.TrimSpace(r.SourceURL)
	r.Notes = strings.TrimSpace(r.Notes)
	for i := range r.Ingredients {
		ing := &r.Ingredients[i] // Zeiger, damit die Änderung im Slice landet
		ing.GroupName = strings.TrimSpace(ing.GroupName)
		ing.Unit = strings.TrimSpace(ing.Unit)
		ing.Name = strings.TrimSpace(ing.Name)
		ing.Note = strings.TrimSpace(ing.Note)
	}
	for i := range r.Steps {
		r.Steps[i].Text = strings.TrimSpace(r.Steps[i].Text)
	}
}

// validate prüft die fachlichen Regeln und liefert den ersten Verstoß.
func (r *Recipe) validate() error {
	if r.Title == "" {
		return &ValidationError{"title", "darf nicht leer sein"}
	}
	if r.BaseServings <= 0 {
		return &ValidationError{"base_servings", "muss größer als 0 sein"}
	}
	if r.PrepMinutes != nil && *r.PrepMinutes < 0 {
		return &ValidationError{"prep_minutes", "darf nicht negativ sein"}
	}
	if r.TotalMinutes != nil && *r.TotalMinutes < 0 {
		return &ValidationError{"total_minutes", "darf nicht negativ sein"}
	}
	// Die Quelle erscheint als Link. Nur Webadressen, damit dort nichts wie
	// "javascript:…" landet, das beim Anklicken im Browser ausgeführt würde.
	if r.SourceURL != "" {
		u, err := url.Parse(r.SourceURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return &ValidationError{"source_url", "muss mit http:// oder https:// beginnen"}
		}
	}
	for i, ing := range r.Ingredients {
		if ing.Name == "" {
			return &ValidationError{fmt.Sprintf("ingredients[%d].name", i), "darf nicht leer sein"}
		}
		if ing.Quantity != nil && *ing.Quantity < 0 {
			return &ValidationError{fmt.Sprintf("ingredients[%d].quantity", i), "darf nicht negativ sein"}
		}
	}
	for i, step := range r.Steps {
		if step.Text == "" {
			return &ValidationError{fmt.Sprintf("steps[%d].text", i), "darf nicht leer sein"}
		}
	}
	return nil
}
