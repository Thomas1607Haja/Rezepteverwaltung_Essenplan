package io.github.thomas1607haja.rezeptplaner.ui

import io.github.thomas1607haja.rezeptplaner.api.PlanInfo
import java.text.DecimalFormat
import java.text.DecimalFormatSymbols
import java.time.LocalDate
import java.time.temporal.ChronoUnit
import java.util.Locale
import kotlin.math.abs
import kotlin.math.floor
import kotlin.math.roundToInt

// Anzeige wie in der Weboberfläche (web/src/lib/format.js).

/** 45 -> "45 Min.", 75 -> "1 Std. 15 Min.". */
fun formatMinutes(minutes: Int?): String {
    if (minutes == null) return ""
    val hours = minutes / 60
    val rest = minutes % 60
    return when {
        hours == 0 -> "$rest Min."
        rest == 0 -> "$hours Std."
        else -> "$hours Std. $rest Min."
    }
}

private val fractions = listOf(1.0 / 4 to "¼", 1.0 / 3 to "⅓", 1.0 / 2 to "½", 2.0 / 3 to "⅔", 3.0 / 4 to "¾")

/** Menge, wie man sie in der Küche liest: 333.3 -> "333", 12.46 -> "12,5", 1.5 -> "1 ½". */
fun formatQuantity(value: Double?): String {
    if (value == null) return ""
    if (value >= 100) return value.roundToInt().toString()
    if (value >= 10) return decimal(value, 1)
    val whole = floor(value).toInt()
    val part = value - whole
    if (part < 0.04) return whole.toString()
    if (part > 0.96) return (whole + 1).toString()
    for ((fraction, symbol) in fractions) {
        if (abs(part - fraction) < 0.04) return if (whole == 0) symbol else "$whole $symbol"
    }
    return decimal(value, 2)
}

private fun decimal(value: Double, digits: Int): String {
    val format = DecimalFormat("0." + "#".repeat(digits), DecimalFormatSymbols(Locale.GERMANY))
    return format.format(value)
}

/** Statushinweis: „heute eingeplant“, „eingeplant in 5 Tagen“, „zuletzt vor 2 Wochen“, „noch nie eingeplant“. */
fun planStatus(info: PlanInfo?, today: LocalDate): String {
    info?.nextPlanned?.let {
        val days = ChronoUnit.DAYS.between(today, LocalDate.parse(it))
        return when {
            days <= 0 -> "heute eingeplant"
            days == 1L -> "morgen eingeplant"
            days < 14 -> "eingeplant in $days Tagen"
            else -> "eingeplant in ${(days / 7.0).roundToInt()} Wochen"
        }
    }
    info?.lastPlanned?.let {
        val days = ChronoUnit.DAYS.between(LocalDate.parse(it), today)
        return when {
            days <= 1 -> "zuletzt gestern"
            days < 14 -> "zuletzt vor $days Tagen"
            days < 60 -> "zuletzt vor ${(days / 7.0).roundToInt()} Wochen"
            days < 365 -> "zuletzt vor ${(days / 30.0).roundToInt()} Monaten"
            days < 730 -> "zuletzt vor über einem Jahr"
            else -> "zuletzt vor über ${days / 365} Jahren"
        }
    }
    return "noch nie eingeplant"
}

private val weekdays = listOf("Mo", "Di", "Mi", "Do", "Fr", "Sa", "So")

/** "2026-10-21" -> "Mi 21.10." */
fun dayLabel(date: String): String {
    val d = LocalDate.parse(date)
    return "${weekdays[d.dayOfWeek.value - 1]} %02d.%02d.".format(d.dayOfMonth, d.monthValue)
}

/** Für die Suche: klein, Umlaute und ß wie in server/internal/database/fold.go. */
fun fold(s: String): String =
    s.lowercase(Locale.GERMAN).replace("ä", "a").replace("ö", "o").replace("ü", "u").replace("ß", "ss")

fun portions(n: Int) = if (n == 1) "1 Portion" else "$n Portionen"

/** Menge mit Ergebnis-Einheit (Schritt 9.2): „4 Portionen“ bzw. „1 Springform 26 cm“, „12 Stück“. */
fun amountText(n: Int, unit: String) = if (unit.isEmpty()) portions(n) else "$n $unit"

/** Vorschläge für die Ergebnis-Einheit im Editor (wie im Web). */
val yieldUnits = listOf("Stück", "Kuchen", "Blech", "Springform 26 cm", "Kastenform", "Laib")
