package io.github.thomas1607haja.rezeptplaner

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import io.github.thomas1607haja.rezeptplaner.ui.RezeptplanerApp
import io.github.thomas1607haja.rezeptplaner.ui.theme.RezeptplanerTheme

/** Einzige Activity; alle Bildschirme sind Compose-Ziele (siehe RezeptplanerApp). */
class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        // Von Rand zu Rand zeichnen (ab Android 16 Pflicht). Die Abstände zu
        // Statusleiste, Navigationsleiste und Tastatur setzt Scaffold.
        enableEdgeToEdge()
        super.onCreate(savedInstanceState)
        setContent {
            RezeptplanerTheme {
                RezeptplanerApp()
            }
        }
    }
}
