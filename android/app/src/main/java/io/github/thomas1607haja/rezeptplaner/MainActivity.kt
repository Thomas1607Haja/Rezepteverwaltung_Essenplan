package io.github.thomas1607haja.rezeptplaner

import android.content.Intent
import android.net.Uri
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.runtime.mutableStateOf
import androidx.core.content.IntentCompat
import androidx.lifecycle.lifecycleScope
import io.github.thomas1607haja.rezeptplaner.data.SetupLink
import io.github.thomas1607haja.rezeptplaner.ui.RezeptplanerApp
import io.github.thomas1607haja.rezeptplaner.ui.theme.RezeptplanerTheme
import kotlinx.coroutines.launch

/** Einzige Activity; alle Bildschirme sind Compose-Ziele (siehe RezeptplanerApp). */
class MainActivity : ComponentActivity() {

    // Geöffneter Einrichtungslink (rezeptplaner://setup?…), z. B. nach dem
    // Scannen des QR-Codes mit der Kamera-App. Die Einrichtung zeigt die
    // Adresse vor dem Speichern zur Bestätigung an.
    private val pendingLink = mutableStateOf<String?>(null)

    // true: Dateien zum Import sind angekommen, die App soll den
    // Import-Bildschirm zeigen.
    private val openImport = mutableStateOf(false)

    override fun onCreate(savedInstanceState: Bundle?) {
        // Von Rand zu Rand zeichnen (ab Android 16 Pflicht). Die Abstände zu
        // Statusleiste, Navigationsleiste und Tastatur setzt Scaffold.
        enableEdgeToEdge()
        super.onCreate(savedInstanceState)
        // Nur beim ersten Start auswerten, nicht nach einer Drehung o. Ä.
        if (savedInstanceState == null) handleIntent(intent)
        setContent {
            RezeptplanerTheme {
                RezeptplanerApp(
                    pendingLink = pendingLink.value,
                    onLinkConsumed = { pendingLink.value = null },
                    openImport = openImport.value,
                    onImportOpened = { openImport.value = false },
                )
            }
        }
    }

    // launchMode singleTask: Ist die App schon offen, kommt der Link hier an.
    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        handleIntent(intent)
    }

    private fun handleIntent(intent: Intent?) {
        intent ?: return
        val data = intent.data
        if (intent.action == Intent.ACTION_VIEW && data?.scheme == SetupLink.SCHEME && data.host == SetupLink.HOST) {
            pendingLink.value = data.toString()
            return
        }
        val files = importUris(intent)
        if (files.isNotEmpty()) {
            // Sofort einlesen: Die Leseberechtigung anderer Apps (WhatsApp)
            // gilt nur, solange diese Activity den Intent hält.
            lifecycleScope.launch { (application as RezeptplanerApplication).importRepository.receive(files) }
            openImport.value = true
        }
    }

    /** Dateien aus „Öffnen mit“ (data) bzw. Teilen (EXTRA_STREAM, auch mehrere). */
    private fun importUris(intent: Intent): List<Uri> = when (intent.action) {
        Intent.ACTION_VIEW -> listOfNotNull(intent.data).filter { it.scheme == "content" || it.scheme == "file" }
        Intent.ACTION_SEND -> listOfNotNull(IntentCompat.getParcelableExtra(intent, Intent.EXTRA_STREAM, Uri::class.java))
        Intent.ACTION_SEND_MULTIPLE ->
            IntentCompat.getParcelableArrayListExtra(intent, Intent.EXTRA_STREAM, Uri::class.java).orEmpty()
        else -> emptyList()
    }
}
