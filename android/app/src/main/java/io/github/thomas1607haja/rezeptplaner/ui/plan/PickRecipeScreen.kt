package io.github.thomas1607haja.rezeptplaner.ui.plan

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.AssistChip
import androidx.compose.material3.Button
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import io.github.thomas1607haja.rezeptplaner.R
import io.github.thomas1607haja.rezeptplaner.api.MealPlanEntry
import io.github.thomas1607haja.rezeptplaner.ui.PickRecipeRoute
import io.github.thomas1607haja.rezeptplaner.ui.common.LocalSnackbar
import io.github.thomas1607haja.rezeptplaner.ui.common.app
import io.github.thomas1607haja.rezeptplaner.ui.dayLabel
import io.github.thomas1607haja.rezeptplaner.ui.recipes.RecipesScreen
import io.github.thomas1607haja.rezeptplaner.ui.recipes.RecipesViewModel
import io.github.thomas1607haja.rezeptplaner.ui.recipes.courseTabs
import kotlinx.coroutines.launch

/** Schnellauswahl für Einträge ohne Rezept – dieselbe wie im Web (RecipePickerPage). */
val quickNotes = listOf("Essen gehen", "Reste", "Pizza bestellen")

/**
 * Die zuletzt verwendeten Texte von Einträgen ohne Rezept (neueste zuerst,
 * ohne die Schnellauswahl). Die App kennt den Plan ab plan_from (14 Tage
 * zurück) und alles Geplante; das reicht für „zuletzt“.
 */
fun recentNotes(entries: List<MealPlanEntry>, limit: Int = 5): List<String> =
    entries.asSequence()
        .filter { it.recipeId == null && it.note.isNotBlank() }
        .sortedByDescending { it.date }
        .map { it.note.trim() }
        .filter { note -> quickNotes.none { it.equals(note, ignoreCase = true) } }
        .distinctBy { it.lowercase() }
        .take(limit)
        .toList()

/**
 * Rezeptauswahl beim Planen: die Rezeptübersicht, Antippen plant ein bzw.
 * ersetzt. Darüber Mittag/Abend und ein Eintrag ohne Rezept (Freitext oder
 * Schnellauswahl), wie in der Weboberfläche.
 */
@OptIn(ExperimentalMaterial3Api::class, ExperimentalLayoutApi::class)
@Composable
fun PickRecipeScreen(route: PickRecipeRoute, onDone: () -> Unit) {
    val app = app()
    val snackbar = LocalSnackbar.current
    val scope = rememberCoroutineScope()
    val snapshot by app.localStore.snapshot.collectAsStateWithLifecycle()
    var busy by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    var slot by rememberSaveable { mutableStateOf(route.slot) }
    var note by rememberSaveable { mutableStateOf("") }
    val replacing = remember(route) { app.localStore.snapshot.value.entries.firstOrNull { it.id == route.replaceEntryId } }
    val recent = remember(snapshot.entries) { recentNotes(snapshot.entries) }
    val pickerFilter = viewModel<RecipesViewModel>(key = "picker").filter
    // Rezept in der Kurzvorschau (Schritt 8.E), null = keine offen
    var previewId by remember { mutableStateOf<Long?>(null) }
    val date = replacing?.date ?: route.date

    fun run(title: String, action: suspend () -> String?) {
        if (busy) return
        busy = true
        scope.launch {
            error = action()
            busy = false
            if (error == null) {
                onDone()
                snackbar.showSnackbar("„$title“ eingeplant am ${dayLabel(date)}")
            }
        }
    }

    fun chooseNote(text: String) {
        val t = text.trim()
        if (t.isEmpty()) return
        run(t) {
            if (replacing != null) app.planRepository.replaceWithNote(replacing, t)
            else app.planRepository.addNote(route.date, slot, t)
        }
    }

    Column(Modifier.fillMaxSize()) {
        TopAppBar(
            title = {
                Text(if (replacing != null) "„${entryTitle(replacing)}“ ersetzen" else "Gericht für ${dayLabel(route.date)}")
            },
            navigationIcon = {
                IconButton(onClick = onDone) { Icon(painterResource(R.drawable.ic_back), contentDescription = "Zurück") }
            },
            windowInsets = WindowInsets(0),
        )
        if (busy) LinearProgressIndicator(Modifier.fillMaxWidth())
        error?.let { Text(it, color = MaterialTheme.colorScheme.error, modifier = Modifier.padding(horizontal = 16.dp)) }
        RecipesScreen(
            onOpen = { recipeId ->
                val recipe = snapshot.recipes.firstOrNull { it.id == recipeId }
                if (wantsPreview(snapshot, recipe)) {
                    error = null
                    previewId = recipeId
                } else {
                    run(recipe?.title.orEmpty()) {
                        if (replacing != null) app.planRepository.replace(replacing, recipeId)
                        else app.planRepository.add(route.date, slot, recipeId)
                    }
                }
            },
            // Eigener Filter, damit die Suche der Rezeptübersicht unberührt bleibt.
            filterState = pickerFilter,
            header = {
                Column(
                    Modifier.padding(horizontal = 16.dp).padding(top = 8.dp),
                    verticalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    // Beim Ersetzen bleibt der Eintrag, wo er ist.
                    if (replacing == null) {
                        SlotChooser(slot) {
                            slot = it
                            // Kaffee & Kuchen: gleich die Backrezepte zeigen.
                            if (it == "coffee" && courseTabs(snapshot.tagGroups).any { t -> t.key == "baking" }) {
                                pickerFilter.value = pickerFilter.value.copy(course = "baking")
                            }
                        }
                    }
                    Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        OutlinedTextField(
                            note, { if (it.length <= 200) note = it },
                            placeholder = { Text("Ohne Rezept, z. B. Essen gehen") },
                            singleLine = true,
                            keyboardOptions = KeyboardOptions(imeAction = ImeAction.Done),
                            keyboardActions = KeyboardActions(onDone = { chooseNote(note) }),
                            modifier = Modifier.weight(1f),
                        )
                        Button(onClick = { chooseNote(note) }, enabled = note.isNotBlank() && !busy) { Text("Eintragen") }
                    }
                    FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        (quickNotes + recent).forEach { text ->
                            AssistChip(onClick = { chooseNote(text) }, label = { Text(text) }, enabled = !busy)
                        }
                    }
                    HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant)
                }
            },
            modifier = Modifier.weight(1f),
        )
    }

    previewId?.let { id ->
        val recipe = snapshot.recipes.firstOrNull { it.id == id } ?: return@let
        PlanPreviewSheet(
            recipe = recipe,
            servings = replacing?.servings ?: if (recipe.yieldUnit.isNotEmpty()) recipe.baseServings else snapshot.settings.familySize,
            excluded = emptyList(),
            confirmLabel = if (replacing != null) "Ersetzen" else "Hinzufügen",
            busy = busy,
            error = error,
            onConfirm = { choice ->
                run(recipe.title) {
                    if (replacing != null) app.planRepository.replace(replacing, id, choice)
                    else app.planRepository.add(route.date, slot, id, choice)
                }
            },
            onDismiss = { previewId = null },
        )
    }
}
