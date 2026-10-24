package io.github.thomas1607haja.rezeptplaner.ui.plan

import io.github.thomas1607haja.rezeptplaner.ui.wrapTitle
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.Checkbox
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedIconButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.unit.dp
import io.github.thomas1607haja.rezeptplaner.api.Ingredient
import io.github.thomas1607haja.rezeptplaner.api.Recipe
import io.github.thomas1607haja.rezeptplaner.data.PlanChoice
import io.github.thomas1607haja.rezeptplaner.data.Snapshot
import io.github.thomas1607haja.rezeptplaner.ui.formatQuantity
import io.github.thomas1607haja.rezeptplaner.ui.amountText

/**
 * Soll vor dem Einplanen die Kurzvorschau kommen? Abgeschaltet in den
 * Einstellungen der Weboberfläche – oder der Server ist älter als 0.11
 * und liefert keine Zutatenschlüssel (dann ginge die Abwahl ins Leere).
 */
fun wantsPreview(snapshot: Snapshot, recipe: Recipe?): Boolean =
    snapshot.settings.planPreview && recipe != null &&
        recipe.ingredients.filter { it.shoppingRelevant }.all { it.itemKey.isNotEmpty() }

/** „2 Zutaten vorhanden“ für den Hinweis am Eintrag; null = nichts abgewählt. */
fun excludedLabel(count: Int): String? = when (count) {
    0 -> null
    1 -> "1 Zutat vorhanden"
    else -> "$count Zutaten vorhanden"
}

/**
 * Kurzvorschau beim Einplanen (Schritt 8.E), wie PlanPreview.svelte im Web:
 * Portionen (Mengen sofort umgerechnet), einkaufsrelevante Zutaten mit
 * Häkchen – abwählen, was noch im Haus ist –, Vorrat ausgegraut darunter,
 * „Alles vorhanden“ und ein Hauptknopf. Gilt nur für diesen Planeintrag.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PlanPreviewSheet(
    recipe: Recipe,
    servings: Int,
    excluded: List<String>,
    confirmLabel: String,
    busy: Boolean,
    error: String?,
    onConfirm: (PlanChoice) -> Unit,
    onDismiss: () -> Unit,
) {
    var portions by remember(recipe.id) { mutableIntStateOf(servings) }
    var off by remember(recipe.id) { mutableStateOf(excluded.toSet()) }
    var showPantry by remember { mutableStateOf(false) }
    val toBuy = recipe.ingredients.filter { it.shoppingRelevant }
    val pantry = recipe.ingredients.filterNot { it.shoppingRelevant }
    // Gezählt je Schlüssel: zwei Zeilen „Zwiebel“ sind ein Artikel.
    val offCount = toBuy.map { it.itemKey }.distinct().count { it in off }

    fun amount(ing: Ingredient): String {
        val q = ing.quantity?.let { formatQuantity(it * portions / recipe.baseServings) }.orEmpty()
        return listOf(q, ing.unit).filter { it.isNotEmpty() }.joinToString(" ")
    }

    fun confirm(allAvailable: Boolean) {
        val keys = off.filter { k -> toBuy.any { it.itemKey == k } }.sorted()
        onConfirm(PlanChoice(portions, if (allAvailable) emptyList() else keys, allAvailable))
    }

    ModalBottomSheet(onDismissRequest = onDismiss, sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)) {
        Column(Modifier.navigationBarsPadding().padding(horizontal = 16.dp).padding(bottom = 16.dp)) {
            Text(wrapTitle(recipe.title), style = MaterialTheme.typography.titleLarge)
            Row(
                Modifier.padding(vertical = 8.dp),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                OutlinedIconButton(onClick = { if (portions > 1) portions-- }, enabled = portions > 1) { Text("−") }
                Text(amountText(portions, recipe.yieldUnit), style = MaterialTheme.typography.titleMedium)
                OutlinedIconButton(onClick = { if (portions < 50) portions++ }, enabled = portions < 50) { Text("+") }
            }
            Column(Modifier.weight(1f, fill = false).verticalScroll(rememberScrollState())) {
                if (toBuy.isEmpty()) {
                    Text("Keine Zutaten zum Einkaufen.", color = MaterialTheme.colorScheme.onSurfaceVariant)
                } else {
                    Text(
                        "Häkchen entfernen, was du noch hast – gilt nur für diesen Eintrag.",
                        style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
                toBuy.forEach { ing ->
                    val on = ing.itemKey !in off
                    Row(
                        Modifier.fillMaxWidth().clickable { off = if (on) off + ing.itemKey else off - ing.itemKey },
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Checkbox(checked = on, onCheckedChange = { off = if (on) off + ing.itemKey else off - ing.itemKey })
                        val style = if (on) TextDecoration.None else TextDecoration.LineThrough
                        val color = if (on) MaterialTheme.colorScheme.onSurface else MaterialTheme.colorScheme.onSurfaceVariant
                        Text(
                            amount(ing), Modifier.width(80.dp), fontWeight = FontWeight.SemiBold, textAlign = TextAlign.End,
                            textDecoration = style, color = color,
                        )
                        Text(
                            ing.name + if (ing.note.isNotEmpty()) ", ${ing.note}" else "",
                            Modifier.padding(start = 10.dp).weight(1f), textDecoration = style, color = color,
                        )
                    }
                    HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant)
                }
                if (pantry.isNotEmpty()) {
                    TextButton(onClick = { showPantry = !showPantry }) {
                        Text((if (showPantry) "▾ " else "▸ ") + "Aus dem Vorrat (${pantry.size}) – wird nicht eingekauft")
                    }
                    if (showPantry) {
                        pantry.forEach { ing ->
                            Row(Modifier.padding(start = 48.dp, top = 2.dp, bottom = 2.dp)) {
                                Text(amount(ing), Modifier.width(80.dp), textAlign = TextAlign.End, color = MaterialTheme.colorScheme.onSurfaceVariant)
                                Text(ing.name, Modifier.padding(start = 10.dp), color = MaterialTheme.colorScheme.onSurfaceVariant)
                            }
                        }
                    }
                }
            }
            error?.let { Text(it, color = MaterialTheme.colorScheme.error, modifier = Modifier.padding(top = 8.dp)) }
            Row(Modifier.fillMaxWidth().padding(top = 12.dp), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                OutlinedButton(onClick = { confirm(true) }, enabled = !busy, modifier = Modifier.weight(1f)) { Text("Alles vorhanden") }
                Button(onClick = { confirm(false) }, enabled = !busy, modifier = Modifier.weight(1f)) {
                    Text(if (busy) "Speichert …" else confirmLabel + if (offCount > 0) " ($offCount)" else "")
                }
            }
        }
    }
}
