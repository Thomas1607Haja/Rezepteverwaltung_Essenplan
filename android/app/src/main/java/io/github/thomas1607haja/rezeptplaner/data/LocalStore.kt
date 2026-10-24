package io.github.thomas1607haja.rezeptplaner.data

import io.github.thomas1607haja.rezeptplaner.api.HouseholdSettings
import io.github.thomas1607haja.rezeptplaner.api.MealPlanEntry
import io.github.thomas1607haja.rezeptplaner.api.Period
import io.github.thomas1607haja.rezeptplaner.api.PlanInfo
import io.github.thomas1607haja.rezeptplaner.api.Recipe
import io.github.thomas1607haja.rezeptplaner.api.SyncResponse
import io.github.thomas1607haja.rezeptplaner.api.TagGroup
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.withContext
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import io.github.thomas1607haja.rezeptplaner.ui.slotOrder
import java.io.File

/**
 * Stand des Rezeptplaners auf dem Handy – alles, was die App ohne
 * Verbindung zeigt. Siehe docs/PLANUNG.md, Abschnitt 7.
 *
 * Gespeichert wird er als eine JSON-Datei (bei einigen hundert Rezepten
 * 1–2 MB) und beim Start ganz in den Speicher geladen. Suchen und Filtern
 * laufen im Speicher; eine Datenbank braucht es dafür nicht, weil die App
 * nur liest – geändert wird online direkt auf dem Server.
 */
@Serializable
data class Snapshot(
    /** Adresse des Servers, von dem der Stand stammt (neuer Server = neu laden). */
    val serverUrl: String = "",
    /** server_time des letzten Abgleichs; null = noch nie abgeglichen. */
    val serverTime: String? = null,
    val today: String = "",
    val planFrom: String = "",
    val shoppingPeriod: Period? = null,
    val recipes: List<Recipe> = emptyList(),
    val planInfo: List<PlanInfo> = emptyList(),
    val entries: List<MealPlanEntry> = emptyList(),
    val tagGroups: List<TagGroup> = emptyList(),
    val settings: HouseholdSettings = HouseholdSettings(),
) {
    val isEmpty: Boolean get() = serverTime == null
}

/**
 * Führt eine Antwort von GET /sync in den bisherigen Stand ein. Rezepte
 * und Planeinträge kommen nur, wenn sie sich geändert haben (Löschungen
 * stehen in deletions); die kleinen Listen kommen immer vollständig.
 */
fun Snapshot.merge(response: SyncResponse, serverUrl: String): Snapshot {
    val full = response.full || this.serverUrl != serverUrl
    val recipes = if (full) linkedMapOf() else recipes.associateByTo(linkedMapOf()) { it.id }
    val entries = if (full) linkedMapOf() else entries.associateByTo(linkedMapOf()) { it.id }
    for (r in response.recipes) recipes[r.id] = r
    for (e in response.mealPlanEntries) entries[e.id] = e
    for (d in response.deletions) {
        when (d.table) {
            "recipes" -> recipes.remove(d.id)
            "meal_plan_entries" -> entries.remove(d.id)
        }
    }
    return Snapshot(
        serverUrl = serverUrl,
        serverTime = response.serverTime,
        today = response.today,
        planFrom = response.planFrom,
        shoppingPeriod = response.shoppingPeriod,
        recipes = recipes.values.sortedBy { it.id },
        planInfo = response.planInfo,
        // Ältere Tage als plan_from braucht die App nicht mehr.
        entries = entries.values.filter { it.date >= response.planFrom }
            .sortedWith(compareBy({ it.date }, { slotOrder(it.slot) }, { it.position }, { it.id })),
        tagGroups = response.tagGroups,
        settings = response.settings,
    )
}

/** Lädt und speichert den [Snapshot] in einer Datei. */
class LocalStore(private val file: File) {
    private val json = Json { ignoreUnknownKeys = true }
    private val _snapshot = MutableStateFlow(Snapshot())
    val snapshot: StateFlow<Snapshot> = _snapshot.asStateFlow()
    private var loaded = false

    /** Liest die Datei einmal ein (beim ersten Aufruf). */
    suspend fun load() = withContext(Dispatchers.IO) {
        if (loaded) return@withContext
        loaded = true
        if (file.exists()) {
            runCatching { json.decodeFromString<Snapshot>(file.readText()) }
                .onSuccess { _snapshot.value = it }
        }
    }

    /** Speichert den neuen Stand: erst in eine Hilfsdatei, dann umbenennen. */
    suspend fun save(snapshot: Snapshot) = withContext(Dispatchers.IO) {
        val tmp = File(file.parentFile, file.name + ".tmp")
        tmp.writeText(json.encodeToString(snapshot))
        if (!tmp.renameTo(file)) {
            file.delete()
            tmp.renameTo(file)
        }
        _snapshot.value = snapshot
    }

    suspend fun clear() = withContext(Dispatchers.IO) {
        file.delete()
        _snapshot.value = Snapshot()
    }
}
