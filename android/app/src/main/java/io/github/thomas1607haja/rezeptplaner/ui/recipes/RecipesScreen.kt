package io.github.thomas1607haja.rezeptplaner.ui.recipes

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.FloatingActionButton
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.ViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import io.github.thomas1607haja.rezeptplaner.R
import io.github.thomas1607haja.rezeptplaner.api.isIllustrated
import io.github.thomas1607haja.rezeptplaner.data.SyncTrigger
import io.github.thomas1607haja.rezeptplaner.ui.common.OfflineBanner
import io.github.thomas1607haja.rezeptplaner.ui.common.RecipeImage
import io.github.thomas1607haja.rezeptplaner.ui.common.app
import io.github.thomas1607haja.rezeptplaner.ui.formatMinutes
import io.github.thomas1607haja.rezeptplaner.ui.planStatus
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.launch
import java.time.LocalDate

/** Eigener Filter für die Rezeptauswahl beim Planen (die Übersicht nutzt RezeptplanerApplication.recipeFilter). */
class RecipesViewModel : ViewModel() {
    val filter = MutableStateFlow(RecipeFilter())
}

@OptIn(ExperimentalMaterial3Api::class, ExperimentalLayoutApi::class)
@Composable
fun RecipesScreen(
    onOpen: (Long) -> Unit,
    modifier: Modifier = Modifier,
    /** Gesetzt: Knopf „+“ mit „Datei importieren“ (nicht in der Rezeptauswahl). */
    onImport: (() -> Unit)? = null,
    onNew: () -> Unit = {},
    onFromUrl: () -> Unit = {},
    onFromPhoto: () -> Unit = {},
    /** Filter dieser Liste; null = der Filter der Rezeptübersicht. */
    filterState: MutableStateFlow<RecipeFilter>? = null,
    /** Inhalt über der Suche, z. B. „Eintrag ohne Rezept“ in der Rezeptauswahl. */
    header: @Composable () -> Unit = {},
) {
    val app = app()
    val snapshot by app.localStore.snapshot.collectAsStateWithLifecycle()
    val sync by app.syncRepository.state.collectAsStateWithLifecycle()
    val state = filterState ?: app.recipeFilter
    val filter by state.collectAsStateWithLifecycle()
    val items = remember(snapshot, filter) { filterRecipes(snapshot, filter) }
    val today = remember(snapshot.today) { snapshot.today.takeIf { it.isNotEmpty() }?.let(LocalDate::parse) ?: LocalDate.now() }
    val scope = rememberCoroutineScope()
    var showTags by remember { mutableStateOf(false) }
    var showSort by remember { mutableStateOf(false) }

    Column(modifier.fillMaxSize()) {
        OfflineBanner(sync.log)
        header()
        Column(Modifier.padding(horizontal = 16.dp, vertical = 8.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            OutlinedTextField(
                value = filter.query,
                onValueChange = { q -> state.value = filter.copy(query = q) },
                placeholder = { Text("Suchen nach Titel oder Zutat") },
                leadingIcon = { Icon(painterResource(R.drawable.ic_search), contentDescription = null) },
                singleLine = true,
                modifier = Modifier.fillMaxWidth(),
            )
            // FlowRow: bricht auf schmalen Handys in eine zweite Zeile um.
            FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp), itemVerticalAlignment = Alignment.CenterVertically) {
                FilterChip(
                    selected = filter.tagIds.isNotEmpty(),
                    onClick = { showTags = true },
                    label = { Text(if (filter.tagIds.isEmpty()) "Kategorien" else "Kategorien (${filter.tagIds.size})") },
                    leadingIcon = { Icon(painterResource(R.drawable.ic_filter), contentDescription = null, Modifier.size(18.dp)) },
                )
                FilterChip(
                    selected = filter.favoritesOnly,
                    onClick = { state.value = filter.copy(favoritesOnly = !filter.favoritesOnly) },
                    label = { Text("Favoriten") },
                    leadingIcon = { Icon(painterResource(R.drawable.ic_heart), contentDescription = null, Modifier.size(18.dp)) },
                )
                // Nur anbieten, wenn es gezeichnete Bilder gibt.
                if (filter.illustrationsOnly || snapshot.recipes.any { it.isIllustrated }) {
                    FilterChip(
                        selected = filter.illustrationsOnly,
                        onClick = { state.value = filter.copy(illustrationsOnly = !filter.illustrationsOnly) },
                        label = { Text("Illustrationen") },
                    )
                }
                Box {
                    TextButton(onClick = { showSort = true }) { Text(filter.sort.label) }
                    DropdownMenu(expanded = showSort, onDismissRequest = { showSort = false }) {
                        RecipeSort.entries.forEach { sort ->
                            DropdownMenuItem(text = { Text(sort.label) }, onClick = {
                                state.value = filter.copy(sort = sort)
                                showSort = false
                            })
                        }
                    }
                }
            }
        }

        PullToRefreshBox(
            isRefreshing = sync.running,
            onRefresh = { scope.launch { app.syncRepository.sync(SyncTrigger.Pull) } },
            modifier = Modifier.fillMaxSize(),
        ) {
            when {
                snapshot.isEmpty -> Message(if (sync.running) "Rezepte werden geladen …" else "Noch keine Daten – nach unten ziehen zum Laden.")
                items.isEmpty() -> Message(
                    if (filter.isActive) "Keine Rezepte zu dieser Suche." else "Noch keine Rezepte.",
                    onReset = if (filter.isActive) ({ state.value = filter.reset() }) else null,
                )
                else -> LazyVerticalGrid(
                    columns = GridCells.Adaptive(minSize = 160.dp),
                    contentPadding = PaddingValues(16.dp),
                    horizontalArrangement = Arrangement.spacedBy(12.dp),
                    verticalArrangement = Arrangement.spacedBy(12.dp),
                    modifier = Modifier.fillMaxSize(),
                ) {
                    // Trefferzahl wie im Web, über die ganze Breite.
                    item(span = { GridItemSpan(maxLineSpan) }) {
                        Text(
                            if (items.size == 1) "1 Rezept" else "${items.size} Rezepte",
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                    items(items, key = { it.recipe.id }) { item -> RecipeTile(item, today) { onOpen(item.recipe.id) } }
                }
            }
            if (onImport != null) {
                // „+“ mit Menü: neues Rezept, aus einer Webadresse, aus Fotos, aus einer Datei.
                var menu by remember { mutableStateOf(false) }
                Box(Modifier.align(Alignment.BottomEnd).padding(16.dp)) {
                    FloatingActionButton(onClick = { menu = true }) {
                        Icon(painterResource(R.drawable.ic_plus), contentDescription = "Rezept hinzufügen")
                    }
                    DropdownMenu(expanded = menu, onDismissRequest = { menu = false }) {
                        DropdownMenuItem(text = { Text("Neues Rezept") }, onClick = { menu = false; onNew() })
                        DropdownMenuItem(text = { Text("Rezept aus Internetadresse") }, onClick = { menu = false; onFromUrl() })
                        DropdownMenuItem(text = { Text("Rezept aus Foto") }, onClick = { menu = false; onFromPhoto() })
                        DropdownMenuItem(text = { Text("Datei importieren") }, onClick = { menu = false; onImport() })
                    }
                }
            }
        }
    }

    if (showTags) {
        ModalBottomSheet(onDismissRequest = { showTags = false }) {
            Column(
                Modifier.verticalScroll(rememberScrollState()).padding(horizontal = 16.dp, vertical = 8.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                snapshot.tagGroups.forEach { group ->
                    Text(group.name, style = MaterialTheme.typography.titleSmall)
                    FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        group.tags.forEach { tag ->
                            val selected = tag.id in filter.tagIds
                            FilterChip(
                                selected = selected,
                                onClick = {
                                    val ids = if (selected) filter.tagIds - tag.id else filter.tagIds + tag.id
                                    state.value = filter.copy(tagIds = ids)
                                },
                                label = { Text(tag.name) },
                            )
                        }
                    }
                }
                Row {
                    TextButton(onClick = { state.value = filter.copy(tagIds = emptySet()) }) { Text("Zurücksetzen") }
                    TextButton(onClick = { showTags = false }) { Text("Fertig") }
                }
            }
        }
    }
}

@Composable
private fun RecipeTile(item: RecipeItem, today: LocalDate, onClick: () -> Unit) {
    val r = item.recipe
    Card(
        Modifier.fillMaxWidth().clickable(onClick = onClick),
        border = CardDefaults.outlinedCardBorder(),
    ) {
        Box {
            RecipeImage(r.thumbnail, Modifier.fillMaxWidth().aspectRatio(4f / 3f).clip(MaterialTheme.shapes.medium))
            if (r.isFavorite) {
                Icon(
                    painterResource(R.drawable.ic_heart_filled), contentDescription = "Favorit",
                    tint = MaterialTheme.colorScheme.primary, modifier = Modifier.align(Alignment.TopEnd).padding(8.dp),
                )
            }
            if (r.isIllustrated) {
                Text(
                    "Illustration",
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onSurface,
                    modifier = Modifier.align(Alignment.BottomStart).padding(6.dp)
                        .background(MaterialTheme.colorScheme.surface.copy(alpha = 0.85f), MaterialTheme.shapes.small)
                        .padding(horizontal = 6.dp, vertical = 1.dp),
                )
            }
        }
        Column(Modifier.padding(10.dp), verticalArrangement = Arrangement.spacedBy(2.dp)) {
            Text(r.title, style = MaterialTheme.typography.titleSmall, maxLines = 2, overflow = TextOverflow.Ellipsis)
            val time = formatMinutes(r.totalMinutes ?: r.prepMinutes)
            val details = listOfNotNull(time.ifEmpty { null }, if (r.parentRecipeId != null) "Variante" else null)
            if (details.isNotEmpty()) {
                Text(details.joinToString(" · "), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
            Text(planStatus(item.planInfo, today), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
    }
}

@Composable
private fun Message(text: String, onReset: (() -> Unit)? = null) {
    // In einer scrollbaren Spalte, damit Herunterziehen auch hier lädt.
    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(32.dp)) {
        Text(text, color = MaterialTheme.colorScheme.onSurfaceVariant)
        if (onReset != null) TextButton(onClick = onReset) { Text("Filter zurücksetzen") }
    }
}
