package io.github.thomas1607haja.rezeptplaner.ui.plan

import androidx.compose.foundation.ExperimentalFoundationApi
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.combinedClickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.Checkbox
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedIconButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.font.FontStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import io.github.thomas1607haja.rezeptplaner.R
import io.github.thomas1607haja.rezeptplaner.api.Ingredient
import io.github.thomas1607haja.rezeptplaner.api.MealPlanEntry
import io.github.thomas1607haja.rezeptplaner.api.Menu
import io.github.thomas1607haja.rezeptplaner.api.Recipe
import io.github.thomas1607haja.rezeptplaner.api.Tag
import io.github.thomas1607haja.rezeptplaner.data.PlanChoice
import io.github.thomas1607haja.rezeptplaner.data.Snapshot
import io.github.thomas1607haja.rezeptplaner.ui.amountText
import io.github.thomas1607haja.rezeptplaner.ui.common.LocalSnackbar
import io.github.thomas1607haja.rezeptplaner.ui.common.RecipeImage
import io.github.thomas1607haja.rezeptplaner.ui.common.app
import io.github.thomas1607haja.rezeptplaner.ui.dayLabel
import io.github.thomas1607haja.rezeptplaner.ui.formatQuantity
import io.github.thomas1607haja.rezeptplaner.ui.slotLabel
import io.github.thomas1607haja.rezeptplaner.ui.slotOrder
import io.github.thomas1607haja.rezeptplaner.ui.slotPrefix
import kotlinx.coroutines.launch

// Menüs im Essensplan (Schritt 9.4), wie MenuCard/MenuDialog/MenuPreview/
// MenuPage im Web. Ein Menü ist eine Klammer um normale Einträge; es
// wandert immer als Block, ein einzelner Gang lässt sich lösen.

/** Ein Eintrag im Plan: einzelnes Gericht oder Menü mit seinen Gängen. */
sealed interface PlanItem {
    val slot: String
    val position: Int
    val key: String

    data class Single(val entry: MealPlanEntry) : PlanItem {
        override val slot get() = entry.slot
        override val position get() = entry.position
        override val key get() = "e${entry.id}"
    }

    data class MenuBlock(val menu: Menu, val courses: List<MealPlanEntry>) : PlanItem {
        override val slot get() = menu.slot
        override val position get() = menu.position
        override val key get() = "m${menu.id}"
    }
}

/**
 * Einträge und Menüs eines Tages in Anzeigereihenfolge. Gänge stecken in
 * ihrem Menü; fehlt das Menü (noch nicht abgeglichen), steht der Gang einzeln.
 */
fun planItems(entries: List<MealPlanEntry>, menus: List<Menu>): List<PlanItem> {
    val ids = menus.map { it.id }.toSet()
    val blocks = menus.map { m -> PlanItem.MenuBlock(m, entries.filter { it.menuId == m.id }.sortedBy { it.position }) }
    val singles = entries.filter { it.menuId == null || it.menuId !in ids }.map { PlanItem.Single(it) }
    return (blocks + singles).sortedWith(compareBy({ slotOrder(it.slot) }, { it.position }))
}

/** Die Kategorien der Gang-Gruppe (Vorspeise, Hauptgericht, …). */
fun courseTags(snapshot: Snapshot): List<Tag> = snapshot.tagGroups.firstOrNull { it.role == "course" }?.tags.orEmpty()

fun courseName(courses: List<Tag>, id: Long?): String = courses.firstOrNull { it.id == id }?.name.orEmpty()

fun courseCount(n: Int) = if (n == 1) "1 Gang" else "$n Gänge"

/** Karte eines Menüs im Plan: Titel, Gänge darunter; Antippen öffnet das Menü. */
@OptIn(ExperimentalFoundationApi::class)
@Composable
fun MenuCard(
    block: PlanItem.MenuBlock,
    courses: List<Tag>,
    onOpen: () -> Unit,
    onOpenCourse: (MealPlanEntry) -> Unit,
) {
    val menu = block.menu
    Card(
        Modifier.fillMaxWidth().combinedClickable(onClick = onOpen, onLongClick = onOpen),
        border = CardDefaults.outlinedCardBorder(),
        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceContainerLow),
    ) {
        Column(Modifier.padding(start = 12.dp, top = 4.dp, bottom = 8.dp, end = 4.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Column(Modifier.weight(1f)) {
                    Text(
                        "MENÜ", style = MaterialTheme.typography.labelSmall,
                        color = MaterialTheme.colorScheme.primary, fontWeight = FontWeight.Bold,
                    )
                    Text(
                        slotPrefix(menu.slot) + menu.displayTitle,
                        style = MaterialTheme.typography.bodyLarge, fontWeight = FontWeight.SemiBold,
                        maxLines = 2, overflow = TextOverflow.Ellipsis,
                    )
                    Text(
                        "${courseCount(block.courses.size)} · ${if (menu.servings == 1) "1 Person" else "${menu.servings} Personen"}",
                        style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
                IconButton(onClick = onOpen) { Icon(painterResource(R.drawable.ic_more), contentDescription = "Menü bearbeiten") }
            }
            block.courses.forEach { e ->
                Row(
                    Modifier.fillMaxWidth().clip(MaterialTheme.shapes.small).clickable { onOpenCourse(e) },
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(10.dp),
                ) {
                    CourseImage(e, 40)
                    Column(Modifier.weight(1f)) {
                        courseName(courses, e.courseTagId).takeIf { it.isNotEmpty() }?.let {
                            Text(it, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.primary)
                        }
                        Text(entryTitle(e), style = MaterialTheme.typography.bodyMedium, maxLines = 1, overflow = TextOverflow.Ellipsis)
                    }
                    if (e.status != "planned") {
                        Icon(painterResource(R.drawable.ic_check), contentDescription = null, tint = MaterialTheme.colorScheme.primary)
                    }
                }
            }
            if (block.courses.isEmpty()) {
                Text("Noch keine Gänge", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
            if (menu.note.isNotEmpty()) {
                Text(
                    menu.note, style = MaterialTheme.typography.bodySmall, fontStyle = FontStyle.Italic,
                    color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 2, overflow = TextOverflow.Ellipsis,
                )
            }
        }
    }
}

@Composable
private fun CourseImage(e: MealPlanEntry, size: Int) {
    if (e.recipeId != null) {
        RecipeImage(e.recipe?.thumbnail.orEmpty(), Modifier.size(size.dp).clip(MaterialTheme.shapes.small))
    } else {
        Box(
            Modifier.size(size.dp).clip(MaterialTheme.shapes.small).background(MaterialTheme.colorScheme.secondaryContainer),
            contentAlignment = Alignment.Center,
        ) {
            Icon(painterResource(R.drawable.ic_utensils), contentDescription = "Ohne Rezept", tint = MaterialTheme.colorScheme.onSecondaryContainer)
        }
    }
}

/** Auswahl des Gangs (Vorspeise, Hauptgericht, …) als kleines Aufklappmenü. */
@Composable
fun CourseChooser(current: Long?, courses: List<Tag>, enabled: Boolean, onChoose: (Long) -> Unit) {
    var open by remember { mutableStateOf(false) }
    Box {
        TextButton(onClick = { open = true }, enabled = enabled) {
            Text(courseName(courses, current).ifEmpty { "Gang wählen" } + " ▾", style = MaterialTheme.typography.labelMedium)
        }
        DropdownMenu(expanded = open, onDismissRequest = { open = false }) {
            DropdownMenuItem(text = { Text("– kein Gang –") }, onClick = { open = false; onChoose(0) })
            courses.forEach { t -> DropdownMenuItem(text = { Text(t.name) }, onClick = { open = false; onChoose(t.id) }) }
        }
    }
}

/**
 * Menü bearbeiten: Titel, Notiz, Personen (auf alle Gänge übertragen),
 * Gänge (Gang ändern, lösen, hinzufügen), Zutaten vorhanden, Kochansicht,
 * verschieben, auflösen, löschen.
 */
@OptIn(ExperimentalMaterial3Api::class, ExperimentalLayoutApi::class)
@Composable
fun MenuSheet(
    menuId: Long,
    onDismiss: () -> Unit,
    onAddCourse: (Menu) -> Unit,
    onMove: (PlanItem.MenuBlock) -> Unit,
    onCook: (Menu) -> Unit,
    onOpenCourse: (MealPlanEntry) -> Unit,
) {
    val app = app()
    val scope = rememberCoroutineScope()
    val snackbar = LocalSnackbar.current
    val snapshot by app.localStore.snapshot.collectAsStateWithLifecycle()
    val menu = snapshot.menus.firstOrNull { it.id == menuId }
    if (menu == null) {
        // Gelöscht (auch auf einem anderen Gerät): Blatt schließen.
        LaunchedEffect(Unit) { onDismiss() }
        return
    }
    val entries = snapshot.entries.filter { it.menuId == menuId }.sortedBy { it.position }
    val courses = courseTags(snapshot)
    var title by remember(menuId) { mutableStateOf(menu.title) }
    var note by remember(menuId) { mutableStateOf(menu.note) }
    var servings by remember(menuId) { mutableIntStateOf(menu.servings) }
    var applyServings by remember { mutableStateOf(true) }
    var busy by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    var confirm by remember { mutableStateOf<String?>(null) } // "dissolve" oder "delete"
    var showPreview by remember { mutableStateOf(false) }
    val changed = title.trim() != menu.title || note.trim() != menu.note || servings != menu.servings
    val withRecipes = entries.mapNotNull { e -> snapshot.recipes.firstOrNull { it.id == e.recipeId }?.let { e to it } }
    val canPreview = withRecipes.isNotEmpty() &&
        withRecipes.all { (_, r) -> r.ingredients.filter { it.shoppingRelevant }.all { it.itemKey.isNotEmpty() } }

    fun run(close: Boolean = false, message: String? = null, action: suspend () -> String?) {
        busy = true
        scope.launch {
            error = action()
            busy = false
            if (error == null) {
                showPreview = false
                if (close) onDismiss()
                message?.let { snackbar.showSnackbar(it) }
            }
        }
    }

    ModalBottomSheet(onDismissRequest = onDismiss, sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)) {
        Column(
            Modifier.navigationBarsPadding().padding(horizontal = 16.dp).padding(bottom = 16.dp).verticalScroll(rememberScrollState()),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            Column {
                Text("Menü: ${menu.displayTitle}", style = MaterialTheme.typography.titleMedium)
                Text("${slotLabel(menu.slot)} am ${dayLabel(menu.date)}", color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
            OutlinedTextField(
                title, { if (it.length <= 200) title = it },
                label = { Text("Titel") }, placeholder = { Text("${menu.displayTitle} (automatisch)") },
                singleLine = true, modifier = Modifier.fillMaxWidth(),
            )
            OutlinedTextField(
                note, { if (it.length <= 2000) note = it },
                label = { Text("Notiz") }, placeholder = { Text("z. B. Suppe vorher kochen") },
                modifier = Modifier.fillMaxWidth(),
            )
            Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                OutlinedIconButton(onClick = { if (servings > 1) servings-- }, enabled = servings > 1) { Text("−") }
                Text(if (servings == 1) "1 Person" else "$servings Personen", style = MaterialTheme.typography.titleMedium)
                OutlinedIconButton(onClick = { if (servings < 50) servings++ }, enabled = servings < 50) { Text("+") }
            }
            if (servings != menu.servings) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text("Auf alle Gänge übertragen", Modifier.weight(1f))
                    Switch(checked = applyServings, onCheckedChange = { applyServings = it })
                }
            }
            if (changed) {
                Button(
                    onClick = { run { app.planRepository.updateMenu(menu, title, note, servings, applyServings && servings != menu.servings) } },
                    enabled = !busy, modifier = Modifier.fillMaxWidth(),
                ) { Text(if (busy) "Speichert …" else "Speichern") }
            }

            HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant)
            Text("Gänge", style = MaterialTheme.typography.titleSmall)
            if (entries.isEmpty()) Text("Noch keine Gänge.", color = MaterialTheme.colorScheme.onSurfaceVariant)
            entries.forEach { e ->
                Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    CourseImage(e, 40)
                    Column(Modifier.weight(1f)) {
                        Text(
                            entryTitle(e), maxLines = 2, overflow = TextOverflow.Ellipsis,
                            modifier = Modifier.clickable { onOpenCourse(e) },
                            color = if (e.recipeId != null) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurface,
                        )
                        val details = buildList {
                            add(if (e.recipe != null) amountText(e.servings, e.recipe.yieldUnit) else "ohne Rezept")
                            excludedLabel(e.excludedIngredients.size)?.let(::add)
                            when (e.status) {
                                "transferred" -> add("an Bring")
                                "have_ingredients" -> add("Zutaten im Haus")
                            }
                        }
                        Text(details.joinToString(" · "), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                        CourseChooser(e.courseTagId, courses, enabled = !busy) { id -> run { app.planRepository.setCourse(e, id) } }
                    }
                    TextButton(
                        onClick = { run(message = "„${entryTitle(e)}“ steht jetzt einzeln im Plan") { app.planRepository.detach(e) } },
                        enabled = !busy,
                    ) { Text("Lösen") }
                }
            }
            FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Button(onClick = { onAddCourse(menu) }, enabled = !busy) { Text("+ Gang hinzufügen") }
                if (canPreview) OutlinedButton(onClick = { error = null; showPreview = true }, enabled = !busy) { Text("Zutaten vorhanden …") }
                if (entries.isNotEmpty()) OutlinedButton(onClick = { onCook(menu) }) { Text("Kochansicht") }
            }
            error?.let { Text(it, color = MaterialTheme.colorScheme.error) }
            HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant)
            FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                OutlinedButton(onClick = { onMove(PlanItem.MenuBlock(menu, entries)) }, enabled = !busy) { Text("Verschieben") }
                OutlinedButton(onClick = { confirm = "dissolve" }, enabled = !busy) { Text("Auflösen") }
                OutlinedButton(onClick = { confirm = "delete" }, enabled = !busy) {
                    Text("Löschen", color = MaterialTheme.colorScheme.error)
                }
            }
        }
    }

    if (showPreview) {
        MenuPreviewSheet(
            courses = withRecipes,
            names = courses,
            busy = busy,
            error = error,
            onConfirm = { choices -> run { app.planRepository.setMenuChoices(choices) } },
            onDismiss = { showPreview = false },
        )
    }

    confirm?.let { kind ->
        val delete = kind == "delete"
        AlertDialog(
            onDismissRequest = { confirm = null },
            title = { Text(if (delete) "Menü löschen?" else "Menü auflösen?") },
            text = {
                Text(
                    if (delete) "„${menu.displayTitle}“ mit ${if (entries.size == 1) "seinem Gang" else "allen ${entries.size} Gängen"} aus dem Plan entfernen?"
                    else "Die Gänge bleiben als einzelne Gerichte im Plan.",
                )
            },
            confirmButton = {
                TextButton(onClick = {
                    confirm = null
                    run(close = true) { app.planRepository.deleteMenu(menu, withEntries = delete) }
                }) { Text(if (delete) "Löschen" else "Auflösen") }
            },
            dismissButton = { TextButton(onClick = { confirm = null }) { Text("Abbrechen") } },
        )
    }
}

/**
 * „Zutaten vorhanden“ fürs ganze Menü: ein Abschnitt je Gang mit Häkchen
 * wie in [PlanPreviewSheet]. Gespeichert wird je Gang.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun MenuPreviewSheet(
    courses: List<Pair<MealPlanEntry, Recipe>>,
    names: List<Tag>,
    busy: Boolean,
    error: String?,
    onConfirm: (Map<MealPlanEntry, PlanChoice>) -> Unit,
    onDismiss: () -> Unit,
) {
    var off by remember { mutableStateOf(courses.associate { (e, _) -> e.id to e.excludedIngredients.toSet() }) }

    fun amount(ing: Ingredient, e: MealPlanEntry, r: Recipe): String {
        val q = ing.quantity?.let { formatQuantity(it * e.servings / r.baseServings) }.orEmpty()
        return listOf(q, ing.unit).filter { it.isNotEmpty() }.joinToString(" ")
    }

    fun confirm(allAvailable: Boolean) {
        onConfirm(
            courses.associate { (e, r) ->
                val keys = off[e.id].orEmpty().filter { k -> r.ingredients.any { it.shoppingRelevant && it.itemKey == k } }.sorted()
                e to PlanChoice(e.servings, if (allAvailable) emptyList() else keys, allAvailable)
            },
        )
    }

    ModalBottomSheet(onDismissRequest = onDismiss, sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)) {
        Column(Modifier.navigationBarsPadding().padding(horizontal = 16.dp).padding(bottom = 16.dp)) {
            Text("Zutaten vorhanden – ganzes Menü", style = MaterialTheme.typography.titleLarge)
            Text(
                "Häkchen entfernen, was du noch hast. Gilt nur für dieses Menü.",
                style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Column(Modifier.weight(1f, fill = false).verticalScroll(rememberScrollState())) {
                courses.forEach { (e, r) ->
                    val label = courseName(names, e.courseTagId)
                    Text(
                        (if (label.isNotEmpty()) "$label: " else "") + r.title + " (${amountText(e.servings, r.yieldUnit)})",
                        style = MaterialTheme.typography.titleSmall, modifier = Modifier.padding(top = 12.dp, bottom = 4.dp),
                    )
                    val toBuy = r.ingredients.filter { it.shoppingRelevant }
                    if (toBuy.isEmpty()) Text("Keine Zutaten zum Einkaufen.", color = MaterialTheme.colorScheme.onSurfaceVariant)
                    toBuy.forEach { ing ->
                        val mine = off[e.id].orEmpty()
                        val on = ing.itemKey !in mine
                        val toggle = { off = off + (e.id to if (on) mine + ing.itemKey else mine - ing.itemKey) }
                        Row(Modifier.fillMaxWidth().clickable { toggle() }, verticalAlignment = Alignment.CenterVertically) {
                            Checkbox(checked = on, onCheckedChange = { toggle() })
                            val style = if (on) TextDecoration.None else TextDecoration.LineThrough
                            val color = if (on) MaterialTheme.colorScheme.onSurface else MaterialTheme.colorScheme.onSurfaceVariant
                            Text(
                                amount(ing, e, r), Modifier.width(80.dp), fontWeight = FontWeight.SemiBold, textAlign = TextAlign.End,
                                textDecoration = style, color = color,
                            )
                            Text(ing.name, Modifier.padding(start = 10.dp).weight(1f), textDecoration = style, color = color)
                        }
                    }
                    HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant)
                }
            }
            error?.let { Text(it, color = MaterialTheme.colorScheme.error, modifier = Modifier.padding(top = 8.dp)) }
            Row(Modifier.fillMaxWidth().padding(top = 12.dp), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                OutlinedButton(onClick = { confirm(true) }, enabled = !busy, modifier = Modifier.weight(1f)) { Text("Alles vorhanden") }
                Button(onClick = { confirm(false) }, enabled = !busy, modifier = Modifier.weight(1f)) {
                    Text(if (busy) "Speichert …" else "Übernehmen")
                }
            }
        }
    }
}

/** Kochansicht eines Menüs: Notiz, Personen, alle Gänge; jeder öffnet seine Kochansicht. */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun MenuScreen(menuId: Long, onBack: () -> Unit, onOpenCourse: (recipeId: Long, entryId: Long) -> Unit) {
    val app = app()
    val snapshot by app.localStore.snapshot.collectAsStateWithLifecycle()
    val menu = snapshot.menus.firstOrNull { it.id == menuId }
    val entries = snapshot.entries.filter { it.menuId == menuId }.sortedBy { it.position }
    val courses = courseTags(snapshot)
    Column(Modifier.fillMaxSize()) {
        TopAppBar(
            title = { Text(menu?.let { "Menü: ${it.displayTitle}" } ?: "Menü", maxLines = 1, overflow = TextOverflow.Ellipsis) },
            navigationIcon = { IconButton(onClick = onBack) { Icon(painterResource(R.drawable.ic_back), contentDescription = "Zurück") } },
            windowInsets = WindowInsets(0),
        )
        if (menu == null) {
            Text("Dieses Menü gibt es nicht mehr.", Modifier.padding(16.dp), color = MaterialTheme.colorScheme.onSurfaceVariant)
            return@Column
        }
        Column(
            Modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Text(
                "${dayLabel(menu.date)} · ${slotLabel(menu.slot)} · ${if (menu.servings == 1) "1 Person" else "${menu.servings} Personen"}",
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            if (menu.note.isNotEmpty()) {
                Text(
                    menu.note,
                    Modifier.fillMaxWidth().clip(MaterialTheme.shapes.small)
                        .background(MaterialTheme.colorScheme.secondaryContainer).padding(12.dp),
                    color = MaterialTheme.colorScheme.onSecondaryContainer,
                )
            }
            if (entries.isEmpty()) Text("Noch keine Gänge.", color = MaterialTheme.colorScheme.onSurfaceVariant)
            entries.forEach { e ->
                val recipeId = e.recipeId
                Row(
                    Modifier.fillMaxWidth().clip(MaterialTheme.shapes.small)
                        .then(if (recipeId != null) Modifier.clickable { onOpenCourse(recipeId, e.id) } else Modifier)
                        .padding(vertical = 4.dp),
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(12.dp),
                ) {
                    CourseImage(e, 64)
                    Column(Modifier.weight(1f)) {
                        courseName(courses, e.courseTagId).takeIf { it.isNotEmpty() }?.let {
                            Text(it.uppercase(), style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.primary)
                        }
                        Text(entryTitle(e), style = MaterialTheme.typography.titleMedium)
                        if (e.recipe != null) {
                            Text(
                                amountText(e.servings, e.recipe.yieldUnit) + if (e.note.isNotEmpty()) " · ${e.note}" else "",
                                style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant,
                            )
                        }
                    }
                }
                HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant)
            }
        }
    }
}
