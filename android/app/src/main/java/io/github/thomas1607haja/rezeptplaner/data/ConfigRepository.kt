package io.github.thomas1607haja.rezeptplaner.data

import android.content.Context
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.stringPreferencesKey
import androidx.datastore.preferences.preferencesDataStore
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.map
import kotlinx.serialization.json.Json

private val Context.dataStore by preferencesDataStore(name = "settings")

/**
 * Speichert die Verbindung zum Rezeptplaner. Sie liegt als JSON im
 * DataStore, verschlüsselt mit [SecureBox]. Die Datei ist zusätzlich von
 * Backups ausgeschlossen (AndroidManifest, data_extraction_rules.xml).
 */
class ConfigRepository(private val context: Context, private val box: SecureBox = SecureBox()) {

    /** Aktuelle Verbindung; null = noch nicht eingerichtet (oder nicht mehr lesbar). */
    val config: Flow<ServerConfig?> = context.dataStore.data.map { prefs ->
        prefs[CONFIG]?.let(box::decrypt)?.let { runCatching { json.decodeFromString<ServerConfig>(it) }.getOrNull() }
    }

    suspend fun save(config: ServerConfig) {
        val encrypted = box.encrypt(json.encodeToString(config))
        context.dataStore.edit { it[CONFIG] = encrypted }
    }

    suspend fun clear() {
        context.dataStore.edit { it.remove(CONFIG) }
    }

    private companion object {
        val CONFIG = stringPreferencesKey("server_config")
        val json = Json { ignoreUnknownKeys = true }
    }
}
