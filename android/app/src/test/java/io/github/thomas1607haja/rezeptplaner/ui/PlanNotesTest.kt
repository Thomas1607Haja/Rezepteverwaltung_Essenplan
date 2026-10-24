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

class CourseTabsTest {
    @Test
    fun tabsFromCourseGroup() {
        val kuchen = io.github.thomas1607haja.rezeptplaner.api.Tag(31, "Kuchen & Torten", "Gang")
        val haupt = io.github.thomas1607haja.rezeptplaner.api.Tag(28, "Hauptgericht", "Gang")
        val groups = listOf(io.github.thomas1607haja.rezeptplaner.api.TagGroup("Gang", listOf(haupt, kuchen), id = 5, role = "course"))
        val tabs = io.github.thomas1607haja.rezeptplaner.ui.recipes.courseTabs(groups)
        assertEquals(listOf("all", "main", "baking"), tabs.map { it.key })
        assertEquals(setOf(31L), tabs.last().ids)
        // Ohne Gang-Gruppe nur „Alle“.
        assertEquals(1, io.github.thomas1607haja.rezeptplaner.ui.recipes.courseTabs(emptyList()).size)

        val snapshot = io.github.thomas1607haja.rezeptplaner.data.Snapshot(
            serverTime = "x", tagGroups = groups,
            recipes = listOf(
                io.github.thomas1607haja.rezeptplaner.api.Recipe(1, "Apfelkuchen", baseServings = 12, tags = listOf(kuchen)),
                io.github.thomas1607haja.rezeptplaner.api.Recipe(2, "Gulasch", baseServings = 4, tags = listOf(haupt)),
            ),
        )
        val baking = io.github.thomas1607haja.rezeptplaner.ui.recipes.filterRecipes(
            snapshot, io.github.thomas1607haja.rezeptplaner.ui.recipes.RecipeFilter(course = "baking"),
        )
        assertEquals(listOf("Apfelkuchen"), baking.map { it.recipe.title })
    }
}

class YieldUnitTest {
    @Test
    fun amountsAndCompatibility() {
        assertEquals("1 Portion", amountText(1, ""))
        assertEquals("1 Springform 26 cm", amountText(1, "Springform 26 cm"))
        val json = kotlinx.serialization.json.Json { encodeDefaults = true }
        val input = io.github.thomas1607haja.rezeptplaner.api.RecipeInput(
            title = "A", description = "", baseServings = 4, prepMinutes = null, totalMinutes = null,
            sourceUrl = "", sourceName = "", isFavorite = false, notes = "",
            ingredients = emptyList(), steps = emptyList(), tags = emptyList(),
        )
        // Ohne Einheit nicht mitschicken (ältere Server lehnen unbekannte Felder ab).
        org.junit.Assert.assertFalse("yield_unit" in json.encodeToString(input))
        org.junit.Assert.assertTrue("\"yield_unit\":\"Kuchen\"" in json.encodeToString(input.copy(yieldUnit = "Kuchen")))
    }
}
