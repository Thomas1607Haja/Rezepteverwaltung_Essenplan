package mealplan

import "time"

// ShoppingPeriod berechnet den Einkaufszeitraum, in dem today liegt: Er
// beginnt am letzten Einkaufstag (today eingeschlossen) und dauert eine
// Woche. Beispiel mit Einkaufstag Samstag: Für Mittwoch, 21.10.2026, ist das
// Samstag, 17.10., bis Freitag, 23.10.
func ShoppingPeriod(today time.Time, shoppingDay time.Weekday) Period {
	// Wie viele Tage liegt der letzte Einkaufstag zurück? Das "+ 7" hält
	// das Ergebnis von % positiv, denn in Go kann es sonst negativ sein.
	daysBack := (int(today.Weekday()) - int(shoppingDay) + 7) % 7
	from := today.AddDate(0, 0, -daysBack)
	to := from.AddDate(0, 0, 6)
	return Period{From: from.Format(dateLayout), To: to.Format(dateLayout)}
}

// Contains meldet, ob der Tag date ("2026-10-24") im Zeitraum liegt. Bei
// diesem Datumsformat entspricht der Textvergleich dem Zeitvergleich.
func (p Period) Contains(date string) bool {
	return date >= p.From && date <= p.To
}
