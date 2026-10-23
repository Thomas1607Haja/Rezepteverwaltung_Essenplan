package mealplan

import "time"

// ShoppingPeriod berechnet den Einkaufszeitraum („eingekaufte Woche“), in
// dem der Zeitpunkt now liegt: Er beginnt am letzten Einkaufstag und dauert
// eine Woche. Am Einkaufstag selbst beginnt der neue Zeitraum erst ab der
// Uhrzeit cutoff (Dauer seit Mitternacht, z. B. 18 Stunden) – vorher ist
// noch nicht eingekauft und es gilt der vorige.
//
// Beispiel mit Einkaufstag Samstag: Für Mittwoch, 21.10.2026, ist das
// Samstag, 17.10., bis Freitag, 23.10. Mit cutoff 18:00 gilt am Samstag,
// 24.10., um 10 Uhr noch der 17.10.–23.10., ab 18 Uhr der 24.10.–30.10.
func ShoppingPeriod(now time.Time, shoppingDay time.Weekday, cutoff time.Duration) Period {
	y, m, d := now.Date()
	today := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	sinceMidnight := time.Duration(now.Hour())*time.Hour + time.Duration(now.Minute())*time.Minute

	// Wie viele Tage liegt der letzte Einkaufstag zurück? Das "+ 7" hält
	// das Ergebnis von % positiv, denn in Go kann es sonst negativ sein.
	daysBack := (int(today.Weekday()) - int(shoppingDay) + 7) % 7
	if daysBack == 0 && sinceMidnight < cutoff {
		daysBack = 7
	}
	from := today.AddDate(0, 0, -daysBack)
	to := from.AddDate(0, 0, 6)
	return Period{From: from.Format(dateLayout), To: to.Format(dateLayout)}
}

// Next ist der folgende Einkaufszeitraum („nächster Einkauf“).
func (p Period) Next() Period {
	from, _ := time.Parse(dateLayout, p.From)
	to, _ := time.Parse(dateLayout, p.To)
	return Period{From: from.AddDate(0, 0, 7).Format(dateLayout), To: to.AddDate(0, 0, 7).Format(dateLayout)}
}

// Contains meldet, ob der Tag date ("2026-10-24") im Zeitraum liegt. Bei
// diesem Datumsformat entspricht der Textvergleich dem Zeitvergleich.
func (p Period) Contains(date string) bool {
	return date >= p.From && date <= p.To
}
