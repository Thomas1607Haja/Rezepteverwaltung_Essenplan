package io.github.thomas1607haja.rezeptplaner.data

import android.util.Log
import io.github.thomas1607haja.rezeptplaner.api.ApiClient
import io.github.thomas1607haja.rezeptplaner.api.ApiException
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import kotlinx.coroutines.withContext
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import java.io.File

/** Anlass eines Abgleichs; wird mit Uhrzeit festgehalten (Samsung schränkt Hintergrund-Apps ein). */
enum class SyncTrigger { Start, Pull, Background, Manual }

/** Protokoll der Abgleiche (Zeiten in Millisekunden seit 1970). */
@Serializable
data class SyncLog(
    val lastSuccess: Long? = null,
    val lastSuccessTrigger: SyncTrigger? = null,
    val lastBackgroundSuccess: Long? = null,
    /** Erster erfolgreicher Abgleich – der Akku-Hinweis kommt frühestens 12 h danach. */
    val firstSuccess: Long? = null,
    val lastError: String? = null,
    /** Letzter Versuch scheiterte, weil keine Verbindung bestand. */
    val offline: Boolean = false,
)

/** Zustand für die Oberfläche. */
data class SyncState(val running: Boolean = false, val log: SyncLog = SyncLog())

/**
 * Gleicht den Stand auf dem Handy mit dem Server ab (GET /api/v1/sync) und
 * lädt danach fehlende Bilder. Immer nur ein Abgleich gleichzeitig.
 */
class SyncRepository(
    private val config: ConfigRepository,
    private val api: ApiClient,
    private val store: LocalStore,
    private val images: ImageStore,
    private val logFile: File,
    private val now: () -> Long = System::currentTimeMillis,
) {
    private val mutex = Mutex()
    private val json = Json { ignoreUnknownKeys = true }
    private val _state = MutableStateFlow(SyncState(log = readLog()))
    val state: StateFlow<SyncState> = _state.asStateFlow()

    /**
     * Gleicht ab. Liefert null bei Erfolg, sonst die Fehlermeldung. Ohne
     * eingerichtete Verbindung passiert nichts.
     */
    suspend fun sync(trigger: SyncTrigger): String? = mutex.withLock {
        val cfg = config.config.first() ?: return@withLock null
        store.load()
        _state.update { it.copy(running = true) }
        val started = now()
        try {
            val old = store.snapshot.value
            val since = old.serverTime.takeIf { old.serverUrl == cfg.url }
            val response = api.sync(cfg, since)
            val merged = old.merge(response, cfg.url)
            store.save(merged)
            val time = now()
            updateLog {
                it.copy(
                    lastSuccess = time,
                    lastSuccessTrigger = trigger,
                    lastBackgroundSuccess = if (trigger == SyncTrigger.Background) time else it.lastBackgroundSuccess,
                    firstSuccess = it.firstSuccess ?: time,
                    lastError = null,
                    offline = false,
                )
            }
            val dataDone = now()
            val loaded = fetchImages(cfg, merged)
            // Messwerte für docs/PLANUNG.md (Messung nach Phase 7) und zur Fehlersuche.
            Log.i(TAG, "Abgleich ($trigger): ${response.recipes.size} Rezepte, ${response.mealPlanEntries.size} Planeinträge geändert; " +
                "Daten ${dataDone - started} ms, $loaded Bilder in ${now() - dataDone} ms")
            null
        } catch (e: ApiException) {
            Log.w(TAG, "Abgleich fehlgeschlagen ($trigger)", e)
            updateLog { it.copy(lastError = e.message, offline = e.offline) }
            e.message
        } finally {
            _state.update { it.copy(running = false) }
        }
    }

    /** Lädt ein großes Bild beim ersten Ansehen nach; true, wenn es danach da ist. */
    suspend fun ensureImage(name: String): Boolean {
        if (name.isEmpty() || images.has(name)) return images.has(name)
        val cfg = config.config.first() ?: return false
        return try {
            images.save(name, api.image(cfg, name))
            true
        } catch (e: ApiException) {
            false
        }
    }

    /** Verbindung getrennt: lokalen Stand und Bilder löschen. */
    suspend fun reset() = mutex.withLock {
        store.clear()
        images.clear()
        updateLog { SyncLog() }
    }

    /**
     * Vorschaubilder aller Rezepte, große Bilder der Rezepte im aktuellen
     * Einkaufszeitraum. Bilder, die kein Rezept mehr nutzt, fliegen raus.
     * Bricht still ab, wenn die Verbindung wegfällt – beim nächsten
     * Abgleich geht es weiter.
     */
    private suspend fun fetchImages(cfg: ServerConfig, snapshot: Snapshot): Int {
        // Eingekaufte Woche und die folgende (nächster Einkauf).
        val from = snapshot.shoppingPeriod?.from
        val to = snapshot.shoppingPeriod?.to?.let { java.time.LocalDate.parse(it).plusDays(7).toString() }
        val planned = snapshot.entries
            .filter { from != null && to != null && it.date >= from && it.date <= to }
            .mapNotNull { it.recipeId }.toSet()
        val wanted = snapshot.recipes.flatMap { r ->
            listOfNotNull(r.thumbnail.ifEmpty { null }, r.image.takeIf { it.isNotEmpty() && r.id in planned })
        }
        var loaded = 0
        for (name in wanted) {
            if (images.has(name)) continue
            try {
                images.save(name, api.image(cfg, name))
                loaded++
            } catch (e: ApiException) {
                if (e.offline) break
                Log.w(TAG, "Bild $name nicht geladen", e)
            }
        }
        // Große Bilder anderer Rezepte, die schon einmal angesehen wurden, bleiben.
        val keep = snapshot.recipes.flatMap { listOf(it.thumbnail, it.image) }.filter { it.isNotEmpty() }.toSet()
        images.prune(keep)
        return loaded
    }

    private fun readLog(): SyncLog =
        runCatching { json.decodeFromString<SyncLog>(logFile.readText()) }.getOrDefault(SyncLog())

    private suspend fun updateLog(change: (SyncLog) -> SyncLog) {
        val log = change(_state.value.log)
        _state.update { it.copy(log = log) }
        withContext(Dispatchers.IO) { runCatching { logFile.writeText(json.encodeToString(log)) } }
    }

    private companion object {
        const val TAG = "Rezeptplaner"
    }
}
