package io.github.thomas1607haja.rezeptplaner.ui

import io.github.thomas1607haja.rezeptplaner.api.EntryChanges
import io.github.thomas1607haja.rezeptplaner.api.HouseholdSettings
import io.github.thomas1607haja.rezeptplaner.api.Ingredient
import io.github.thomas1607haja.rezeptplaner.api.NewEntry
import io.github.thomas1607haja.rezeptplaner.api.Recipe
import io.github.thomas1607haja.rezeptplaner.data.Snapshot
import io.github.thomas1607haja.rezeptplaner.ui.plan.excludedLabel
import io.github.thomas1607haja.rezeptplaner.ui.plan.wantsPreview
import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class PlanPreviewTest {
    // Wie ApiClient: Standardwerte werden mitgeschickt – außer bei den neuen Feldern.
    private val json = Json { ignoreUnknownKeys = true; encodeDefaults = true }

    @Test
    fun newFieldsAreOnlySentWhenUsed() {
        // Ein älterer Server lehnt unbekannte Felder ab: ohne Abwahl nichts schicken.
        val plain = json.encodeToString(NewEntry(date = "2026-10-24", recipeId = 1))
        assertFalse("excluded_ingredients" in plain)
        val changes = json.encodeToString(EntryChanges(1, 4, "", "planned"))
        assertFalse("excluded_ingredients" in changes)
        assertFalse("item_key" in json.encodeToString(Ingredient(name = "Salz")))

        assertTrue("\"excluded_ingredients\":[\"tomat\"]" in json.encodeToString(NewEntry(date = "x", recipeId = 1, excludedIngredients = listOf("tomat"))))
        // „Alles vorhanden“: leere Liste wird geschickt (nicht „unverändert“).
        assertTrue("\"excluded_ingredients\":[]" in json.encodeToString(EntryChanges(1, 4, "", "have_ingredients", emptyList())))
    }

    @Test
    fun previewOnlyWithKeysAndSetting() {
        val withKeys = Recipe(1, "A", baseServings = 2, ingredients = listOf(Ingredient(name = "Tomaten", itemKey = "tomat")))
        val oldServer = Recipe(2, "B", baseServings = 2, ingredients = listOf(Ingredient(name = "Tomaten")))
        val on = Snapshot(serverTime = "x")
        val off = Snapshot(serverTime = "x", settings = HouseholdSettings(planPreview = false))
        assertTrue(wantsPreview(on, withKeys))
        assertFalse(wantsPreview(on, oldServer))
        assertFalse(wantsPreview(off, withKeys))
        assertEquals(null, excludedLabel(0))
        assertEquals("2 Zutaten vorhanden", excludedLabel(2))
    }
}
