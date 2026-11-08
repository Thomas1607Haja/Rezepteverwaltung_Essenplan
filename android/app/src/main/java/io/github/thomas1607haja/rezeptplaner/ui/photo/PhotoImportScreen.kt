package io.github.thomas1607haja.rezeptplaner.ui.photo

import android.net.Uri
import androidx.activity.compose.BackHandler
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.PickVisualMediaRequest
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.background
import androidx.compose.foundation.ExperimentalFoundationApi
import androidx.compose.foundation.combinedClickable
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
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
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import io.github.thomas1607haja.rezeptplaner.R
import io.github.thomas1607haja.rezeptplaner.ui.common.app
import io.github.thomas1607haja.rezeptplaner.ui.common.tr
import io.github.thomas1607haja.rezeptplaner.ui.edit.cameraFile
import io.github.thomas1607haja.rezeptplaner.ui.edit.readImage

/**
 * Rezept aus Fotos (7.7): Seiten fotografieren oder aus der Galerie wählen,
 * erkannte Zeilen zuordnen, dann weiter im Bearbeiten-Bildschirm.
 */
@OptIn(ExperimentalMaterial3Api::class, ExperimentalLayoutApi::class, ExperimentalFoundationApi::class)
@Composable
fun PhotoImportScreen(onDraft: () -> Unit, onClose: () -> Unit) {
    val app = app()
    val vm = viewModel { PhotoImportViewModel(app) }
    val state by vm.state.collectAsStateWithLifecycle()
    val context = LocalContext.current
    var confirmLeave by remember { mutableStateOf(false) }

    val leave = { if (state.lines.isNotEmpty()) confirmLeave = true else onClose() }
    BackHandler(onBack = leave)

    // Kamera-App und Fotoauswahl wie beim Rezeptbild: keine Berechtigung nötig.
    var cameraUri by rememberSaveable { mutableStateOf<Uri?>(null) }
    var cameraForDish by rememberSaveable { mutableStateOf(false) }
    val camera = rememberLauncherForActivityResult(ActivityResultContracts.TakePicture()) { ok ->
        val uri = cameraUri
        if (ok && uri != null) {
            if (cameraForDish) vm.setDish(readImage(context, uri)) else vm.addPages(listOf(uri))
        }
    }
    val pages = rememberLauncherForActivityResult(ActivityResultContracts.PickMultipleVisualMedia(5)) { uris -> vm.addPages(uris) }
    val dishPicker = rememberLauncherForActivityResult(ActivityResultContracts.PickVisualMedia()) { uri ->
        if (uri != null) vm.setDish(readImage(context, uri))
    }
    fun takePhoto(dish: Boolean) {
        val uri = cameraFile(context)
        cameraUri = uri
        cameraForDish = dish
        camera.launch(uri)
    }
    val imagesOnly = PickVisualMediaRequest(ActivityResultContracts.PickVisualMedia.ImageOnly)

    Column(Modifier.fillMaxSize()) {
        TopAppBar(
            title = { Text(tr(R.string.photo_title)) },
            navigationIcon = { IconButton(onClick = leave) { Icon(painterResource(R.drawable.ic_back), contentDescription = tr(R.string.common_back)) } },
            windowInsets = WindowInsets(0),
        )

        // Eine Liste für alles: Knöpfe, Zeilen und Gerichtfoto scrollen
        // zusammen (im Querformat blieb für die Zeilen sonst kaum Platz).
        // Die Auswahl der Art haftet oben, fest bleibt nur „Weiter“.
        LazyColumn(Modifier.weight(1f).fillMaxWidth()) {
            item {
                Column(Modifier.padding(horizontal = 16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    if (state.lines.isEmpty()) {
                        Text(tr(R.string.photo_intro))
                    }
                    FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        OutlinedButton(onClick = { takePhoto(dish = false) }, enabled = state.busy == null) {
                            Text(tr(if (state.pages == 0) R.string.photo_take_page else R.string.photo_take_another))
                        }
                        OutlinedButton(onClick = { pages.launch(imagesOnly) }, enabled = state.busy == null) {
                            Text(tr(R.string.photo_from_gallery))
                        }
                    }
                    state.busy?.let {
                        Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                            CircularProgressIndicator(Modifier.size(24.dp))
                            Text(it)
                        }
                    }
                    state.error?.let { Text(it, color = MaterialTheme.colorScheme.error) }
                }
            }

            if (state.lines.isNotEmpty()) {
                stickyHeader {
                    Column(Modifier.fillMaxWidth().background(MaterialTheme.colorScheme.background)) {
                        Column(Modifier.padding(16.dp, 8.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                            FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                                LineKind.entries.forEach { kind ->
                                    FilterChip(
                                        selected = state.brush == kind,
                                        onClick = { vm.setBrush(kind) },
                                        label = { Text(tr(kind.label), color = if (kind == LineKind.Ignore) Color.Unspecified else kindColor(kind)) },
                                    )
                                }
                            }
                            Text(
                                tr(R.string.photo_tap_hint),
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                            )
                        }
                        HorizontalDivider()
                    }
                }
                itemsIndexed(state.lines) { i, line ->
                    val kind = state.kinds[i]
                    if (i > 0 && line.page != state.lines[i - 1].page) {
                        Text(
                            tr(R.string.photo_page, "n" to line.page + 1),
                            style = MaterialTheme.typography.labelMedium,
                            modifier = Modifier.padding(16.dp, 12.dp, 16.dp, 4.dp),
                        )
                    }
                    Row(
                        Modifier.fillMaxWidth()
                            .combinedClickable(onClick = { vm.tap(i) }, onLongClick = { vm.markRange(i) })
                            .padding(horizontal = 16.dp, vertical = 6.dp),
                        verticalAlignment = Alignment.Top,
                    ) {
                        Text(
                            tr(kind.label),
                            style = MaterialTheme.typography.labelMedium,
                            color = kindColor(kind),
                            modifier = Modifier.width(76.dp),
                        )
                        Text(
                            line.text,
                            style = MaterialTheme.typography.bodyMedium,
                            color = if (kind == LineKind.Ignore) MaterialTheme.colorScheme.onSurfaceVariant else MaterialTheme.colorScheme.onSurface,
                            textDecoration = if (kind == LineKind.Ignore) TextDecoration.LineThrough else null,
                        )
                    }
                }
                item {
                    HorizontalDivider()
                    // Optionales Rezeptbild: Foto vom fertigen Gericht.
                    FlowRow(Modifier.padding(16.dp, 4.dp), itemVerticalAlignment = Alignment.CenterVertically) {
                        Text(
                            tr(if (state.dish != null) R.string.photo_dish_chosen else R.string.photo_dish_optional),
                            style = MaterialTheme.typography.bodySmall,
                        )
                        TextButton(onClick = { takePhoto(dish = true) }) { Text(tr(R.string.photo_camera)) }
                        TextButton(onClick = { dishPicker.launch(imagesOnly) }) { Text(tr(R.string.photo_gallery)) }
                        if (state.dish != null) TextButton(onClick = { vm.setDish(null) }) { Text(tr(R.string.common_remove)) }
                    }
                }
            }
        }

        if (state.lines.isNotEmpty()) {
            HorizontalDivider()
            Button(
                onClick = { vm.next(onDraft) },
                enabled = state.busy == null,
                modifier = Modifier.fillMaxWidth().padding(16.dp, 8.dp),
            ) { Text(tr(R.string.photo_next)) }
        }
    }

    if (confirmLeave) {
        AlertDialog(
            onDismissRequest = { confirmLeave = false },
            title = { Text(tr(R.string.photo_leave_title)) },
            text = { Text(tr(R.string.photo_leave_text)) },
            confirmButton = { TextButton(onClick = { confirmLeave = false; vm.clear(); onClose() }) { Text(tr(R.string.photo_discard)) } },
            dismissButton = { TextButton(onClick = { confirmLeave = false }) { Text(tr(R.string.photo_keep)) } },
        )
    }
}

/** Gut unterscheidbare Farben je Art, hell und dunkel lesbar. */
@Composable
private fun kindColor(kind: LineKind): Color {
    val dark = isSystemInDarkTheme()
    return when (kind) {
        LineKind.Title -> MaterialTheme.colorScheme.primary
        LineKind.Ingredient -> if (dark) Color(0xFF8BC48A) else Color(0xFF2E7D32)
        LineKind.Step -> if (dark) Color(0xFF8AB4E0) else Color(0xFF2F6CA8)
        LineKind.Ignore -> MaterialTheme.colorScheme.outline
    }
}
