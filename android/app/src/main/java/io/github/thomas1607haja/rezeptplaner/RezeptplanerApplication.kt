package io.github.thomas1607haja.rezeptplaner

import android.app.Application
import io.github.thomas1607haja.rezeptplaner.api.ApiClient
import io.github.thomas1607haja.rezeptplaner.data.ConfigRepository
import io.github.thomas1607haja.rezeptplaner.data.ImageStore
import io.github.thomas1607haja.rezeptplaner.data.ImportRepository
import io.github.thomas1607haja.rezeptplaner.data.LocalStore
import io.github.thomas1607haja.rezeptplaner.data.PlanRepository
import io.github.thomas1607haja.rezeptplaner.data.RecipeRepository
import io.github.thomas1607haja.rezeptplaner.data.ShoppingRepository
import io.github.thomas1607haja.rezeptplaner.data.StatisticsRepository
import io.github.thomas1607haja.rezeptplaner.data.SyncRepository
import io.github.thomas1607haja.rezeptplaner.ui.recipes.RecipeFilter
import kotlinx.coroutines.flow.MutableStateFlow
import java.io.File

/**
 * Hält die Objekte, die die ganze App teilt. Ohne Framework für
 * Abhängigkeiten: Bei dieser Größe genügt eine Stelle, an der alles
 * einmal erzeugt wird.
 */
class RezeptplanerApplication : Application() {
    val configRepository by lazy { ConfigRepository(this) }
    val apiClient by lazy { ApiClient() }
    val localStore by lazy { LocalStore(File(filesDir, "snapshot.json")) }
    val imageStore by lazy { ImageStore(File(filesDir, "images")) }
    val syncRepository by lazy {
        SyncRepository(configRepository, apiClient, localStore, imageStore, File(filesDir, "sync_log.json"))
    }
    val planRepository by lazy { PlanRepository(configRepository, apiClient, syncRepository) }
    val shoppingRepository by lazy { ShoppingRepository(configRepository, apiClient, syncRepository) }
    val recipeRepository by lazy { RecipeRepository(configRepository, apiClient, syncRepository) }
    val statisticsRepository by lazy { StatisticsRepository(configRepository, apiClient) }
    val importRepository by lazy { ImportRepository(contentResolver, configRepository, apiClient, syncRepository) }

    /**
     * Filter der Rezeptübersicht, solange die App läuft (wie recipeFilter im
     * Web). Hier statt im ViewModel, damit z. B. ein Kategorie-Chip in der
     * Kochansicht die Übersicht gefiltert öffnen kann.
     */
    val recipeFilter = MutableStateFlow(RecipeFilter())

    override fun onCreate() {
        super.onCreate()
        CrashLog.install(this)
    }
}
