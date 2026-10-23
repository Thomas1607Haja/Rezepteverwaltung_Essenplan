package io.github.thomas1607haja.rezeptplaner.ui.common

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.unit.dp
import coil3.compose.AsyncImage
import io.github.thomas1607haja.rezeptplaner.R
import io.github.thomas1607haja.rezeptplaner.RezeptplanerApplication
import io.github.thomas1607haja.rezeptplaner.data.SyncLog
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter

/** Zugriff auf die gemeinsamen Objekte der App aus einer Composable. */
@Composable
fun app(): RezeptplanerApplication = LocalContext.current.applicationContext as RezeptplanerApplication

/**
 * Rezeptbild aus dem lokalen Speicher. Fehlt die Datei und ist [load] true,
 * wird sie einmal vom Server geholt (große Bilder beim ersten Ansehen).
 * Ohne Bild erscheint ein schlichter Platzhalter.
 */
@Composable
fun RecipeImage(name: String, modifier: Modifier = Modifier, load: Boolean = false) {
    val app = app()
    val version by app.imageStore.version.collectAsState()
    val current = remember(name, version) { app.imageStore.file(name) }
    if (current == null && load && name.isNotEmpty()) {
        LaunchedEffect(name) { app.syncRepository.ensureImage(name) }
    }
    if (current != null) {
        AsyncImage(model = current, contentDescription = null, contentScale = ContentScale.Crop, modifier = modifier)
    } else {
        Box(modifier.background(MaterialTheme.colorScheme.surfaceVariant), contentAlignment = Alignment.Center) {
            Icon(
                painterResource(R.drawable.ic_dish), contentDescription = null,
                tint = MaterialTheme.colorScheme.outline,
            )
        }
    }
}

private val timeFormat = DateTimeFormatter.ofPattern("dd.MM. HH:mm").withZone(ZoneId.systemDefault())

fun formatTime(millis: Long?): String = millis?.let { timeFormat.format(Instant.ofEpochMilli(it)) } ?: "noch nie"

/** „Offline – Stand von …“, wenn der letzte Abgleich keine Verbindung bekam. */
@Composable
fun OfflineBanner(log: SyncLog) {
    if (!log.offline) return
    Text(
        "Offline – Stand von ${formatTime(log.lastSuccess)}",
        style = MaterialTheme.typography.bodySmall,
        color = MaterialTheme.colorScheme.onSecondaryContainer,
        modifier = Modifier
            .fillMaxWidth()
            .background(MaterialTheme.colorScheme.secondaryContainer)
            .padding(horizontal = 16.dp, vertical = 6.dp),
    )
}
