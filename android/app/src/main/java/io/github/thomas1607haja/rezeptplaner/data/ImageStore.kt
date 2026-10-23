package io.github.thomas1607haja.rezeptplaner.data

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.withContext
import java.io.File

/**
 * Rezeptbilder auf dem Handy, je Bild eine Datei unter dem Namen vom
 * Server (z. B. "0123…ef_thumb.jpg"). Vorschaubilder lädt der Abgleich
 * immer, große Bilder für den Einkaufszeitraum vorab, sonst beim ersten
 * Ansehen.
 */
class ImageStore(private val dir: File) {

    // Steigt bei jeder Änderung; die Anzeige (RecipeImage) schaut dann neu
    // nach, ob ein bisher fehlendes Bild inzwischen da ist.
    private val _version = MutableStateFlow(0)
    val version: StateFlow<Int> = _version.asStateFlow()

    fun file(name: String): File? = if (valid(name)) File(dir, name).takeIf { it.exists() } else null

    fun has(name: String) = file(name) != null

    suspend fun save(name: String, data: ByteArray) = withContext(Dispatchers.IO) {
        require(valid(name)) { "ungültiger Bildname" }
        dir.mkdirs()
        val tmp = File(dir, "$name.tmp")
        tmp.writeBytes(data)
        tmp.renameTo(File(dir, name))
        _version.update { it + 1 }
    }

    /** Löscht alle Bilder, die nicht in [keep] stehen (gelöschte Rezepte, neue Bilder). */
    suspend fun prune(keep: Set<String>) = withContext(Dispatchers.IO) {
        dir.listFiles()?.filter { it.name !in keep }?.forEach { it.delete() }
        _version.update { it + 1 }
    }

    suspend fun clear() = withContext(Dispatchers.IO) {
        dir.deleteRecursively()
        _version.update { it + 1 }
    }

    companion object {
        // Wie validName in server/internal/images: kein Pfad, kein "..".
        private val pattern = Regex("^[0-9a-f]{32}(_thumb)?\\.jpg$")

        fun valid(name: String) = pattern.matches(name)
    }
}
