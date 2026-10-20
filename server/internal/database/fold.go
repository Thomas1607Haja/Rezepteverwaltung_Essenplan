package database

import (
	"database/sql/driver"
	"strings"

	"modernc.org/sqlite"
)

// foldReplacer ersetzt Umlaute und ß durch ihre Grundbuchstaben.
var foldReplacer = strings.NewReplacer("ä", "a", "ö", "o", "ü", "u", "ß", "ss")

// Fold bereitet einen Text für Suche und Sortierung vor: Kleinbuchstaben,
// Umlaute ohne Punkte, ß als ss. So findet "apfel" auch "Äpfel", und
// "Äpfel" wird wie "Apfel" einsortiert (wie im Telefonbuch).
func Fold(s string) string {
	return foldReplacer.Replace(strings.ToLower(s))
}

// init läuft automatisch einmal beim Programmstart, bevor main beginnt.
// Hier meldet es Fold als SQL-Funktion fold(text) beim Treiber an, denn
// SQLites eigene Funktionen lower() und LIKE kennen nur ASCII.
func init() {
	sqlite.MustRegisterDeterministicScalarFunction("fold", 1,
		func(_ *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
			// args[0] ist vom Typ any; der Typ-Switch prüft, was drinsteckt.
			switch v := args[0].(type) {
			case string:
				return Fold(v), nil
			case []byte:
				return Fold(string(v)), nil
			default:
				return v, nil // NULL und Zahlen unverändert lassen
			}
		})
}
