package io.github.thomas1607haja.rezeptplaner.data

import android.content.ContentResolver
import android.net.Uri
import android.provider.OpenableColumns
import io.github.thomas1607haja.rezeptplaner.api.ApiClient
import io.github.thomas1607haja.rezeptplaner.api.ApiException
import io.github.thomas1607haja.rezeptplaner.api.CategoryChoice
import io.github.thomas1607haja.rezeptplaner.api.ImportPreview
import io.github.thomas1607haja.rezeptplaner.api.ImportResult
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.withContext

/** Eine eingelesene Importdatei: Anzeigename und Inhalt. */
class ImportFile(val name: String, val data: ByteArray)

/** Ablauf des Datei-Imports, siehe docs/PLANUNG.md „Datei-Import in der App (7.6b)“. */
sealed interface ImportState {
    data object Idle : ImportState
    data object Reading : ImportState

    /** Dateien liegen bereit, die App ist aber noch nicht verbunden. */
    data class WaitingForSetup(val files: List<ImportFile>) : ImportState
    data object Uploading : ImportState
    data class Preview(
        val preview: ImportPreview,
        val rejected: List<String>,
        val selected: Set<Int>,
        /** Wahl je unbekannter Kategorie; fehlt ein Name, wird sie weggelassen. */
        val categories: Map<String, CategoryChoice> = emptyMap(),
        val importing: Boolean = false,
        /** Fehler beim Bestätigen; die Vorschau bleibt, man kann es erneut versuchen. */
        val error: String? = null,
    ) : ImportState
    data class Done(val result: ImportResult) : ImportState
    data class Failed(val message: String) : ImportState
}

/**
 * Nimmt Importdateien entgegen (Teilen, „Öffnen mit“, Dateiauswahl),
 * prüft sie am Inhalt und schickt sie zur Vorschau an den Server. Import
 * nur online; gespeichert wird erst nach Bestätigung (confirm).
 */
class ImportRepository(
    private val resolver: ContentResolver,
    private val config: ConfigRepository,
    private val api: ApiClient,
    private val sync: SyncRepository,
) {
    private val _state = MutableStateFlow<ImportState>(ImportState.Idle)
    val state: StateFlow<ImportState> = _state.asStateFlow()

    /**
     * Liest die Dateien hinter [uris] sofort ein – andere Apps (WhatsApp)
     * geben nur eine befristete Leseberechtigung – und lädt sie hoch.
     */
    suspend fun receive(uris: List<Uri>) {
        if (uris.isEmpty()) return
        _state.value = ImportState.Reading
        val files = mutableListOf<ImportFile>()
        val rejected = mutableListOf<String>()
        var total = 0L
        withContext(Dispatchers.IO) {
            for (uri in uris.take(MAX_FILES)) {
                val name = displayName(uri)
                val data = try {
                    read(uri, MAX_BYTES - total)
                } catch (e: TooLargeException) {
                    rejected += "„$name“ ist zu groß (höchstens ${MAX_BYTES shr 20} MB zusammen)."
                    continue
                } catch (e: Exception) {
                    rejected += "„$name“ ließ sich nicht lesen."
                    continue
                }
                if (isImportFile(data)) {
                    files += ImportFile(name, data)
                    total += data.size
                } else {
                    rejected += "„$name“ ist keine Rezeptplaner-Importdatei."
                }
            }
        }
        if (uris.size > MAX_FILES) rejected += "Höchstens $MAX_FILES Dateien auf einmal."
        if (files.isEmpty()) {
            _state.value = ImportState.Failed(rejected.joinToString("\n").ifEmpty { "Keine Datei erhalten." })
            return
        }
        if (config.config.first() == null) {
            _state.value = ImportState.WaitingForSetup(files)
            return
        }
        upload(files, rejected)
    }

    /** Nach der Einrichtung mit den wartenden Dateien weitermachen. */
    suspend fun resume() {
        val waiting = _state.value as? ImportState.WaitingForSetup ?: return
        upload(waiting.files, emptyList())
    }

    private suspend fun upload(files: List<ImportFile>, rejected: List<String>) {
        val cfg = config.config.first() ?: return
        _state.value = ImportState.Uploading
        _state.value = try {
            val preview = api.uploadImport(cfg, files.map { it.name to it.data })
            // Vorauswahl wie im Web: alles Importierbare außer Dubletten.
            val selected = preview.recipes.filter { it.importable && it.duplicates.isEmpty() }.map { it.index }.toSet()
            ImportState.Preview(preview, rejected, selected)
        } catch (e: ApiException) {
            ImportState.Failed(if (e.offline) OFFLINE else e.message ?: "Fehler beim Hochladen.")
        }
    }

    fun toggle(index: Int) {
        val p = _state.value as? ImportState.Preview ?: return
        _state.value = p.copy(selected = if (index in p.selected) p.selected - index else p.selected + index)
    }

    /** [choice] null = weglassen. */
    fun choose(category: String, choice: CategoryChoice?) {
        val p = _state.value as? ImportState.Preview ?: return
        _state.value = p.copy(categories = if (choice == null) p.categories - category else p.categories + (category to choice))
    }

    suspend fun confirm() {
        val p = _state.value as? ImportState.Preview ?: return
        val cfg = config.config.first() ?: return
        _state.value = p.copy(importing = true, error = null)
        _state.value = try {
            val result = api.confirmImport(cfg, p.preview.token, p.selected.sorted(), p.categories)
            sync.sync(SyncTrigger.Manual)
            ImportState.Done(result)
        } catch (e: ApiException) {
            p.copy(importing = false, error = if (e.offline) OFFLINE else e.message)
        }
    }

    suspend fun thumbnail(token: String, index: Int): ByteArray? {
        val cfg = config.config.first() ?: return null
        return runCatching { api.importThumbnail(cfg, token, index) }.getOrNull()
    }

    fun reset() {
        _state.value = ImportState.Idle
    }

    private fun displayName(uri: Uri): String =
        runCatching {
            resolver.query(uri, arrayOf(OpenableColumns.DISPLAY_NAME), null, null, null)?.use { c ->
                if (c.moveToFirst()) c.getString(0) else null
            }
        }.getOrNull() ?: uri.lastPathSegment ?: "Datei"

    private fun read(uri: Uri, limit: Long): ByteArray {
        val stream = resolver.openInputStream(uri) ?: error("kein Inhalt")
        return stream.use { input ->
            val out = java.io.ByteArrayOutputStream()
            val buffer = ByteArray(64 * 1024)
            var total = 0L
            while (true) {
                val n = input.read(buffer)
                if (n < 0) break
                total += n
                if (total > limit) throw TooLargeException()
                out.write(buffer, 0, n)
            }
            out.toByteArray()
        }
    }

    private class TooLargeException : Exception()

    companion object {
        // Wie die Grenzen des Servers (server/internal/fileimport/archive.go).
        const val MAX_BYTES = 30L shl 20
        const val MAX_FILES = 10
        private const val OFFLINE = "Import geht nur mit Verbindung zum Rezeptplaner."

        // Byte-Order-Mark am Dateianfang, wie ihn manche Editoren schreiben.
        private val BOM = Char(0xFEFF).toString()

        /**
         * Erkennt eine Importdatei am Inhalt – Dateiname und MIME-Typ sind beim
         * Teilen oft unbrauchbar: ZIP-Kennung oder JSON mit Formatkennung.
         */
        fun isImportFile(data: ByteArray): Boolean {
            if (data.size >= 4 && data[0] == 'P'.code.toByte() && data[1] == 'K'.code.toByte() &&
                ((data[2].toInt() == 3 && data[3].toInt() == 4) || (data[2].toInt() == 5 && data[3].toInt() == 6))
            ) {
                return true
            }
            val head = String(data, 0, minOf(data.size, 4096), Charsets.UTF_8).removePrefix(BOM).trimStart()
            return head.startsWith("{") && Regex("\"format\"\\s*:\\s*\"rezeptplaner-import\"").containsMatchIn(head)
        }
    }
}
