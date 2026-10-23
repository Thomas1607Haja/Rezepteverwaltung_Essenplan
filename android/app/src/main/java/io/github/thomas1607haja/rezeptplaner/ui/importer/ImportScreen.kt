package io.github.thomas1607haja.rezeptplaner.ui.importer

import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.Checkbox
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.produceState
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import coil3.compose.AsyncImage
import io.github.thomas1607haja.rezeptplaner.R
import io.github.thomas1607haja.rezeptplaner.api.ImportCandidate
import io.github.thomas1607haja.rezeptplaner.data.ImportState
import io.github.thomas1607haja.rezeptplaner.ui.common.app
import io.github.thomas1607haja.rezeptplaner.ui.formatMinutes
import io.github.thomas1607haja.rezeptplaner.ui.portions
import kotlinx.coroutines.launch

/**
 * Datei-Import: Dateien kommen über Teilen/„Öffnen mit“ (MainActivity) oder
 * die Dateiauswahl hier. Vorschau wie in der Weboberfläche, gespeichert
 * wird erst nach „Importieren“.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ImportScreen(onClose: () -> Unit, onOpenRecipe: (Long) -> Unit) {
    val app = app()
    val repo = app.importRepository
    val state by repo.state.collectAsStateWithLifecycle()
    val scope = rememberCoroutineScope()
    // Android-Dateiauswahl, mehrere Dateien; alle Typen, weil WhatsApp-
    // Dokumente oft einen allgemeinen Typ tragen – geprüft wird der Inhalt.
    val picker = rememberLauncherForActivityResult(ActivityResultContracts.OpenMultipleDocuments()) { uris ->
        if (uris.isNotEmpty()) scope.launch { repo.receive(uris) }
    }
    val pick = { picker.launch(arrayOf("*/*")) }
    val close = {
        repo.reset()
        onClose()
    }

    Column(Modifier.fillMaxSize()) {
        TopAppBar(
            title = { Text("Datei importieren") },
            navigationIcon = { IconButton(onClick = close) { Icon(painterResource(R.drawable.ic_back), contentDescription = "Zurück") } },
            windowInsets = WindowInsets(0),
        )
        when (val s = state) {
            ImportState.Idle -> Start(pick, error = null)
            is ImportState.Failed -> Start(pick, error = s.message)
            ImportState.Reading, ImportState.Uploading -> Busy(if (s == ImportState.Reading) "Datei wird gelesen …" else "Wird geprüft …")
            is ImportState.WaitingForSetup -> Busy("Erst die App mit dem Rezeptplaner verbinden – danach geht es weiter.")
            is ImportState.Preview -> Preview(
                s,
                thumbnail = { index -> repo.thumbnail(s.preview.token, index) },
                onToggle = repo::toggle,
                onImport = { scope.launch { repo.confirm() } },
                onOther = pick,
            )
            is ImportState.Done -> Done(s, onOpenRecipe, onAgain = { repo.reset() }, onClose = close)
        }
    }
}

@Composable
private fun Start(pick: () -> Unit, error: String?) {
    Column(Modifier.fillMaxWidth().padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
        Text(
            "Importdatei (ZIP oder JSON, z. B. von der Claude-Skill „rezeptplaner-import“) wählen. Gespeichert wird erst nach einer Vorschau.",
        )
        error?.let { Text(it, color = MaterialTheme.colorScheme.error) }
        Button(onClick = pick, modifier = Modifier.fillMaxWidth()) { Text("Datei auswählen") }
        Card(Modifier.fillMaxWidth(), border = CardDefaults.outlinedCardBorder()) {
            Column(Modifier.padding(12.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                Text("Aus WhatsApp", style = MaterialTheme.typography.titleSmall)
                Text("Datei lange drücken → Teilen → Rezeptplaner.", style = MaterialTheme.typography.bodySmall)
                Text(
                    "Oder hier „Datei auswählen“: WhatsApp-Dokumente liegen unter Android → media → com.whatsapp → " +
                        "WhatsApp → Media → WhatsApp Documents (meist auch unter „Zuletzt“).",
                    style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }
    }
}

@Composable
private fun Busy(text: String) {
    Row(Modifier.padding(24.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(12.dp)) {
        CircularProgressIndicator(Modifier.size(24.dp))
        Text(text)
    }
}

@Composable
private fun Preview(
    s: ImportState.Preview,
    thumbnail: suspend (Int) -> ByteArray?,
    onToggle: (Int) -> Unit,
    onImport: () -> Unit,
    onOther: () -> Unit,
) {
    val count = s.selected.size
    Column(Modifier.fillMaxSize()) {
        LazyColumn(Modifier.weight(1f), contentPadding = PaddingValues(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            val messages = s.rejected + s.preview.fileErrors.map { "${it.file}: ${it.error}" }
            items(messages) { Text(it, color = MaterialTheme.colorScheme.error) }
            if (s.preview.recipes.isEmpty()) item { Text("Keine Rezepte gefunden.") }
            items(s.preview.recipes, key = { it.index }) { c ->
                CandidateCard(c, selected = c.index in s.selected, enabled = !s.importing, thumbnail = thumbnail, onToggle = { onToggle(c.index) })
            }
        }
        Column(Modifier.fillMaxWidth().padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            s.error?.let { Text(it, color = MaterialTheme.colorScheme.error) }
            if (s.preview.recipes.isNotEmpty()) {
                Button(onClick = onImport, enabled = count > 0 && !s.importing, modifier = Modifier.fillMaxWidth()) {
                    Text(if (s.importing) "Importiert …" else if (count == 1) "1 Rezept importieren" else "$count Rezepte importieren")
                }
            }
            OutlinedButton(onClick = onOther, enabled = !s.importing, modifier = Modifier.fillMaxWidth()) { Text("Andere Datei") }
        }
    }
}

@Composable
private fun CandidateCard(
    c: ImportCandidate,
    selected: Boolean,
    enabled: Boolean,
    thumbnail: suspend (Int) -> ByteArray?,
    onToggle: () -> Unit,
) {
    val image by produceState<ByteArray?>(null, c.index, c.thumbnail) { if (c.thumbnail) value = thumbnail(c.index) }
    val r = c.recipe
    Card(
        Modifier.fillMaxWidth().then(if (c.importable && enabled) Modifier.clickable(onClick = onToggle) else Modifier),
        border = CardDefaults.outlinedCardBorder(),
    ) {
        Row(Modifier.padding(8.dp), horizontalArrangement = Arrangement.spacedBy(10.dp)) {
            Checkbox(checked = selected, onCheckedChange = { onToggle() }, enabled = c.importable && enabled)
            Box(Modifier.size(64.dp).clip(MaterialTheme.shapes.small).background(MaterialTheme.colorScheme.surfaceVariant)) {
                image?.let { AsyncImage(model = it, contentDescription = null, contentScale = ContentScale.Crop, modifier = Modifier.fillMaxSize()) }
            }
            Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                Text(r.title.ifEmpty { "(ohne Titel)" }, fontWeight = FontWeight.SemiBold, maxLines = 2, overflow = TextOverflow.Ellipsis)
                val details = buildList {
                    add(portions(r.baseServings))
                    add(if (r.ingredients.size == 1) "1 Zutat" else "${r.ingredients.size} Zutaten")
                    add(if (r.steps.size == 1) "1 Schritt" else "${r.steps.size} Schritte")
                    formatMinutes(r.totalMinutes).takeIf { it.isNotEmpty() }?.let(::add)
                }
                Text(details.joinToString(" · "), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                r.sourceName.takeIf { it.isNotEmpty() }?.let {
                    Text("Quelle: $it", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
                if (r.tags.isNotEmpty()) {
                    Text(r.tags.joinToString(", ") { it.name }, style = MaterialTheme.typography.bodySmall)
                }
                c.errors.forEach { Text(it, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall) }
                if (c.duplicates.isNotEmpty()) {
                    Text(
                        "Gibt es schon: " + c.duplicates.joinToString(", ") { it.title },
                        color = MaterialTheme.colorScheme.tertiary, style = MaterialTheme.typography.bodySmall,
                    )
                }
                c.warnings.forEach { Text(it, color = MaterialTheme.colorScheme.tertiary, style = MaterialTheme.typography.bodySmall) }
            }
        }
    }
}

@Composable
private fun Done(s: ImportState.Done, onOpenRecipe: (Long) -> Unit, onAgain: () -> Unit, onClose: () -> Unit) {
    LazyColumn(contentPadding = PaddingValues(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
        val created = s.result.created
        item {
            Text(
                if (created.size == 1) "1 Rezept importiert" else "${created.size} Rezepte importiert",
                style = MaterialTheme.typography.titleMedium,
            )
        }
        items(created, key = { it.id }) { c ->
            Column(Modifier.fillMaxWidth().clickable { onOpenRecipe(c.id) }.padding(vertical = 6.dp)) {
                Text(c.title, color = MaterialTheme.colorScheme.primary)
                c.warnings.forEach { Text(it, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.tertiary) }
            }
        }
        if (s.result.failed.isNotEmpty()) {
            item { Text("Nicht importiert", style = MaterialTheme.typography.titleMedium) }
            items(s.result.failed, key = { "f${it.index}" }) { f ->
                Text("${f.title.ifEmpty { "Rezept ${f.index + 1}" }}: ${f.error}", color = MaterialTheme.colorScheme.error)
            }
        }
        item {
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp), modifier = Modifier.padding(top = 8.dp)) {
                Button(onClick = onClose) { Text("Fertig") }
                OutlinedButton(onClick = onAgain) { Text("Weitere Datei") }
            }
        }
    }
}
