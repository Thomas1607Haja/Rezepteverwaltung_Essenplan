// Package itemkey bildet den Vergleichsschlüssel einer Zutat, unter dem die
// Einkaufsliste gleiche Zutaten zusammenfasst ("Zwiebel" = "Zwiebeln (rot)").
// Derselbe Schlüssel kennzeichnet abgewählte Zutaten eines Planeintrags
// (meal_plan_entries.excluded_ingredients); die Rezepte liefern ihn je Zutat
// als item_key mit, damit Weboberfläche und App ihn nicht nachbauen müssen.
package itemkey

import (
	"regexp"
	"strings"

	"github.com/Thomas1607Haja/Rezepteverwaltung_Essenplan/server/internal/database"
)

// parenthesized trifft Klammerzusätze wie in "Zwiebel(n)" oder
// "Tomaten (aus der Dose)".
var parenthesized = regexp.MustCompile(`\s*\([^)]*\)`)

// Clean entfernt Klammerzusätze und doppelte Leerzeichen.
func Clean(name string) string {
	return strings.Join(strings.Fields(parenthesized.ReplaceAllString(name, "")), " ")
}

// Of liefert den Schlüssel eines Zutatennamens.
func Of(name string) string { return NameKey(Clean(name)) }

// NameKey bildet den Vergleichsschlüssel eines (bereinigten) Namens:
// gefaltet und das letzte Wort ohne Pluralendung, damit "Zwiebel" und
// "Zwiebeln" oder "Ei" und "Eier" zusammenfallen.
func NameKey(name string) string {
	words := strings.Fields(database.Fold(name))
	if len(words) == 0 {
		return ""
	}
	words[len(words)-1] = Stem(words[len(words)-1])
	return strings.Join(words, " ")
}

// pluralEndings in der Reihenfolge, in der sie geprüft werden.
var pluralEndings = []string{"en", "er", "n", "e", "s"}

// Stem entfernt eine typische Pluralendung, wenn danach noch mindestens
// zwei Buchstaben übrig bleiben ("eier" → "ei", "tomaten" → "tomat",
// "tomate" → "tomat"). Das ist bewusst grob: Es geht nur darum, dass
// Singular und Plural denselben Schlüssel bekommen.
func Stem(word string) string {
	for _, end := range pluralEndings {
		if rest, ok := strings.CutSuffix(word, end); ok && len([]rune(rest)) >= 2 {
			return rest
		}
	}
	return word
}
