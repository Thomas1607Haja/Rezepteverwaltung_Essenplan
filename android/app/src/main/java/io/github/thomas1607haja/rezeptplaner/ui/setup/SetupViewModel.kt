package io.github.thomas1607haja.rezeptplaner.ui.setup

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import io.github.thomas1607haja.rezeptplaner.BuildConfig
import io.github.thomas1607haja.rezeptplaner.api.ApiClient
import io.github.thomas1607haja.rezeptplaner.api.ApiException
import io.github.thomas1607haja.rezeptplaner.data.ConfigRepository
import io.github.thomas1607haja.rezeptplaner.data.InvalidSetupException
import io.github.thomas1607haja.rezeptplaner.data.ServerConfig
import io.github.thomas1607haja.rezeptplaner.data.SetupLink
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

/**
 * Ablauf der Einrichtung: QR-Code scannen (oder von Hand eingeben) →
 * Adresse bestätigen → Verbindung testen → speichern. Gespeichert wird nur
 * eine Verbindung, die der Server bestätigt hat.
 */
class SetupViewModel(
    private val repository: ConfigRepository,
    private val api: ApiClient,
) : ViewModel() {

    sealed interface State {
        /** Start: scannen oder von Hand eingeben. */
        data class Choose(val error: String? = null) : State

        /** Eingabe von Hand. */
        data class Manual(val error: String? = null) : State

        /** Gelesene Verbindung, wartet auf Bestätigung bzw. wird getestet. */
        data class Confirm(val config: ServerConfig, val testing: Boolean = false, val error: String? = null) : State

        /** Verbindung steht und ist gespeichert. */
        data class Done(val config: ServerConfig, val serverVersion: String) : State
    }

    private val _state = MutableStateFlow<State>(State.Choose())
    val state: StateFlow<State> = _state.asStateFlow()

    /** Text aus dem QR-Scanner oder einem geöffneten Link. */
    fun onLink(text: String) {
        _state.value = try {
            State.Confirm(SetupLink.parse(text, allowLocalHttp = BuildConfig.ALLOW_LOCAL_HTTP))
        } catch (e: InvalidSetupException) {
            State.Choose(e.message)
        }
    }

    fun onScanFailed(message: String) {
        _state.value = State.Choose(message)
    }

    fun showManual() {
        _state.value = State.Manual()
    }

    fun back() {
        _state.value = State.Choose()
    }

    fun onManual(url: String, key: String, cfId: String, cfSecret: String) {
        _state.value = try {
            State.Confirm(SetupLink.validate(url, key, cfId, cfSecret, allowLocalHttp = BuildConfig.ALLOW_LOCAL_HTTP))
        } catch (e: InvalidSetupException) {
            State.Manual(e.message)
        }
    }

    /** Testet die Verbindung und speichert sie, wenn der Server sie annimmt. */
    fun confirm() {
        val current = _state.value as? State.Confirm ?: return
        _state.value = current.copy(testing = true, error = null)
        viewModelScope.launch {
            try {
                val info = api.system(current.config)
                repository.save(current.config)
                _state.value = State.Done(current.config, info.version)
            } catch (e: ApiException) {
                _state.update { (it as? State.Confirm)?.copy(testing = false, error = e.message) ?: it }
            }
        }
    }
}
