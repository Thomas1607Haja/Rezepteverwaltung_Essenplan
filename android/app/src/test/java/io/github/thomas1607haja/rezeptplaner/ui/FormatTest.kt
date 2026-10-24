package io.github.thomas1607haja.rezeptplaner.ui

import io.github.thomas1607haja.rezeptplaner.api.PlanInfo
import org.junit.Assert.assertEquals
import org.junit.Test
import java.time.LocalDate

class FormatTest {
    @Test
    fun quantitiesLikeWeb() {
        // Dieselben Beispiele wie in web/src/lib/format.js.
        assertEquals("333", formatQuantity(333.33))
        assertEquals("12,5", formatQuantity(12.46))
        assertEquals("1 ½", formatQuantity(1.5))
        assertEquals("⅓", formatQuantity(0.33))
        assertEquals("2,2", formatQuantity(2.2))
        assertEquals("2", formatQuantity(1.98))
        assertEquals("", formatQuantity(null))
    }

    @Test
    fun minutes() {
        assertEquals("45 Min.", formatMinutes(45))
        assertEquals("1 Std. 15 Min.", formatMinutes(75))
        assertEquals("2 Std.", formatMinutes(120))
    }

    @Test
    fun status() {
        val today = LocalDate.parse("2026-10-23")
        assertEquals("noch nie eingeplant", planStatus(null, today))
        assertEquals("heute eingeplant", planStatus(PlanInfo(1, nextPlanned = "2026-10-23"), today))
        assertEquals("eingeplant in 5 Tagen", planStatus(PlanInfo(1, nextPlanned = "2026-10-28"), today))
        assertEquals("zuletzt vor 3 Wochen", planStatus(PlanInfo(1, lastPlanned = "2026-10-02"), today))
    }

    @Test
    fun foldAndDay() {
        assertEquals("kasespatzle mit soße".replace("ß", "ss"), fold("Käsespätzle mit Soße"))
        assertEquals("Fr 23.10.", dayLabel("2026-10-23"))
    }

    @Test
    fun wrapTitleAllowsBreakAfterHyphen() {
        assertEquals("Romanesco-\u200BTomaten-\u200BPasta", wrapTitle("Romanesco-Tomaten-Pasta"))
        // Zahlen und Gedankenstriche bleiben unverändert.
        assertEquals("Pizza 2-3 Personen – schnell", wrapTitle("Pizza 2-3 Personen – schnell"))
    }
}
