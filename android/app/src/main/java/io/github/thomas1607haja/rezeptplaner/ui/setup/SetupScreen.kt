package io.github.thomas1607haja.rezeptplaner.ui.setup

import android.util.Log
import androidx.activity.compose.BackHandler
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.safeDrawingPadding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalResources
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import com.google.mlkit.vision.barcode.common.Barcode
import com.google.mlkit.vision.codescanner.GmsBarcodeScannerOptions
import com.google.mlkit.vision.codescanner.GmsBarcodeScanning
import io.github.thomas1607haja.rezeptplaner.R
import io.github.thomas1607haja.rezeptplaner.ui.CrashReportCard
import io.github.thomas1607haja.rezeptplaner.ui.common.tr

private const val TAG = "Rezeptplaner"

/**
 * Einrichtung der Verbindung. [onFinished] wird nach erfolgreichem
 * Speichern aufgerufen, [onCancel] (falls gesetzt) bricht eine erneute
 * Einrichtung ab und behält die alte Verbindung.
 */
@Composable
fun SetupScreen(
    viewModel: SetupViewModel,
    state: SetupViewModel.State,
    onFinished: () -> Unit,
    onCancel: (() -> Unit)?,
) {
    val context = LocalContext.current
    val resources = LocalResources.current
    val scan = {
        val unavailable = resources.getString(R.string.setup_scan_unavailable)
        try {
            val options = GmsBarcodeScannerOptions.Builder()
                .setBarcodeFormats(Barcode.FORMAT_QR_CODE)
                .build()
            GmsBarcodeScanning.getClient(context, options).startScan()
                .addOnSuccessListener { barcode -> barcode.rawValue?.let(viewModel::onLink) }
                .addOnFailureListener { e ->
                    Log.w(TAG, "QR-Scanner meldet Fehler", e)
                    viewModel.onScanFailed("$unavailable (${e.javaClass.simpleName}: ${e.message})")
                }
        } catch (e: RuntimeException) {
            // Fehler der Play-Dienste beim Start sollen die App nicht beenden.
            Log.w(TAG, "QR-Scanner startet nicht", e)
            viewModel.onScanFailed("$unavailable (${e.javaClass.simpleName}: ${e.message})")
        }
        Unit
    }

    // Zurück: innerhalb der Einrichtung einen Schritt zurück, am Anfang
    // abbrechen (nur bei erneuter Einrichtung; sonst schließt die App).
    BackHandler(enabled = state !is SetupViewModel.State.Choose || onCancel != null) {
        if (state is SetupViewModel.State.Choose) onCancel?.invoke() else viewModel.back()
    }

    Surface(Modifier.fillMaxSize()) {
        Column(
            Modifier
                .safeDrawingPadding()
                .imePadding()
                .verticalScroll(rememberScrollState())
                .padding(24.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            Text(tr(R.string.setup_title), style = MaterialTheme.typography.headlineSmall)
            CrashReportCard()
            when (state) {
                is SetupViewModel.State.Choose -> ChooseStep(state, scan, viewModel::showManual, onCancel)
                is SetupViewModel.State.Manual -> ManualStep(state, viewModel::onManual, viewModel::back)
                is SetupViewModel.State.Confirm -> ConfirmStep(state, viewModel::confirm, viewModel::back)
                is SetupViewModel.State.Done -> DoneStep(state, onFinished)
            }
        }
    }
}

@Composable
private fun ChooseStep(
    state: SetupViewModel.State.Choose,
    onScan: () -> Unit,
    onManual: () -> Unit,
    onCancel: (() -> Unit)?,
) {
    Text(tr(R.string.setup_intro))
    state.error?.let { ErrorText(it) }
    Button(onClick = onScan, modifier = Modifier.fillMaxWidth()) { Text(tr(R.string.setup_scan)) }
    OutlinedButton(onClick = onManual, modifier = Modifier.fillMaxWidth()) { Text(tr(R.string.setup_manual)) }
    onCancel?.let { TextButton(onClick = it, modifier = Modifier.fillMaxWidth()) { Text(tr(R.string.common_cancel)) } }
}

@Composable
private fun ManualStep(
    state: SetupViewModel.State.Manual,
    onSubmit: (String, String, String, String) -> Unit,
    onBack: () -> Unit,
) {
    var url by rememberSaveable { mutableStateOf("https://") }
    var key by rememberSaveable { mutableStateOf("") }
    var cfId by rememberSaveable { mutableStateOf("") }
    var cfSecret by rememberSaveable { mutableStateOf("") }

    OutlinedTextField(
        value = url, onValueChange = { url = it }, label = { Text(tr(R.string.setup_app_url)) },
        singleLine = true, keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Uri),
        modifier = Modifier.fillMaxWidth(),
    )
    OutlinedTextField(
        value = key, onValueChange = { key = it }, label = { Text(tr(R.string.setup_api_key)) },
        singleLine = true, visualTransformation = PasswordVisualTransformation(),
        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password),
        modifier = Modifier.fillMaxWidth(),
    )
    Text(tr(R.string.setup_cf_title), style = MaterialTheme.typography.titleSmall)
    OutlinedTextField(
        value = cfId, onValueChange = { cfId = it }, label = { Text("Client ID") },
        singleLine = true, modifier = Modifier.fillMaxWidth(),
    )
    OutlinedTextField(
        value = cfSecret, onValueChange = { cfSecret = it }, label = { Text("Client Secret") },
        singleLine = true, visualTransformation = PasswordVisualTransformation(),
        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password),
        modifier = Modifier.fillMaxWidth(),
    )
    state.error?.let { ErrorText(it) }
    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        OutlinedButton(onClick = onBack) { Text(tr(R.string.common_back)) }
        Button(onClick = { onSubmit(url, key, cfId, cfSecret) }) { Text(tr(R.string.setup_next)) }
    }
}

@Composable
private fun ConfirmStep(state: SetupViewModel.State.Confirm, onConfirm: () -> Unit, onBack: () -> Unit) {
    Text(tr(R.string.setup_confirm_question))
    Card(Modifier.fillMaxWidth(), border = CardDefaults.outlinedCardBorder()) {
        Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
            Text(state.config.host, style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.SemiBold)
            Text(state.config.url, style = MaterialTheme.typography.bodySmall)
            if (state.config.hasCloudflareAccess) {
                Text(tr(R.string.setup_with_cf), style = MaterialTheme.typography.bodySmall)
            }
        }
    }
    Text(
        tr(R.string.setup_confirm_warning),
        style = MaterialTheme.typography.bodySmall,
        color = MaterialTheme.colorScheme.onSurfaceVariant,
    )
    state.error?.let { ErrorText(it) }
    Row(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalAlignment = Alignment.CenterVertically) {
        OutlinedButton(onClick = onBack, enabled = !state.testing) { Text(tr(R.string.common_cancel)) }
        Button(onClick = onConfirm, enabled = !state.testing) {
            Text(tr(if (state.error == null) R.string.setup_connect else R.string.setup_retry))
        }
        if (state.testing) CircularProgressIndicator(Modifier.size(24.dp), strokeWidth = 3.dp)
    }
}

@Composable
private fun DoneStep(state: SetupViewModel.State.Done, onFinished: () -> Unit) {
    Text(tr(R.string.setup_connected_to, "host" to state.config.host), style = MaterialTheme.typography.titleMedium)
    Text(tr(R.string.setup_server_version, "version" to state.serverVersion), color = MaterialTheme.colorScheme.onSurfaceVariant)
    Spacer(Modifier.height(8.dp))
    Button(onClick = onFinished, modifier = Modifier.fillMaxWidth()) { Text(tr(R.string.setup_start)) }
}

@Composable
private fun ErrorText(message: String) {
    Text(message, color = MaterialTheme.colorScheme.error)
}
