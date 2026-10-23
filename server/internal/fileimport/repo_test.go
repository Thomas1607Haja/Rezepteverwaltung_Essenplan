package fileimport

import (
	"bytes"
	"context"
	"os"
	"slices"
	"strings"
	"testing"
)

// Pfade relativ zu diesem Paket (server/internal/fileimport).
const (
	repoRoot   = "../../../"
	skillDir   = repoRoot + "skills/rezeptplaner-import/"
	docsSchema = repoRoot + "docs/importformat.schema.json"
)

// Die Skill bringt eine Kopie des Schemas mit; sie muss gleich bleiben.
func TestSkillSchemaMatchesDocs(t *testing.T) {
	docs, err := os.ReadFile(docsSchema)
	if err != nil {
		t.Fatal(err)
	}
	skill, err := os.ReadFile(skillDir + "schema.json")
	if err != nil {
		t.Fatal(err)
	}
	norm := func(b []byte) []byte { return bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n")) }
	if !bytes.Equal(norm(docs), norm(skill)) {
		t.Error("skills/rezeptplaner-import/schema.json weicht von docs/importformat.schema.json ab – Kopie aktualisieren")
	}
}

// Die Kategorieliste der Skill muss die Startkategorien enthalten.
func TestSkillCategoriesMatchSeed(t *testing.T) {
	f := newFixture(t)
	groups, err := f.recipes.ListTags(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var seed []string
	for _, g := range groups {
		for _, tag := range g.Tags {
			seed = append(seed, g.Name+"/"+tag.Name)
		}
	}

	data, err := os.ReadFile(skillDir + "categories.md")
	if err != nil {
		t.Fatal(err)
	}
	var listed []string
	group := ""
	for _, line := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		switch {
		case strings.HasPrefix(line, "## Hinweise"):
			group = ""
		case strings.HasPrefix(line, "## "):
			group = strings.TrimPrefix(line, "## ")
		case group != "" && strings.HasPrefix(line, "- "):
			listed = append(listed, group+"/"+strings.TrimPrefix(line, "- "))
		}
	}
	slices.Sort(seed)
	slices.Sort(listed)
	if !slices.Equal(seed, listed) {
		t.Errorf("categories.md passt nicht zu den Startkategorien\nDatenbank: %v\nSkill:     %v", seed, listed)
	}
}

// Das Beispiel der Skill muss ohne Fehler importierbar sein.
func TestSkillExampleIsValid(t *testing.T) {
	data, err := os.ReadFile(skillDir + "example/recipes.json")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := parseDocument(data)
	if err != nil {
		t.Fatal(err)
	}
	for i, raw := range doc.Recipes {
		if p := parseRecipe(raw); len(p.errors) > 0 {
			t.Errorf("Rezept %d: %v", i+1, p.errors)
		}
	}
}
