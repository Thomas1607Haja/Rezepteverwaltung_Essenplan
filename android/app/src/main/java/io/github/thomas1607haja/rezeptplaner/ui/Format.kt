package io.github.thomas1607haja.rezeptplaner.ui

import io.github.thomas1607haja.rezeptplaner.R
import io.github.thomas1607haja.rezeptplaner.api.PlanInfo
import io.github.thomas1607haja.rezeptplaner.ui.common.Texts
import java.text.DecimalFormat
import java.text.DecimalFormatSymbols
import java.time.LocalDate
import java.time.format.DateTimeFormatter
import java.time.format.TextStyle
import java.time.temporal.ChronoUnit
import java.util.Locale
import kotlin.math.abs
import kotlin.math.floor
import kotlin.math.roundToInt

// Anzeige wie in der Weboberfläche (web/src/lib/format.js). Texte und
// Formate folgen der Sprache der App (Texts, res/values…/strings_shared.xml).

/** 45 -> "45 Min.", 75 -> "1 Std. 15 Min." (bzw. "1 h 15 min"). */
fun formatMinutes(minutes: Int?): String {
    if (minutes == null) return ""
    val hours = minutes / 60
    val rest = minutes % 60
    return when {
        hours == 0 -> Texts.get(R.string.format_minutes, "count" to rest)
        rest == 0 -> Texts.get(R.string.format_hours, "count" to hours)
        else -> Texts.get(R.string.format_hours_minutes, "hours" to hours, "minutes" to rest)
    }
}

private val fractions = listOf(1.0 / 4 to "¼", 1.0 / 3 to "⅓", 1.0 / 2 to "½", 2.0 / 3 to "⅔", 3.0 / 4 to "¾")

/** Menge, wie man sie in der Küche liest: 333.3 -> "333", 12.46 -> "12,5" (bzw. "12.5"), 1.5 -> "1 ½". */
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

/** Zahl mit höchstens [digits] Nachkommastellen im Format der Sprache. */
fun formatNumber(value: Double, digits: Int = 2): String = decimal(value, digits)

private fun decimal(value: Double, digits: Int): String {
    val format = DecimalFormat("0." + "#".repeat(digits), DecimalFormatSymbols(Texts.locale()))
    return format.format(value)
}

/** Statushinweis: „heute eingeplant“, „eingeplant in 5 Tagen“, „zuletzt vor 2 Wochen“, „noch nie eingeplant“. */
fun planStatus(info: PlanInfo?, today: LocalDate): String {
    info?.nextPlanned?.let {
        val days = ChronoUnit.DAYS.between(today, LocalDate.parse(it)).toInt()
        return when {
            days <= 0 -> Texts.get(R.string.format_planned_today)
            days == 1 -> Texts.get(R.string.format_planned_tomorrow)
            days < 14 -> Texts.plural(R.plurals.format_planned_in_days, days)
            else -> Texts.plural(R.plurals.format_planned_in_weeks, (days / 7.0).roundToInt())
        }
    }
    info?.lastPlanned?.let {
        val days = ChronoUnit.DAYS.between(LocalDate.parse(it), today).toInt()
        return when {
            days <= 1 -> Texts.get(R.string.format_last_yesterday)
            days < 14 -> Texts.plural(R.plurals.format_last_days, days)
            days < 60 -> Texts.plural(R.plurals.format_last_weeks, (days / 7.0).roundToInt())
            days < 365 -> Texts.plural(R.plurals.format_last_months, (days / 30.0).roundToInt())
            else -> Texts.plural(R.plurals.format_last_years, days / 365)
        }
    }
    return Texts.get(R.string.format_never_planned)
}

/**
 * "2026-10-21" -> "Mi 21.10." bzw. "Wed 10/21", "Mer 21/10": kurzer
 * Wochentag ohne Punkt, Tag und Monat in der Reihenfolge der Sprache.
 */
fun dayLabel(date: String): String {
    val d = LocalDate.parse(date)
    val locale = Texts.locale()
    val weekday = d.dayOfWeek.getDisplayName(TextStyle.SHORT, locale).removeSuffix(".")
        .replaceFirstChar { it.titlecase(locale) }
    return "$weekday ${d.format(DateTimeFormatter.ofPattern(dayMonthPattern(locale), locale))}"
}

/** Tag und Monat: Deutsch "21.10.", amerikanisches Englisch "10/21", sonst "21/10". */
private fun dayMonthPattern(locale: Locale): String = when {
    locale.language == "de" -> "dd.MM."
    locale.language == "en" && (locale.country.isEmpty() || locale.country == "US") -> "MM/dd"
    else -> "dd/MM"
}

/** Für die Suche: klein, Umlaute und ß wie in server/internal/database/fold.go. */
fun fold(s: String): String =
    s.lowercase(Locale.GERMAN).replace("ä", "a").replace("ö", "o").replace("ü", "u").replace("ß", "ss")

/** Mahlzeiten eines Tages (Schritt 9.3), wie web/src/lib/slots.js. */
fun slotLabel(slot: String) = when (slot) {
    "lunch" -> Texts.get(R.string.slots_lunch)
    "coffee" -> Texts.get(R.string.slots_coffee)
    else -> Texts.get(R.string.slots_dinner)
}

/** Vorsatz im Plan: „Mittag: “, „Kaffee: “; Abendessen ohne. */
fun slotPrefix(slot: String) = when (slot) {
    "lunch" -> Texts.get(R.string.slots_lunch_short) + ": "
    "coffee" -> Texts.get(R.string.slots_coffee_short) + ": "
    else -> ""
}

/** „mittags“, „nachmittags“ bzw. leer fürs Abendessen. */
fun slotWhen(slot: String) = when (slot) {
    "lunch" -> Texts.get(R.string.slots_lunch_when)
    "coffee" -> Texts.get(R.string.slots_coffee_when)
    else -> ""
}

/** Reihenfolge der Slots an einem Tag: Mittag, Kaffee, Abend. */
fun slotOrder(slot: String) = when (slot) {
    "lunch" -> 0
    "coffee" -> 1
    else -> 2
}

fun portions(n: Int) = Texts.plural(R.plurals.format_servings, n)

/** Menge mit Ergebnis-Einheit (Schritt 9.2): „4 Portionen“ bzw. „1 Springform 26 cm“, „12 Stück“. */
fun amountText(n: Int, unit: String) = if (unit.isEmpty()) portions(n) else "$n $unit"

/** Vorschläge für die Ergebnis-Einheit im Editor (wie im Web, in der Sprache der App). */
val yieldUnits: List<String>
    get() = Texts.get(R.string.format_yield_units).split("|")

/**
 * Titel für mehrzeilige Anzeige: Android bricht zusammengesetzte Wörter
 * nicht am Bindestrich um, sondern mitten im Wort („Romanesco-Tomaten-Pa“ /
 * „sta“). Ein unsichtbares Leerzeichen (U+200B) nach dem Bindestrich erlaubt
 * dort den Umbruch.
 */
fun wrapTitle(title: String): String = title.replace(Regex("-(?=\\p{L})"), "-​")
