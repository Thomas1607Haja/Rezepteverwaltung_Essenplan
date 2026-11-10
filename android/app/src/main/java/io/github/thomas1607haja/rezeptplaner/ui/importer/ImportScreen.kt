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
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.produceState
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalUriHandler
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import coil3.compose.AsyncImage
import io.github.thomas1607haja.rezeptplaner.R
import io.github.thomas1607haja.rezeptplaner.api.CategoryChoice
import io.github.thomas1607haja.rezeptplaner.api.ImportCandidate
import io.github.thomas1607haja.rezeptplaner.api.Note
import io.github.thomas1607haja.rezeptplaner.api.SkillCheck
import io.github.thomas1607haja.rezeptplaner.api.TagGroup
import io.github.thomas1607haja.rezeptplaner.data.ImportState
import io.github.thomas1607haja.rezeptplaner.ui.amountText
import io.github.thomas1607haja.rezeptplaner.ui.common.app
import io.github.thomas1607haja.rezeptplaner.ui.common.tr
import io.github.thomas1607haja.rezeptplaner.ui.common.trPlural
import io.github.thomas1607haja.rezeptplaner.ui.formatMinutes
import io.github.thomas1607haja.rezeptplaner.ui.wrapTitle
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
    val snapshot by app.localStore.snapshot.collectAsStateWithLifecycle()
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
            title = { Text(tr(R.string.import_title)) },
            navigationIcon = { IconButton(onClick = close) { Icon(painterResource(R.drawable.ic_back), contentDescription = tr(R.string.common_back)) } },
            windowInsets = WindowInsets(0),
        )
        when (val s = state) {
            ImportState.Idle -> Start(pick, error = null)
            is ImportState.Failed -> Start(pick, error = s.message)
            ImportState.Reading, ImportState.Uploading -> Busy(tr(if (s == ImportState.Reading) R.string.import_reading else R.string.import_checking))
            is ImportState.WaitingForSetup -> Busy(tr(R.string.import_waiting_setup))
            is ImportState.Preview -> Preview(
                s,
                thumbnail = { index -> repo.thumbnail(s.preview.token, index) },
                onToggle = repo::toggle,
                tagGroups = snapshot.tagGroups,
                onChoose = repo::choose,
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
        Text(tr(R.string.import_intro))
        error?.let { Text(it, color = MaterialTheme.colorScheme.error) }
        Button(onClick = pick, modifier = Modifier.fillMaxWidth()) { Text(tr(R.string.import_choose_file)) }
        Card(Modifier.fillMaxWidth(), border = CardDefaults.outlinedCardBorder()) {
            Column(Modifier.padding(12.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                Text(tr(R.string.import_whatsapp_title), style = MaterialTheme.typography.titleSmall)
                Text(tr(R.string.import_whatsapp_share), style = MaterialTheme.typography.bodySmall)
                Text(
                    tr(R.string.import_whatsapp_folder),
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
    tagGroups: List<TagGroup>,
    onChoose: (String, CategoryChoice?) -> Unit,
    onImport: () -> Unit,
    onOther: () -> Unit,
) {
    val count = s.selected.size
    Column(Modifier.fillMaxSize()) {
        LazyColumn(Modifier.weight(1f), contentPadding = PaddingValues(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            items(s.rejected) { Text(it, color = MaterialTheme.colorScheme.error) }
            items(s.preview.fileErrors) {
                Text(
                    tr(R.string.import_item_error, "item" to it.file, "error" to (it.errorDetail?.text() ?: it.error)),
                    color = MaterialTheme.colorScheme.error,
                )
            }
            s.preview.skill?.takeIf { it.updateAvailable && it.downloadUrl.isNotEmpty() }?.let { skill ->
                item { SkillUpdateCard(skill) }
            }
            if (s.preview.recipes.isEmpty()) item { Text(tr(R.string.import_no_recipes)) }
            items(s.preview.recipes, key = { it.index }) { c ->
                CandidateCard(c, selected = c.index in s.selected, enabled = !s.importing, thumbnail = thumbnail, onToggle = { onToggle(c.index) })
            }
            if (s.preview.unknownCategories.isNotEmpty()) {
                item {
                    UnknownCategories(s.preview.unknownCategories, s.categories, tagGroups, enabled = !s.importing, onChoose = onChoose)
                }
            }
            // Am Ende der Liste statt fest unten: Im Querformat blieb für die
            // Vorschau sonst kaum Platz.
            item {
                OutlinedButton(onClick = onOther, enabled = !s.importing, modifier = Modifier.fillMaxWidth()) { Text(tr(R.string.import_other_file)) }
            }
        }
        // Fest bleibt nur, was zum Abschluss gebraucht wird.
        if (s.error != null || s.preview.recipes.isNotEmpty()) {
            Column(Modifier.fillMaxWidth().padding(16.dp, 8.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                s.error?.let { Text(it, color = MaterialTheme.colorScheme.error) }
                if (s.preview.recipes.isNotEmpty()) {
                    Button(onClick = onImport, enabled = count > 0 && !s.importing, modifier = Modifier.fillMaxWidth()) {
                        Text(if (s.importing) tr(R.string.import_importing) else trPlural(R.plurals.import_import_count, count))
                    }
                }
            }
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
                Text(wrapTitle(r.title).ifEmpty { tr(R.string.import_untitled) }, fontWeight = FontWeight.SemiBold, maxLines = 2, overflow = TextOverflow.Ellipsis)
                val details = buildList {
                    add(amountText(r.baseServings, r.yieldUnit))
                    add(trPlural(R.plurals.import_ingredient_count, r.ingredients.size))
                    add(trPlural(R.plurals.import_step_count, r.steps.size))
                    formatMinutes(r.totalMinutes).takeIf { it.isNotEmpty() }?.let(::add)
                }
                Text(details.joinToString(" · "), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                r.sourceName.takeIf { it.isNotEmpty() }?.let {
                    Text(tr(R.string.import_source, "source" to it), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
                if (r.tags.isNotEmpty()) {
                    Text(r.tags.joinToString(", ") { it.name }, style = MaterialTheme.typography.bodySmall)
                }
                texts(c.errorDetails, c.errors).forEach { Text(it, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall) }
                if (c.duplicates.isNotEmpty()) {
                    Text(
                        tr(R.string.import_exists, "titles" to c.duplicates.joinToString(", ") { it.title }),
                        color = MaterialTheme.colorScheme.tertiary, style = MaterialTheme.typography.bodySmall,
                    )
                }
                if (c.unknownCategories.isNotEmpty()) {
                    Text(
                        tr(R.string.import_unknown_in_recipe, "names" to c.unknownCategories.joinToString(", ")),
                        color = MaterialTheme.colorScheme.tertiary, style = MaterialTheme.typography.bodySmall,
                    )
                }
                // Hinweise zu unbekannten Kategorien stehen gesammelt unten.
                val warnings = if (c.warningDetails.isNotEmpty()) {
                    c.warningDetails.filterNot { it.code == UNKNOWN_CATEGORY }.map { it.text() }
                } else {
                    // Server vor 0.20: nur Texte (deutsch), am Wortlaut erkennen.
                    c.warnings.filterNot { it.startsWith("Kategorie „") && "gibt es nicht" in it }
                }
                warnings.forEach {
                    Text(it, color = MaterialTheme.colorScheme.tertiary, style = MaterialTheme.typography.bodySmall)
                }
            }
        }
    }
}

/** Code des Hinweises „Kategorie gibt es nicht“ (server/internal/apierr); steht gesammelt unten. */
private const val UNKNOWN_CATEGORY = "import_unknown_category"

/** Hinweise in der Sprache der App; ältere Server (vor 0.20) liefern nur [fallback]. */
private fun texts(details: List<Note>, fallback: List<String>): List<String> =
    if (details.isNotEmpty()) details.map { it.text() } else fallback

/** Text der Wahl für eine unbekannte Kategorie. */
@Composable
private fun choiceLabel(name: String, choice: CategoryChoice?, groups: List<TagGroup>): String = when {
    choice == null -> tr(R.string.import_skip)
    choice.tagId > 0 -> tr(R.string.import_assign_to, "name" to (groups.flatMap { it.tags }.firstOrNull { it.id == choice.tagId }?.name ?: "?"))
    else -> tr(R.string.import_new_in, "name" to name, "group" to (groups.firstOrNull { it.id == choice.newInGroup }?.name ?: "?"))
}

/**
 * Unbekannte Kategorien wie in der Weboberfläche: einer vorhandenen
 * zuordnen, in einer Gruppe neu anlegen oder weglassen (Standard).
 */
@Composable
private fun UnknownCategories(
    names: List<String>,
    choices: Map<String, CategoryChoice>,
    groups: List<TagGroup>,
    enabled: Boolean,
    onChoose: (String, CategoryChoice?) -> Unit,
) {
    // Ohne Gruppen-IDs (Abgleich mit einem Server vor 0.9.0) lässt sich nichts neu anlegen.
    val withIds = groups.filter { it.id > 0 }
    Card(Modifier.fillMaxWidth(), border = CardDefaults.outlinedCardBorder()) {
        Column(Modifier.padding(12.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
            Text(tr(R.string.import_unknown_title), style = MaterialTheme.typography.titleSmall)
            Text(
                tr(R.string.import_unknown_hint),
                style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            names.forEach { name ->
                var open by remember(name) { mutableStateOf(false) }
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text(tr(R.string.import_quoted, "name" to name), fontWeight = FontWeight.SemiBold, modifier = Modifier.weight(1f))
                    Box {
                        TextButton(onClick = { open = true }, enabled = enabled) { Text(choiceLabel(name, choices[name], groups)) }
                        DropdownMenu(expanded = open, onDismissRequest = { open = false }) {
                            DropdownMenuItem(text = { Text(tr(R.string.import_skip)) }, onClick = { open = false; onChoose(name, null) })
                            groups.filter { it.tags.isNotEmpty() }.forEach { g ->
                                HorizontalDivider()
                                Text(
                                    tr(R.string.import_assign_group, "group" to g.name), style = MaterialTheme.typography.labelMedium,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.padding(horizontal = 12.dp, vertical = 4.dp),
                                )
                                g.tags.forEach { t ->
                                    DropdownMenuItem(text = { Text(t.name) }, onClick = { open = false; onChoose(name, CategoryChoice(tagId = t.id)) })
                                }
                            }
                            if (withIds.isNotEmpty()) {
                                HorizontalDivider()
                                Text(
                                    tr(R.string.import_create), style = MaterialTheme.typography.labelMedium,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.padding(horizontal = 12.dp, vertical = 4.dp),
                                )
                                withIds.forEach { g ->
                                    DropdownMenuItem(text = { Text(g.name) }, onClick = { open = false; onChoose(name, CategoryChoice(newInGroup = g.id)) })
                                }
                            }
                        }
                    }
                }
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
                trPlural(R.plurals.import_imported, created.size),
                style = MaterialTheme.typography.titleMedium,
            )
        }
        items(created, key = { it.id }) { c ->
            Column(Modifier.fillMaxWidth().clickable { onOpenRecipe(c.id) }.padding(vertical = 6.dp)) {
                Text(c.title, color = MaterialTheme.colorScheme.primary)
                texts(c.warningDetails, c.warnings).forEach { Text(it, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.tertiary) }
            }
        }
        items(texts(s.result.warningDetails, s.result.warnings)) { Text(it, color = MaterialTheme.colorScheme.tertiary) }
        if (s.result.failed.isNotEmpty()) {
            item { Text(tr(R.string.import_not_imported), style = MaterialTheme.typography.titleMedium) }
            items(s.result.failed, key = { "f${it.index}" }) { f ->
                Text(
                    tr(
                        R.string.import_item_error,
                        "item" to f.title.ifEmpty { tr(R.string.import_recipe_n, "n" to f.index + 1) },
                        "error" to (f.errorDetail?.text() ?: f.error),
                    ),
                    color = MaterialTheme.colorScheme.error,
                )
            }
        }
        item {
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp), modifier = Modifier.padding(top = 8.dp)) {
                Button(onClick = onClose) { Text(tr(R.string.import_done)) }
                OutlinedButton(onClick = onAgain) { Text(tr(R.string.import_another)) }
            }
        }
    }
}

/** Die Datei stammt von einer älteren Claude-Skill: nur ein Hinweis, der Import geht weiter. */
@Composable
private fun SkillUpdateCard(skill: SkillCheck) {
    val uriHandler = LocalUriHandler.current
    Card(Modifier.fillMaxWidth(), border = CardDefaults.outlinedCardBorder()) {
        Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Text(
                tr(R.string.import_skill_update, "version" to skill.version, "current" to skill.current),
                style = MaterialTheme.typography.bodyMedium,
            )
            OutlinedButton(onClick = { uriHandler.openUri(skill.downloadUrl) }) {
                Text(tr(R.string.import_skill_download, "current" to skill.current))
            }
        }
    }
}
