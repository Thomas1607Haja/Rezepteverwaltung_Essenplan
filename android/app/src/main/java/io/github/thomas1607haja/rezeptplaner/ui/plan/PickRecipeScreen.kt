package io.github.thomas1607haja.rezeptplaner.ui.plan

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.WindowInsets
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.unit.dp
import androidx.lifecycle.viewmodel.compose.viewModel
import io.github.thomas1607haja.rezeptplaner.R
import io.github.thomas1607haja.rezeptplaner.ui.PickRecipeRoute
import io.github.thomas1607haja.rezeptplaner.ui.common.LocalSnackbar
import io.github.thomas1607haja.rezeptplaner.ui.common.app
import io.github.thomas1607haja.rezeptplaner.ui.dayLabel
import io.github.thomas1607haja.rezeptplaner.ui.recipes.RecipesScreen
import io.github.thomas1607haja.rezeptplaner.ui.recipes.RecipesViewModel
import kotlinx.coroutines.launch

/** Rezeptauswahl beim Planen: die Rezeptübersicht, Antippen plant ein bzw. ersetzt. */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PickRecipeScreen(route: PickRecipeRoute, onDone: () -> Unit) {
    val app = app()
    val snackbar = LocalSnackbar.current
    val scope = rememberCoroutineScope()
    var busy by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    val replacing = remember(route) { app.localStore.snapshot.value.entries.firstOrNull { it.id == route.replaceEntryId } }

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
                if (!busy) {
                    busy = true
                    scope.launch {
                        val title = app.localStore.snapshot.value.recipes.firstOrNull { it.id == recipeId }?.title.orEmpty()
                        error = if (replacing != null) {
                            app.planRepository.replace(replacing, recipeId)
                        } else {
                            app.planRepository.add(route.date, route.slot, recipeId)
                        }
                        busy = false
                        if (error == null) {
                            onDone()
                            snackbar.showSnackbar("„$title“ eingeplant am ${dayLabel(route.date)}")
                        }
                    }
                }
            },
            // Eigener Filter, damit die Suche der Rezeptübersicht unberührt bleibt.
            viewModel = viewModel<RecipesViewModel>(key = "picker"),
            modifier = Modifier.weight(1f),
        )
    }
}
