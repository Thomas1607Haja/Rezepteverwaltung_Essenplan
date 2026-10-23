package io.github.thomas1607haja.rezeptplaner.ui.recipe

import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AssistChip
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedIconButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalUriHandler
import androidx.compose.ui.platform.LocalView
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import io.github.thomas1607haja.rezeptplaner.R
import io.github.thomas1607haja.rezeptplaner.api.Ingredient
import io.github.thomas1607haja.rezeptplaner.api.Recipe
import io.github.thomas1607haja.rezeptplaner.ui.common.RecipeImage
import io.github.thomas1607haja.rezeptplaner.ui.common.app
import io.github.thomas1607haja.rezeptplaner.ui.formatMinutes
import io.github.thomas1607haja.rezeptplaner.ui.formatQuantity
import io.github.thomas1607haja.rezeptplaner.ui.planStatus
import io.github.thomas1607haja.rezeptplaner.ui.portions
import java.net.URI
import java.time.LocalDate

/**
 * Kochansicht. [planServings] > 0: geöffnet aus dem Essensplan, die
 * Portionen kommen aus dem Eintrag.
 */
@OptIn(ExperimentalMaterial3Api::class, ExperimentalLayoutApi::class)
@Composable
fun RecipeScreen(recipeId: Long, planServings: Int, onBack: () -> Unit, onOpen: (Long) -> Unit) {
    val snapshot by app().localStore.snapshot.collectAsStateWithLifecycle()
    val recipe = snapshot.recipes.firstOrNull { it.id == recipeId }

    // Bildschirm bleibt an, solange die Kochansicht offen ist.
    val view = LocalView.current
    DisposableEffect(view) {
        view.keepScreenOn = true
        onDispose { view.keepScreenOn = false }
    }

    Column(Modifier.fillMaxSize()) {
        TopAppBar(
            title = { Text(recipe?.title.orEmpty(), maxLines = 1, overflow = TextOverflow.Ellipsis) },
            navigationIcon = {
                IconButton(onClick = onBack) { Icon(painterResource(R.drawable.ic_back), contentDescription = "Zurück") }
            },
            // Den Platz für die Statusleiste hält schon das Scaffold frei.
            windowInsets = WindowInsets(0),
        )
        if (recipe == null) {
            Text("Dieses Rezept gibt es nicht mehr.", Modifier.padding(24.dp))
            return@Column
        }
        RecipeContent(recipe, planServings, snapshot.recipes, snapshot.planInfo.firstOrNull { it.recipeId == recipe.id }?.let {
            planStatus(it, snapshot.today.takeIf(String::isNotEmpty)?.let(LocalDate::parse) ?: LocalDate.now())
        } ?: "noch nie eingeplant", onOpen)
    }
}

@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun RecipeContent(recipe: Recipe, planServings: Int, all: List<Recipe>, status: String, onOpen: (Long) -> Unit) {
    var servings by rememberSaveable(recipe.id) { mutableIntStateOf(if (planServings > 0) planServings else recipe.baseServings) }
    val factor = servings.toDouble() / recipe.baseServings
    val uri = LocalUriHandler.current

    // Familie: Ursprungsrezept und alle Varianten (nur eine Ebene).
    val origin = recipe.parentRecipeId ?: recipe.id
    val family = remember(all, recipe.id) {
        all.filter { (it.id == origin || it.parentRecipeId == origin) && it.id != recipe.id }
    }

    Column(
        Modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(bottom = 24.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        if (recipe.image.isNotEmpty() || recipe.thumbnail.isNotEmpty()) {
            RecipeImage(
                recipe.image.ifEmpty { recipe.thumbnail },
                Modifier.fillMaxWidth().aspectRatio(16f / 10f),
                load = true,
            )
        }
        Column(Modifier.padding(horizontal = 16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Row(verticalAlignment = Alignment.Top) {
                Text(recipe.title, style = MaterialTheme.typography.headlineSmall, modifier = Modifier.weight(1f))
                if (recipe.isFavorite) {
                    Icon(painterResource(R.drawable.ic_heart_filled), contentDescription = "Favorit", tint = MaterialTheme.colorScheme.primary)
                }
            }
            if (recipe.parentRecipeId != null) {
                all.firstOrNull { it.id == recipe.parentRecipeId }?.let { parent ->
                    Text(
                        "Variante von ${parent.title}", color = MaterialTheme.colorScheme.primary,
                        modifier = Modifier.clickable { onOpen(parent.id) },
                    )
                }
            }
            if (family.isNotEmpty()) FamilyStrip(family, origin, onOpen)
            if (recipe.description.isNotEmpty()) Text(recipe.description)
            Meta(recipe, status) { url -> runCatching { uri.openUri(url) } }
            if (recipe.tags.isNotEmpty()) {
                FlowRow(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                    recipe.tags.forEach { AssistChip(onClick = {}, label = { Text(it.name) }) }
                }
            }
        }

        SectionCard("Zutaten") {
            Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                OutlinedIconButton(onClick = { if (servings > 1) servings-- }, enabled = servings > 1) { Text("−") }
                Text(portions(servings), style = MaterialTheme.typography.titleMedium)
                OutlinedIconButton(onClick = { servings++ }) { Text("+") }
            }
            if (servings != recipe.baseServings) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text(
                        "Umgerechnet von ${portions(recipe.baseServings)}",
                        style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                    TextButton(onClick = { servings = recipe.baseServings }) { Text("zurücksetzen") }
                }
            }
            Ingredients(recipe.ingredients, factor)
        }

        if (recipe.steps.isNotEmpty()) {
            SectionCard("Zubereitung") {
                recipe.steps.forEachIndexed { i, step ->
                    Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                        Text("${i + 1}.", color = MaterialTheme.colorScheme.primary, fontWeight = FontWeight.SemiBold)
                        Text(step.text)
                    }
                }
            }
        }
        if (recipe.notes.isNotEmpty()) {
            SectionCard("Notizen") { Text(recipe.notes) }
        }
    }
}

@Composable
private fun Meta(recipe: Recipe, status: String, open: (String) -> Unit) {
    val muted = MaterialTheme.colorScheme.onSurfaceVariant
    val parts = buildList {
        add(status)
        recipe.prepMinutes?.let { add("Vorbereitung ${formatMinutes(it)}") }
        recipe.totalMinutes?.let { add("Gesamt ${formatMinutes(it)}") }
    }
    Text(parts.joinToString(" · "), style = MaterialTheme.typography.bodySmall, color = muted)
    val web = recipe.sourceUrl.takeIf { it.startsWith("https://") || it.startsWith("http://") }
    if (web != null || recipe.sourceName.isNotEmpty()) {
        val label = recipe.sourceName.ifEmpty { runCatching { URI(web).host }.getOrNull() ?: web.orEmpty() }
        Text(
            "Quelle: $label",
            style = MaterialTheme.typography.bodySmall,
            color = if (web != null) MaterialTheme.colorScheme.primary else muted,
            modifier = if (web != null) Modifier.clickable { open(web) } else Modifier,
        )
    }
}

@Composable
private fun Ingredients(ingredients: List<Ingredient>, factor: Double) {
    var group: String? = null
    ingredients.forEach { ing ->
        if (ing.groupName.isNotEmpty() && ing.groupName != group) {
            Text(ing.groupName, style = MaterialTheme.typography.titleSmall, modifier = Modifier.padding(top = 8.dp))
        }
        group = ing.groupName
        Row(Modifier.fillMaxWidth().padding(vertical = 4.dp)) {
            val amount = listOf(formatQuantity(ing.quantity?.times(factor)), ing.unit).filter { it.isNotEmpty() }.joinToString(" ")
            Text(amount, fontWeight = FontWeight.SemiBold, textAlign = TextAlign.End, modifier = Modifier.width(88.dp))
            Spacer(Modifier.width(12.dp))
            Text(
                ing.name + if (ing.note.isNotEmpty()) ", ${ing.note}" else "",
                modifier = Modifier.weight(1f),
            )
        }
        HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant)
    }
}

@Composable
private fun FamilyStrip(family: List<Recipe>, origin: Long, onOpen: (Long) -> Unit) {
    Row(Modifier.horizontalScroll(rememberScrollState()), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        family.forEach { r ->
            Card(
                Modifier.widthIn(max = 160.dp).clickable { onOpen(r.id) },
                border = CardDefaults.outlinedCardBorder(),
            ) {
                Row(Modifier.padding(6.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    RecipeImage(r.thumbnail, Modifier.size(40.dp).clip(MaterialTheme.shapes.small))
                    Column {
                        Text(r.title, style = MaterialTheme.typography.bodySmall, maxLines = 2, overflow = TextOverflow.Ellipsis)
                        Text(
                            if (r.id == origin) "Ursprungsrezept" else "Variante",
                            style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun SectionCard(title: String, content: @Composable () -> Unit) {
    Card(Modifier.fillMaxWidth().padding(horizontal = 16.dp), border = CardDefaults.outlinedCardBorder()) {
        Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
            Text(title, style = MaterialTheme.typography.titleMedium)
            content()
        }
    }
}
