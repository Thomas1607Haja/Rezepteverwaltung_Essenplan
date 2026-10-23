package io.github.thomas1607haja.rezeptplaner.ui.shopping

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyListScope
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.Checkbox
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.SegmentedButton
import androidx.compose.material3.SegmentedButtonDefaults
import androidx.compose.material3.SingleChoiceSegmentedButtonRow
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import io.github.thomas1607haja.rezeptplaner.api.ShoppingItem
import io.github.thomas1607haja.rezeptplaner.ui.common.OfflineBanner
import io.github.thomas1607haja.rezeptplaner.ui.common.app
import io.github.thomas1607haja.rezeptplaner.ui.dayLabel

/**
 * Einkauf (7.8): Liste der eingekauften Woche bzw. des nächsten Einkaufs,
 * Artikel abwählen, an Bring senden. Nur online; gerechnet wird auf dem
 * Server wie in der Weboberfläche.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ShoppingScreen(onOpenRecipe: (Long, Int) -> Unit) {
    val app = app()
    val vm = viewModel { ShoppingViewModel(app.shoppingRepository) }
    val ui by vm.ui.collectAsStateWithLifecycle()
    val sync by app.syncRepository.state.collectAsStateWithLifecycle()
    val preview = ui.preview

    Column(Modifier.fillMaxSize()) {
        OfflineBanner(sync.log)
        SingleChoiceSegmentedButtonRow(Modifier.fillMaxWidth().padding(16.dp, 12.dp, 16.dp, 4.dp)) {
            listOf(false to "Eingekaufte Woche", true to "Nächster Einkauf").forEachIndexed { i, (next, label) ->
                SegmentedButton(
                    selected = ui.next == next,
                    onClick = { vm.switchPeriod(next) },
                    shape = SegmentedButtonDefaults.itemShape(i, 2),
                    label = { Text(label) },
                )
            }
        }
        PullToRefreshBox(
            isRefreshing = ui.loading && preview != null,
            onRefresh = { vm.load(keepResult = true) },
            modifier = Modifier.weight(1f),
        ) {
            LazyColumn(Modifier.fillMaxSize()) {
                when {
                    ui.error != null && preview == null -> item { Message(ui.error!!, error = true) }
                    preview == null -> item { Message("Lädt …") }
                    else -> {
                        item {
                            Text(
                                "${dayLabel(preview.period.from)} bis ${dayLabel(preview.period.to)}",
                                style = MaterialTheme.typography.bodyMedium,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                                modifier = Modifier.padding(horizontal = 16.dp),
                            )
                        }
                        ui.error?.let { item { Message(it, error = true) } }
                        entries(ui, onOpenRecipe)
                        shoppingList(ui, vm)
                        differences(ui, vm)
                    }
                }
            }
        }
        // Senden immer sichtbar, auch bei langen Listen.
        if (preview != null && preview.items.isNotEmpty()) {
            Surface(tonalElevation = 3.dp) {
                Column(Modifier.fillMaxWidth().padding(16.dp, 8.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                    SendStatus(ui, vm)
                    val count = preview.items.size - ui.deselected.size
                    Button(onClick = { vm.send(missing = false) }, enabled = !ui.sending && count > 0, modifier = Modifier.fillMaxWidth()) {
                        Text(if (ui.sending) "Sendet …" else "An Bring senden ($count)")
                    }
                }
            }
        }
    }

    ui.lists?.let { lists ->
        AlertDialog(
            onDismissRequest = vm::closeLists,
            title = { Text("Bring-Liste wählen") },
            text = {
                Column {
                    lists.forEach { list ->
                        Text(
                            list.name,
                            style = MaterialTheme.typography.bodyLarge,
                            modifier = Modifier.fillMaxWidth().clickable { vm.setList(list) }.padding(vertical = 12.dp),
                        )
                    }
                }
            },
            confirmButton = {},
            dismissButton = { TextButton(onClick = vm::closeLists) { Text("Abbrechen") } },
        )
    }
}

/** Gerichte des Zeitraums; nicht eingehende mit Grund. */
private fun LazyListScope.entries(ui: ShoppingUi, onOpenRecipe: (Long, Int) -> Unit) {
    val preview = ui.preview ?: return
    item { Heading("Gerichte") }
    if (preview.entries.isEmpty()) {
        item { Message("Für diesen Zeitraum ist nichts geplant.") }
        return
    }
    items(preview.entries, key = { "e${it.id}" }) { entry ->
        val reason = entryReason(entry, preview.today)
        val muted = if (entry.included) MaterialTheme.colorScheme.onSurface else MaterialTheme.colorScheme.onSurfaceVariant
        Row(
            Modifier.fillMaxWidth()
                .then(if (entry.recipeId != null) Modifier.clickable { onOpenRecipe(entry.recipeId, entry.servings) } else Modifier)
                .padding(horizontal = 16.dp, vertical = 6.dp),
            verticalAlignment = Alignment.Top,
        ) {
            Text(
                dayLabel(entry.date) + if (entry.slot == "lunch") "\nmittags" else "",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.width(76.dp),
            )
            Column(Modifier.weight(1f)) {
                Text(entry.title, color = muted)
                val details = listOfNotNull(
                    if (entry.recipeId != null) "${entry.servings} Port." else null,
                    reason.ifEmpty { null },
                )
                if (details.isNotEmpty()) {
                    Text(details.joinToString(" · "), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
            }
        }
    }
}

/** Die Einkaufsliste mit Häkchen. */
private fun LazyListScope.shoppingList(ui: ShoppingUi, vm: ShoppingViewModel) {
    val preview = ui.preview ?: return
    item {
        Row(Modifier.fillMaxWidth().padding(end = 8.dp), verticalAlignment = Alignment.CenterVertically) {
            Heading("Einkaufsliste", Modifier.weight(1f))
            if (preview.items.isNotEmpty()) {
                TextButton(onClick = { vm.selectAll(true) }) { Text("Alle") }
                TextButton(onClick = { vm.selectAll(false) }) { Text("Keine") }
            }
        }
    }
    ui.result?.let { result ->
        item {
            Column(Modifier.padding(horizontal = 16.dp, vertical = 4.dp)) {
                Text(sentSummary(result), color = MaterialTheme.colorScheme.primary)
                if (result.failed.isNotEmpty()) {
                    Text("Nicht angekommen – später nachsenden oder von Hand eintragen:", color = MaterialTheme.colorScheme.error)
                    result.failed.forEach { Text("• ${it.name}${if (it.spec.isNotEmpty()) " (${it.spec})" else ""}") }
                }
            }
        }
    }
    if (preview.items.isEmpty()) {
        item { Message("Nichts einzukaufen.") }
        return
    }
    item {
        Text(
            "${preview.items.size - ui.deselected.size} von ${preview.items.size} ausgewählt. Abgewählte Artikel gelten nach dem Senden als erledigt.",
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = Modifier.padding(horizontal = 16.dp),
        )
    }
    items(preview.items, key = { "i${it.key}" }) { item ->
        ItemRow(item, checked = item.key !in ui.deselected) { vm.toggle(item.key) }
    }
}

/** Nach dem Senden geändert: Differenz nachsenden, zu viel Gesendetes. */
private fun LazyListScope.differences(ui: ShoppingUi, vm: ShoppingViewModel) {
    val preview = ui.preview ?: return
    if (preview.missing.isEmpty() && preview.surplus.isEmpty()) return
    item {
        HorizontalDivider(Modifier.padding(top = 12.dp))
        Heading("Nach dem Senden geändert")
        Text(
            "Gerichte wurden nach dem Senden geändert, verschoben oder entfernt.",
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = Modifier.padding(horizontal = 16.dp),
        )
    }
    if (preview.missing.isNotEmpty()) {
        item { SubHeading("Fehlt noch auf der Bring-Liste") }
        items(preview.missing, key = { "m${it.key}" }) { item ->
            ItemRow(item, checked = item.key !in ui.deselectedMissing) { vm.toggle(item.key, missing = true) }
        }
        item {
            Column(Modifier.padding(horizontal = 16.dp, vertical = 4.dp)) {
                if (preview.items.isEmpty()) SendStatus(ui, vm)
                Button(onClick = { vm.send(missing = true) }, enabled = !ui.sending) {
                    Text(if (ui.sending) "Sendet …" else "Differenz nachsenden (${preview.missing.size - ui.deselectedMissing.size})")
                }
            }
        }
    }
    if (preview.surplus.isNotEmpty()) {
        item {
            SubHeading("Zu viel gesendet")
            Text(
                "Wird nicht aus Bring gelöscht – vielleicht schon gekauft. Am besten für ein anderes Gericht einplanen.",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.padding(horizontal = 16.dp),
            )
        }
        items(preview.surplus, key = { "s${it.key}" }) { item ->
            Text("${item.name}  ${item.spec}", modifier = Modifier.padding(horizontal = 16.dp, vertical = 4.dp))
        }
    }
}

@Composable
private fun ItemRow(item: ShoppingItem, checked: Boolean, onToggle: () -> Unit) {
    Row(
        Modifier.fillMaxWidth().clickable(onClick = onToggle).padding(start = 4.dp, end = 16.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Checkbox(checked = checked, onCheckedChange = { onToggle() })
        Column(Modifier.weight(1f).padding(vertical = 6.dp)) {
            Row {
                Text(
                    item.name,
                    modifier = Modifier.weight(1f),
                    color = if (checked) MaterialTheme.colorScheme.onSurface else MaterialTheme.colorScheme.onSurfaceVariant,
                )
                Text(item.spec, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
            if (item.recipes.isNotEmpty()) {
                Text(item.recipes.joinToString(", "), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
        }
    }
}

/** Fehler oder Hinweis zur Übertragung; fehlt die Bring-Liste, gleich die Auswahl anbieten. */
@Composable
private fun SendStatus(ui: ShoppingUi, vm: ShoppingViewModel) {
    ui.notice?.let { Text(it, color = MaterialTheme.colorScheme.primary, style = MaterialTheme.typography.bodySmall) }
    ui.sendError?.let { Text(it, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall) }
    if (ui.needsList) TextButton(onClick = vm::chooseList) { Text("Bring-Liste wählen") }
}

@Composable
private fun Heading(text: String, modifier: Modifier = Modifier) {
    Text(text, style = MaterialTheme.typography.titleMedium, modifier = modifier.padding(16.dp, 16.dp, 16.dp, 4.dp))
}

@Composable
private fun SubHeading(text: String) {
    Text(text, style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.SemiBold, modifier = Modifier.padding(16.dp, 12.dp, 16.dp, 4.dp))
}

@Composable
private fun Message(text: String, error: Boolean = false) {
    Text(
        text,
        color = if (error) MaterialTheme.colorScheme.error else MaterialTheme.colorScheme.onSurfaceVariant,
        modifier = Modifier.padding(16.dp),
    )
}
