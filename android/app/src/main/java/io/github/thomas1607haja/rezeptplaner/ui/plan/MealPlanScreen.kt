package io.github.thomas1607haja.rezeptplaner.ui.plan

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
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
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.SmallFloatingActionButton
import androidx.compose.material3.Text
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
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
import java.time.LocalDate

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
fun MealPlanScreen(onOpenRecipe: (recipeId: Long, servings: Int) -> Unit) {
    val app = app()
    val snapshot by app.localStore.snapshot.collectAsStateWithLifecycle()
    val sync by app.syncRepository.state.collectAsStateWithLifecycle()
    val today = remember(snapshot.today) { snapshot.today.takeIf { it.isNotEmpty() }?.let(LocalDate::parse) ?: LocalDate.now() }
    val days = remember(snapshot, today) { planDays(snapshot, today) }
    val todayIndex = days.indexOfFirst { it.date == today.toString() }.coerceAtLeast(0)
    val listState = rememberLazyListState()
    val scope = rememberCoroutineScope()
    val period = snapshot.shoppingPeriod

    // Beim ersten Anzeigen mit dem heutigen Tag oben beginnen.
    LaunchedEffect(days.isNotEmpty()) { if (days.isNotEmpty()) listState.scrollToItem(todayIndex) }

    Column(Modifier.fillMaxSize()) {
        OfflineBanner(sync.log)
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
                        val inPeriod = period != null && day.date >= period.from && day.date <= period.to
                        DayRow(day, isToday = day.date == today.toString(), inPeriod = inPeriod, onOpenRecipe = onOpenRecipe)
                    }
                }
                SmallFloatingActionButton(
                    onClick = { scope.launch { listState.animateScrollToItem(todayIndex) } },
                    modifier = Modifier.align(Alignment.BottomEnd).padding(16.dp),
                ) { Text("Heute", Modifier.padding(horizontal = 12.dp)) }
            }
        }
    }
}

@Composable
private fun DayRow(day: PlanDay, isToday: Boolean, inPeriod: Boolean, onOpenRecipe: (Long, Int) -> Unit) {
    val background = if (inPeriod) MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.45f) else MaterialTheme.colorScheme.background
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
        }
        if (day.entries.isEmpty()) {
            Text("–", color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
        day.entries.forEach { entry -> EntryCard(entry, onOpenRecipe) }
    }
}

@Composable
private fun EntryCard(entry: MealPlanEntry, onOpenRecipe: (Long, Int) -> Unit) {
    val recipeId = entry.recipeId
    Card(
        Modifier.fillMaxWidth().then(if (recipeId != null) Modifier.clickable { onOpenRecipe(recipeId, entry.servings) } else Modifier),
        border = CardDefaults.outlinedCardBorder(),
    ) {
        Row(Modifier.padding(8.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(12.dp)) {
            Box {
                RecipeImage(entry.recipe?.thumbnail.orEmpty(), Modifier.size(56.dp).clip(MaterialTheme.shapes.small))
            }
            Column(Modifier.weight(1f)) {
                val title = entry.recipe?.title ?: entry.note
                Text(
                    (if (entry.slot == "lunch") "Mittag: " else "") + title,
                    style = MaterialTheme.typography.bodyLarge, maxLines = 2, overflow = TextOverflow.Ellipsis,
                )
                val details = buildList {
                    if (entry.recipe != null) add(portions(entry.servings))
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
        }
    }
}
