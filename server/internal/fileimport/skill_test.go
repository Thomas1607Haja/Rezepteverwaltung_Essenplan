package fileimport

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"testing"
)

func TestSkillVersion(t *testing.T) {
	cases := []struct {
		generator, want string
		ok              bool
	}{
		{"claude-skill rezeptplaner-import 2.4", "2.4", true},
		{"  claude-skill rezeptplaner-import 2.10.1 ", "2.10.1", true},
		{"claude-skill rezeptplaner-import", "", false},
		{"claude-skill rezeptplaner-import beta", "", false},
		{"test", "", false},
		{"", "", false},
	}
	for _, c := range cases {
		got, ok := skillVersion(c.generator)
		if got != c.want || ok != c.ok {
			t.Errorf("skillVersion(%q) = %q, %v; erwartet %q, %v", c.generator, got, ok, c.want, c.ok)
		}
	}
}

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		sign int
	}{
		{"2.4", "2.5", -1},
		{"2.10", "2.9", 1},
		{"2.4", "2.4.0", 0},
		{"3", "2.99", 1},
	}
	for _, c := range cases {
		got := compareVersions(c.a, c.b)
		if (got < 0 && c.sign >= 0) || (got > 0 && c.sign <= 0) || (got == 0 && c.sign != 0) {
			t.Errorf("compareVersions(%q, %q) = %d, erwartet Vorzeichen %d", c.a, c.b, got, c.sign)
		}
	}
}

func TestCheckSkill(t *testing.T) {
	if got := checkSkill([]string{"test", ""}, "u"); got != nil {
		t.Errorf("ohne Skill-Datei: %+v, erwartet nil", got)
	}
	got := checkSkill([]string{"claude-skill rezeptplaner-import " + CurrentSkillVersion, "claude-skill rezeptplaner-import 2.0", "test"}, "https://example.org/skill.zip")
	if got == nil || got.Version != "2.0" || !got.UpdateAvailable || got.Current != CurrentSkillVersion || got.DownloadURL != "https://example.org/skill.zip" {
		t.Errorf("älteste Version: %+v", got)
	}
	got = checkSkill([]string{"claude-skill rezeptplaner-import " + CurrentSkillVersion}, "u")
	if got == nil || got.UpdateAvailable {
		t.Errorf("aktuelle Version: %+v", got)
	}
	got = checkSkill([]string{"claude-skill rezeptplaner-import 99.0"}, "u")
	if got == nil || got.UpdateAvailable {
		t.Errorf("neuere Skill als Server: %+v", got)
	}
}

// Die Vorschau meldet die Skill-Version der hochgeladenen Datei.
func TestPreviewReportsSkill(t *testing.T) {
	f := newFixture(t)
	doc := func(generator string) []byte {
		return fmt.Appendf(nil, `{"format":"rezeptplaner-import","version":3,"generator":%q,"recipes":[{"title":"Linsensuppe","base_servings":4}]}`, generator)
	}
	code, p, body := f.upload(t, namedFile{"alt.json", doc("claude-skill rezeptplaner-import 2.0")})
	if code != http.StatusOK || p.Skill == nil || p.Skill.Version != "2.0" || !p.Skill.UpdateAvailable {
		t.Fatalf("alte Skill: %d %s", code, body)
	}
	code, p, body = f.upload(t, namedFile{"fremd.json", doc("test")})
	if code != http.StatusOK || p.Skill != nil {
		t.Fatalf("ohne Skill: %d %s", code, body)
	}
}

// CurrentSkillVersion muss zur Skill im Repository passen.
func TestSkillVersionMatchesBuildScript(t *testing.T) {
	data, err := os.ReadFile("../../../skills/rezeptplaner-import/scripts/build_import.py")
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`(?m)^GENERATOR = "claude-skill rezeptplaner-import ([0-9.]+)"`).FindSubmatch(data)
	if m == nil {
		t.Fatal("GENERATOR in build_import.py nicht gefunden")
	}
	if string(m[1]) != CurrentSkillVersion {
		t.Errorf("build_import.py hat Skill-Version %s, CurrentSkillVersion ist %s", m[1], CurrentSkillVersion)
	}
	skill, err := os.ReadFile("../../../skills/rezeptplaner-import/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`Skill-Version ` + regexp.QuoteMeta(CurrentSkillVersion) + `\b`).Match(skill) {
		t.Errorf("SKILL.md nennt nicht Skill-Version %s", CurrentSkillVersion)
	}
}
