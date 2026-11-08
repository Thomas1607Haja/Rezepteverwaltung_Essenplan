package io.github.thomas1607haja.rezeptplaner.ui.stats

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.SegmentedButton
import androidx.compose.material3.SegmentedButtonDefaults
import androidx.compose.material3.SingleChoiceSegmentedButtonRow
import androidx.compose.material3.Text
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import io.github.thomas1607haja.rezeptplaner.R
import io.github.thomas1607haja.rezeptplaner.api.StatRecipe
import io.github.thomas1607haja.rezeptplaner.api.Statistics
import io.github.thomas1607haja.rezeptplaner.data.localStatistics
import io.github.thomas1607haja.rezeptplaner.ui.common.RecipeImage
import io.github.thomas1607haja.rezeptplaner.ui.common.Texts
import io.github.thomas1607haja.rezeptplaner.ui.common.app
import io.github.thomas1607haja.rezeptplaner.ui.common.tr
import io.github.thomas1607haja.rezeptplaner.ui.common.trPlural
import io.github.thomas1607haja.rezeptplaner.ui.recipes.RecipeFilter
import kotlinx.coroutines.launch
import java.time.LocalDate
import java.time.format.DateTimeFormatter
import java.time.format.FormatStyle
import java.time.temporal.ChronoUnit

/** Zeiträume wie in der Weboberfläche. */
private val periods = listOf("3m" to R.string.stats_period_3m, "12m" to R.string.stats_period_12m, "all" to R.string.stats_period_all)

/**
 * Statistik (Schritt 8.D), nur lesend. Online rechnet der Server für jeden
 * Zeitraum; ohne Verbindung zeigt die App „gesamt“ aus ihren Daten.
 * [onRecipes] öffnet die Rezeptübersicht mit einem Filter, [onOpen] ein Rezept.
 */
@OptIn(ExperimentalMaterial3Api::class, ExperimentalLayoutApi::class)
@Composable
fun StatisticsScreen(onRecipes: (RecipeFilter) -> Unit, onOpen: (Long) -> Unit) {
    val app = app()
    val snapshot by app.localStore.snapshot.collectAsStateWithLifecycle()
    val scope = rememberCoroutineScope()
    var period by rememberSaveable { mutableStateOf("3m") }
    var server by remember { mutableStateOf<Statistics?>(null) }
    var offline by remember { mutableStateOf(false) }
    var loading by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    // Hochzählen lädt neu (Herunterziehen).
    var reload by remember { mutableIntStateOf(0) }

    LaunchedEffect(period, reload) {
        loading = true
        error = null
        try {
            val result = app.statisticsRepository.load(period)
            offline = result == null
            server = result
        } catch (e: Exception) {
            error = e.message
        }
        loading = false
    }
    // Offline: aus dem Abgleich, nur gesamt.
    val stats = server ?: remember(snapshot) { localStatistics(snapshot) }

    PullToRefreshBox(isRefreshing = loading, onRefresh = { scope.launch { reload++ } }, modifier = Modifier.fillMaxSize()) {
        Column(
            Modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Text(tr(R.string.nav_statistics), style = MaterialTheme.typography.headlineSmall)
            if (offline) {
                Text(
                    tr(R.string.stats_offline_hint),
                    style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.tertiary,
                )
            }
            error?.let { Text(it, color = MaterialTheme.colorScheme.error) }

            // Zwei Kacheln je Zeile, gleich breit.
            FlowRow(
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalArrangement = Arrangement.spacedBy(8.dp),
                maxItemsInEachRow = 2,
            ) {
                val tile = Modifier.weight(1f)
                Tile(tile, stats.recipes, trPlural(R.plurals.stats_recipes, stats.recipes))
                Tile(tile, stats.planned, tr(R.string.stats_planned, "percent" to percent(stats.planned, stats.recipes)))
                Tile(tile, stats.neverPlanned, tr(R.string.stats_never_planned)) { onRecipes(RecipeFilter(neverPlanned = true)) }
                Tile(tile, stats.illustrated, tr(R.string.stats_illustrated)) { onRecipes(RecipeFilter(illustrationsOnly = true)) }
                Tile(tile, stats.favorites, tr(R.string.stats_favorites)) { onRecipes(RecipeFilter(favoritesOnly = true)) }
                Box(tile) // Platzhalter, damit die letzte Kachel nicht die ganze Breite nimmt
            }

            Section(tr(R.string.stats_top_title)) {
                SingleChoiceSegmentedButtonRow(Modifier.fillMaxWidth()) {
                    periods.forEachIndexed { i, (value, label) ->
                        SegmentedButton(
                            selected = stats.period == value || (offline && value == "all"),
                            onClick = { period = value },
                            enabled = !offline || value == "all",
                            shape = SegmentedButtonDefaults.itemShape(i, periods.size),
                        ) { Text(tr(label)) }
                    }
                }
                val entries = stats.from?.let { trPlural(R.plurals.stats_entries_since, stats.entries, "date" to formatDate(it)) }
                    ?: trPlural(R.plurals.stats_entries_total, stats.entries)
                val menus = if (stats.menus > 0) trPlural(R.plurals.stats_menus, stats.menus) else ""
                Text(
                    entries + menus,
                    style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
                if (stats.top.isEmpty()) Muted(tr(R.string.stats_top_empty))
                val max = stats.top.maxOfOrNull { it.count }?.coerceAtLeast(1) ?: 1
                stats.top.forEachIndexed { i, r -> TopRow(i + 1, r, max) { onOpen(r.id) } }
            }

            if (stats.withoutRecipe.isNotEmpty()) {
                Section(tr(R.string.stats_without_recipe_title)) {
                    Muted(tr(R.string.stats_without_recipe_hint, "example" to tr(R.string.quick_eating_out)))
                    stats.withoutRecipe.forEach { Text("${it.count}×  ${it.note}") }
                }
            }

            Section(tr(R.string.stats_favorites_waiting_title)) {
                if (stats.favoritesWaiting.isEmpty()) Muted(tr(R.string.stats_favorites_waiting_empty))
                stats.favoritesWaiting.forEach { r ->
                    LinkRow(r.title, lastMade(r.lastPlanned, stats.today)) { onOpen(r.id) }
                }
            }

            Section(tr(R.string.stats_recent_title)) {
                stats.recentlyAdded.forEach { r ->
                    val detail = tr(
                        R.string.stats_recent_detail,
                        "date" to formatDate(r.createdAt.take(10)),
                        "last" to lastMade(r.lastPlanned, stats.today),
                    )
                    LinkRow(r.title, detail) { onOpen(r.id) }
                }
            }

            Section(tr(R.string.stats_categories_title)) {
                val max = stats.categories.flatMap { g -> g.tags.map { it.count } }.maxOrNull()?.coerceAtLeast(1) ?: 1
                stats.categories.forEach { g ->
                    Text(g.name, style = MaterialTheme.typography.titleSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    g.tags.forEach { t ->
                        BarRow(t.name, t.count, max) { onRecipes(RecipeFilter(tagIds = setOf(t.id))) }
                    }
                }
                Muted(trPlural(R.plurals.stats_without_category, stats.withoutCategory))
            }
        }
    }
}

private fun percent(part: Int, whole: Int) = if (whole == 0) 0 else Math.round(100.0 * part / whole).toInt()

/** Datum in der Sprache der App, z. B. „24.10.2026“ bzw. „Oct 24, 2026“. */
private fun formatDate(date: String) = runCatching {
    LocalDate.parse(date).format(DateTimeFormatter.ofLocalizedDate(FormatStyle.MEDIUM).withLocale(Texts.locale()))
}.getOrDefault(date)

/** „zuletzt vor 3 Wochen“ wie im Web; ohne Datum „noch nie eingeplant“. */
private fun lastMade(date: String?, today: String): String {
    if (date == null) return Texts.get(R.string.stats_last_never)
    val days = runCatching { ChronoUnit.DAYS.between(LocalDate.parse(date), LocalDate.parse(today)) }.getOrDefault(0L)
    return when {
        days == 0L -> Texts.get(R.string.stats_last_today)
        days < 14 -> Texts.plural(R.plurals.stats_last_days, days.toInt())
        days < 60 -> Texts.plural(R.plurals.stats_last_weeks, Math.round(days / 7.0).toInt())
        else -> Texts.plural(R.plurals.stats_last_months, Math.round(days / 30.0).toInt())
    }
}

@Composable
private fun Tile(modifier: Modifier, value: Int, label: String, onClick: (() -> Unit)? = null) {
    Card(
        modifier.then(if (onClick != null) Modifier.clickable(onClick = onClick) else Modifier),
        border = CardDefaults.outlinedCardBorder(),
    ) {
        Column(Modifier.padding(12.dp)) {
            Text("$value", style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.Bold)
            Text(
                label, style = MaterialTheme.typography.bodySmall,
                color = if (onClick != null) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
    }
}

@Composable
private fun Section(title: String, content: @Composable () -> Unit) {
    Card(Modifier.fillMaxWidth(), border = CardDefaults.outlinedCardBorder()) {
        Column(Modifier.padding(12.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
            Text(title, style = MaterialTheme.typography.titleMedium)
            content()
        }
    }
}

@Composable
private fun Muted(text: String) {
    Text(text, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
}

@Composable
private fun TopRow(rank: Int, r: StatRecipe, max: Int, onClick: () -> Unit) {
    Row(
        Modifier.fillMaxWidth().clickable(onClick = onClick).padding(vertical = 2.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Text("$rank.", Modifier.width(28.dp), color = MaterialTheme.colorScheme.onSurfaceVariant)
        RecipeImage(r.thumbnail, Modifier.size(36.dp).clip(MaterialTheme.shapes.small))
        Column(Modifier.weight(1f)) {
            Text(r.title, maxLines = 1, overflow = TextOverflow.Ellipsis, color = MaterialTheme.colorScheme.primary)
            Bar(r.count, max)
        }
        Text("${r.count}×", color = MaterialTheme.colorScheme.onSurfaceVariant)
    }
}

@Composable
private fun BarRow(label: String, count: Int, max: Int, onClick: () -> Unit) {
    Row(
        Modifier.fillMaxWidth().clickable(onClick = onClick).padding(vertical = 2.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Text(label, Modifier.width(130.dp), maxLines = 1, overflow = TextOverflow.Ellipsis, color = MaterialTheme.colorScheme.primary)
        Box(Modifier.weight(1f)) { Bar(count, max) }
        Text("$count", color = MaterialTheme.colorScheme.onSurfaceVariant)
    }
}

/** Balken im Verhältnis [count] / [max]. */
@Composable
private fun Bar(count: Int, max: Int) {
    Box(Modifier.fillMaxWidth().height(8.dp)) {
        if (count > 0) {
            Box(
                Modifier.fillMaxWidth(count.toFloat() / max).fillMaxHeight().clip(MaterialTheme.shapes.extraSmall)
                    .background(MaterialTheme.colorScheme.primary.copy(alpha = 0.75f)),
            )
        }
    }
}

@Composable
private fun LinkRow(title: String, detail: String, onClick: () -> Unit) {
    Column(Modifier.fillMaxWidth().clickable(onClick = onClick).padding(vertical = 4.dp)) {
        Text(title, color = MaterialTheme.colorScheme.primary)
        Muted(detail)
    }
}
