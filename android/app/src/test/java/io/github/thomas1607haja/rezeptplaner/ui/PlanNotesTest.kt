package io.github.thomas1607haja.rezeptplaner.ui

import io.github.thomas1607haja.rezeptplaner.api.MealPlanEntry
import io.github.thomas1607haja.rezeptplaner.ui.plan.recentNotes
import io.github.thomas1607haja.rezeptplaner.ui.recipes.RecipeFilter
import io.github.thomas1607haja.rezeptplaner.ui.recipes.RecipeSort
import org.junit.Assert.assertEquals
import org.junit.Test

class PlanNotesTest {
    private fun note(id: Long, date: String, text: String) = MealPlanEntry(id, date, servings = 2, note = text)

    @Test
    fun recentNotesNewestFirstWithoutQuickPicksAndDuplicates() {
        val entries = listOf(
            note(1, "2026-10-10", "Grillen bei Oma"),
            note(2, "2026-10-20", "Essen gehen"), // Schnellauswahl
            note(3, "2026-10-18", "grillen bei oma"), // doppelt, andere Schreibweise
            note(4, "2026-10-15", "Döner"),
            MealPlanEntry(5, "2026-10-21", recipeId = 7, servings = 2, note = "mit Salat"), // hat ein Rezept
        )
        assertEquals(listOf("grillen bei oma", "Döner"), recentNotes(entries))
        assertEquals(listOf("grillen bei oma"), recentNotes(entries, limit = 1))
    }

    @Test
    fun resetKeepsSort() {
        val f = RecipeFilter(query = "x", tagIds = setOf(1), favoritesOnly = true, sort = RecipeSort.Newest)
        assertEquals(RecipeFilter(sort = RecipeSort.Newest), f.reset())
    }
}

class QrUrlTest {
    @Test
    fun onlyHttpAddresses() {
        org.junit.Assert.assertEquals(
            "https://www.chefkoch.de/rezepte/123/Gulasch.html",
            io.github.thomas1607haja.rezeptplaner.ui.edit.urlFromQr(" https://www.chefkoch.de/rezepte/123/Gulasch.html\n"),
        )
        org.junit.Assert.assertEquals("http://x.de/a", io.github.thomas1607haja.rezeptplaner.ui.edit.urlFromQr("Rezept: http://x.de/a."))
        org.junit.Assert.assertEquals(null, io.github.thomas1607haja.rezeptplaner.ui.edit.urlFromQr("WIFI:S:Zuhause;T:WPA;P:geheim;;"))
        org.junit.Assert.assertEquals(null, io.github.thomas1607haja.rezeptplaner.ui.edit.urlFromQr("rezeptplaner://setup?url=x"))
    }
}
