package io.github.thomas1607haja.rezeptplaner.ui.edit

import io.github.thomas1607haja.rezeptplaner.api.Ingredient
import io.github.thomas1607haja.rezeptplaner.api.Recipe
import io.github.thomas1607haja.rezeptplaner.api.RecipeInput
import io.github.thomas1607haja.rezeptplaner.api.Step
import io.github.thomas1607haja.rezeptplaner.api.TagRef
import java.text.DecimalFormat
import java.text.DecimalFormatSymbols
import java.util.Locale

// Umwandlung zwischen Rezept und Formular, wie web/src/lib/recipeForm.js:
// Im Formular sind Gruppen eigene Zeilen (Überschriften); die API kennt nur
// group_name an jeder Zutat.

private var nextKey = 1L
private fun key() = nextKey++

/** Zutat oder (heading = true) Gruppenüberschrift. Alle Felder als Text wie eingegeben. */
data class IngredientRow(
    val key: Long = key(),
    val heading: Boolean = false,
    val quantity: String = "",
    val unit: String = "",
    val name: String = "",
    val note: String = "",
    val shoppingRelevant: Boolean = true,
)

data class StepRow(val key: Long = key(), val text: String = "")

data class RecipeForm(
    val title: String = "",
    val description: String = "",
    val baseServings: String = "4",
    /** Ergebnis-Einheit, leer = Portionen. */
    val yieldUnit: String = "",
    val prepMinutes: String = "",
    val totalMinutes: String = "",
    val sourceUrl: String = "",
    val sourceName: String = "",
    val notes: String = "",
    val isFavorite: Boolean = false,
    val tagIds: Set<Long> = emptySet(),
    val items: List<IngredientRow> = listOf(IngredientRow()),
    val steps: List<StepRow> = listOf(StepRow()),
)

fun Recipe.toForm(): RecipeForm {
    val items = mutableListOf<IngredientRow>()
    var group = ""
    for (ing in ingredients) {
        if (ing.groupName != group) {
            group = ing.groupName
            // Zurück zu „ohne Gruppe“ ist mitten in der Liste nicht darstellbar.
            if (group.isNotEmpty()) items += IngredientRow(heading = true, name = group)
        }
        items += IngredientRow(
            quantity = quantityText(ing.quantity), unit = ing.unit, name = ing.name, note = ing.note,
            shoppingRelevant = ing.shoppingRelevant,
        )
    }
    return RecipeForm(
        title = title, description = description, baseServings = baseServings.toString(), yieldUnit = yieldUnit,
        prepMinutes = prepMinutes?.toString().orEmpty(), totalMinutes = totalMinutes?.toString().orEmpty(),
        sourceUrl = sourceUrl, sourceName = sourceName, notes = notes, isFavorite = isFavorite,
        tagIds = tags.map { it.id }.toSet(),
        items = items.ifEmpty { listOf(IngredientRow()) },
        steps = steps.map { StepRow(text = it.text) }.ifEmpty { listOf(StepRow()) },
    )
}

/** Menge fürs Eingabefeld: 1.5 -> "1,5", 250.0 -> "250". */
fun quantityText(value: Double?): String =
    value?.let { DecimalFormat("0.###", DecimalFormatSymbols(Locale.GERMANY)).format(it) }.orEmpty()

private val unicodeFractions = mapOf('¼' to 0.25, '½' to 0.5, '¾' to 0.75, '⅓' to 1.0 / 3, '⅔' to 2.0 / 3)

/**
 * Liest eine Menge: "1,5", "1.5", "1/2", "1 1/2", "½", "1½". Leer = null
 * (ohne Menge); nicht lesbar = NaN.
 */
fun parseQuantity(text: String): Double? {
    var s = text.trim()
    if (s.isEmpty()) return null
    var value = 0.0
    unicodeFractions[s.last()]?.let {
        value += it
        s = s.dropLast(1).trim()
        if (s.isEmpty()) return value
    }
    Regex("""^(?:(\d+)\s+)?(\d+)\s*/\s*(\d+)$""").matchEntire(s)?.let { m ->
        val (whole, num, den) = m.destructured
        if (den.toInt() == 0) return Double.NaN
        return value + (whole.toIntOrNull() ?: 0) + num.toDouble() / den.toDouble()
    }
    if (!Regex("""^\d+([.,]\d+)?$""").matches(s)) return Double.NaN
    return value + s.replace(',', '.').toDouble()
}

/** Prüft das Formular. Ergebnis: Eingabe für die API oder Fehler je Feld ("title", "item:<key>", …). */
fun RecipeForm.toInput(): Pair<RecipeInput?, Map<String, String>> {
    val errors = mutableMapOf<String, String>()
    if (title.isBlank()) errors["title"] = "Bitte einen Titel eingeben."
    val servings = baseServings.trim().toIntOrNull()
    if (servings == null || servings < 1) errors["baseServings"] = "Mindestens 1."
    fun minutes(field: String, text: String): Int? {
        if (text.isBlank()) return null
        val n = text.trim().toIntOrNull()
        if (n == null || n < 0) errors[field] = "Minuten als ganze Zahl."
        return n
    }
    val prep = minutes("prepMinutes", prepMinutes)
    val total = minutes("totalMinutes", totalMinutes)
    val url = sourceUrl.trim()
    if (url.isNotEmpty() && !(url.startsWith("http://") || url.startsWith("https://"))) {
        errors["sourceUrl"] = "Muss mit http:// oder https:// beginnen."
    }

    val ingredients = mutableListOf<Ingredient>()
    var group = ""
    for (row in items) {
        if (row.heading) {
            group = row.name.trim()
            continue
        }
        if (listOf(row.quantity, row.unit, row.name, row.note).all { it.isBlank() }) continue
        val q = parseQuantity(row.quantity)
        when {
            q != null && q.isNaN() -> errors["item:${row.key}"] = "Menge nicht lesbar, z. B. 1,5 oder 1/2."
            row.name.isBlank() -> errors["item:${row.key}"] = "Bitte einen Namen eingeben."
        }
        ingredients += Ingredient(
            groupName = group, quantity = q?.takeUnless { it.isNaN() }, unit = row.unit.trim(), name = row.name.trim(),
            note = row.note.trim(), shoppingRelevant = row.shoppingRelevant,
        )
    }
    if (errors.isNotEmpty()) return null to errors
    return RecipeInput(
        title = title.trim(), description = description.trim(), baseServings = servings!!,
        yieldUnit = yieldUnit.trim().replace(Regex("\\s+"), " ").take(40),
        prepMinutes = prep, totalMinutes = total, sourceUrl = url, sourceName = sourceName.trim(),
        isFavorite = isFavorite, notes = notes.trim(), ingredients = ingredients,
        steps = steps.map { it.text.trim() }.filter { it.isNotEmpty() }.map { Step(it) },
        tags = tagIds.sorted().map { TagRef(it) },
    ) to emptyMap()
}

/** Verschiebt das Element an [index] um [delta] (−1 hoch, +1 runter). */
fun <T> List<T>.move(index: Int, delta: Int): List<T> {
    val target = index + delta
    if (index !in indices || target !in indices) return this
    return toMutableList().apply { add(target, removeAt(index)) }
}
