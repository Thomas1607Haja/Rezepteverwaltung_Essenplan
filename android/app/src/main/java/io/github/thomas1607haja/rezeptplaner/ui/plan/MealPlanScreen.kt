package io.github.thomas1607haja.rezeptplaner.ui.plan

import androidx.compose.foundation.background
import androidx.compose.foundation.ExperimentalFoundationApi
import androidx.compose.foundation.combinedClickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.SmallFloatingActionButton
import androidx.compose.material3.Text
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import io.github.thomas1607haja.rezeptplaner.R
import io.github.thomas1607haja.rezeptplaner.api.MealPlanEntry
import io.github.thomas1607haja.rezeptplaner.data.Snapshot
import io.github.thomas1607haja.rezeptplaner.data.SyncTrigger
import io.github.thomas1607haja.rezeptplaner.ui.common.OfflineBanner
import io.github.thomas1607haja.rezeptplaner.ui.common.RecipeImage
import io.github.thomas1607haja.rezeptplaner.ui.common.app
import io.github.thomas1607haja.rezeptplaner.ui.dayLabel
import io.github.thomas1607haja.rezeptplaner.ui.portions
import kotlinx.coroutines.launch
import io.github.thomas1607haja.rezeptplaner.api.Period
import io.github.thomas1607haja.rezeptplaner.ui.theme.nextShoppingColor
import androidx.compose.ui.graphics.Color
import java.time.LocalDate
import java.time.LocalDateTime

/** Ein Tag im Plan mit seinen Einträgen (Mittag vor Abend). */
data class PlanDay(val date: String, val entries: List<MealPlanEntry>)

/**
 * Tage von plan_from bis mindestens zwei Wochen nach heute bzw. bis zum
 * letzten geplanten Tag – auch leere, damit Lücken im Plan sichtbar sind.
 */
fun planDays(snapshot: Snapshot, today: LocalDate): List<PlanDay> {
    if (snapshot.planFrom.isEmpty()) return emptyList()
    val byDate = snapshot.entries.groupBy { it.date }
    var day = LocalDate.parse(snapshot.planFrom)
    val lastPlanned = snapshot.entries.maxOfOrNull { it.date }?.let(LocalDate::parse) ?: today
    val end = maxOf(today.plusDays(13), lastPlanned)
    return buildList {
        while (!day.isAfter(end)) {
            val date = day.toString()
            add(PlanDay(date, byDate[date].orEmpty()))
            day = day.plusDays(1)
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun MealPlanScreen(
    onOpenRecipe: (recipeId: Long, entryId: Long) -> Unit,
    onAdd: (date: String) -> Unit,
    onReplace: (MealPlanEntry) -> Unit,
) {
    val app = app()
    val snapshot by app.localStore.snapshot.collectAsStateWithLifecycle()
    val sync by app.syncRepository.state.collectAsStateWithLifecycle()
    val today = remember(snapshot.today) { snapshot.today.takeIf { it.isNotEmpty() }?.let(LocalDate::parse) ?: LocalDate.now() }
    val days = remember(snapshot, today) { planDays(snapshot, today) }
    val todayIndex = days.indexOfFirst { it.date == today.toString() }.coerceAtLeast(0)
    val listState = rememberLazyListState()
    val scope = rememberCoroutineScope()
    // Eingekaufte Woche und nächster Einkauf, mit der Uhr des Handys
    // berechnet (Wechsel zur Einkaufsuhrzeit auch ohne Verbindung).
    val periods = remember(snapshot.settings, snapshot.today) {
        shoppingPeriods(snapshot.settings, LocalDateTime.now())
    }
    // Geöffnete Blätter: Eintrag bearbeiten bzw. verschieben.
    var editing by remember { mutableStateOf<MealPlanEntry?>(null) }
    var moving by remember { mutableStateOf<MealPlanEntry?>(null) }

    // Beim ersten Anzeigen mit dem heutigen Tag oben beginnen.
    LaunchedEffect(days.isNotEmpty()) { if (days.isNotEmpty()) listState.scrollToItem(todayIndex) }

    Column(Modifier.fillMaxSize()) {
        OfflineBanner(sync.log)
        // Fest über dem Plan, damit sie beim Öffnen auf „heute“ sichtbar ist.
        if (periods != null) Legend(periods)
        PullToRefreshBox(
            isRefreshing = sync.running,
            onRefresh = { scope.launch { app.syncRepository.sync(SyncTrigger.Pull) } },
            modifier = Modifier.fillMaxSize(),
        ) {
            if (days.isEmpty()) {
                Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(32.dp)) {
                    Text(
                        if (sync.running) "Essensplan wird geladen …" else "Noch keine Daten – nach unten ziehen zum Laden.",
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            } else {
                LazyColumn(state = listState, contentPadding = PaddingValues(bottom = 80.dp), modifier = Modifier.fillMaxSize()) {
                    items(days, key = { it.date }) { day ->
                        val shading = when {
                            periods == null -> null
                            day.date >= periods.bought.from && day.date <= periods.bought.to -> Shading.Bought
                            day.date >= periods.next.from && day.date <= periods.next.to -> Shading.Next
                            else -> null
                        }
                        DayRow(
                            day, isToday = day.date == today.toString(), shading = shading,
                            canAdd = day.date >= today.toString(),
                            onOpenRecipe = onOpenRecipe, onAdd = { onAdd(day.date) }, onEdit = { editing = it },
                        )
                    }
                }
                SmallFloatingActionButton(
                    onClick = { scope.launch { listState.animateScrollToItem(todayIndex) } },
                    modifier = Modifier.align(Alignment.BottomEnd).padding(16.dp),
                ) { Text("Heute", Modifier.padding(horizontal = 12.dp)) }
            }
        }
    }

    editing?.let { entry ->
        EntrySheet(
            entry,
            onDismiss = { editing = null },
            onReplace = { editing = null; onReplace(entry) },
            onMove = { editing = null; moving = entry },
        )
    }
    moving?.let { entry -> MoveSheet(entry, snapshot, today, onDismiss = { moving = null }) }
}

private enum class Shading { Bought, Next }

@Composable
private fun Legend(periods: ShoppingPeriods) {
    Column(
        Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp),
        verticalArrangement = Arrangement.spacedBy(4.dp),
    ) {
        LegendItem(MaterialTheme.colorScheme.primaryContainer, "Eingekauft: ${range(periods.bought)}")
        LegendItem(nextShoppingColor(), "Nächster Einkauf: ${range(periods.next)}")
    }
}

@Composable
private fun LegendItem(color: Color, text: String) {
    Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        Box(Modifier.size(14.dp).clip(MaterialTheme.shapes.extraSmall).background(color))
        Text(text, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
    }
}

private fun range(p: Period) = "${dayLabel(p.from)}–${dayLabel(p.to)}"

@Composable
private fun DayRow(
    day: PlanDay,
    isToday: Boolean,
    shading: Shading?,
    canAdd: Boolean,
    onOpenRecipe: (Long, Long) -> Unit,
    onAdd: () -> Unit,
    onEdit: (MealPlanEntry) -> Unit,
) {
    val background = when (shading) {
        Shading.Bought -> MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.45f)
        Shading.Next -> nextShoppingColor()
        null -> MaterialTheme.colorScheme.background
    }
    Column(
        Modifier.fillMaxWidth().background(background).padding(horizontal = 16.dp, vertical = 10.dp),
        verticalArrangement = Arrangement.spacedBy(6.dp),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            Text(
                dayLabel(day.date),
                style = MaterialTheme.typography.titleSmall,
                fontWeight = if (isToday) FontWeight.Bold else FontWeight.Medium,
                color = if (isToday) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onBackground,
            )
            if (isToday) Text("Heute", style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.primary)
            Spacer(Modifier.weight(1f))
            if (canAdd) {
                IconButton(onClick = onAdd, modifier = Modifier.size(32.dp)) {
                    Icon(painterResource(R.drawable.ic_plus), contentDescription = "Gericht für ${dayLabel(day.date)} einplanen")
                }
            }
        }
        if (day.entries.isEmpty()) {
            Text("–", color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
        day.entries.forEach { entry -> EntryCard(entry, onOpenRecipe, onEdit) }
    }
}

@OptIn(ExperimentalFoundationApi::class)
@Composable
private fun EntryCard(entry: MealPlanEntry, onOpenRecipe: (Long, Long) -> Unit, onEdit: (MealPlanEntry) -> Unit) {
    val recipeId = entry.recipeId
    Card(
        // Antippen: Kochansicht; langes Drücken (oder ⋮): bearbeiten.
        Modifier.fillMaxWidth().combinedClickable(
            onClick = { if (recipeId != null) onOpenRecipe(recipeId, entry.id) else onEdit(entry) },
            onLongClick = { onEdit(entry) },
        ),
        border = CardDefaults.outlinedCardBorder(),
    ) {
        Row(Modifier.padding(8.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(12.dp)) {
            if (recipeId != null) {
                RecipeImage(entry.recipe?.thumbnail.orEmpty(), Modifier.size(56.dp).clip(MaterialTheme.shapes.small))
            } else {
                // Eintrag ohne Rezept (z. B. „Essen gehen“): Besteck statt Bild.
                Box(
                    Modifier.size(56.dp).clip(MaterialTheme.shapes.small).background(MaterialTheme.colorScheme.secondaryContainer),
                    contentAlignment = Alignment.Center,
                ) {
                    Icon(
                        painterResource(R.drawable.ic_utensils), contentDescription = "Ohne Rezept",
                        tint = MaterialTheme.colorScheme.onSecondaryContainer,
                    )
                }
            }
            Column(Modifier.weight(1f)) {
                val title = entry.recipe?.title ?: entry.note
                Text(
                    (if (entry.slot == "lunch") "Mittag: " else "") + title,
                    style = MaterialTheme.typography.bodyLarge, maxLines = 2, overflow = TextOverflow.Ellipsis,
                )
                val details = buildList {
                    if (entry.recipe != null) add(portions(entry.servings))
                    excludedLabel(entry.excludedIngredients.size)?.let(::add)
                    if (entry.recipe != null && entry.note.isNotEmpty()) add(entry.note)
                    when (entry.status) {
                        "transferred" -> add("an Bring gesendet")
                        "have_ingredients" -> add("Zutaten im Haus")
                    }
                }
                if (details.isNotEmpty()) {
                    Text(details.joinToString(" · "), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
            }
            if (entry.status != "planned") {
                Icon(painterResource(R.drawable.ic_check), contentDescription = null, tint = MaterialTheme.colorScheme.primary)
            }
            if (entry.recipe?.isFavorite == true) {
                Icon(painterResource(R.drawable.ic_heart_filled), contentDescription = "Favorit", tint = MaterialTheme.colorScheme.primary, modifier = Modifier.size(18.dp))
            }
            IconButton(onClick = { onEdit(entry) }) {
                Icon(painterResource(R.drawable.ic_more), contentDescription = "Eintrag bearbeiten")
            }
        }
    }
}
