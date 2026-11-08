// Package locales enthält nur Tests: Sie prüfen, dass alle Sprachdateien
// des Repositorys (Weboberfläche, Dashboard-Karte, Integration, Add-on,
// Android-App) dieselben Schlüssel und Platzhalter haben. Die Dateien
// liegen außerhalb des Go-Moduls; die Tests lesen sie über relative Pfade
// (go test läuft im Ordner des Pakets).
package locales

import (
	"encoding/json"
	"encoding/xml"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/apierr"
)

// repo ist der Hauptordner des Repositorys, von server/internal/locales aus.
const repo = "../../.."

// languages sind alle unterstützten Sprachen; Englisch ist die Rückfallsprache.
var languages = []string{"en", "de", "fr", "es", "it"}

// pluralForms sind die Kategorien von Intl.PluralRules bzw. CLDR.
var pluralForms = map[string]bool{"zero": true, "one": true, "two": true, "few": true, "many": true, "other": true}

// entries ist eine Sprachdatei als flache Liste: Schlüssel → Text. Pluralformen
// erscheinen als "schlüssel#one", "schlüssel#other" usw.
type entries map[string]string

// flattenJSON macht aus verschachteltem JSON flache Schlüssel ("a.b.c").
// Ein Objekt nur aus Pluralformen gilt als ein Eintrag mit Pluralformen.
func flattenJSON(prefix string, v any, out entries) {
	switch t := v.(type) {
	case map[string]any:
		if isPlural(t) {
			for form, text := range t {
				s, _ := text.(string)
				out[prefix+"#"+form] = s
			}
			return
		}
		for k, child := range t {
			key := k
			if prefix != "" {
				key = prefix + "." + k
			}
			flattenJSON(key, child, out)
		}
	case string:
		out[prefix] = t
	default:
		out[prefix] = ""
	}
}

func isPlural(m map[string]any) bool {
	if len(m) == 0 {
		return false
	}
	for k := range m {
		if !pluralForms[k] {
			return false
		}
	}
	return true
}

// baseKeys liefert die Schlüssel ohne Pluralform; Pluralformen dürfen sich
// je Sprache unterscheiden (Französisch kennt "many", Deutsch nicht).
func baseKeys(e entries) []string {
	set := map[string]bool{}
	for k := range e {
		base, _, _ := strings.Cut(k, "#")
		set[base] = true
	}
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// placeholders findet {name} (Web, Karte, Integration) und %1$s, %d (Android).
var placeholderPattern = regexp.MustCompile(`\{[a-z_][a-z0-9_]*\}|%(?:\d+\$)?[sd]`)

func placeholders(text string) []string {
	found := placeholderPattern.FindAllString(text, -1)
	slices.Sort(found)
	return slices.Compact(found)
}

// placeholdersOf sammelt die Platzhalter aller Pluralformen eines Schlüssels.
// "{count}" darf in der Form "one" fehlen ("1 Portion" / "one serving").
func placeholdersOf(e entries, base string) []string {
	var all []string
	for k, text := range e {
		if k == base || strings.HasPrefix(k, base+"#") {
			all = append(all, placeholders(text)...)
		}
	}
	slices.Sort(all)
	return slices.Compact(all)
}

// compare prüft alle Sprachen gegen Englisch.
func compare(t *testing.T, name string, files map[string]entries) {
	t.Helper()
	if len(files) == 0 {
		t.Skipf("%s: noch keine Sprachdateien", name)
	}
	en, ok := files["en"]
	if !ok {
		t.Fatalf("%s: englische Datei fehlt (Rückfallsprache)", name)
	}
	enKeys := baseKeys(en)
	for _, lang := range languages {
		e, ok := files[lang]
		if !ok {
			t.Errorf("%s: Sprache %s fehlt", name, lang)
			continue
		}
		keys := baseKeys(e)
		for _, k := range enKeys {
			if !slices.Contains(keys, k) {
				t.Errorf("%s [%s]: Schlüssel %q fehlt", name, lang, k)
			}
		}
		for _, k := range keys {
			if !slices.Contains(enKeys, k) {
				t.Errorf("%s [%s]: Schlüssel %q gibt es im Englischen nicht", name, lang, k)
				continue
			}
			if _, plural := e[k+"#other"]; !plural {
				if _, plain := e[k]; !plain {
					t.Errorf("%s [%s]: %q hat Pluralformen, aber keine Form \"other\"", name, lang, k)
				}
			}
			if strings.TrimSpace(firstText(e, k)) == "" {
				t.Errorf("%s [%s]: %q ist leer", name, lang, k)
			}
			want, got := placeholdersOf(en, k), placeholdersOf(e, k)
			if !slices.Equal(want, got) {
				t.Errorf("%s [%s]: %q hat Platzhalter %v, Englisch %v", name, lang, k, got, want)
			}
		}
	}
}

func firstText(e entries, base string) string {
	if s, ok := e[base]; ok {
		return s
	}
	return e[base+"#other"]
}

// readNamespaces liest <dir>/<bereich>/<sprache>.json und fasst die Bereiche
// je Sprache zusammen (Schlüssel "bereich.…", wie t() in der Weboberfläche).
func readNamespaces(t *testing.T, dir string) map[string]entries {
	t.Helper()
	files := map[string]entries{}
	areas, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return files
	}
	if err != nil {
		t.Fatal(err)
	}
	for _, area := range areas {
		if !area.IsDir() {
			continue
		}
		for lang, e := range readJSONDir(t, filepath.Join(dir, area.Name())) {
			if files[lang] == nil {
				files[lang] = entries{}
			}
			for k, v := range e {
				files[lang][area.Name()+"."+k] = v
			}
		}
	}
	return files
}

// readJSONDir liest <dir>/<sprache>.json für alle Sprachen, die es gibt.
func readJSONDir(t *testing.T, dir string) map[string]entries {
	t.Helper()
	files := map[string]entries{}
	for _, lang := range languages {
		data, err := os.ReadFile(filepath.Join(dir, lang+".json"))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		var v any
		if err := json.Unmarshal(data, &v); err != nil {
			t.Fatalf("%s/%s.json: %v", dir, lang, err)
		}
		e := entries{}
		flattenJSON("", v, e)
		files[lang] = e
	}
	return files
}

func TestWebLocales(t *testing.T) {
	compare(t, "web/src/locales", readNamespaces(t, filepath.Join(repo, "web/src/locales")))
}

// Jeder Fehlercode des Servers braucht einen Text in der Weboberfläche.
func TestWebKnowsAllErrorCodes(t *testing.T) {
	en := readNamespaces(t, filepath.Join(repo, "web/src/locales"))["en"]
	for _, code := range apierr.Codes() {
		if _, ok := en["errors."+code]; !ok {
			t.Errorf("web/src/locales/errors: Code %q fehlt", code)
		}
	}
}

func TestIntegrationTranslations(t *testing.T) {
	compare(t, "custom_components/rezeptplaner/translations",
		readJSONDir(t, filepath.Join(repo, "custom_components/rezeptplaner/translations")))
}

// Texte der Kalendertermine und des Sensors (custom_components/…/texts).
func TestIntegrationTexts(t *testing.T) {
	compare(t, "custom_components/rezeptplaner/texts",
		readJSONDir(t, filepath.Join(repo, "custom_components/rezeptplaner/texts")))
}

// Die Karte ist eine einzelne JS-Datei ohne Build. Ihre Texte stehen als
// reines JSON zwischen den Kommentaren "// i18n:begin" und "// i18n:end"
// ({"en": {...}, "de": {...}, ...}).
func TestCardTranslations(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repo, "custom_components/rezeptplaner/frontend/rezeptplaner-card.js"))
	if err != nil {
		t.Fatal(err)
	}
	_, rest, ok := strings.Cut(string(data), "// i18n:begin")
	if !ok {
		t.Skip("Karte hat noch keine Übersetzungen")
	}
	block, _, ok := strings.Cut(rest, "// i18n:end")
	if !ok {
		t.Fatal("// i18n:end fehlt")
	}
	// Zwischen den Markierungen steht "const TEXTS = {...};".
	start, end := strings.Index(block, "{"), strings.LastIndex(block, "}")
	var all map[string]any
	if err := json.Unmarshal([]byte(block[start:end+1]), &all); err != nil {
		t.Fatalf("Texte der Karte sind kein reines JSON: %v", err)
	}
	files := map[string]entries{}
	for lang, v := range all {
		e := entries{}
		flattenJSON("", v, e)
		files[lang] = e
	}
	compare(t, "rezeptplaner-card.js", files)
}

// Add-on-Optionen: rezeptplaner/translations/<sprache>.yaml. Verglichen
// werden die eingerückten Schlüssel (einfaches YAML ohne Listen).
func TestAddonTranslations(t *testing.T) {
	files := map[string]entries{}
	for _, lang := range languages {
		data, err := os.ReadFile(filepath.Join(repo, "rezeptplaner/translations", lang+".yaml"))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		e := entries{}
		var path []string
		for _, line := range strings.Split(string(data), "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || strings.HasPrefix(trimmed, "#") {
				continue
			}
			depth := (len(line) - len(strings.TrimLeft(line, " "))) / 2
			key, value, _ := strings.Cut(trimmed, ":")
			path = append(path[:min(depth, len(path))], key)
			if v := strings.TrimSpace(value); v != "" {
				e[strings.Join(path, ".")] = strings.Trim(v, `"'`)
			}
		}
		files[lang] = e
	}
	compare(t, "rezeptplaner/translations", files)
}

// Android: res/values (Englisch) und res/values-<sprache>.
func TestAndroidStrings(t *testing.T) {
	res := filepath.Join(repo, "android/app/src/main/res")
	files := map[string]entries{}
	for _, lang := range languages {
		dir := "values-" + lang
		if lang == "en" {
			dir = "values"
		}
		paths, err := filepath.Glob(filepath.Join(res, dir, "strings*.xml"))
		if err != nil {
			t.Fatal(err)
		}
		if len(paths) == 0 {
			continue
		}
		e := entries{}
		for _, path := range paths {
			readAndroid(t, path, e)
		}
		files[lang] = e
	}
	if len(files) < 2 {
		t.Skip("App hat noch keine Übersetzungen")
	}
	compare(t, "android strings*.xml", files)

	// Die Fehlertexte erzeugt android/tools/gen_errors.py aus denen des Webs.
	web := readNamespaces(t, filepath.Join(repo, "web/src/locales"))["en"]
	for key := range web {
		code, ok := strings.CutPrefix(key, "errors.")
		if !ok {
			continue
		}
		if _, ok := files["en"]["error_"+code]; !ok {
			t.Errorf("Android: error_%s fehlt – python android/tools/gen_errors.py ausführen", code)
		}
	}
}

// readAndroid liest eine strings*.xml nach e (Plurals als name#form).
func readAndroid(t *testing.T, path string, e entries) {
	t.Helper()
	{
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var doc struct {
			Strings []struct {
				Name         string `xml:"name,attr"`
				Translatable string `xml:"translatable,attr"`
				Text         string `xml:",chardata"`
			} `xml:"string"`
			Plurals []struct {
				Name  string `xml:"name,attr"`
				Items []struct {
					Quantity string `xml:"quantity,attr"`
					Text     string `xml:",chardata"`
				} `xml:"item"`
			} `xml:"plurals"`
		}
		if err := xml.Unmarshal(data, &doc); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		for _, s := range doc.Strings {
			if s.Translatable == "false" {
				continue
			}
			e[s.Name] = s.Text
		}
		for _, p := range doc.Plurals {
			for _, it := range p.Items {
				e[p.Name+"#"+it.Quantity] = it.Text
			}
		}
	}
}
