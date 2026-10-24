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
import io.github.thomas1607haja.rezeptplaner.api.StatRecipe
import io.github.thomas1607haja.rezeptplaner.api.Statistics
import io.github.thomas1607haja.rezeptplaner.data.localStatistics
import io.github.thomas1607haja.rezeptplaner.ui.common.RecipeImage
import io.github.thomas1607haja.rezeptplaner.ui.common.app
import io.github.thomas1607haja.rezeptplaner.ui.recipes.RecipeFilter
import kotlinx.coroutines.launch
import java.time.LocalDate
import java.time.format.DateTimeFormatter
import java.time.temporal.ChronoUnit

/** Zeiträume wie in der Weboberfläche. */
private val periods = listOf("3m" to "3 Monate", "12m" to "1 Jahr", "all" to "Gesamt")

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
            Text("Statistik", style = MaterialTheme.typography.headlineSmall)
            if (offline) {
                Text(
                    "Ohne Verbindung: Stand des letzten Abgleichs, nur „Gesamt“. Zeitraum und Einträge ohne Rezept gibt es online.",
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
                Tile(tile, stats.recipes, "Rezepte")
                Tile(tile, stats.planned, "schon eingeplant (${percent(stats.planned, stats.recipes)} %)")
                Tile(tile, stats.neverPlanned, "noch nie eingeplant →") { onRecipes(RecipeFilter(neverPlanned = true)) }
                Tile(tile, stats.illustrated, "mit Illustration →") { onRecipes(RecipeFilter(illustrationsOnly = true)) }
                Tile(tile, stats.favorites, "Favoriten →") { onRecipes(RecipeFilter(favoritesOnly = true)) }
                Box(tile) // Platzhalter, damit die letzte Kachel nicht die ganze Breite nimmt
            }

            Section("Am häufigsten gegessen") {
                SingleChoiceSegmentedButtonRow(Modifier.fillMaxWidth()) {
                    periods.forEachIndexed { i, (value, label) ->
                        SegmentedButton(
                            selected = stats.period == value || (offline && value == "all"),
                            onClick = { period = value },
                            enabled = !offline || value == "all",
                            shape = SegmentedButtonDefaults.itemShape(i, periods.size),
                        ) { Text(label) }
                    }
                }
                Text(
                    "${stats.entries} ${if (stats.entries == 1) "Gericht" else "Gerichte"} im Essensplan " +
                        (stats.from?.let { "seit ${formatDate(it)}" } ?: "insgesamt") + " (bis heute)" +
                        when (stats.menus) {
                            0 -> ""
                            1 -> ", davon ein Menü"
                            else -> ", davon ${stats.menus} Menüs"
                        },
                    style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
                if (stats.top.isEmpty()) Muted("In diesem Zeitraum stand noch nichts im Essensplan.")
                val max = stats.top.maxOfOrNull { it.count }?.coerceAtLeast(1) ?: 1
                stats.top.forEachIndexed { i, r -> TopRow(i + 1, r, max) { onOpen(r.id) } }
            }

            if (stats.withoutRecipe.isNotEmpty()) {
                Section("Ohne Rezept") {
                    Muted("Einträge wie „Essen gehen“ im Zeitraum – sie zählen nicht als Rezept.")
                    stats.withoutRecipe.forEach { Text("${it.count}×  ${it.note}") }
                }
            }

            Section("Lieblingsgerichte, die lange nicht dran waren") {
                if (stats.favoritesWaiting.isEmpty()) Muted("Alle Favoriten waren in den letzten 8 Wochen dran oder sind eingeplant.")
                stats.favoritesWaiting.forEach { r ->
                    LinkRow(r.title, "zuletzt ${ago(r.lastPlanned, stats.today)}") { onOpen(r.id) }
                }
            }

            Section("Zuletzt hinzugefügt") {
                stats.recentlyAdded.forEach { r ->
                    val planned = r.lastPlanned?.let { "zuletzt ${ago(it, stats.today)}" } ?: "noch nie eingeplant"
                    LinkRow(r.title, "${formatDate(r.createdAt.take(10))}, $planned") { onOpen(r.id) }
                }
            }

            Section("Rezepte nach Kategorien") {
                val max = stats.categories.flatMap { g -> g.tags.map { it.count } }.maxOrNull()?.coerceAtLeast(1) ?: 1
                stats.categories.forEach { g ->
                    Text(g.name, style = MaterialTheme.typography.titleSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    g.tags.forEach { t ->
                        BarRow(t.name, t.count, max) { onRecipes(RecipeFilter(tagIds = setOf(t.id))) }
                    }
                }
                Muted("${stats.withoutCategory} ${if (stats.withoutCategory == 1) "Rezept" else "Rezepte"} ohne Kategorie")
            }
        }
    }
}

private fun percent(part: Int, whole: Int) = if (whole == 0) 0 else Math.round(100.0 * part / whole).toInt()

private val dateFormat = DateTimeFormatter.ofPattern("dd.MM.yyyy")

private fun formatDate(date: String) = runCatching { LocalDate.parse(date).format(dateFormat) }.getOrDefault(date)

/** „vor 3 Wochen“ wie im Web. */
private fun ago(date: String?, today: String): String {
    if (date == null) return "noch nie"
    val days = runCatching { ChronoUnit.DAYS.between(LocalDate.parse(date), LocalDate.parse(today)) }.getOrDefault(0)
    return when {
        days == 0L -> "heute"
        days < 14 -> "vor $days Tagen"
        days < 60 -> "vor ${Math.round(days / 7.0)} Wochen"
        else -> "vor ${Math.round(days / 30.0)} Monaten"
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
