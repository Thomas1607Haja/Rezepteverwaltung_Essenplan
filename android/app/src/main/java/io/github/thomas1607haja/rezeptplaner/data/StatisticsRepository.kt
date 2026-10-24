package io.github.thomas1607haja.rezeptplaner.data

import io.github.thomas1607haja.rezeptplaner.api.ApiClient
import io.github.thomas1607haja.rezeptplaner.api.ApiException
import io.github.thomas1607haja.rezeptplaner.api.Recipe
import io.github.thomas1607haja.rezeptplaner.api.StatGroup
import io.github.thomas1607haja.rezeptplaner.api.StatRecipe
import io.github.thomas1607haja.rezeptplaner.api.StatTag
import io.github.thomas1607haja.rezeptplaner.api.Statistics
import kotlinx.coroutines.flow.first
import java.time.LocalDate

/**
 * Statistik: online vom Server (jeder Zeitraum), ohne Verbindung aus den
 * Daten auf dem Handy – dann nur „gesamt“ und ohne Einträge ohne Rezept,
 * weil die App den Plan nur ab plan_from kennt. Siehe docs/PLANUNG.md, 8.D.
 */
class StatisticsRepository(private val config: ConfigRepository, private val api: ApiClient) {
    /** null = nicht verbunden bzw. offline; dann [local] verwenden. */
    suspend fun load(period: String): Statistics? {
        val cfg = config.config.first() ?: return null
        return try {
            api.statistics(cfg, period)
        } catch (e: ApiException) {
            if (e.offline) null else throw e
        }
    }
}

/** Ab so vielen Tagen ohne Termin gilt ein Favorit als „lange nicht dran“ (wie im Server). */
private const val WAITING_AFTER_DAYS = 56L

/**
 * Dieselbe Statistik wie der Server für „gesamt“, berechnet aus dem
 * Abgleich: plan_info kommt für alle Rezepte mit (plan_count zählt bis
 * heute), dazu Rezepte und Kategorien.
 */
fun localStatistics(snapshot: Snapshot): Statistics {
    val today = snapshot.today.ifEmpty { LocalDate.now().toString() }
    val info = snapshot.planInfo.associateBy { it.recipeId }
    val recipes = snapshot.recipes
    // Letzter Termin bis heute: last_planned liegt vor heute, ein Termin
    // heute steht in next_planned.
    fun lastUntilToday(id: Long): String? = info[id]?.let { if (it.nextPlanned == today) today else it.lastPlanned }
    fun stat(id: Long, title: String, thumbnail: String, count: Int = 0, createdAt: String = "") =
        StatRecipe(id, title, thumbnail, count, lastUntilToday(id), createdAt)

    val waitingBefore = LocalDate.parse(today).minusDays(WAITING_AFTER_DAYS).toString()
    val tagCounts = recipes.flatMap { r -> r.tags.map { it.id } }.groupingBy { it }.eachCount()
    return Statistics(
        period = "all",
        today = today,
        recipes = recipes.size,
        planned = recipes.count { it.id in info },
        neverPlanned = recipes.count { it.id !in info },
        illustrated = recipes.count { it.imageKind == "illustration" },
        favorites = recipes.count { it.isFavorite },
        entries = recipes.sumOf { info[it.id]?.planCount ?: 0 },
        top = recipes.filter { (info[it.id]?.planCount ?: 0) > 0 }
            .sortedWith(compareByDescending<Recipe> { info[it.id]!!.planCount }.thenBy { it.title.lowercase() })
            .take(20)
            .map { stat(it.id, it.title, it.thumbnail, info[it.id]!!.planCount) },
        favoritesWaiting = recipes.filter { r ->
            val next = info[r.id]?.nextPlanned
            val last = lastUntilToday(r.id)
            r.isFavorite && (next == null || next <= today) && (last == null || last < waitingBefore)
        }
            .sortedWith(compareBy({ lastUntilToday(it.id) ?: "" }, { it.title.lowercase() }))
            .take(10)
            .map { stat(it.id, it.title, it.thumbnail) },
        recentlyAdded = recipes.sortedWith(compareByDescending<Recipe> { it.createdAt }.thenByDescending { it.id })
            .take(5)
            .map { stat(it.id, it.title, it.thumbnail, createdAt = it.createdAt) },
        categories = snapshot.tagGroups.map { g -> StatGroup(g.name, g.tags.map { StatTag(it.id, it.name, tagCounts[it.id] ?: 0) }) },
        withoutCategory = recipes.count { it.tags.isEmpty() },
    )
}
