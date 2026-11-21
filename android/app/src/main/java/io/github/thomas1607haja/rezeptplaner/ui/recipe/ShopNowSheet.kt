package io.github.thomas1607haja.rezeptplaner.ui.recipe

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.unit.dp
import io.github.thomas1607haja.rezeptplaner.R
import io.github.thomas1607haja.rezeptplaner.api.Recipe
import io.github.thomas1607haja.rezeptplaner.api.SendResult
import io.github.thomas1607haja.rezeptplaner.api.TodoList
import io.github.thomas1607haja.rezeptplaner.data.Outcome
import io.github.thomas1607haja.rezeptplaner.ui.common.Texts
import io.github.thomas1607haja.rezeptplaner.ui.common.app
import io.github.thomas1607haja.rezeptplaner.ui.common.tr
import io.github.thomas1607haja.rezeptplaner.ui.plan.PlanPreviewSheet
import kotlinx.coroutines.launch

/** Text der Bestätigung nach „Jetzt einkaufen“ (ohne Fehlschläge). */
fun shopNowSummary(result: SendResult, listName: String): List<String> = buildList {
    add(Texts.plural(R.plurals.shop_now_sent, result.sent.size, "list" to listName))
    val merged = result.sent.count { it.merged }
    if (merged > 0) add(Texts.plural(R.plurals.shop_now_merged, merged))
}

/**
 * „Jetzt einkaufen“ (wie ShopNowDialog.svelte im Web): Kurzvorschau mit
 * Portionen und Zutaten, dann „An Bring senden · Liste“ und eine
 * Bestätigung mit „Auch im Essensplan eintragen“. Nur online: Ohne
 * Verbindung zeigt die Vorschau den Hinweis, und Senden bleibt gesperrt.
 */
@Composable
fun ShopNowSheet(recipe: Recipe, familySize: Int, onPlan: (servings: Int) -> Unit, onDismiss: () -> Unit) {
    val app = app()
    val scope = rememberCoroutineScope()
    var list by remember { mutableStateOf<TodoList?>(null) }
    var error by remember { mutableStateOf<String?>(null) }
    var busy by remember { mutableStateOf(false) }
    var result by remember { mutableStateOf<Pair<SendResult, Int>?>(null) }

    LaunchedEffect(Unit) {
        when (val r = app.shoppingRepository.bringList()) {
            is Outcome.Ok -> {
                list = r.value
                if (r.value.entityId.isEmpty()) error = Texts.get(R.string.shop_now_no_list)
            }
            is Outcome.Error -> error = r.message
        }
    }

    val done = result
    if (done == null) {
        val name = list?.takeIf { it.entityId.isNotEmpty() }?.name?.ifEmpty { list?.entityId }
        PlanPreviewSheet(
            recipe = recipe,
            // Vorbelegt wie beim Einplanen: Kuchen & Co. mit der Grundmenge.
            servings = if (recipe.yieldUnit.isNotEmpty()) recipe.baseServings else familySize,
            excluded = emptyList(),
            confirmLabel = if (name != null) tr(R.string.shop_now_send, "list" to name) else tr(R.string.shop_now_send_plain),
            busy = busy,
            error = error,
            shop = true,
            confirmEnabled = name != null,
            onConfirm = { choice ->
                busy = true
                error = null
                scope.launch {
                    when (val r = app.shoppingRepository.sendRecipe(recipe.id, choice.servings, choice.excluded)) {
                        is Outcome.Ok -> result = r.value to choice.servings
                        is Outcome.Error -> error = r.message
                    }
                    busy = false
                }
            },
            onDismiss = onDismiss,
        )
    } else {
        val (sent, servings) = done
        val listName = list?.name?.ifEmpty { null } ?: sent.list
        AlertDialog(
            onDismissRequest = onDismiss,
            title = { Text(tr(R.string.shop_now_title)) },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    shopNowSummary(sent, listName).forEach { Text(it) }
                    if (sent.failed.isNotEmpty()) {
                        Text(
                            tr(R.string.shop_now_failed, "items" to sent.failed.joinToString { it.name }),
                            color = MaterialTheme.colorScheme.error,
                        )
                    }
                    TextButton(onClick = { onPlan(servings) }) { Text(tr(R.string.shop_now_also_plan)) }
                    Text(
                        tr(R.string.shop_now_also_plan_hint),
                        style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            },
            confirmButton = { TextButton(onClick = onDismiss) { Text(tr(R.string.common_close)) } },
        )
    }
}
