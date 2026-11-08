package io.github.thomas1607haja.rezeptplaner.ui.settings

import android.annotation.SuppressLint
import android.content.Context
import android.content.Intent
import androidx.core.net.toUri
import android.os.PowerManager
import android.provider.Settings
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.compose.LifecycleEventEffect
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import io.github.thomas1607haja.rezeptplaner.R
import io.github.thomas1607haja.rezeptplaner.data.SyncLog
import io.github.thomas1607haja.rezeptplaner.data.SyncTrigger
import io.github.thomas1607haja.rezeptplaner.ui.common.app
import io.github.thomas1607haja.rezeptplaner.ui.common.formatTime
import io.github.thomas1607haja.rezeptplaner.ui.common.tr
import kotlinx.coroutines.launch

private const val HOURS_12 = 12 * 60 * 60 * 1000L

/**
 * Soll der Hinweis zur Akku-Optimierung erscheinen? Erst wenn der erste
 * Abgleich mindestens 12 Stunden her ist und seitdem kein Abgleich im
 * Hintergrund innerhalb von 12 Stunden gelang – dann hält One UI die App
 * offenbar an. Siehe docs/PLANUNG.md, Abschnitt 7.
 */
fun backgroundSyncStalled(log: SyncLog, now: Long): Boolean {
    val first = log.firstSuccess ?: return false
    if (now - first < HOURS_12) return false
    val last = log.lastBackgroundSuccess ?: return true
    return now - last > HOURS_12
}

private fun Context.ignoresBatteryOptimizations(): Boolean =
    getSystemService(PowerManager::class.java).isIgnoringBatteryOptimizations(packageName)

@Composable
fun SyncCard() {
    val app = app()
    val context = LocalContext.current
    val state by app.syncRepository.state.collectAsStateWithLifecycle()
    val scope = rememberCoroutineScope()
    var unrestricted by remember { mutableStateOf(context.ignoresBatteryOptimizations()) }
    // Nach der Rückkehr aus den Android-Einstellungen neu prüfen.
    LifecycleEventEffect(Lifecycle.Event.ON_RESUME) { unrestricted = context.ignoresBatteryOptimizations() }

    val log = state.log
    val stalled = !unrestricted && backgroundSyncStalled(log, System.currentTimeMillis())

    Card(
        Modifier.fillMaxWidth(),
        border = CardDefaults.outlinedCardBorder(),
        colors = if (stalled) CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.errorContainer) else CardDefaults.cardColors(),
    ) {
        Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Text(tr(R.string.settings_sync_title), style = MaterialTheme.typography.titleMedium)
            val last = formatTime(log.lastSuccess)
            Text(
                log.lastSuccessTrigger?.let { tr(R.string.settings_sync_last_trigger, "time" to last, "trigger" to tr(label(it))) }
                    ?: tr(R.string.settings_sync_last, "time" to last),
            )
            Text(tr(R.string.settings_sync_background, "time" to formatTime(log.lastBackgroundSuccess)), style = MaterialTheme.typography.bodySmall)
            log.lastError?.let { Text(it, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall) }
            OutlinedButton(
                onClick = { scope.launch { app.syncRepository.sync(SyncTrigger.Manual) } },
                enabled = !state.running,
            ) { Text(tr(if (state.running) R.string.settings_sync_running else R.string.settings_sync_now)) }

            if (!unrestricted) {
                Text(
                    tr(if (stalled) R.string.settings_sync_stalled else R.string.settings_sync_battery_hint),
                    style = MaterialTheme.typography.bodySmall,
                )
                Button(onClick = { requestUnrestricted(context) }) { Text(tr(R.string.settings_sync_battery_button)) }
                Text(
                    tr(R.string.settings_sync_samsung_hint),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            } else {
                Text(tr(R.string.settings_sync_unrestricted), style = MaterialTheme.typography.bodySmall)
            }
        }
    }
}

private fun label(trigger: SyncTrigger): Int = when (trigger) {
    SyncTrigger.Start -> R.string.settings_trigger_start
    SyncTrigger.Pull -> R.string.settings_trigger_pull
    SyncTrigger.Background -> R.string.settings_trigger_background
    SyncTrigger.Manual -> R.string.settings_trigger_manual
}

// Für eine App außerhalb des Play Store ist der direkte Dialog erlaubt
// und für die Familie der einfachste Weg.
@SuppressLint("BatteryLife")
private fun requestUnrestricted(context: Context) {
    val intent = Intent(Settings.ACTION_REQUEST_IGNORE_BATTERY_OPTIMIZATIONS, "package:${context.packageName}".toUri())
    runCatching { context.startActivity(intent) }
        .onFailure { context.startActivity(Intent(Settings.ACTION_IGNORE_BATTERY_OPTIMIZATION_SETTINGS)) }
}
