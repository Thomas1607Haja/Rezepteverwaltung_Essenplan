package io.github.thomas1607haja.rezeptplaner.ui.recipes

import io.github.thomas1607haja.rezeptplaner.api.PlanInfo
import io.github.thomas1607haja.rezeptplaner.api.Recipe
import io.github.thomas1607haja.rezeptplaner.api.isIllustrated
import io.github.thomas1607haja.rezeptplaner.data.Snapshot
import io.github.thomas1607haja.rezeptplaner.ui.fold

/** Sortierungen wie in der Weboberfläche (server/internal/recipes/list.go). */
enum class RecipeSort(val label: String) {
    Name("Name"),
    Newest("Neu hinzugefügt"),
    LastPlanned("Zuletzt eingeplant"),
    MostPlanned("Am häufigsten eingeplant"),
}

data class RecipeFilter(
    val query: String = "",
    val tagIds: Set<Long> = emptySet(),
    val favoritesOnly: Boolean = false,
    /** Nur gezeichnete Bilder – zum Ersetzen durch ein Foto. */
    val illustrationsOnly: Boolean = false,
    val sort: RecipeSort = RecipeSort.Name,
) {
    val isActive: Boolean get() = query.isNotBlank() || tagIds.isNotEmpty() || favoritesOnly || illustrationsOnly
}

/** Ein Rezept in der Liste mit seinen Terminen. */
data class RecipeItem(val recipe: Recipe, val planInfo: PlanInfo?)

/**
 * Wendet den Filter an – dieselben Regeln wie der Server: Jedes Suchwort
 * muss im Titel, in der Beschreibung oder in einer Zutat vorkommen
 * (ohne Rücksicht auf Groß-/Kleinschreibung und Umlaute). Kategorien:
 * innerhalb einer Gruppe genügt eine (Pasta oder Reis), über Gruppen
 * hinweg müssen alle passen (Vegetarisch und Pasta).
 */
fun filterRecipes(snapshot: Snapshot, filter: RecipeFilter): List<RecipeItem> {
    val info = snapshot.planInfo.associateBy { it.recipeId }
    val words = fold(filter.query).split(Regex("\\s+")).filter { it.isNotEmpty() }
    val groupOf = snapshot.tagGroups.flatMap { g -> g.tags.map { it.id to g.name } }.toMap()
    val wantedByGroup = filter.tagIds.groupBy { groupOf[it] ?: "" }.mapValues { it.value.toSet() }

    val items = snapshot.recipes.asSequence()
        .filter { !filter.favoritesOnly || it.isFavorite }
        .filter { !filter.illustrationsOnly || it.isIllustrated }
        .filter { r ->
            if (words.isEmpty()) return@filter true
            val text = fold(r.title + " " + r.description + " " + r.ingredients.joinToString(" ") { it.name })
            words.all { it in text }
        }
        .filter { r ->
            val ids = r.tags.map { it.id }.toSet()
            wantedByGroup.values.all { wanted -> wanted.any { it in ids } }
        }
        .map { RecipeItem(it, info[it.id]) }
        .toList()

    val byName = compareBy<RecipeItem> { fold(it.recipe.title) }.thenBy { it.recipe.id }
    return when (filter.sort) {
        RecipeSort.Name -> items.sortedWith(byName)
        RecipeSort.Newest -> items.sortedWith(compareByDescending<RecipeItem> { it.recipe.createdAt }.thenBy { it.recipe.id })
        // Nie eingeplante ans Ende.
        RecipeSort.LastPlanned -> items.sortedWith(
            compareBy<RecipeItem> { it.planInfo?.lastPlanned == null }
                .thenByDescending { it.planInfo?.lastPlanned }.then(byName),
        )
        RecipeSort.MostPlanned -> items.sortedWith(compareByDescending<RecipeItem> { it.planInfo?.planCount ?: 0 }.then(byName))
    }
}
