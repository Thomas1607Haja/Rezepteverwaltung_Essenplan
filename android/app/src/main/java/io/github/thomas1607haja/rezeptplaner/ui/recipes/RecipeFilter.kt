package io.github.thomas1607haja.rezeptplaner.ui.recipes

import androidx.annotation.StringRes
import io.github.thomas1607haja.rezeptplaner.R
import io.github.thomas1607haja.rezeptplaner.api.PlanInfo
import io.github.thomas1607haja.rezeptplaner.api.Recipe
import io.github.thomas1607haja.rezeptplaner.api.isIllustrated
import io.github.thomas1607haja.rezeptplaner.data.Snapshot
import io.github.thomas1607haja.rezeptplaner.ui.fold

/** Sortierungen wie in der Weboberfläche (server/internal/recipes/list.go). */
enum class RecipeSort(@StringRes val label: Int) {
    Name(R.string.recipe_sort_name),
    Newest(R.string.recipe_sort_newest),
    LastPlanned(R.string.recipe_sort_last_planned),
    MostPlanned(R.string.recipe_sort_most_planned),
}

data class RecipeFilter(
    val query: String = "",
    val tagIds: Set<Long> = emptySet(),
    val favoritesOnly: Boolean = false,
    /** Nur gezeichnete Bilder – zum Ersetzen durch ein Foto. */
    val illustrationsOnly: Boolean = false,
    /** Nur Rezepte ohne jeden Planeintrag, auch ohne zukünftigen (Link aus der Statistik). */
    val neverPlanned: Boolean = false,
    /** Schnellfilter-Reiter nach Gang ([courseTabs]); "all" = alle. */
    val course: String = "all",
    val sort: RecipeSort = RecipeSort.Name,
) {
    val isActive: Boolean get() = query.isNotBlank() || tagIds.isNotEmpty() || favoritesOnly || illustrationsOnly || neverPlanned || course != "all"

    /** Suche und Filter zurücksetzen, die Sortierung bleibt (wie resetFilter im Web). */
    fun reset() = RecipeFilter(sort = sort)
}

/** Ein Schnellfilter-Reiter mit den IDs seiner Gänge; [label] aus courses_* (strings_shared). */
data class CourseTab(val key: String, @StringRes val label: Int, val ids: Set<Long>)

/**
 * Reiter wie web/src/lib/courseTabs.js: Gänge der Gang-Gruppe nach ihrem
 * festen Schlüssel (Tag.courseKey), nie nach dem Namen – Namen sind Daten.
 */
private val courseTabDefs = listOf(
    Triple("main", R.string.courses_main, setOf("main")),
    Triple("starters", R.string.courses_starters, setOf("starter", "soup", "salad")),
    Triple("desserts", R.string.courses_desserts, setOf("dessert")),
    Triple("baking", R.string.courses_baking, setOf("cake", "pastry", "bread")),
)

/** Reiter mit IDs, vorn „Alle“; Reiter ohne passende Kategorie fallen weg. */
fun courseTabs(groups: List<io.github.thomas1607haja.rezeptplaner.api.TagGroup>): List<CourseTab> {
    val all = CourseTab("all", R.string.courses_all, emptySet())
    val course = groups.firstOrNull { it.role == "course" } ?: return listOf(all)
    return listOf(all) + courseTabDefs.mapNotNull { (key, label, keys) ->
        val ids = course.tags.filter { it.courseKey in keys }.map { it.id }.toSet()
        if (ids.isEmpty()) null else CourseTab(key, label, ids)
    }
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
    val courseIds = courseTabs(snapshot.tagGroups).firstOrNull { it.key == filter.course }?.ids.orEmpty()

    val items = snapshot.recipes.asSequence()
        .filter { !filter.favoritesOnly || it.isFavorite }
        .filter { !filter.illustrationsOnly || it.isIllustrated }
        // plan_info gibt es für jedes Rezept mit mindestens einem Eintrag.
        .filter { !filter.neverPlanned || it.id !in info }
        .filter { r -> courseIds.isEmpty() || r.tags.any { it.id in courseIds } }
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
