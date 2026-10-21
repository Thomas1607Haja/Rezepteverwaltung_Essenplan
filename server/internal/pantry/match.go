package pantry

import (
	"context"
	"regexp"
	"strings"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/database"
)

// Matcher prüft Zutatennamen gegen die Vorratsliste. Er enthält einen
// Schnappschuss der Liste und wird pro Anfrage neu erzeugt.
type Matcher struct {
	// terms enthält alle Namen und Synonyme in gefalteter Form (siehe
	// database.Fold). Eine Map mit leerem struct{} als Wert ist in Go die
	// übliche Form einer Menge: Sie belegt für den Wert keinen Speicher.
	terms map[string]struct{}
}

// Matcher lädt die aktuelle Vorratsliste.
func (s *Store) Matcher(ctx context.Context) (*Matcher, error) {
	items, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	m := &Matcher{terms: make(map[string]struct{})}
	for _, it := range items {
		m.terms[foldTerm(it.Name)] = struct{}{}
		for _, syn := range it.Synonyms {
			m.terms[foldTerm(syn)] = struct{}{}
		}
	}
	return m, nil
}

var (
	// parenthesized trifft Klammerzusätze wie in "Zwiebel(n)" oder
	// "Öl (zum Braten)".
	parenthesized = regexp.MustCompile(`\([^)]*\)`)
	// combined trennt Aufzählungen wie "Salz und Pfeffer" oder "Salz/Pfeffer".
	combined = regexp.MustCompile(`\s+(?:und|oder|&)\s+|/|\+`)
)

// foldTerm bringt einen Namen in die Vergleichsform: gefaltet und mit
// einfachen Leerzeichen.
func foldTerm(s string) string {
	return strings.Join(strings.Fields(database.Fold(s)), " ")
}

// Matches meldet, ob eine Zutat mit diesem Namen im Vorrat ist. Verglichen
// wird der ganze Name ohne Klammerzusätze und ohne alles nach dem ersten
// Komma ("Olivenöl, extra vergine" → "Olivenöl"). Eine Aufzählung wie
// "Salz und Pfeffer" passt, wenn alle Teile im Vorrat sind.
// Teilwörter zählen bewusst nicht: "Salzbrezel" ist kein Salz.
func (m *Matcher) Matches(name string) bool {
	name = parenthesized.ReplaceAllString(name, " ")
	name, _, _ = strings.Cut(name, ",")
	name = foldTerm(name)
	if name == "" {
		return false
	}
	if _, ok := m.terms[name]; ok {
		return true
	}
	parts := combined.Split(name, -1)
	if len(parts) < 2 {
		return false
	}
	for _, part := range parts {
		if _, ok := m.terms[strings.TrimSpace(part)]; !ok {
			return false
		}
	}
	return true
}
