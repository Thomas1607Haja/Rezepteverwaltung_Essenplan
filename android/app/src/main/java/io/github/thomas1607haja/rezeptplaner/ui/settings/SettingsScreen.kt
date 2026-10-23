package io.github.thomas1607haja.rezeptplaner.ui.settings

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
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import io.github.thomas1607haja.rezeptplaner.BuildConfig
import io.github.thomas1607haja.rezeptplaner.ui.CrashReportCard
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
        Text("Einstellungen", style = MaterialTheme.typography.headlineSmall)
        CrashReportCard()

        Card(Modifier.fillMaxWidth(), border = CardDefaults.outlinedCardBorder()) {
            Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text("Verbindung", style = MaterialTheme.typography.titleMedium)
                Text(config?.host ?: "–", fontWeight = FontWeight.SemiBold)
                config?.let { Text(it.url, style = MaterialTheme.typography.bodySmall) }
                Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    when (val c = check) {
                        SettingsViewModel.Check.None -> {}
                        SettingsViewModel.Check.Running -> {
                            CircularProgressIndicator(Modifier.size(18.dp), strokeWidth = 2.dp)
                            Text("Prüfe …")
                        }
                        is SettingsViewModel.Check.Ok -> Text("Verbunden · Rezeptplaner ${c.serverVersion}", color = MaterialTheme.colorScheme.primary)
                        is SettingsViewModel.Check.Failed -> Text(c.message, color = MaterialTheme.colorScheme.error)
                    }
                }
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    OutlinedButton(onClick = viewModel::testConnection, enabled = check != SettingsViewModel.Check.Running) {
                        Text("Verbindung testen")
                    }
                    OutlinedButton(onClick = onReconnect) { Text("Neu verbinden") }
                }
            }
        }

        CloudflareCard(viewModel, configured = config?.hasCloudflareAccess == true)

        Card(Modifier.fillMaxWidth(), border = CardDefaults.outlinedCardBorder()) {
            Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text("Gerät trennen", style = MaterialTheme.typography.titleMedium)
                Text(
                    "Entfernt die Verbindung von diesem Handy. Den Schlüssel in der Weboberfläche unter " +
                        "„Verbundene Geräte“ zusätzlich sperren, wenn das Gerät weitergegeben wird.",
                    style = MaterialTheme.typography.bodySmall,
                )
                OutlinedButton(onClick = { confirmDisconnect = true }) { Text("Verbindung trennen") }
            }
        }

        Text(
            "Rezeptplaner für Android ${BuildConfig.VERSION_NAME}",
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }

    if (confirmDisconnect) {
        AlertDialog(
            onDismissRequest = { confirmDisconnect = false },
            title = { Text("Verbindung trennen?") },
            text = { Text("Danach muss die App neu verbunden werden.") },
            confirmButton = {
                TextButton(onClick = { confirmDisconnect = false; viewModel.disconnect() }) { Text("Trennen") }
            },
            dismissButton = { TextButton(onClick = { confirmDisconnect = false }) { Text("Abbrechen") } },
        )
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
                if (configured) "Service Token ist eingerichtet." else "Kein Service Token (nur nötig, wenn Access aktiv ist).",
                style = MaterialTheme.typography.bodySmall,
            )
            if (!editing) {
                OutlinedButton(onClick = { editing = true; id = ""; secret = ""; error = null }) {
                    Text(if (configured) "Ändern" else "Einrichten")
                }
            } else {
                OutlinedTextField(id, { id = it }, label = { Text("Client ID") }, singleLine = true, modifier = Modifier.fillMaxWidth())
                OutlinedTextField(
                    secret, { secret = it }, label = { Text("Client Secret") }, singleLine = true,
                    visualTransformation = PasswordVisualTransformation(), modifier = Modifier.fillMaxWidth(),
                )
                Text("Beide leer lassen und speichern = ohne Cloudflare Access.", style = MaterialTheme.typography.bodySmall)
                error?.let { Text(it, color = MaterialTheme.colorScheme.error) }
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    OutlinedButton(onClick = { editing = false }) { Text("Abbrechen") }
                    Button(onClick = {
                        scope.launch {
                            error = viewModel.saveCloudflareAccess(id, secret)
                            if (error == null) editing = false
                        }
                    }) { Text("Speichern") }
                }
            }
        }
    }
}
