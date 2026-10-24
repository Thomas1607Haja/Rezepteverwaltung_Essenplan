package io.github.thomas1607haja.rezeptplaner.ui.plan

import io.github.thomas1607haja.rezeptplaner.ui.wrapTitle
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DatePicker
import androidx.compose.material3.DatePickerDialog
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedIconButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.SegmentedButton
import androidx.compose.material3.SegmentedButtonDefaults
import androidx.compose.material3.SingleChoiceSegmentedButtonRow
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.rememberDatePickerState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import io.github.thomas1607haja.rezeptplaner.R
import io.github.thomas1607haja.rezeptplaner.api.MealPlanEntry
import io.github.thomas1607haja.rezeptplaner.api.MoveWarning
import io.github.thomas1607haja.rezeptplaner.data.Outcome
import io.github.thomas1607haja.rezeptplaner.data.Snapshot
import io.github.thomas1607haja.rezeptplaner.ui.common.LocalSnackbar
import io.github.thomas1607haja.rezeptplaner.ui.common.app
import io.github.thomas1607haja.rezeptplaner.ui.dayLabel
import io.github.thomas1607haja.rezeptplaner.ui.slotLabel
import io.github.thomas1607haja.rezeptplaner.ui.amountText
import kotlinx.coroutines.launch
import java.time.Instant
import java.time.LocalDate
import java.time.ZoneOffset

// Blätter zum Ändern des Essensplans. Alles geht direkt an den Server
// (PlanRepository); ohne Verbindung erscheint eine Meldung.

fun entryTitle(e: MealPlanEntry) = e.recipe?.title ?: e.note

/** Die nächsten [count] Tage ab heute mit ihren Einträgen und Menüs. */
fun upcomingDays(snapshot: Snapshot, today: LocalDate, count: Int): List<PlanDay> {
    val byDate = snapshot.entries.groupBy { it.date }
    val menus = snapshot.menus.groupBy { it.date }
    return (0 until count).map { i ->
        val date = today.plusDays(i.toLong()).toString()
        PlanDay(date, byDate[date].orEmpty(), menus[date].orEmpty())
    }
}

fun itemTitle(item: PlanItem) = when (item) {
    is PlanItem.Single -> entryTitle(item.entry)
    is PlanItem.MenuBlock -> "Menü: ${item.menu.displayTitle}"
}

fun itemDate(item: PlanItem) = when (item) {
    is PlanItem.Single -> item.entry.date
    is PlanItem.MenuBlock -> item.menu.date
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
internal fun SlotChooser(slot: String, onChange: (String) -> Unit) {
    // Kaffee nur, wenn in den Einstellungen eingeschaltet (oder schon gewählt).
    val coffee = app().localStore.snapshot.collectAsStateWithLifecycle().value.settings.coffeeSlot || slot == "coffee"
    val options = buildList {
        add("dinner" to "Abend")
        add("lunch" to "Mittag")
        if (coffee) add("coffee" to "Kaffee")
    }
    SingleChoiceSegmentedButtonRow(Modifier.fillMaxWidth()) {
        options.forEachIndexed { i, (value, label) ->
            SegmentedButton(
                selected = slot == value,
                onClick = { onChange(value) },
                shape = SegmentedButtonDefaults.itemShape(i, options.size),
            ) { Text(label) }
        }
    }
}

/**
 * Liste der Tage zum Auswählen. Ein leerer Tag (im gewählten Slot) wird
 * mit einem Tippen gewählt; bei belegten Tagen gibt es „Dazu“ und je
 * Gericht oder Menü [action] (z. B. „Tauschen“ oder „Ersetzen“; null =
 * kein Knopf).
 */
@Composable
private fun DayChooser(
    days: List<PlanDay>,
    slot: String,
    exclude: String?,
    action: (PlanItem) -> String?,
    onAppend: (String) -> Unit,
    onItem: (PlanItem) -> Unit,
) {
    LazyColumn(Modifier.fillMaxWidth()) {
        items(days, key = { it.date }) { day ->
            val inSlot = planItems(day.entries, day.menus).filter { it.slot == slot && it.key != exclude }
            Column(
                Modifier.fillMaxWidth()
                    .then(if (inSlot.isEmpty()) Modifier.clickable { onAppend(day.date) } else Modifier)
                    .padding(horizontal = 16.dp, vertical = 10.dp),
                verticalArrangement = Arrangement.spacedBy(4.dp),
            ) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text(dayLabel(day.date), fontWeight = FontWeight.SemiBold, modifier = Modifier.weight(1f))
                    if (inSlot.isEmpty()) {
                        Text("frei", color = MaterialTheme.colorScheme.primary)
                    } else {
                        TextButton(onClick = { onAppend(day.date) }) { Text("Dazu") }
                    }
                }
                inSlot.forEach { item ->
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Text(itemTitle(item), modifier = Modifier.weight(1f), color = MaterialTheme.colorScheme.onSurfaceVariant)
                        action(item)?.let { label -> TextButton(onClick = { onItem(item) }) { Text(label) } }
                    }
                }
            }
            HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant)
        }
    }
}

/** „Zum Essensplan hinzufügen“ aus der Kochansicht. */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AddToPlanSheet(recipeId: Long, recipeTitle: String, snapshot: Snapshot, today: LocalDate, onDismiss: () -> Unit) {
    val app = app()
    val snackbar = LocalSnackbar.current
    val scope = rememberCoroutineScope()
    var slot by rememberSaveable { mutableStateOf("dinner") }
    var busy by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    var replacing by remember { mutableStateOf<MealPlanEntry?>(null) }
    // Kurzvorschau: Tag zum Hinzufügen bzw. Eintrag zum Ersetzen
    var previewDate by remember { mutableStateOf<String?>(null) }
    var previewReplace by remember { mutableStateOf<MealPlanEntry?>(null) }
    val recipe = snapshot.recipes.firstOrNull { it.id == recipeId }
    val preview = wantsPreview(snapshot, recipe)

    fun run(date: String, action: suspend () -> String?) {
        busy = true
        scope.launch {
            error = action()
            busy = false
            if (error == null) {
                onDismiss()
                snackbar.showSnackbar("„$recipeTitle“ eingeplant am ${dayLabel(date)}")
            }
        }
    }

    ModalBottomSheet(onDismissRequest = onDismiss) {
        Column(Modifier.navigationBarsPadding(), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Column(Modifier.padding(horizontal = 16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text("Zum Essensplan hinzufügen", style = MaterialTheme.typography.titleMedium)
                SlotChooser(slot) { slot = it }
                error?.let { Text(it, color = MaterialTheme.colorScheme.error) }
                if (busy) CircularProgressIndicator(Modifier.size(24.dp))
            }
            DayChooser(
                days = upcomingDays(snapshot, today, 14), slot = slot, exclude = null,
                // Ersetzen nur bei einzelnen Gerichten; ein Menü ergänzt man im Menü.
                action = { if (it is PlanItem.Single) "Ersetzen" else null },
                onAppend = { date ->
                    when {
                        busy -> {}
                        preview -> { error = null; previewDate = date }
                        else -> run(date) { app.planRepository.add(date, slot, recipeId) }
                    }
                },
                onItem = { item ->
                    val e = (item as PlanItem.Single).entry
                    if (preview) { error = null; previewReplace = e } else replacing = e
                },
            )
        }
    }

    if (recipe != null && (previewDate != null || previewReplace != null)) {
        val target = previewReplace
        PlanPreviewSheet(
            recipe = recipe,
            servings = target?.servings ?: if (recipe.yieldUnit.isNotEmpty()) recipe.baseServings else snapshot.settings.familySize,
            excluded = emptyList(),
            confirmLabel = if (target != null) "Ersetzen" else "Hinzufügen",
            busy = busy,
            error = error,
            onConfirm = { choice ->
                if (target != null) run(target.date) { app.planRepository.replace(target, recipeId, choice) }
                else previewDate?.let { date -> run(date) { app.planRepository.add(date, slot, recipeId, choice) } }
            },
            onDismiss = { previewDate = null; previewReplace = null },
        )
    }

    replacing?.let { e ->
        AlertDialog(
            onDismissRequest = { replacing = null },
            title = { Text("Ersetzen?") },
            text = { Text("„${entryTitle(e)}“ am ${dayLabel(e.date)} durch „$recipeTitle“ ersetzen?") },
            confirmButton = {
                TextButton(onClick = {
                    replacing = null
                    run(e.date) { app.planRepository.replace(e, recipeId) }
                }) { Text("Ersetzen") }
            },
            dismissButton = { TextButton(onClick = { replacing = null }) { Text("Abbrechen") } },
        )
    }
}

/** Langes Drücken auf einen Eintrag: Portionen, Notiz, Status, Favorit, Ersetzen, Verschieben, Entfernen. */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun EntrySheet(
    entry: MealPlanEntry,
    onDismiss: () -> Unit,
    onReplace: () -> Unit,
    onMove: () -> Unit,
    onMenu: (menuId: Long) -> Unit = {},
) {
    val app = app()
    val scope = rememberCoroutineScope()
    var servings by remember(entry.id) { mutableIntStateOf(entry.servings) }
    var note by remember(entry.id) { mutableStateOf(entry.note) }
    var haveIngredients by remember(entry.id) { mutableStateOf(entry.status == "have_ingredients") }
    var busy by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    var confirmDelete by remember { mutableStateOf(false) }
    var favorite by remember(entry.id) { mutableStateOf(entry.recipe?.isFavorite == true) }
    var showPreview by remember { mutableStateOf(false) }
    val snapshot by app.localStore.snapshot.collectAsStateWithLifecycle()
    val recipe = snapshot.recipes.firstOrNull { it.id == entry.recipeId }
    val transferred = entry.status == "transferred"
    val changed = servings != entry.servings || note.trim() != entry.note ||
        haveIngredients != (entry.status == "have_ingredients")

    fun run(action: suspend () -> String?) {
        busy = true
        scope.launch {
            error = action()
            busy = false
            if (error == null) onDismiss()
        }
    }

    ModalBottomSheet(onDismissRequest = onDismiss) {
        Column(
            Modifier.navigationBarsPadding().padding(horizontal = 16.dp).padding(bottom = 16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Column(Modifier.weight(1f)) {
                    Text(wrapTitle(entryTitle(entry)), style = MaterialTheme.typography.titleMedium)
                    Text(
                        slotLabel(entry.slot) + " am ${dayLabel(entry.date)}",
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
                // Favorit direkt im Plan umschalten, wie im Web (EntryDialog).
                val recipeId = entry.recipeId
                if (recipeId != null && entry.recipe != null) {
                    IconButton(
                        onClick = {
                            busy = true
                            scope.launch {
                                error = (app.recipeRepository.setFavorite(recipeId, !favorite) as? Outcome.Error)?.message
                                if (error == null) favorite = !favorite
                                busy = false
                            }
                        },
                        enabled = !busy,
                    ) {
                        Icon(
                            painterResource(if (favorite) R.drawable.ic_heart_filled else R.drawable.ic_heart),
                            contentDescription = if (favorite) "Favorit entfernen" else "Als Favorit merken",
                            tint = MaterialTheme.colorScheme.primary,
                        )
                    }
                }
            }
            if (entry.recipeId != null) {
                Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                    OutlinedIconButton(onClick = { if (servings > 1) servings-- }, enabled = servings > 1) { Text("−") }
                    Text(amountText(servings, entry.recipe?.yieldUnit.orEmpty()), style = MaterialTheme.typography.titleMedium)
                    OutlinedIconButton(onClick = { servings++ }) { Text("+") }
                }
            }
            OutlinedTextField(
                note, { note = it },
                label = { Text(if (entry.recipeId != null) "Notiz" else "Text (z. B. Essen gehen)") },
                modifier = Modifier.fillMaxWidth(),
            )
            if (entry.recipeId != null) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Column(Modifier.weight(1f)) {
                        Text("Zutaten sind im Haus")
                        Text(
                            if (transferred) "Schon an Bring gesendet." else "Wird dann nicht eingekauft.",
                            style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                    Switch(checked = haveIngredients, onCheckedChange = { haveIngredients = it }, enabled = !transferred)
                }
            }
            // „Zutaten vorhanden …“: dieselbe Kurzvorschau wie beim Einplanen.
            if (recipe != null && recipe.ingredients.filter { it.shoppingRelevant }.all { it.itemKey.isNotEmpty() }) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    excludedLabel(entry.excludedIngredients.size)?.let {
                        Text(it, color = MaterialTheme.colorScheme.primary, modifier = Modifier.weight(1f))
                    }
                    TextButton(onClick = { showPreview = true }, enabled = !busy) { Text("Zutaten vorhanden …") }
                }
                if (transferred) {
                    Text(
                        "Schon an Bring gesendet – Änderungen erscheinen im Einkauf als Differenz.",
                        style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }
            error?.let { Text(it, color = MaterialTheme.colorScheme.error) }
            Button(
                onClick = { run { app.planRepository.update(entry, servings, note, haveIngredients) } },
                enabled = changed && !busy && (entry.recipeId != null || note.isNotBlank()),
                modifier = Modifier.fillMaxWidth(),
            ) { Text(if (busy) "Speichert …" else "Speichern") }
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                // Ohne Rezept: „Rezept wählen“ macht daraus ein Gericht (wie im Web).
                OutlinedButton(onClick = onReplace, enabled = !busy) {
                    Text(if (entry.recipeId != null) "Ersetzen" else "Rezept wählen")
                }
                OutlinedButton(onClick = onMove, enabled = !busy) { Text("Verschieben") }
                OutlinedButton(onClick = { confirmDelete = true }, enabled = !busy) {
                    Text("Entfernen", color = MaterialTheme.colorScheme.error)
                }
            }
            // Menüs (Schritt 9.4, Server 0.16)
            val menu = snapshot.menus.firstOrNull { it.id == entry.menuId }
            if (menu != null) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text(
                        "Gang im Menü „${menu.displayTitle}“", Modifier.weight(1f),
                        style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                    TextButton(onClick = { onDismiss(); onMenu(menu.id) }, enabled = !busy) { Text("Menü") }
                    TextButton(onClick = { run { app.planRepository.detach(entry) } }, enabled = !busy) { Text("Aus Menü lösen") }
                }
            } else if (snapshot.menusSupported && entry.menuId == null) {
                TextButton(
                    onClick = {
                        busy = true
                        scope.launch {
                            // Gang des Rezepts (z. B. Kuchen & Torten), sonst Hauptgericht;
                            // ein Text (z. B. „Grillen“) bleibt ohne Gang.
                            val courses = courseTags(snapshot)
                            val main = courses.firstOrNull { it.name.equals("Hauptgericht", ignoreCase = true) }
                            val own = recipe?.tags?.firstOrNull { t -> courses.any { it.id == t.id } }
                            val course = if (entry.recipeId != null) (own ?: main)?.id ?: 0 else 0
                            val (created, err) = app.planRepository.toMenu(entry, course)
                            busy = false
                            error = err
                            if (created != null) { onDismiss(); onMenu(created.id) }
                        }
                    },
                    enabled = !busy,
                ) { Text("Zu einem Menü erweitern") }
            }
        }
    }

    if (showPreview && recipe != null) {
        PlanPreviewSheet(
            recipe = recipe,
            servings = servings,
            excluded = entry.excludedIngredients,
            confirmLabel = "Übernehmen",
            busy = busy,
            error = error,
            onConfirm = { choice -> run { app.planRepository.setChoice(entry, choice) } },
            onDismiss = { showPreview = false },
        )
    }

    if (confirmDelete) {
        AlertDialog(
            onDismissRequest = { confirmDelete = false },
            title = { Text("Entfernen?") },
            text = { Text("„${entryTitle(entry)}“ am ${dayLabel(entry.date)} aus dem Plan entfernen?") },
            confirmButton = {
                TextButton(onClick = { confirmDelete = false; run { app.planRepository.delete(entry) } }) { Text("Entfernen") }
            },
            dismissButton = { TextButton(onClick = { confirmDelete = false }) { Text("Abbrechen") } },
        )
    }
}

/**
 * Verschieben oder Tauschen eines Gerichts oder Menüs ([subject]; ein Menü
 * wandert immer als Block). Meldet der Server Warnungen (z. B. das Gericht
 * rutscht in den Einkaufszeitraum), wird erst nach Bestätigung gespeichert
 * – wie in der Weboberfläche (MoveWarningDialog).
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun MoveSheet(subject: PlanItem, snapshot: Snapshot, today: LocalDate, onDismiss: () -> Unit) {
    val app = app()
    val snackbar = LocalSnackbar.current
    val scope = rememberCoroutineScope()
    var slot by rememberSaveable { mutableStateOf(subject.slot) }
    var busy by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    // Angefragte Verschiebung, die noch bestätigt werden muss.
    var pending by remember { mutableStateOf<Triple<String, PlanItem?, List<MoveWarning>>?>(null) }
    var pickDate by remember { mutableStateOf(false) }
    val title = when (subject) {
        is PlanItem.Single -> "„${entryTitle(subject.entry)}“"
        is PlanItem.MenuBlock -> "Menü „${subject.menu.displayTitle}“"
    }

    /** [sendEntries]: danach die Zutaten dieser Gerichte an Bring senden. */
    fun move(
        date: String,
        target: PlanItem?,
        confirm: Boolean,
        have: List<Long> = emptyList(),
        sendEntries: List<Long> = emptyList(),
    ) {
        busy = true
        scope.launch {
            val swapWith = (target as? PlanItem.Single)?.entry?.id
            val swapWithMenu = (target as? PlanItem.MenuBlock)?.menu?.id
            val (result, err) = when (subject) {
                is PlanItem.Single -> app.planRepository.move(subject.entry, date, slot, swapWith, confirm, have, swapWithMenu)
                is PlanItem.MenuBlock -> app.planRepository.moveMenu(subject.menu, date, slot, swapWith, swapWithMenu, confirm, have)
            }
            error = err
            when {
                err != null -> busy = false
                result != null && !result.applied -> {
                    busy = false
                    pending = Triple(date, target, result.warnings)
                }
                else -> {
                    var message = "$title ist jetzt am ${dayLabel(date)}"
                    if (sendEntries.isNotEmpty()) {
                        message = when (val sent = app.shoppingRepository.sendEntries(sendEntries)) {
                            is Outcome.Error -> "Verschoben, aber Senden an Bring fehlgeschlagen: ${sent.message}"
                            is Outcome.Ok -> "Verschoben, ${sent.value.sent.size} Artikel an Bring gesendet" +
                                if (sent.value.failed.isNotEmpty()) " (${sent.value.failed.size} nicht angekommen, siehe Einkauf)" else ""
                        }
                    }
                    busy = false
                    onDismiss()
                    snackbar.showSnackbar(message)
                }
            }
        }
    }

    ModalBottomSheet(onDismissRequest = onDismiss) {
        Column(Modifier.navigationBarsPadding(), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Column(Modifier.padding(horizontal = 16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text("$title verschieben", style = MaterialTheme.typography.titleMedium)
                SlotChooser(slot) { slot = it }
                // Die Liste zeigt vier Wochen; jedes andere Datum über den Kalender (wie im Web).
                TextButton(onClick = { pickDate = true }, enabled = !busy) { Text("Anderes Datum wählen …") }
                error?.let { Text(it, color = MaterialTheme.colorScheme.error) }
                if (busy) CircularProgressIndicator(Modifier.size(24.dp))
            }
            DayChooser(
                days = upcomingDays(snapshot, today, 28), slot = slot, exclude = subject.key, action = { "Tauschen" },
                onAppend = { date -> if (!busy) move(date, null, confirm = false) },
                onItem = { other -> if (!busy) move(itemDate(other), other, confirm = false) },
            )
        }
    }

    pending?.let { (date, target, warnings) ->
        val canMarkAvailable = warnings.any { it.type == "planned_into_period" }
        AlertDialog(
            onDismissRequest = { pending = null },
            title = { Text("Wirklich verschieben?") },
            text = { Column(verticalArrangement = Arrangement.spacedBy(8.dp)) { warnings.forEach { Text(it.message) } } },
            confirmButton = {
                Column(horizontalAlignment = Alignment.End) {
                    TextButton(onClick = { pending = null; move(date, target, confirm = true) }) { Text("Trotzdem verschieben") }
                    if (canMarkAvailable) {
                        val ids = warnings.filter { it.type == "planned_into_period" }.map { it.entryId }
                        TextButton(onClick = {
                            pending = null
                            move(date, target, confirm = true, have = ids)
                        }) { Text("Zutaten sind vorhanden") }
                        TextButton(onClick = {
                            pending = null
                            move(date, target, confirm = true, sendEntries = ids)
                        }) { Text("Zutaten an Bring senden") }
                    }
                    TextButton(onClick = { pending = null }) { Text("Abbrechen") }
                }
            },
        )
    }

    if (pickDate) {
        // Der DatePicker rechnet in UTC-Millisekunden.
        val state = rememberDatePickerState(
            initialSelectedDateMillis = LocalDate.parse(itemDate(subject)).atStartOfDay(ZoneOffset.UTC).toInstant().toEpochMilli(),
        )
        DatePickerDialog(
            onDismissRequest = { pickDate = false },
            confirmButton = {
                TextButton(
                    onClick = {
                        pickDate = false
                        state.selectedDateMillis?.let { millis ->
                            val date = Instant.ofEpochMilli(millis).atZone(ZoneOffset.UTC).toLocalDate().toString()
                            if (!busy) move(date, null, confirm = false)
                        }
                    },
                    enabled = state.selectedDateMillis != null,
                ) { Text("Verschieben") }
            },
            dismissButton = { TextButton(onClick = { pickDate = false }) { Text("Abbrechen") } },
        ) { DatePicker(state) }
    }
}
