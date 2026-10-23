package io.github.thomas1607haja.rezeptplaner.data

import io.github.thomas1607haja.rezeptplaner.api.Deletion
import io.github.thomas1607haja.rezeptplaner.api.MealPlanEntry
import io.github.thomas1607haja.rezeptplaner.api.Period
import io.github.thomas1607haja.rezeptplaner.api.Recipe
import io.github.thomas1607haja.rezeptplaner.api.SyncResponse
import org.junit.Assert.assertEquals
import org.junit.Test

class MergeTest {
    private val url = "https://rezepte.example.org"

    private fun response(
        full: Boolean,
        recipes: List<Recipe> = emptyList(),
        entries: List<MealPlanEntry> = emptyList(),
        deletions: List<Deletion> = emptyList(),
    ) = SyncResponse(
        serverTime = "2026-10-23T10:00:00.000Z", full = full, planFrom = "2026-10-09", today = "2026-10-23",
        shoppingPeriod = Period("2026-10-17", "2026-10-23"),
        recipes = recipes, mealPlanEntries = entries, deletions = deletions,
    )

    private fun recipe(id: Long, title: String) = Recipe(id = id, title = title, baseServings = 4)
    private fun entry(id: Long, date: String) = MealPlanEntry(id = id, date = date, recipeId = 1, servings = 4)

    @Test
    fun fullThenIncremental() {
        val first = Snapshot().merge(
            response(full = true, recipes = listOf(recipe(1, "Lasagne"), recipe(2, "Suppe")), entries = listOf(entry(5, "2026-10-24"))),
            url,
        )
        assertEquals(listOf("Lasagne", "Suppe"), first.recipes.map { it.title })

        // Geändert: 1, neu: 3, gelöscht: 2 und Eintrag 5; Eintrag 6 liegt vor plan_from.
        val second = first.merge(
            response(
                full = false,
                recipes = listOf(recipe(1, "Lasagne al forno"), recipe(3, "Curry")),
                entries = listOf(entry(6, "2026-09-01"), entry(7, "2026-10-25")),
                deletions = listOf(Deletion("recipes", 2), Deletion("meal_plan_entries", 5)),
            ),
            url,
        )
        assertEquals(listOf("Lasagne al forno", "Curry"), second.recipes.map { it.title })
        assertEquals(listOf(7L), second.entries.map { it.id })
        assertEquals("2026-10-23T10:00:00.000Z", second.serverTime)
    }

    @Test
    fun otherServerReplacesEverything() {
        val first = Snapshot().merge(response(full = true, recipes = listOf(recipe(1, "Lasagne"))), url)
        val other = first.merge(response(full = false, recipes = listOf(recipe(9, "Anderes"))), "https://anders.example.org")
        assertEquals(listOf(9L), other.recipes.map { it.id })
    }
}
