package io.github.thomas1607haja.rezeptplaner

import android.app.Application
import io.github.thomas1607haja.rezeptplaner.api.ApiClient
import io.github.thomas1607haja.rezeptplaner.data.ConfigRepository

/**
 * Hält die Objekte, die die ganze App teilt. Ohne Framework für
 * Abhängigkeiten: Bei dieser Größe genügt eine Stelle, an der alles
 * einmal erzeugt wird.
 */
class RezeptplanerApplication : Application() {
    val configRepository by lazy { ConfigRepository(this) }
    val apiClient by lazy { ApiClient() }
}
