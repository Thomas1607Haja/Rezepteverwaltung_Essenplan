package io.github.thomas1607haja.rezeptplaner

import android.content.Intent
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.runtime.mutableStateOf
import io.github.thomas1607haja.rezeptplaner.data.SetupLink
import io.github.thomas1607haja.rezeptplaner.ui.RezeptplanerApp
import io.github.thomas1607haja.rezeptplaner.ui.theme.RezeptplanerTheme

/** Einzige Activity; alle Bildschirme sind Compose-Ziele (siehe RezeptplanerApp). */
class MainActivity : ComponentActivity() {

    // Geöffneter Einrichtungslink (rezeptplaner://setup?…), z. B. nach dem
    // Scannen des QR-Codes mit der Kamera-App. Die Einrichtung zeigt die
    // Adresse vor dem Speichern zur Bestätigung an.
    private val pendingLink = mutableStateOf<String?>(null)

    override fun onCreate(savedInstanceState: Bundle?) {
        // Von Rand zu Rand zeichnen (ab Android 16 Pflicht). Die Abstände zu
        // Statusleiste, Navigationsleiste und Tastatur setzt Scaffold.
        enableEdgeToEdge()
        super.onCreate(savedInstanceState)
        // Nur beim ersten Start auswerten, nicht nach einer Drehung o. Ä.
        if (savedInstanceState == null) handleIntent(intent)
        setContent {
            RezeptplanerTheme {
                RezeptplanerApp(pendingLink = pendingLink.value, onLinkConsumed = { pendingLink.value = null })
            }
        }
    }

    // launchMode singleTask: Ist die App schon offen, kommt der Link hier an.
    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        handleIntent(intent)
    }

    private fun handleIntent(intent: Intent?) {
        val data = intent?.data ?: return
        if (intent.action == Intent.ACTION_VIEW && data.scheme == SetupLink.SCHEME && data.host == SetupLink.HOST) {
            pendingLink.value = data.toString()
        }
    }
}
