package io.github.thomas1607haja.rezeptplaner.ui

import io.github.thomas1607haja.rezeptplaner.api.Period
import io.github.thomas1607haja.rezeptplaner.api.SendResult
import io.github.thomas1607haja.rezeptplaner.api.SentItem
import io.github.thomas1607haja.rezeptplaner.api.ShoppingEntry
import io.github.thomas1607haja.rezeptplaner.api.ShoppingItem
import io.github.thomas1607haja.rezeptplaner.api.ShoppingPreview
import io.github.thomas1607haja.rezeptplaner.ui.shopping.entryReason
import io.github.thomas1607haja.rezeptplaner.ui.shopping.nothingLeft
import io.github.thomas1607haja.rezeptplaner.ui.shopping.sentSummary
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class ShoppingLogicTest {
    private val period = Period("2026-10-23", "2026-10-29")

    @Test
    fun switchesToNextPurchaseOnlyWhenNothingIsLeft() {
        assertTrue(nothingLeft(ShoppingPreview(period, "2026-10-24")))
        assertFalse(nothingLeft(ShoppingPreview(period, "2026-10-24", items = listOf(ShoppingItem("mehl", "Mehl")))))
        assertFalse(nothingLeft(ShoppingPreview(period, "2026-10-24", missing = listOf(ShoppingItem("ei", "Eier")))))
    }

    @Test
    fun reasonsLikeTheWebPage() {
        val base = ShoppingEntry(1, "2026-10-25", recipeId = 3, title = "Lasagne", status = "planned")
        assertEquals("", entryReason(base.copy(included = true), "2026-10-24"))
        assertEquals("ohne Rezept", entryReason(base.copy(recipeId = null), "2026-10-24"))
        assertEquals("schon an Bring gesendet", entryReason(base.copy(status = "transferred"), "2026-10-24"))
        assertEquals("Zutaten vorhanden", entryReason(base.copy(status = "have_ingredients"), "2026-10-24"))
        assertEquals("vorbei", entryReason(base.copy(date = "2026-10-23"), "2026-10-24"))
    }

    @Test
    fun summaryCountsMergedItems() {
        assertEquals("Ein Artikel an Bring gesendet.", sentSummary(SendResult(sent = listOf(SentItem("Mehl")))))
        assertEquals(
            "2 Artikel an Bring gesendet, davon 1 bei schon vorhandenen ergänzt.",
            sentSummary(SendResult(sent = listOf(SentItem("Mehl"), SentItem("Eier", merged = true)))),
        )
    }
}
