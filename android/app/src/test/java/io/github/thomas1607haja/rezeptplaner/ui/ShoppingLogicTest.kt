package io.github.thomas1607haja.rezeptplaner.ui

import io.github.thomas1607haja.rezeptplaner.api.Period
import io.github.thomas1607haja.rezeptplaner.api.ResetEntry
import io.github.thomas1607haja.rezeptplaner.api.ResetResult
import io.github.thomas1607haja.rezeptplaner.api.TodoList
import io.github.thomas1607haja.rezeptplaner.api.SendResult
import io.github.thomas1607haja.rezeptplaner.api.SentItem
import io.github.thomas1607haja.rezeptplaner.api.ShoppingEntry
import io.github.thomas1607haja.rezeptplaner.api.ShoppingItem
import io.github.thomas1607haja.rezeptplaner.api.ShoppingPreview
import io.github.thomas1607haja.rezeptplaner.ui.shopping.entryReason
import io.github.thomas1607haja.rezeptplaner.ui.shopping.nothingLeft
import io.github.thomas1607haja.rezeptplaner.ui.shopping.resetSummary
import io.github.thomas1607haja.rezeptplaner.ui.shopping.resetTarget
import io.github.thomas1607haja.rezeptplaner.ui.shopping.sentSummary
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith
import androidx.test.ext.junit.runners.AndroidJUnit4
import org.robolectric.annotation.Config

// Robolectric: die Texte kommen aus den Ressourcen (Deutsch).
@RunWith(AndroidJUnit4::class)
@Config(sdk = [36], qualifiers = "de")
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

    @Test
    fun resetShowsTargetListAndOutcome() {
        val e = ResetEntry(1, "2026-11-07", "Gulasch", lists = listOf(TodoList("todo.test", "Test")), items = 3)
        assertEquals("an „Test“", resetTarget(e))
        assertEquals("an unbekannte Liste", resetTarget(e.copy(lists = emptyList(), unknownList = true)))
        assertEquals("keine Artikel gesendet", resetTarget(e.copy(lists = emptyList())))
        assertEquals(
            listOf(
                "2 Gerichte wieder geplant.",
                "Von der alten Liste entfernt: Eier, Milch.",
                "Nicht entfernt, Liste unbekannt: Mehl.",
                "Jetzt „An Bring senden“ – es geht an „Einkauf Familie“.",
            ),
            resetSummary(ResetResult(entries = 2, removed = listOf("Eier", "Milch"), skipped = listOf("Mehl")), "Einkauf Familie"),
        )
    }
}
