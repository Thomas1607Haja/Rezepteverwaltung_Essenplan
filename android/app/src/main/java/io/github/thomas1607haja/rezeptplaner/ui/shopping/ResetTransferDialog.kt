package io.github.thomas1607haja.rezeptplaner.ui.shopping

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Checkbox
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import io.github.thomas1607haja.rezeptplaner.api.ResetEntry
import io.github.thomas1607haja.rezeptplaner.api.ResetPreview
import io.github.thomas1607haja.rezeptplaner.api.ResetResult
import io.github.thomas1607haja.rezeptplaner.data.Outcome
import io.github.thomas1607haja.rezeptplaner.ui.common.app
import io.github.thomas1607haja.rezeptplaner.ui.dayLabel
import kotlinx.coroutines.launch

/** „an „Testliste““; unbekannt = vor Server 0.18 gesendet. */
fun resetTarget(e: ResetEntry): String {
    val names = e.lists.map { "„${it.name}“" } + if (e.unknownList) listOf("unbekannte Liste") else emptyList()
    return if (names.isEmpty()) "keine Artikel gesendet" else "an " + names.joinToString(", ")
}

/** Ergebnis als Zeilen, wie in der Weboberfläche (ResetTransferDialog.svelte). */
fun resetSummary(r: ResetResult, current: String): List<String> = listOfNotNull(
    if (r.entries == 1) "1 Gericht wieder geplant." else "${r.entries} Gerichte wieder geplant.",
    r.removed.takeIf { it.isNotEmpty() }?.let { "Von der alten Liste entfernt: ${it.joinToString(", ")}." },
    r.reduced.takeIf { it.isNotEmpty() }?.let { "Menge verringert (andere Gerichte brauchen sie noch): ${it.joinToString(", ")}." },
    r.notFound.takeIf { it.isNotEmpty() }?.let { "Nicht mehr offen auf der Liste, unverändert: ${it.joinToString(", ")}." },
    r.skipped.takeIf { it.isNotEmpty() }?.let { "Nicht entfernt, Liste unbekannt: ${it.joinToString(", ")}." },
    r.failed.takeIf { it.isNotEmpty() }?.let { "Nicht entfernt (bitte von Hand löschen): ${it.joinToString(", ") { f -> f.name }}." },
    current.takeIf { it.isNotEmpty() }?.let { "Jetzt „An Bring senden“ – es geht an „$it“." },
)

/**
 * Übertragung zurücksetzen: übertragene Gerichte des Einkaufszeitraums
 * ([next]) bzw. nur [entryId] wieder auf „geplant“, z. B. nach dem Senden
 * an die falsche Bring-Liste. [onDone] meldet, dass zurückgesetzt wurde.
 */
@Composable
fun ResetTransferDialog(next: Boolean, entryId: Long?, onDone: () -> Unit, onDismiss: () -> Unit) {
    val app = app()
    val scope = rememberCoroutineScope()
    var preview by remember { mutableStateOf<ResetPreview?>(null) }
    var error by remember { mutableStateOf<String?>(null) }
    var busy by remember { mutableStateOf(false) }
    var result by remember { mutableStateOf<ResetResult?>(null) }
    var selected by remember { mutableStateOf(emptySet<Long>()) }
    var remove by remember { mutableStateOf(true) }
    // Liste für Gerichte ohne gespeicherte Liste; leer = nicht entfernen.
    var fallback by remember { mutableStateOf("") }
    var chooseFallback by remember { mutableStateOf(false) }

    LaunchedEffect(next, entryId) {
        when (val r = app.shoppingRepository.resetPreview(next, entryId)) {
            is Outcome.Error -> error = r.message
            is Outcome.Ok -> {
                preview = r.value
                selected = r.value.entries.map { it.id }.toSet()
            }
        }
    }

    val p = preview
    val done = result
    AlertDialog(
        onDismissRequest = { if (!busy) onDismiss() },
        title = { Text("Übertragung zurücksetzen") },
        text = {
            Column(Modifier.verticalScroll(rememberScrollState()), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                when {
                    done != null -> resetSummary(done, p?.currentList?.name.orEmpty()).forEach { Text(it) }
                    p == null -> Text(error ?: "Lädt …", color = if (error != null) MaterialTheme.colorScheme.error else MaterialTheme.colorScheme.onSurfaceVariant)
                    else -> {
                        Text("Die Gerichte gelten danach wieder als geplant und lassen sich neu an Bring senden. Abgewählte Zutaten („Zutaten vorhanden“) bleiben erhalten.")
                        if (p.entries.isEmpty()) Text("In diesem Zeitraum ist nichts übertragen.", color = MaterialTheme.colorScheme.onSurfaceVariant)
                        p.entries.forEach { e ->
                            val on = e.id in selected
                            Row(
                                Modifier.fillMaxWidth().clickable { selected = if (on) selected - e.id else selected + e.id },
                                verticalAlignment = Alignment.CenterVertically,
                            ) {
                                Checkbox(checked = on, onCheckedChange = { selected = if (on) selected - e.id else selected + e.id })
                                Column {
                                    Text(e.title, fontWeight = FontWeight.SemiBold)
                                    Text(
                                        "${dayLabel(e.date)} · ${resetTarget(e)}" + if (e.items > 0) " · ${e.items} Artikel" else "",
                                        style = MaterialTheme.typography.bodySmall,
                                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                                    )
                                }
                            }
                        }
                        if (p.entries.isNotEmpty()) {
                            Row(Modifier.fillMaxWidth().clickable { remove = !remove }, verticalAlignment = Alignment.CenterVertically) {
                                Checkbox(checked = remove, onCheckedChange = { remove = it })
                                Text("Artikel aus der alten Liste entfernen")
                            }
                            val unknown = p.entries.count { it.id in selected && it.unknownList }
                            if (remove) {
                                Text(
                                    "Was andere, weiter übertragene Gerichte brauchen, bleibt mit kleinerer Menge stehen. Schon abgehakte Artikel bleiben unberührt.",
                                    style = MaterialTheme.typography.bodySmall,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                )
                            }
                            if (remove && unknown > 0) {
                                Text(
                                    (if (unknown == 1) "Für 1 Gericht" else "Für $unknown Gerichte") +
                                        " ist die damalige Liste nicht gespeichert (vor Version 0.18 gesendet). Entfernen aus:",
                                    style = MaterialTheme.typography.bodyMedium,
                                )
                                Box {
                                    OutlinedButton(onClick = { chooseFallback = true }, modifier = Modifier.fillMaxWidth()) {
                                        Text(p.lists.firstOrNull { it.entityId == fallback }?.name ?: "– nicht entfernen –")
                                    }
                                    DropdownMenu(expanded = chooseFallback, onDismissRequest = { chooseFallback = false }) {
                                        DropdownMenuItem(text = { Text("– nicht entfernen –") }, onClick = { fallback = ""; chooseFallback = false })
                                        p.lists.forEach { l ->
                                            DropdownMenuItem(text = { Text(l.name) }, onClick = { fallback = l.entityId; chooseFallback = false })
                                        }
                                    }
                                }
                                if (p.listsError.isNotEmpty()) {
                                    Text("Home Assistant nicht erreichbar – Listen nicht geladen.", color = MaterialTheme.colorScheme.error)
                                }
                            }
                            if (p.currentList.entityId.isNotEmpty()) {
                                Text(
                                    "Neu senden geht danach an „${p.currentList.name}“ (Einstellungen).",
                                    style = MaterialTheme.typography.bodySmall,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                )
                            }
                        }
                        error?.let { Text(it, color = MaterialTheme.colorScheme.error) }
                    }
                }
            }
        },
        confirmButton = {
            when {
                done != null -> TextButton(onClick = onDismiss) { Text("Schließen") }
                p != null && p.entries.isNotEmpty() -> TextButton(
                    onClick = {
                        busy = true
                        error = null
                        scope.launch {
                            when (val r = app.shoppingRepository.reset(p.entries.map { it.id }.filter { it in selected }, remove, fallback)) {
                                is Outcome.Error -> error = r.message
                                is Outcome.Ok -> {
                                    result = r.value
                                    onDone()
                                }
                            }
                            busy = false
                        }
                    },
                    enabled = !busy && selected.isNotEmpty(),
                ) { Text(if (busy) "Setzt zurück …" else "Zurücksetzen (${selected.size})") }
            }
        },
        dismissButton = {
            if (done == null) TextButton(onClick = onDismiss, enabled = !busy) { Text("Abbrechen") }
        },
    )
}
