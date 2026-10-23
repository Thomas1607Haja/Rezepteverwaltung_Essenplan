package io.github.thomas1607haja.rezeptplaner.ui.theme

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color

// Farben wie in der Weboberfläche (web/src/app.css), damit App und Web
// gleich aussehen. Hell und dunkel folgen der Systemeinstellung.
private val LightColors = lightColorScheme(
    primary = Color(0xFFC2552D),
    onPrimary = Color(0xFFFFFFFF),
    primaryContainer = Color(0xFFF6E3DA),
    onPrimaryContainer = Color(0xFF1F1D1A),
    secondaryContainer = Color(0xFFF6E3DA),
    onSecondaryContainer = Color(0xFF1F1D1A),
    background = Color(0xFFF7F6F3),
    onBackground = Color(0xFF1F1D1A),
    surface = Color(0xFFF7F6F3),
    onSurface = Color(0xFF1F1D1A),
    surfaceContainer = Color(0xFFFFFFFF),
    surfaceVariant = Color(0xFFEFEDE8),
    onSurfaceVariant = Color(0xFF6B665E),
    outline = Color(0xFFE2DFD8),
    error = Color(0xFFB3261E),
)

private val DarkColors = darkColorScheme(
    primary = Color(0xFFE8825A),
    onPrimary = Color(0xFF1F1D1A),
    primaryContainer = Color(0xFF3D2A21),
    onPrimaryContainer = Color(0xFFECE9E4),
    secondaryContainer = Color(0xFF3D2A21),
    onSecondaryContainer = Color(0xFFECE9E4),
    background = Color(0xFF171614),
    onBackground = Color(0xFFECE9E4),
    surface = Color(0xFF171614),
    onSurface = Color(0xFFECE9E4),
    surfaceContainer = Color(0xFF211F1C),
    surfaceVariant = Color(0xFF2C2925),
    onSurfaceVariant = Color(0xFFA39D93),
    outline = Color(0xFF3A3631),
    error = Color(0xFFF2B8B5),
)

@Composable
fun RezeptplanerTheme(
    darkTheme: Boolean = isSystemInDarkTheme(),
    content: @Composable () -> Unit,
) {
    MaterialTheme(
        colorScheme = if (darkTheme) DarkColors else LightColors,
        content = content,
    )
}
