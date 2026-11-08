package io.github.thomas1607haja.rezeptplaner.ui.edit

import android.content.Context
import android.net.Uri
import androidx.activity.compose.BackHandler
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.PickVisualMediaRequest
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.material3.AssistChip
import io.github.thomas1607haja.rezeptplaner.ui.yieldUnits
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.Checkbox
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.core.content.FileProvider
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import coil3.compose.AsyncImage
import io.github.thomas1607haja.rezeptplaner.R
import io.github.thomas1607haja.rezeptplaner.data.ImageChange
import io.github.thomas1607haja.rezeptplaner.ui.common.RecipeImage
import io.github.thomas1607haja.rezeptplaner.ui.common.app
import io.github.thomas1607haja.rezeptplaner.ui.common.tr
import java.io.File

/**
 * Rezept bearbeiten bzw. anlegen (recipeId 0). [freshVariant]: eben als
 * Variante angelegt – Abbrechen fragt, ob sie wieder gelöscht werden soll.
 */
@OptIn(ExperimentalMaterial3Api::class, ExperimentalLayoutApi::class)
@Composable
fun RecipeEditScreen(
    recipeId: Long,
    fromDraft: Boolean,
    freshVariant: Boolean,
    onSaved: (Long, String?) -> Unit,
    onCancel: () -> Unit,
    onDeleted: () -> Unit,
    /** Vorhandenes Rezept öffnen (Dubletten beim URL-Import). */
    onOpenRecipe: (Long) -> Unit = {},
) {
    val app = app()
    val vm = viewModel(key = "edit-$recipeId-$fromDraft") { RecipeEditViewModel(app, recipeId, fromDraft) }
    val form by vm.form.collectAsStateWithLifecycle()
    val errors by vm.errors.collectAsStateWithLifecycle()
    val message by vm.message.collectAsStateWithLifecycle()
    val saving by vm.saving.collectAsStateWithLifecycle()
    val image by vm.image.collectAsStateWithLifecycle()
    val snapshot by app.localStore.snapshot.collectAsStateWithLifecycle()
    val context = LocalContext.current

    var confirmLeave by remember { mutableStateOf(false) }
    var confirmDelete by remember { mutableStateOf(false) }
    var pasting by remember { mutableStateOf(false) }

    // Zurück: Bei einer frischen Variante fragen, ob sie verworfen wird,
    // sonst bei ungespeicherten Änderungen nachfragen.
    val leave = { if (freshVariant || vm.changed) confirmLeave = true else onCancel() }
    BackHandler(onBack = leave)

    // Kamera-App (TakePicture) und Fotoauswahl: keine Berechtigung nötig.
    var cameraUri by rememberSaveable { mutableStateOf<Uri?>(null) }
    val camera = rememberLauncherForActivityResult(ActivityResultContracts.TakePicture()) { ok ->
        val uri = cameraUri
        if (ok && uri != null) readImage(context, uri)?.let(vm::setImage)
    }
    val gallery = rememberLauncherForActivityResult(ActivityResultContracts.PickVisualMedia()) { uri ->
        if (uri != null) readImage(context, uri)?.let(vm::setImage)
    }

    Column(Modifier.fillMaxSize()) {
        TopAppBar(
            title = { Text(tr(if (vm.isNew) R.string.recipe_new else R.string.recipe_edit_title)) },
            navigationIcon = { IconButton(onClick = leave) { Icon(painterResource(R.drawable.ic_back), contentDescription = tr(R.string.common_back)) } },
            actions = {
                if (saving) CircularProgressIndicator(Modifier.size(24.dp).padding(end = 8.dp))
                TextButton(onClick = { vm.save(onSaved) }, enabled = !saving) { Text(tr(R.string.common_save)) }
            },
            windowInsets = WindowInsets(0),
        )
        Column(
            Modifier.fillMaxSize().imePadding().verticalScroll(rememberScrollState()).padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            message?.let { Text(it, color = MaterialTheme.colorScheme.error) }
            if (vm.duplicates.isNotEmpty()) {
                // Wie im Web: Antippen öffnet das vorhandene Rezept, der Entwurf bleibt erhalten.
                Text(tr(R.string.recipe_edit_duplicates), color = MaterialTheme.colorScheme.tertiary)
                vm.duplicates.forEach { d ->
                    Text(
                        d.title, color = MaterialTheme.colorScheme.primary,
                        modifier = Modifier.clickable { onOpenRecipe(d.id) }.padding(vertical = 2.dp),
                    )
                }
            }

            Field(tr(R.string.recipe_edit_field_title), form.title, errors["title"]) { v -> vm.update { it.copy(title = v) } }
            Field(tr(R.string.recipe_edit_field_description), form.description, null, singleLine = false) { v -> vm.update { it.copy(description = v) } }
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Field(tr(if (form.yieldUnit.isBlank()) R.string.recipe_edit_field_servings else R.string.recipe_edit_field_amount), form.baseServings, errors["baseServings"], Modifier.weight(1f), number = true) { v -> vm.update { it.copy(baseServings = v) } }
                Field(tr(R.string.recipe_edit_field_prep), form.prepMinutes, errors["prepMinutes"], Modifier.weight(1f), number = true) { v -> vm.update { it.copy(prepMinutes = v) } }
                Field(tr(R.string.recipe_edit_field_total), form.totalMinutes, errors["totalMinutes"], Modifier.weight(1f), number = true) { v -> vm.update { it.copy(totalMinutes = v) } }
            }

            // Ergebnis-Einheit (Kuchen & Co.): leer = Portionen; Vorschläge als Chips.
            Field(tr(R.string.recipe_edit_field_yield_unit), form.yieldUnit, null) { v -> vm.update { it.copy(yieldUnit = v) } }
            FlowRow(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                yieldUnits.forEach { u ->
                    AssistChip(onClick = { vm.update { it.copy(yieldUnit = u) } }, label = { Text(u) })
                }
            }

            Section(tr(R.string.recipe_edit_section_image))
            ImageBlock(image, vm.currentImage)
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                OutlinedButton(onClick = {
                    val uri = cameraFile(context)
                    cameraUri = uri
                    camera.launch(uri)
                }) { Text(tr(R.string.recipe_edit_take_photo)) }
                OutlinedButton(onClick = {
                    gallery.launch(PickVisualMediaRequest(ActivityResultContracts.PickVisualMedia.ImageOnly))
                }) { Text(tr(R.string.recipe_edit_gallery)) }
                if (image !is ImageChange.Remove && (vm.currentImage.isNotEmpty() || image !is ImageChange.Keep)) {
                    TextButton(onClick = { vm.setImage(if (vm.currentImage.isNotEmpty()) ImageChange.Remove else ImageChange.Keep) }) {
                        Text(tr(R.string.common_remove))
                    }
                }
            }

            Section(tr(R.string.recipe_categories))
            snapshot.tagGroups.forEach { group ->
                Text(group.name, style = MaterialTheme.typography.labelLarge, color = MaterialTheme.colorScheme.onSurfaceVariant)
                FlowRow(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                    group.tags.forEach { tag ->
                        val on = tag.id in form.tagIds
                        FilterChip(
                            selected = on,
                            onClick = { vm.update { it.copy(tagIds = if (on) it.tagIds - tag.id else it.tagIds + tag.id) } },
                            label = { Text(tag.name) },
                        )
                    }
                }
            }

            Section(tr(R.string.recipe_ingredients))
            form.items.forEachIndexed { index, row ->
                IngredientEditor(
                    row, errors["item:${row.key}"],
                    onChange = { changed ->
                        vm.updateItem(row.key) { changed }
                        if (!row.heading && changed.name != row.name) vm.checkPantry(row.key, changed.name)
                    },
                    onMove = { delta -> vm.update { it.copy(items = it.items.move(index, delta)) } },
                    onDelete = { vm.update { it.copy(items = it.items.filterNot { r -> r.key == row.key }) } },
                )
            }
            FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                OutlinedButton(onClick = { vm.update { it.copy(items = it.items + IngredientRow()) } }) { Text(tr(R.string.recipe_edit_add_ingredient)) }
                OutlinedButton(onClick = { vm.update { it.copy(items = it.items + IngredientRow(heading = true)) } }) { Text(tr(R.string.recipe_edit_add_heading)) }
                TextButton(onClick = { pasting = true }) { Text(tr(R.string.recipe_edit_paste_lines)) }
            }

            Section(tr(R.string.recipe_steps))
            form.steps.forEachIndexed { index, step ->
                Row(verticalAlignment = Alignment.Top) {
                    Text("${index + 1}.", Modifier.padding(top = 16.dp, end = 8.dp), color = MaterialTheme.colorScheme.primary)
                    OutlinedTextField(
                        step.text,
                        { v -> vm.update { f -> f.copy(steps = f.steps.map { if (it.key == step.key) it.copy(text = v) else it }) } },
                        modifier = Modifier.weight(1f),
                    )
                    RowActions(
                        onUp = { vm.update { it.copy(steps = it.steps.move(index, -1)) } },
                        onDown = { vm.update { it.copy(steps = it.steps.move(index, 1)) } },
                        onDelete = { vm.update { it.copy(steps = it.steps.filterNot { s -> s.key == step.key }) } },
                    )
                }
            }
            OutlinedButton(onClick = { vm.update { it.copy(steps = it.steps + StepRow()) } }) { Text(tr(R.string.recipe_edit_add_step)) }

            Section(tr(R.string.recipe_edit_section_source))
            Field(tr(R.string.recipe_edit_field_source_url), form.sourceUrl, errors["sourceUrl"], keyboard = KeyboardType.Uri) { v -> vm.update { it.copy(sourceUrl = v) } }
            Field(tr(R.string.recipe_edit_field_source_name), form.sourceName, null) { v -> vm.update { it.copy(sourceName = v) } }
            Field(tr(R.string.recipe_notes), form.notes, null, singleLine = false) { v -> vm.update { it.copy(notes = v) } }

            if (!vm.isNew) {
                TextButton(onClick = { confirmDelete = true }, enabled = !saving) {
                    Text(tr(R.string.recipe_delete), color = MaterialTheme.colorScheme.error)
                }
            }
        }
    }

    if (pasting) PasteDialog(onDismiss = { pasting = false }) { text -> pasting = false; vm.pasteLines(text) }

    if (confirmLeave) {
        AlertDialog(
            onDismissRequest = { confirmLeave = false },
            title = { Text(tr(if (freshVariant) R.string.recipe_edit_discard_variant_title else R.string.recipe_edit_discard_changes_title)) },
            text = { Text(tr(if (freshVariant) R.string.recipe_edit_discard_variant_text else R.string.recipe_edit_discard_changes_text)) },
            confirmButton = {
                TextButton(onClick = {
                    confirmLeave = false
                    if (freshVariant) vm.delete(onDeleted) else onCancel()
                }) { Text(tr(R.string.recipe_edit_discard)) }
            },
            dismissButton = { TextButton(onClick = { confirmLeave = false }) { Text(tr(R.string.recipe_edit_keep_editing)) } },
        )
    }
    if (confirmDelete) {
        val hasVariants = snapshot.recipes.any { it.parentRecipeId == recipeId }
        AlertDialog(
            onDismissRequest = { confirmDelete = false },
            title = { Text(tr(R.string.recipe_delete_title)) },
            text = {
                Text(
                    tr(R.string.recipe_delete_text, "title" to form.title) +
                        if (hasVariants) " " + tr(R.string.recipe_delete_variants) else "",
                )
            },
            confirmButton = { TextButton(onClick = { confirmDelete = false; vm.delete(onDeleted) }) { Text(tr(R.string.common_delete)) } },
            dismissButton = { TextButton(onClick = { confirmDelete = false }) { Text(tr(R.string.common_cancel)) } },
        )
    }
}

@Composable
private fun Section(title: String) {
    Text(title, style = MaterialTheme.typography.titleMedium, modifier = Modifier.padding(top = 8.dp))
}

@Composable
private fun Field(
    label: String,
    value: String,
    error: String?,
    modifier: Modifier = Modifier,
    singleLine: Boolean = true,
    number: Boolean = false,
    keyboard: KeyboardType = if (number) KeyboardType.Number else KeyboardType.Text,
    onChange: (String) -> Unit,
) {
    OutlinedTextField(
        value, onChange, label = { Text(label) }, singleLine = singleLine, isError = error != null,
        supportingText = error?.let { { Text(it) } },
        keyboardOptions = KeyboardOptions(keyboardType = keyboard),
        modifier = Modifier.fillMaxWidth().then(modifier),
    )
}

@Composable
private fun ImageBlock(image: ImageChange, current: String) {
    val modifier = Modifier.fillMaxWidth().aspectRatio(16f / 9f).clip(MaterialTheme.shapes.medium)
    when (image) {
        is ImageChange.Upload -> AsyncImage(model = image.data, contentDescription = null, contentScale = ContentScale.Crop, modifier = modifier)
        is ImageChange.FromUrl -> Text(tr(R.string.recipe_edit_image_from_url), style = MaterialTheme.typography.bodySmall)
        ImageChange.Remove -> Text(tr(R.string.recipe_edit_image_remove), style = MaterialTheme.typography.bodySmall)
        ImageChange.Keep -> if (current.isNotEmpty()) RecipeImage(current, modifier) else Text(tr(R.string.recipe_edit_no_image), style = MaterialTheme.typography.bodySmall)
    }
}

@Composable
private fun IngredientEditor(
    row: IngredientRow,
    error: String?,
    onChange: (IngredientRow) -> Unit,
    onMove: (Int) -> Unit,
    onDelete: () -> Unit,
) {
    Card(Modifier.fillMaxWidth(), border = CardDefaults.outlinedCardBorder()) {
        Column(Modifier.padding(8.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
            if (row.heading) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    OutlinedTextField(
                        row.name, { onChange(row.copy(name = it)) }, label = { Text(tr(R.string.recipe_edit_heading_label)) },
                        singleLine = true, modifier = Modifier.weight(1f),
                    )
                    RowActions({ onMove(-1) }, { onMove(1) }, onDelete)
                }
            } else {
                Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                    OutlinedTextField(
                        row.quantity, { onChange(row.copy(quantity = it)) }, label = { Text(tr(R.string.recipe_edit_quantity)) }, singleLine = true,
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal), modifier = Modifier.width(84.dp),
                    )
                    OutlinedTextField(row.unit, { onChange(row.copy(unit = it)) }, label = { Text(tr(R.string.recipe_edit_unit)) }, singleLine = true, modifier = Modifier.width(88.dp))
                    OutlinedTextField(row.name, { onChange(row.copy(name = it)) }, label = { Text(tr(R.string.recipe_edit_ingredient)) }, singleLine = true, modifier = Modifier.weight(1f))
                }
                Row(verticalAlignment = Alignment.CenterVertically) {
                    OutlinedTextField(
                        row.note, { onChange(row.copy(note = it)) }, label = { Text(tr(R.string.recipe_edit_note)) }, singleLine = true,
                        modifier = Modifier.weight(1f),
                    )
                    Checkbox(checked = row.shoppingRelevant, onCheckedChange = { onChange(row.copy(shoppingRelevant = it)) })
                    Text(tr(R.string.recipe_edit_buy), style = MaterialTheme.typography.bodySmall)
                    RowActions({ onMove(-1) }, { onMove(1) }, onDelete)
                }
                error?.let { Text(it, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall) }
            }
        }
    }
}

@Composable
private fun RowActions(onUp: () -> Unit, onDown: () -> Unit, onDelete: () -> Unit) {
    Row {
        IconButton(onClick = onUp, modifier = Modifier.size(36.dp)) { Icon(painterResource(R.drawable.ic_up), contentDescription = tr(R.string.recipe_edit_move_up)) }
        IconButton(onClick = onDown, modifier = Modifier.size(36.dp)) { Icon(painterResource(R.drawable.ic_down), contentDescription = tr(R.string.recipe_edit_move_down)) }
        IconButton(onClick = onDelete, modifier = Modifier.size(36.dp)) { Icon(painterResource(R.drawable.ic_delete), contentDescription = tr(R.string.common_remove)) }
    }
}

@Composable
private fun PasteDialog(onDismiss: () -> Unit, onPaste: (String) -> Unit) {
    var text by rememberSaveable { mutableStateOf("") }
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(tr(R.string.recipe_edit_paste_title)) },
        text = {
            OutlinedTextField(
                text, { text = it }, label = { Text(tr(R.string.recipe_edit_paste_label)) },
                singleLine = false, modifier = Modifier.fillMaxWidth(),
            )
        },
        confirmButton = { TextButton(onClick = { onPaste(text) }, enabled = text.isNotBlank()) { Text(tr(R.string.recipe_edit_paste)) } },
        dismissButton = { TextButton(onClick = onDismiss) { Text(tr(R.string.common_cancel)) } },
    )
}

/** Datei im Cache für die Kamera-App (über FileProvider freigegeben). */
internal fun cameraFile(context: Context): Uri {
    val dir = File(context.cacheDir, "camera").apply { mkdirs() }
    val file = File(dir, "foto-${System.currentTimeMillis()}.jpg")
    return FileProvider.getUriForFile(context, "${context.packageName}.files", file)
}

/** Liest ein gewähltes Bild ein (höchstens 20 MB wie beim Server). */
internal fun readImage(context: Context, uri: Uri): ImageChange.Upload? = runCatching {
    val type = context.contentResolver.getType(uri) ?: "image/jpeg"
    val data = context.contentResolver.openInputStream(uri)?.use { it.readBytes() } ?: return null
    if (data.size > 20 * 1024 * 1024) return null
    ImageChange.Upload(data, type)
}.getOrNull()
