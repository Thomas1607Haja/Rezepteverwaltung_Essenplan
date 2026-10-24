package io.github.thomas1607haja.rezeptplaner.ui.recipes

import io.github.thomas1607haja.rezeptplaner.ui.wrapTitle
import androidx.compose.foundation.ExperimentalFoundationApi
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.ui.focus.onFocusChanged
import androidx.compose.ui.platform.LocalDensity
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

@OptIn(ExperimentalMaterial3Api::class, ExperimentalLayoutApi::class, ExperimentalFoundationApi::class)
@Composable
fun RecipesScreen(
    onOpen: (Long) -> Unit,
    modifier: Modifier = Modifier,
    /** Gesetzt: Knopf „+“ mit „Datei importieren“ (nicht in der Rezeptauswahl). */
    onImport: (() -> Unit)? = null,
    onNew: () -> Unit = {},
    onFromUrl: () -> Unit = {},
    onFromPhoto: () -> Unit = {},
    onFromQr: () -> Unit = {},
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
    val listState = rememberLazyListState()
    var showTags by remember { mutableStateOf(false) }

    val tabs = remember(snapshot.tagGroups) { courseTabs(snapshot.tagGroups) }

    // Eine einzige scrollbare Liste: Kopfbereich, Suche, Filter und Rezepte
    // scrollen zusammen. Früher stand der Kopf fest darüber und das Raster
    // bekam nur den Restplatz – mit vielen Schnellauswahlen in der
    // Rezeptauswahl blieb für die Rezepte nichts übrig.
    BoxWithConstraints(modifier.fillMaxSize()) {
        // Gang-Reiter und Suche haften oben, solange darunter genug Platz
        // bleibt (nicht im Querformat und nicht bei sehr großer Schrift).
        val sticky = maxHeight >= 440.dp * LocalDensity.current.fontScale.coerceAtLeast(1f)
        // Spalten wie früher GridCells.Adaptive(160.dp): 16 dp Rand, 12 dp Abstand.
        val columns = ((maxWidth - 32.dp + 12.dp) / (160.dp + 12.dp)).toInt().coerceAtLeast(1)
        val rows = remember(items, columns) { items.chunked(columns) }
        PullToRefreshBox(
            isRefreshing = sync.running,
            onRefresh = { scope.launch { app.syncRepository.sync(SyncTrigger.Pull) } },
            modifier = Modifier.fillMaxSize(),
        ) {
            LazyColumn(
                state = listState,
                contentPadding = PaddingValues(bottom = if (onImport != null) 88.dp else 16.dp),
                // imePadding: Die Tastatur verdeckt weder Suchfeld noch Liste.
                modifier = Modifier.fillMaxSize().imePadding(),
            ) {
                item(contentType = "header") {
                    Column {
                        OfflineBanner(sync.log)
                        header()
                    }
                }
                val search: @Composable () -> Unit = {
                    SearchBar(
                        filter, tabs, onChange = { state.value = it },
                        // Beim Tippen die Suche nach oben holen, damit die
                        // Treffer zwischen Suchfeld und Tastatur sichtbar sind.
                        onFocus = { scope.launch { listState.animateScrollToItem(1) } },
                    )
                }
                if (sticky) stickyHeader(key = "search", contentType = "search") { search() }
                else item(key = "search", contentType = "search") { search() }
                item(contentType = "filters") {
                    Filters(filter, snapshot.recipes.any { it.isIllustrated }, onChange = { state.value = it }, onTags = { showTags = true })
                }
                when {
                    snapshot.isEmpty -> item {
                        Message(if (sync.running) "Rezepte werden geladen …" else "Noch keine Daten – nach unten ziehen zum Laden.")
                    }
                    items.isEmpty() -> item {
                        Message(
                            if (filter.isActive) "Keine Rezepte zu dieser Suche." else "Noch keine Rezepte.",
                            onReset = if (filter.isActive) ({ state.value = filter.reset() }) else null,
                        )
                    }
                    else -> {
                        // Trefferzahl wie im Web, über die ganze Breite.
                        item(contentType = "count") {
                            Text(
                                if (items.size == 1) "1 Rezept" else "${items.size} Rezepte",
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                                modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp),
                            )
                        }
                        items(rows, key = { row -> row.first().recipe.id }, contentType = { "recipes" }) { row ->
                            Row(
                                Modifier.padding(horizontal = 16.dp).padding(bottom = 12.dp),
                                horizontalArrangement = Arrangement.spacedBy(12.dp),
                            ) {
                                row.forEach { item ->
                                    Box(Modifier.weight(1f)) { RecipeTile(item, today) { onOpen(item.recipe.id) } }
                                }
                                // Letzte Zeile: leere Plätze auffüllen, damit Kacheln gleich breit bleiben.
                                repeat(columns - row.size) { Spacer(Modifier.weight(1f)) }
                            }
                        }
                    }
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
                        DropdownMenuItem(text = { Text("Rezept aus QR-Code") }, onClick = { menu = false; onFromQr() })
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

/** Gang-Reiter und Suchfeld; haftet beim Scrollen oben, daher mit Hintergrund. */
@Composable
private fun SearchBar(filter: RecipeFilter, tabs: List<CourseTab>, onChange: (RecipeFilter) -> Unit, onFocus: () -> Unit) {
    Column(
        Modifier.fillMaxWidth().background(MaterialTheme.colorScheme.background).padding(horizontal = 16.dp, vertical = 8.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        // Schnellfilter nach Gang (Schritt 9.1), seitlich scrollbar.
        if (tabs.size > 1) {
            Row(Modifier.horizontalScroll(rememberScrollState()), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                tabs.forEach { tab ->
                    FilterChip(
                        selected = filter.course == tab.key,
                        onClick = { onChange(filter.copy(course = tab.key)) },
                        label = { Text(tab.label) },
                    )
                }
            }
        }
        OutlinedTextField(
            value = filter.query,
            onValueChange = { q -> onChange(filter.copy(query = q)) },
            placeholder = { Text("Suchen nach Titel oder Zutat") },
            leadingIcon = { Icon(painterResource(R.drawable.ic_search), contentDescription = null) },
            singleLine = true,
            modifier = Modifier.fillMaxWidth().onFocusChanged { if (it.isFocused) onFocus() },
        )
    }
}

/** Kategorien, Favoriten usw. und die Sortierung; scrollt mit der Liste. */
@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun Filters(filter: RecipeFilter, hasIllustrations: Boolean, onChange: (RecipeFilter) -> Unit, onTags: () -> Unit) {
    var showSort by remember { mutableStateOf(false) }
    // FlowRow: bricht auf schmalen Handys in eine zweite Zeile um.
    FlowRow(
        Modifier.padding(horizontal = 16.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        itemVerticalAlignment = Alignment.CenterVertically,
    ) {
        FilterChip(
            selected = filter.tagIds.isNotEmpty(),
            onClick = onTags,
            label = { Text(if (filter.tagIds.isEmpty()) "Kategorien" else "Kategorien (${filter.tagIds.size})") },
            leadingIcon = { Icon(painterResource(R.drawable.ic_filter), contentDescription = null, Modifier.size(18.dp)) },
        )
        FilterChip(
            selected = filter.favoritesOnly,
            onClick = { onChange(filter.copy(favoritesOnly = !filter.favoritesOnly)) },
            label = { Text("Favoriten") },
            leadingIcon = { Icon(painterResource(R.drawable.ic_heart), contentDescription = null, Modifier.size(18.dp)) },
        )
        FilterChip(
            selected = filter.neverPlanned,
            onClick = { onChange(filter.copy(neverPlanned = !filter.neverPlanned)) },
            label = { Text("Nie eingeplant") },
        )
        // Nur anbieten, wenn es gezeichnete Bilder gibt.
        if (filter.illustrationsOnly || hasIllustrations) {
            FilterChip(
                selected = filter.illustrationsOnly,
                onClick = { onChange(filter.copy(illustrationsOnly = !filter.illustrationsOnly)) },
                label = { Text("Illustrationen") },
            )
        }
        Box {
            TextButton(onClick = { showSort = true }) { Text(filter.sort.label) }
            DropdownMenu(expanded = showSort, onDismissRequest = { showSort = false }) {
                RecipeSort.entries.forEach { sort ->
                    DropdownMenuItem(text = { Text(sort.label) }, onClick = {
                        onChange(filter.copy(sort = sort))
                        showSort = false
                    })
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
            Text(wrapTitle(r.title), style = MaterialTheme.typography.titleSmall, maxLines = 2, overflow = TextOverflow.Ellipsis)
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
    // Steht in der Liste, damit Herunterziehen auch hier lädt.
    Column(Modifier.fillMaxWidth().padding(32.dp)) {
        Text(text, color = MaterialTheme.colorScheme.onSurfaceVariant)
        if (onReset != null) TextButton(onClick = onReset) { Text("Filter zurücksetzen") }
    }
}
