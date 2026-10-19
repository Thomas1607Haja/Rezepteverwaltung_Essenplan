package database

import "time"

// timeLayout ist das Format aller Zeitstempel in der Datenbank: UTC mit
// Millisekunden und immer gleicher Länge. Dadurch sortieren und vergleichen
// sich die Zeitstempel als Text genauso wie als Zeitpunkt.
//
// Go beschreibt Formate mit einem festen Beispieldatum statt mit Platzhaltern
// wie "YYYY": Montag, 2. Januar 2006, 15:04:05 (merken: 1 2 3 4 5 6).
const timeLayout = "2006-01-02T15:04:05.000Z"

// FormatTime wandelt t in das Datenbankformat um.
func FormatTime(t time.Time) string {
	return t.UTC().Format(timeLayout)
}

// ParseTime liest einen Zeitstempel im Datenbankformat.
func ParseTime(s string) (time.Time, error) {
	return time.Parse(timeLayout, s)
}
