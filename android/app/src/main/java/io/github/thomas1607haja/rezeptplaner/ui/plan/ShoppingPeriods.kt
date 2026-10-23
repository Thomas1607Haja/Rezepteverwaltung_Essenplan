package io.github.thomas1607haja.rezeptplaner.ui.plan

import io.github.thomas1607haja.rezeptplaner.api.HouseholdSettings
import io.github.thomas1607haja.rezeptplaner.api.Period
import java.time.DayOfWeek
import java.time.LocalDateTime
import java.time.LocalTime

/** Eingekaufte Woche und nächster Einkauf. */
data class ShoppingPeriods(val bought: Period, val next: Period)

/**
 * Berechnet beide Zeiträume wie der Server (server/internal/mealplan/
 * period.go): Die eingekaufte Woche beginnt am letzten Einkaufstag; am
 * Einkaufstag selbst erst ab der eingestellten Uhrzeit. Die App rechnet
 * selbst mit der Uhr des Handys, damit der Wechsel auch ohne Verbindung
 * pünktlich passiert.
 */
fun shoppingPeriods(settings: HouseholdSettings, now: LocalDateTime): ShoppingPeriods? {
    val day = runCatching { DayOfWeek.valueOf(settings.shoppingDay.uppercase()) }.getOrNull() ?: return null
    val cutoff = runCatching { LocalTime.parse(settings.shoppingTime) }.getOrDefault(LocalTime.MIDNIGHT)
    val today = now.toLocalDate()
    var daysBack = ((today.dayOfWeek.value - day.value) + 7) % 7
    if (daysBack == 0 && now.toLocalTime() < cutoff) daysBack = 7
    val from = today.minusDays(daysBack.toLong())
    val bought = Period(from.toString(), from.plusDays(6).toString())
    val next = Period(from.plusDays(7).toString(), from.plusDays(13).toString())
    return ShoppingPeriods(bought, next)
}
