package io.github.thomas1607haja.rezeptplaner.sync

import android.content.Context
import androidx.work.Constraints
import androidx.work.CoroutineWorker
import androidx.work.ExistingPeriodicWorkPolicy
import androidx.work.NetworkType
import androidx.work.PeriodicWorkRequestBuilder
import androidx.work.WorkManager
import androidx.work.WorkerParameters
import io.github.thomas1607haja.rezeptplaner.RezeptplanerApplication
import io.github.thomas1607haja.rezeptplaner.data.SyncTrigger
import java.util.concurrent.TimeUnit

/**
 * Abgleich im Hintergrund, alle 4 Stunden mit Netz. Samsung (One UI)
 * schiebt solche Aufgaben bei akku-optimierten Apps oft weit hinaus; die
 * Einstellungen zeigen dann einen Hinweis (siehe SettingsScreen). Beim
 * Start der App und beim Herunterziehen wird ohnehin abgeglichen.
 */
class SyncWorker(context: Context, params: WorkerParameters) : CoroutineWorker(context, params) {

    override suspend fun doWork(): Result {
        val app = applicationContext as RezeptplanerApplication
        app.syncRepository.sync(SyncTrigger.Background) ?: return Result.success()
        // Ohne Verbindung später erneut versuchen, sonst bis zum nächsten Lauf warten.
        return if (app.syncRepository.state.value.log.offline) Result.retry() else Result.success()
    }

    companion object {
        private const val NAME = "sync"

        fun schedule(context: Context) {
            val request = PeriodicWorkRequestBuilder<SyncWorker>(4, TimeUnit.HOURS)
                .setConstraints(Constraints.Builder().setRequiredNetworkType(NetworkType.CONNECTED).build())
                .build()
            WorkManager.getInstance(context).enqueueUniquePeriodicWork(NAME, ExistingPeriodicWorkPolicy.KEEP, request)
        }

        fun cancel(context: Context) {
            WorkManager.getInstance(context).cancelUniqueWork(NAME)
        }
    }
}
