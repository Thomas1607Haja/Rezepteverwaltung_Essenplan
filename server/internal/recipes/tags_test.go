package recipes

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"testing"
)

func TestListTags(t *testing.T) {
	s := newTestStore(t)

	groups, err := s.ListTags(context.Background())
	if err != nil {
		t.Fatalf("ListTags: %v", err)
	}

	// Gruppen in der Reihenfolge der Startkategorien, nicht alphabetisch.
	var names []string
	total := 0
	for _, g := range groups {
		names = append(names, g.Name)
		total += len(g.Tags)
		for _, tag := range g.Tags {
			if tag.Group != g.Name {
				t.Errorf("Kategorie %q mit Gruppe %q steht in Gruppe %q", tag.Name, tag.Group, g.Name)
			}
		}
	}
	if want := []string{"Gang", "Hauptzutat", "Geschmack", "Art", "Eigenschaften"}; !slices.Equal(names, want) {
		t.Errorf("Gruppen = %q, erwartet %q", names, want)
	}
	if total != 31 {
		t.Errorf("%d Kategorien, erwartet 31", total)
	}
	// Die Gang-Gruppe ist gekennzeichnet (Migration 013).
	if groups[0].Role != RoleCourse || groups[1].Role != "" || groups[0].Tags[0].Name != "Vorspeise" {
		t.Errorf("Gang = %+v", groups[0])
	}

	// Innerhalb der Gruppe nach position.
	first := groups[1].Tags
	if first[0].Name != "Vegetarisch" || first[len(first)-1].Name != "Meeresfrüchte" {
		t.Errorf("Hauptzutat = %+v", first)
	}
}

func TestListTagsOverHTTP(t *testing.T) {
	rec := do(newTestMux(t), "GET", "/api/v1/tags", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("Status = %d, erwartet 200; Antwort: %s", rec.Code, rec.Body)
	}
	var resp struct {
		Groups []TagGroup `json:"groups"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Antwort ist keine Kategorieliste: %v", err)
	}
	if len(resp.Groups) != 5 || resp.Groups[0].Tags[0].ID == 0 || resp.Groups[0].Role != RoleCourse {
		t.Errorf("Antwort = %s", rec.Body)
	}
}
