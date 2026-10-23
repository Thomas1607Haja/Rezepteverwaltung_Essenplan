package io.github.thomas1607haja.rezeptplaner.ui.settings

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import io.github.thomas1607haja.rezeptplaner.api.ApiClient
import io.github.thomas1607haja.rezeptplaner.api.ApiException
import io.github.thomas1607haja.rezeptplaner.data.ConfigRepository
import io.github.thomas1607haja.rezeptplaner.data.InvalidSetupException
import io.github.thomas1607haja.rezeptplaner.data.ServerConfig
import io.github.thomas1607haja.rezeptplaner.data.SetupLink
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.stateIn
import kotlinx.coroutines.launch

/** Einstellungen der App: Verbindung anzeigen, testen, Cloudflare Access ändern, trennen. */
class SettingsViewModel(
    private val repository: ConfigRepository,
    private val api: ApiClient,
) : ViewModel() {

    /** Ergebnis des letzten Verbindungstests. */
    sealed interface Check {
        data object None : Check
        data object Running : Check
        data class Ok(val serverVersion: String) : Check
        data class Failed(val message: String) : Check
    }

    val config: StateFlow<ServerConfig?> =
        repository.config.stateIn(viewModelScope, SharingStarted.WhileSubscribed(5_000), null)

    private val _check = MutableStateFlow<Check>(Check.None)
    val check: StateFlow<Check> = _check.asStateFlow()

    fun testConnection() {
        viewModelScope.launch {
            val current = repository.config.first() ?: return@launch
            _check.value = Check.Running
            _check.value = try {
                Check.Ok(api.system(current).version)
            } catch (e: ApiException) {
                Check.Failed(e.message ?: "Fehler")
            }
        }
    }

    /**
     * Speichert ein neues Service Token für Cloudflare Access (beide leer =
     * ohne) und testet die Verbindung damit. Liefert eine Fehlermeldung oder null.
     */
    suspend fun saveCloudflareAccess(id: String, secret: String): String? {
        val current = repository.config.first() ?: return "Nicht verbunden."
        val updated = try {
            SetupLink.validate(current.url, current.apiKey, id, secret)
        } catch (e: InvalidSetupException) {
            return e.message
        }
        repository.save(updated)
        testConnection()
        return null
    }

    fun disconnect() {
        viewModelScope.launch {
            repository.clear()
            _check.value = Check.None
        }
    }
}
