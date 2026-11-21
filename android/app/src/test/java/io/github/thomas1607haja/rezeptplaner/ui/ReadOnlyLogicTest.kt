package io.github.thomas1607haja.rezeptplaner.ui

import io.github.thomas1607haja.rezeptplaner.api.HouseholdSettings
import io.github.thomas1607haja.rezeptplaner.api.Ingredient
import io.github.thomas1607haja.rezeptplaner.api.MealPlanEntry
import io.github.thomas1607haja.rezeptplaner.api.PlanInfo
import io.github.thomas1607haja.rezeptplaner.api.Recipe
import io.github.thomas1607haja.rezeptplaner.api.Tag
import io.github.thomas1607haja.rezeptplaner.api.TagGroup
import io.github.thomas1607haja.rezeptplaner.data.Snapshot
import io.github.thomas1607haja.rezeptplaner.data.SyncLog
import io.github.thomas1607haja.rezeptplaner.ui.plan.planDays
import io.github.thomas1607haja.rezeptplaner.ui.recipes.RecipeFilter
import io.github.thomas1607haja.rezeptplaner.ui.recipes.RecipeSort
import io.github.thomas1607haja.rezeptplaner.ui.recipes.filterRecipes
import io.github.thomas1607haja.rezeptplaner.ui.settings.backgroundSyncStalled
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import java.time.LocalDate

class ReadOnlyLogicTest {
    private val veg = Tag(1, "Vegetarisch", "Hauptzutat")
    private val rind = Tag(3, "Rind", "Hauptzutat")
    private val pasta = Tag(11, "Pasta", "Art")
    private val reis = Tag(12, "Reis", "Art")

    private val snapshot = Snapshot(
        serverTime = "x", today = "2026-10-23", planFrom = "2026-10-20",
        tagGroups = listOf(TagGroup("Hauptzutat", listOf(veg, rind)), TagGroup("Art", listOf(pasta, reis))),
        recipes = listOf(
            Recipe(1, "Käsespätzle", baseServings = 4, tags = listOf(veg), ingredients = listOf(Ingredient(name = "Spätzle")), createdAt = "2026-01"),
            Recipe(2, "Bolognese", baseServings = 4, tags = listOf(rind, pasta), isFavorite = true, createdAt = "2026-03"),
            Recipe(3, "Gemüsereis", baseServings = 4, tags = listOf(veg, reis), createdAt = "2026-02", image = "g.jpg", imageKind = "illustration"),
        ),
        planInfo = listOf(PlanInfo(2, lastPlanned = "2026-10-01", planCount = 5), PlanInfo(3, lastPlanned = "2026-10-15", planCount = 1)),
        entries = listOf(MealPlanEntry(9, "2026-11-20", recipeId = 2, servings = 4)),
    )

    private fun titles(filter: RecipeFilter) = filterRecipes(snapshot, filter).map { it.recipe.title }

    @Test
    fun searchIgnoresCaseAndUmlauts() {
        assertEquals(listOf("Käsespätzle"), titles(RecipeFilter(query = "SPATZLE")))
        assertEquals(listOf("Käsespätzle"), titles(RecipeFilter(query = "kase spätzle")))
    }

    @Test
    fun tagsOrWithinGroupAndAcrossGroups() {
        assertEquals(listOf("Bolognese", "Gemüsereis"), titles(RecipeFilter(tagIds = setOf(11, 12))))
        assertEquals(listOf("Gemüsereis"), titles(RecipeFilter(tagIds = setOf(1, 11, 12))))
    }

    @Test
    fun favoritesAndSorting() {
        assertEquals(listOf("Bolognese"), titles(RecipeFilter(favoritesOnly = true)))
        assertEquals(listOf("Gemüsereis"), titles(RecipeFilter(illustrationsOnly = true)))
        assertEquals(listOf("Bolognese", "Gemüsereis", "Käsespätzle"), titles(RecipeFilter()))
        assertEquals(listOf("Bolognese", "Gemüsereis", "Käsespätzle"), titles(RecipeFilter(sort = RecipeSort.Newest)))
        assertEquals(listOf("Gemüsereis", "Bolognese", "Käsespätzle"), titles(RecipeFilter(sort = RecipeSort.LastPlanned)))
        assertEquals(listOf("Bolognese", "Gemüsereis", "Käsespätzle"), titles(RecipeFilter(sort = RecipeSort.MostPlanned)))
    }

    @Test
    fun planDaysCoverGapsUntilLastEntry() {
        val days = planDays(snapshot, LocalDate.parse("2026-10-23"))
        assertEquals("2026-10-20", days.first().date)
        assertEquals("2026-11-20", days.last().date)
        assertEquals(1, days.single { it.date == "2026-11-20" }.entries.size)
    }

    @Test
    fun planDaysFollowPlanWeeksAndKeepLaterEntries() {
        val today = LocalDate.parse("2026-10-23")
        // 8 Wochen: bis heute + 55 Tage, über den letzten Eintrag (20.11.) hinaus.
        val wide = planDays(snapshot.copy(settings = HouseholdSettings(planWeeks = 8)), today)
        assertEquals("2026-12-17", wide.last().date)
        // 1 Woche: Der Eintrag am 20.11. bleibt sichtbar, nichts verschwindet.
        val narrow = planDays(snapshot.copy(settings = HouseholdSettings(planWeeks = 1)), today)
        assertEquals("2026-11-20", narrow.last().date)
        // Ohne späte Einträge endet der Plan genau am Ende des Zeitraums.
        val empty = planDays(snapshot.copy(entries = emptyList(), settings = HouseholdSettings(planWeeks = 1)), today)
        assertEquals("2026-10-29", empty.last().date)
        // Werte außerhalb 1–12 werden begrenzt.
        assertEquals(84, HouseholdSettings(planWeeks = 99).planDays())
        assertEquals(7, HouseholdSettings(planWeeks = 0).planDays())
    }

    @Test
    fun batteryHintOnlyWhenBackgroundSyncStalls() {
        val h = 60 * 60 * 1000L
        val now = 100 * h
        assertFalse(backgroundSyncStalled(SyncLog(), now))
        assertFalse(backgroundSyncStalled(SyncLog(firstSuccess = now - 2 * h), now))
        assertTrue(backgroundSyncStalled(SyncLog(firstSuccess = now - 20 * h), now))
        assertFalse(backgroundSyncStalled(SyncLog(firstSuccess = now - 20 * h, lastBackgroundSuccess = now - 3 * h), now))
        assertTrue(backgroundSyncStalled(SyncLog(firstSuccess = now - 40 * h, lastBackgroundSuccess = now - 13 * h), now))
    }
}
