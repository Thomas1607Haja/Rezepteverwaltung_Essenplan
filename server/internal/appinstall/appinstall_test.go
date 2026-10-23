package appinstall

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestLoadRepository(t *testing.T) {
	dir := t.TempDir()
	if got := LoadRepository(dir); got != DefaultRepository {
		t.Errorf("ohne options.json: %q", got)
	}
	tests := map[string]string{
		`{"app_repository": "beispiel/rezepte"}`:                     "beispiel/rezepte",
		`{"app_repository": "https://github.com/beispiel/rezepte/"}`: "beispiel/rezepte",
		`{"app_repository": ""}`:                                     DefaultRepository,
		`{"app_repository": "../../etc"}`:                            DefaultRepository,
		`{"app_repository": "a/b?x=1"}`:                              DefaultRepository,
		`kein json`:                                                  DefaultRepository,
	}
	for content, want := range tests {
		if err := os.WriteFile(filepath.Join(dir, "options.json"), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		if got := LoadRepository(dir); got != want {
			t.Errorf("%s: %q, erwartet %q", content, got, want)
		}
	}
}

func TestLinks(t *testing.T) {
	l := NewLinks("beispiel/rezepte")
	if l.APKURL != "https://github.com/beispiel/rezepte/releases/latest/download/rezeptplaner.apk" {
		t.Errorf("APK: %s", l.APKURL)
	}
	if l.SkillURL != "https://github.com/beispiel/rezepte/releases/latest/download/rezeptplaner-import-skill.zip" {
		t.Errorf("Skill: %s", l.SkillURL)
	}

	// Obtainium-Link zurückverfolgen: Weiterleitungsseite → obtainium://app/<JSON>.
	u, err := url.Parse(l.ObtainiumURL)
	if err != nil || u.Host != "apps.obtainium.imranr.dev" || u.Path != "/redirect" {
		t.Fatalf("Obtainium-Link: %s", l.ObtainiumURL)
	}
	raw, ok := strings.CutPrefix(u.Query().Get("r"), "obtainium://app/")
	if !ok {
		t.Fatalf("r = %q", u.Query().Get("r"))
	}
	var config struct {
		ID                 string `json:"id"`
		URL                string `json:"url"`
		Author             string `json:"author"`
		AdditionalSettings string `json:"additionalSettings"` // JSON als Text
	}
	if err := json.Unmarshal([]byte(raw), &config); err != nil {
		t.Fatalf("JSON: %v\n%s", err, raw)
	}
	if config.ID != PackageName || config.URL != "https://github.com/beispiel/rezepte" || config.Author != "beispiel" {
		t.Errorf("config: %+v", config)
	}
	var settings map[string]any
	if err := json.Unmarshal([]byte(config.AdditionalSettings), &settings); err != nil {
		t.Fatalf("additionalSettings: %v", err)
	}
	// Die Version muss aus dem Tag der GitHub Action kommen.
	version := regexp.MustCompile(settings["versionExtractionRegEx"].(string))
	if got := version.FindString("android-v0.1.0"); got != "0.1.0" {
		t.Errorf("Version aus Tag: %q", got)
	}
	if len(l.ObtainiumURL) > 500 {
		t.Errorf("Obtainium-Link %d Zeichen – QR-Code wird zu dicht", len(l.ObtainiumURL))
	}
}
