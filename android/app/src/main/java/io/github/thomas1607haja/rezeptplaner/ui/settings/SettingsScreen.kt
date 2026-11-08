package io.github.thomas1607haja.rezeptplaner.ui.settings

import android.content.Context
import android.content.Intent
import android.net.Uri
import android.os.Build
import android.provider.Settings
import androidx.annotation.RequiresApi
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalUriHandler
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import io.github.thomas1607haja.rezeptplaner.BuildConfig
import io.github.thomas1607haja.rezeptplaner.R
import io.github.thomas1607haja.rezeptplaner.ui.CrashReportCard
import io.github.thomas1607haja.rezeptplaner.ui.common.tr
import kotlinx.coroutines.launch

@Composable
fun SettingsScreen(viewModel: SettingsViewModel, onReconnect: () -> Unit) {
    val config by viewModel.config.collectAsStateWithLifecycle()
    val check by viewModel.check.collectAsStateWithLifecycle()
    var confirmDisconnect by remember { mutableStateOf(false) }

    // Beim Öffnen einmal prüfen, ob die Verbindung steht.
    LaunchedEffect(Unit) { viewModel.testConnection() }

    Column(
        Modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        Text(tr(R.string.nav_settings), style = MaterialTheme.typography.headlineSmall)
        CrashReportCard()

        Card(Modifier.fillMaxWidth(), border = CardDefaults.outlinedCardBorder()) {
            Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text(tr(R.string.settings_connection), style = MaterialTheme.typography.titleMedium)
                Text(config?.host ?: "–", fontWeight = FontWeight.SemiBold)
                config?.let { Text(it.url, style = MaterialTheme.typography.bodySmall) }
                Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    when (val c = check) {
                        SettingsViewModel.Check.None -> {}
                        SettingsViewModel.Check.Running -> {
                            CircularProgressIndicator(Modifier.size(18.dp), strokeWidth = 2.dp)
                            Text(tr(R.string.settings_checking))
                        }
                        is SettingsViewModel.Check.Ok -> Text(tr(R.string.settings_connected_version, "version" to c.serverVersion), color = MaterialTheme.colorScheme.primary)
                        is SettingsViewModel.Check.Failed -> Text(c.message, color = MaterialTheme.colorScheme.error)
                    }
                }
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    OutlinedButton(onClick = viewModel::testConnection, enabled = check != SettingsViewModel.Check.Running) {
                        Text(tr(R.string.settings_test_connection))
                    }
                    OutlinedButton(onClick = onReconnect) { Text(tr(R.string.settings_reconnect)) }
                }
            }
        }

        SyncCard()

        LanguageCard()

        CloudflareCard(viewModel, configured = config?.hasCloudflareAccess == true)

        Card(Modifier.fillMaxWidth(), border = CardDefaults.outlinedCardBorder()) {
            Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text(tr(R.string.settings_disconnect_title), style = MaterialTheme.typography.titleMedium)
                Text(
                    tr(R.string.settings_disconnect_text),
                    style = MaterialTheme.typography.bodySmall,
                )
                OutlinedButton(onClick = { confirmDisconnect = true }) { Text(tr(R.string.settings_disconnect_button)) }
            }
        }

        // Nur in der GitHub-Variante gesetzt, in „play“ leer (build.gradle.kts).
        if (BuildConfig.SUPPORT_URL.isNotEmpty()) SupportCard(BuildConfig.SUPPORT_URL)

        Text(
            tr(R.string.settings_app_version, "version" to BuildConfig.VERSION_NAME),
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }

    if (confirmDisconnect) {
        AlertDialog(
            onDismissRequest = { confirmDisconnect = false },
            title = { Text(tr(R.string.settings_disconnect_confirm_title)) },
            text = { Text(tr(R.string.settings_disconnect_confirm_text)) },
            confirmButton = {
                TextButton(onClick = { confirmDisconnect = false; viewModel.disconnect() }) { Text(tr(R.string.settings_disconnect_confirm)) }
            },
            dismissButton = { TextButton(onClick = { confirmDisconnect = false }) { Text(tr(R.string.common_cancel)) } },
        )
    }
}

/** Link zu Buy Me a Coffee; öffnet die Seite im Browser. */
@Composable
private fun SupportCard(url: String) {
    val uriHandler = LocalUriHandler.current
    Card(Modifier.fillMaxWidth(), border = CardDefaults.outlinedCardBorder()) {
        Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Text(tr(R.string.support_title), style = MaterialTheme.typography.titleMedium)
            Text(tr(R.string.support_text), style = MaterialTheme.typography.bodySmall)
            OutlinedButton(onClick = { uriHandler.openUri(url) }) { Text(tr(R.string.support_button)) }
        }
    }
}

@Composable
private fun CloudflareCard(viewModel: SettingsViewModel, configured: Boolean) {
    var editing by rememberSaveable { mutableStateOf(false) }
    var id by rememberSaveable { mutableStateOf("") }
    var secret by rememberSaveable { mutableStateOf("") }
    var error by rememberSaveable { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    Card(Modifier.fillMaxWidth(), border = CardDefaults.outlinedCardBorder()) {
        Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Text("Cloudflare Access", style = MaterialTheme.typography.titleMedium)
            Text(
                tr(if (configured) R.string.settings_cf_configured else R.string.settings_cf_none),
                style = MaterialTheme.typography.bodySmall,
            )
            if (!editing) {
                OutlinedButton(onClick = { editing = true; id = ""; secret = ""; error = null }) {
                    Text(tr(if (configured) R.string.settings_cf_change else R.string.settings_cf_setup))
                }
            } else {
                OutlinedTextField(id, { id = it }, label = { Text("Client ID") }, singleLine = true, modifier = Modifier.fillMaxWidth())
                OutlinedTextField(
                    secret, { secret = it }, label = { Text("Client Secret") }, singleLine = true,
                    visualTransformation = PasswordVisualTransformation(), modifier = Modifier.fillMaxWidth(),
                )
                Text(tr(R.string.settings_cf_empty_hint), style = MaterialTheme.typography.bodySmall)
                error?.let { Text(it, color = MaterialTheme.colorScheme.error) }
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    OutlinedButton(onClick = { editing = false }) { Text(tr(R.string.common_cancel)) }
                    Button(onClick = {
                        scope.launch {
                            error = viewModel.saveCloudflareAccess(id, secret)
                            if (error == null) editing = false
                        }
                    }) { Text(tr(R.string.common_save)) }
                }
            }
        }
    }
}

/**
 * Sprache: Die App folgt Android. Ab Android 13 lässt sie sich pro App in den
 * Android-Einstellungen wählen; der Knopf öffnet diese Seite (darunter fehlt er).
 */
@Composable
private fun LanguageCard() {
    val context = LocalContext.current
    Card(Modifier.fillMaxWidth(), border = CardDefaults.outlinedCardBorder()) {
        Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Text(tr(R.string.settings_language_title), style = MaterialTheme.typography.titleMedium)
            Text(tr(R.string.settings_language_text), style = MaterialTheme.typography.bodySmall)
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
                OutlinedButton(onClick = { openAppLanguageSettings(context) }) { Text(tr(R.string.settings_language_button)) }
            }
        }
    }
}

@RequiresApi(Build.VERSION_CODES.TIRAMISU)
private fun openAppLanguageSettings(context: Context) {
    val intent = Intent(Settings.ACTION_APP_LOCALE_SETTINGS, Uri.fromParts("package", context.packageName, null))
    // Manche Hersteller kennen die Seite nicht: dann die allgemeine App-Info.
    runCatching { context.startActivity(intent) }
        .onFailure {
            runCatching {
                context.startActivity(Intent(Settings.ACTION_APPLICATION_DETAILS_SETTINGS, Uri.fromParts("package", context.packageName, null)))
            }
        }
}
