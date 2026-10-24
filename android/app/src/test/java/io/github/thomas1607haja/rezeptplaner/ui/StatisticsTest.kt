package io.github.thomas1607haja.rezeptplaner.ui

import io.github.thomas1607haja.rezeptplaner.api.PlanInfo
import io.github.thomas1607haja.rezeptplaner.api.Recipe
import io.github.thomas1607haja.rezeptplaner.api.Tag
import io.github.thomas1607haja.rezeptplaner.api.TagGroup
import io.github.thomas1607haja.rezeptplaner.data.Snapshot
import io.github.thomas1607haja.rezeptplaner.data.localStatistics
import io.github.thomas1607haja.rezeptplaner.ui.recipes.RecipeFilter
import io.github.thomas1607haja.rezeptplaner.ui.recipes.filterRecipes
import org.junit.Assert.assertEquals
import org.junit.Test

/** Dieselben Regeln wie server/internal/recipes/stats_test.go für „gesamt“. */
class StatisticsTest {
    private val pasta = Tag(11, "Pasta", "Art")
    private val snapshot = Snapshot(
        serverTime = "x", today = "2026-10-24", planFrom = "2026-10-10",
        tagGroups = listOf(TagGroup("Art", listOf(pasta), id = 3)),
        recipes = listOf(
            Recipe(1, "Spaghetti", baseServings = 4, isFavorite = true, tags = listOf(pasta), createdAt = "2026-01-01"),
            Recipe(2, "Risotto", baseServings = 4, isFavorite = true, createdAt = "2026-02-01"),
            Recipe(3, "Steak", baseServings = 2, imageKind = "illustration", createdAt = "2026-03-01"),
            Recipe(4, "Neu", baseServings = 2, isFavorite = true, createdAt = "2026-10-01"),
            Recipe(5, "Nur geplant", baseServings = 2, createdAt = "2026-04-01"),
        ),
        planInfo = listOf(
            PlanInfo(1, nextPlanned = "2026-10-24", lastPlanned = "2026-10-01", planCount = 3), // heute zählt
            PlanInfo(2, lastPlanned = "2026-03-01", planCount = 1),
            PlanInfo(3, nextPlanned = "2026-10-30", lastPlanned = "2026-10-20", planCount = 1),
            PlanInfo(5, nextPlanned = "2026-11-02", planCount = 0), // nur Zukunft
        ),
    )

    @Test
    fun localStatisticsMatchesServerRules() {
        val st = localStatistics(snapshot)
        assertEquals(5, st.recipes)
        assertEquals(4, st.planned) // auch „Nur geplant“
        assertEquals(1, st.neverPlanned)
        assertEquals(1, st.illustrated)
        assertEquals(3, st.favorites)
        assertEquals(5, st.entries)
        assertEquals(listOf("Spaghetti" to 3, "Risotto" to 1, "Steak" to 1), st.top.map { it.title to it.count })
        assertEquals("2026-10-24", st.top[0].lastPlanned)
        assertEquals(listOf("Neu", "Risotto"), st.favoritesWaiting.map { it.title })
        assertEquals("Neu", st.recentlyAdded[0].title)
        assertEquals(1, st.categories.single().tags.single().count)
        assertEquals(4, st.withoutCategory)
    }

    @Test
    fun neverPlannedFilter() {
        assertEquals(listOf("Neu"), filterRecipes(snapshot, RecipeFilter(neverPlanned = true)).map { it.recipe.title })
    }
}
