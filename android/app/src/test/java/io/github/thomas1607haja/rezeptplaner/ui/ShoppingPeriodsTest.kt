package io.github.thomas1607haja.rezeptplaner.ui

import io.github.thomas1607haja.rezeptplaner.api.HouseholdSettings
import io.github.thomas1607haja.rezeptplaner.api.Period
import io.github.thomas1607haja.rezeptplaner.ui.plan.shoppingPeriods
import org.junit.Assert.assertEquals
import org.junit.Test
import java.time.LocalDateTime

/** Dieselben Fälle wie TestShoppingPeriodWithTime im Server. */
class ShoppingPeriodsTest {
    private val thursday18 = HouseholdSettings(shoppingDay = "thursday", shoppingTime = "18:00")

    private fun bought(now: String) = shoppingPeriods(thursday18, LocalDateTime.parse(now))!!.bought

    @Test
    fun cutoffOnShoppingDay() {
        assertEquals(Period("2026-10-15", "2026-10-21"), bought("2026-10-21T23:59"))
        assertEquals(Period("2026-10-15", "2026-10-21"), bought("2026-10-22T09:30"))
        assertEquals(Period("2026-10-15", "2026-10-21"), bought("2026-10-22T17:59"))
        assertEquals(Period("2026-10-22", "2026-10-28"), bought("2026-10-22T18:00"))
        assertEquals(Period("2026-10-22", "2026-10-28"), bought("2026-10-23T08:00"))
    }

    @Test
    fun nextAndDefaults() {
        val p = shoppingPeriods(thursday18, LocalDateTime.parse("2026-10-23T08:00"))!!
        assertEquals(Period("2026-10-29", "2026-11-04"), p.next)
        // Ohne Uhrzeit (älterer Server): ab Mitternacht wie bisher.
        val old = shoppingPeriods(HouseholdSettings(shoppingDay = "saturday"), LocalDateTime.parse("2026-10-24T00:01"))!!
        assertEquals(Period("2026-10-24", "2026-10-30"), old.bought)
    }
}
