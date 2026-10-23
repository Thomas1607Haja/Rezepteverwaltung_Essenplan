package io.github.thomas1607haja.rezeptplaner.ui

import io.github.thomas1607haja.rezeptplaner.api.Ingredient
import io.github.thomas1607haja.rezeptplaner.api.Recipe
import io.github.thomas1607haja.rezeptplaner.api.Step
import io.github.thomas1607haja.rezeptplaner.api.Tag
import io.github.thomas1607haja.rezeptplaner.ui.edit.IngredientRow
import io.github.thomas1607haja.rezeptplaner.ui.edit.RecipeForm
import io.github.thomas1607haja.rezeptplaner.ui.edit.move
import io.github.thomas1607haja.rezeptplaner.ui.edit.parseQuantity
import io.github.thomas1607haja.rezeptplaner.ui.edit.toForm
import io.github.thomas1607haja.rezeptplaner.ui.edit.toInput
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class RecipeFormTest {
    @Test
    fun quantities() {
        assertEquals(1.5, parseQuantity("1,5")!!, 1e-9)
        assertEquals(1.5, parseQuantity("1 1/2")!!, 1e-9)
        assertEquals(0.5, parseQuantity("½")!!, 1e-9)
        assertEquals(1.5, parseQuantity("1½")!!, 1e-9)
        assertNull(parseQuantity("  "))
        assertTrue(parseQuantity("etwas")!!.isNaN())
        assertTrue(parseQuantity("1/0")!!.isNaN())
    }

    @Test
    fun roundTripWithGroups() {
        val recipe = Recipe(
            id = 1, title = "Lasagne", baseServings = 4, prepMinutes = 20,
            ingredients = listOf(
                Ingredient(quantity = 250.0, unit = "g", name = "Nudelplatten"),
                Ingredient(groupName = "Für die Soße", quantity = 1.5, unit = "EL", name = "Öl", shoppingRelevant = false),
                Ingredient(groupName = "Für die Soße", name = "Salz"),
            ),
            steps = listOf(Step("Schichten"), Step("Backen")),
            tags = listOf(Tag(11, "Pasta")),
        )
        val form = recipe.toForm()
        assertEquals(listOf(false, true, false, false), form.items.map { it.heading })
        assertEquals("1,5", form.items[2].quantity)

        val (input, errors) = form.toInput()
        assertTrue(errors.isEmpty())
        input!!
        assertEquals(recipe.ingredients, input.ingredients)
        assertEquals(listOf("Schichten", "Backen"), input.steps.map { it.text })
        assertEquals(listOf(11L), input.tags.map { it.id })
        assertEquals(20, input.prepMinutes)
    }

    @Test
    fun validation() {
        val bad = RecipeForm(
            title = " ", baseServings = "0", sourceUrl = "javascript:alert(1)",
            items = listOf(IngredientRow(key = 99, quantity = "viel", name = "Mehl")),
        )
        val (input, errors) = bad.toInput()
        assertNull(input)
        assertEquals(setOf("title", "baseServings", "sourceUrl", "item:99"), errors.keys)
        // Leere Zeilen werden einfach weggelassen.
        val (ok, _) = RecipeForm(title = "Suppe", items = listOf(IngredientRow())).toInput()
        assertEquals(0, ok!!.ingredients.size)
    }

    @Test
    fun moving() {
        assertEquals(listOf("b", "a", "c"), listOf("a", "b", "c").move(1, -1))
        assertEquals(listOf("a", "b", "c"), listOf("a", "b", "c").move(0, -1))
    }
}
