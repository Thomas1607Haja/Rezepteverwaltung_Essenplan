package io.github.thomas1607haja.rezeptplaner.ui

import androidx.annotation.DrawableRes
import androidx.annotation.StringRes
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import androidx.navigation.NavDestination.Companion.hasRoute
import androidx.navigation.NavDestination.Companion.hierarchy
import androidx.navigation.NavGraph.Companion.findStartDestination
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.currentBackStackEntryAsState
import androidx.navigation.compose.rememberNavController
import androidx.navigation.toRoute
import io.github.thomas1607haja.rezeptplaner.data.ImportState
import io.github.thomas1607haja.rezeptplaner.data.SyncTrigger
import io.github.thomas1607haja.rezeptplaner.ui.importer.ImportScreen
import io.github.thomas1607haja.rezeptplaner.sync.SyncWorker
import io.github.thomas1607haja.rezeptplaner.ui.common.LocalSnackbar
import io.github.thomas1607haja.rezeptplaner.ui.plan.MealPlanScreen
import io.github.thomas1607haja.rezeptplaner.ui.plan.PickRecipeScreen
import io.github.thomas1607haja.rezeptplaner.ui.recipe.RecipeScreen
import io.github.thomas1607haja.rezeptplaner.ui.recipes.RecipesScreen
import io.github.thomas1607haja.rezeptplaner.R
import io.github.thomas1607haja.rezeptplaner.RezeptplanerApplication
import io.github.thomas1607haja.rezeptplaner.data.ServerConfig
import io.github.thomas1607haja.rezeptplaner.ui.settings.SettingsScreen
import io.github.thomas1607haja.rezeptplaner.ui.settings.SettingsViewModel
import io.github.thomas1607haja.rezeptplaner.ui.setup.SetupScreen
import io.github.thomas1607haja.rezeptplaner.ui.setup.SetupViewModel
import kotlinx.coroutines.flow.map
import kotlinx.serialization.Serializable

// Ziele der Navigation. @Serializable macht sie zu typsicheren Routen
// (Navigation Compose); später kommen Ziele mit Parametern hinzu, z. B.
// ein Rezept mit seiner ID.
@Serializable data object MealPlanRoute
@Serializable data object RecipesRoute
@Serializable data object ShoppingRoute
@Serializable data object SettingsRoute

/**
 * Rezeptauswahl beim Planen: für [date]/[slot] einplanen oder – mit
 * [replaceEntryId] > 0 – das Gericht dieses Eintrags ersetzen.
 */
@Serializable data class PickRecipeRoute(val date: String, val slot: String = "dinner", val replaceEntryId: Long = 0)

/** Datei-Import (Teilen, „Öffnen mit“, Dateiauswahl). */
@Serializable data object ImportRoute

/** Kochansicht; planServings > 0 = aus dem Essensplan geöffnet. */
@Serializable data class RecipeRoute(val id: Long, val planServings: Int = 0)

/** Die vier Hauptbereiche in der unteren Leiste, wie in der Weboberfläche. */
private enum class Section(val route: Any, @StringRes val label: Int, @DrawableRes val icon: Int) {
    MealPlan(MealPlanRoute, R.string.nav_meal_plan, R.drawable.ic_nav_calendar),
    Recipes(RecipesRoute, R.string.nav_recipes, R.drawable.ic_nav_book),
    Shopping(ShoppingRoute, R.string.nav_shopping, R.drawable.ic_nav_cart),
    Settings(SettingsRoute, R.string.nav_settings, R.drawable.ic_nav_gear),
}

/** Gespeicherte Verbindung; Loading, solange der DataStore noch liest. */
private sealed interface ConfigState {
    data object Loading : ConfigState
    data class Loaded(val config: ServerConfig?) : ConfigState
}

/**
 * Wurzel der Oberfläche. Ohne Verbindung (oder wenn ein Einrichtungslink
 * geöffnet wurde) erscheint die Einrichtung, sonst die App mit der
 * unteren Leiste.
 *
 * @param pendingLink geöffneter Link rezeptplaner://setup?… (z. B. aus der
 *   Kamera-App), null = keiner
 */
@Composable
fun RezeptplanerApp(
    pendingLink: String?,
    onLinkConsumed: () -> Unit,
    openImport: Boolean = false,
    onImportOpened: () -> Unit = {},
) {
    val app = LocalContext.current.applicationContext as RezeptplanerApplication
    val configFlow = remember { app.configRepository.config.map<ServerConfig?, ConfigState> { ConfigState.Loaded(it) } }
    val configState by configFlow.collectAsStateWithLifecycle(ConfigState.Loading)
    val setupViewModel = viewModel { SetupViewModel(app.configRepository, app.apiClient) }
    val setupState by setupViewModel.state.collectAsStateWithLifecycle()
    // Erneute Einrichtung bei bestehender Verbindung (Einstellungen oder Link).
    var reconnecting by rememberSaveable { mutableStateOf(false) }

    LaunchedEffect(pendingLink) {
        if (pendingLink != null) {
            setupViewModel.onLink(pendingLink)
            reconnecting = true
            onLinkConsumed()
        }
    }

    val state = configState
    if (state !is ConfigState.Loaded) {
        Surface(Modifier.fillMaxSize()) {}
        return
    }
    val hasConfig = state.config != null
    if (!hasConfig || reconnecting || setupState is SetupViewModel.State.Done) {
        SetupScreen(
            viewModel = setupViewModel,
            state = setupState,
            onFinished = {
                reconnecting = false
                setupViewModel.back()
            },
            onCancel = if (hasConfig) {
                {
                    reconnecting = false
                    setupViewModel.back()
                }
            } else {
                null
            },
        )
    } else {
        // Einmal je App-Start abgleichen und den Abgleich im Hintergrund
        // einplanen (bleibt bestehen, bis die Verbindung getrennt wird).
        val context = LocalContext.current
        var synced by rememberSaveable { mutableStateOf(false) }
        LaunchedEffect(Unit) {
            if (!synced) {
                synced = true
                SyncWorker.schedule(context)
                app.syncRepository.sync(SyncTrigger.Start)
            }
        }
        // Kamen Dateien vor der Einrichtung, geht es jetzt mit ihnen weiter.
        // Einmal starten und auf den Zustand warten: Ein LaunchedEffect mit dem
        // Zustand als Schlüssel würde beim Wechsel auf „Uploading“ neu starten
        // und dabei den laufenden Upload abbrechen.
        val importState by app.importRepository.state.collectAsStateWithLifecycle()
        LaunchedEffect(Unit) {
            app.importRepository.state.collect { if (it is ImportState.WaitingForSetup) app.importRepository.resume() }
        }
        MainScaffold(
            onReconnect = { reconnecting = true },
            openImport = openImport || importState !is ImportState.Idle,
            onImportOpened = onImportOpened,
        )
    }
}

/** App mit unterer Leiste; internal für die Oberflächentests. */
@Composable
internal fun MainScaffold(onReconnect: () -> Unit, openImport: Boolean = false, onImportOpened: () -> Unit = {}) {
    val app = LocalContext.current.applicationContext as RezeptplanerApplication
    val navController = rememberNavController()
    val backStackEntry by navController.currentBackStackEntryAsState()
    val destination = backStackEntry?.destination

    val snackbar = remember { SnackbarHostState() }
    // Neue Dateien zum Import: Import-Bildschirm zeigen (nur einmal je Eingang).
    LaunchedEffect(openImport) {
        if (openImport && destination?.hasRoute(ImportRoute::class) != true) navController.navigate(ImportRoute)
        if (openImport) onImportOpened()
    }
    Scaffold(
        snackbarHost = { SnackbarHost(snackbar) },
        bottomBar = {
            NavigationBar {
                Section.entries.forEach { section ->
                    val selected = destination?.hierarchy?.any { it.hasRoute(section.route::class) } == true
                    NavigationBarItem(
                        selected = selected,
                        onClick = {
                            navController.navigate(section.route) {
                                // Wie bei den Google-Apps: Jeder Bereich behält
                                // seinen Zustand, Zurück führt zum Essensplan.
                                popUpTo(navController.graph.findStartDestination().id) { saveState = true }
                                launchSingleTop = true
                                restoreState = true
                            }
                        },
                        icon = { Icon(painterResource(section.icon), contentDescription = null) },
                        label = { Text(stringResource(section.label)) },
                    )
                }
            }
        },
    ) { innerPadding ->
        CompositionLocalProvider(LocalSnackbar provides snackbar) {
        NavHost(
            navController = navController,
            startDestination = MealPlanRoute,
            modifier = Modifier.padding(innerPadding),
        ) {
            composable<MealPlanRoute> {
                MealPlanScreen(
                    onOpenRecipe = { id, servings -> navController.navigate(RecipeRoute(id, servings)) },
                    onAdd = { date -> navController.navigate(PickRecipeRoute(date)) },
                    onReplace = { e -> navController.navigate(PickRecipeRoute(e.date, e.slot, e.id)) },
                )
            }
            composable<PickRecipeRoute> { entry ->
                PickRecipeScreen(entry.toRoute(), onDone = { navController.popBackStack() })
            }
            composable<RecipesRoute> {
                RecipesScreen(
                    onOpen = { id -> navController.navigate(RecipeRoute(id)) },
                    onImport = { navController.navigate(ImportRoute) },
                )
            }
            composable<ImportRoute> {
                ImportScreen(
                    onClose = { navController.popBackStack() },
                    onOpenRecipe = { id -> navController.navigate(RecipeRoute(id)) },
                )
            }
            composable<RecipeRoute> { entry ->
                val route = entry.toRoute<RecipeRoute>()
                RecipeScreen(
                    recipeId = route.id,
                    planServings = route.planServings,
                    onBack = { navController.popBackStack() },
                    onOpen = { id -> navController.navigate(RecipeRoute(id)) },
                )
            }
            composable<ShoppingRoute> { Placeholder(stringResource(R.string.nav_shopping)) }
            composable<SettingsRoute> {
                SettingsScreen(
                    viewModel = viewModel {
                        SettingsViewModel(app.configRepository, app.apiClient, app.syncRepository) { SyncWorker.cancel(app) }
                    },
                    onReconnect = onReconnect,
                )
            }
        }
        }
    }
}

/** Platzhalter, bis die Bereiche in den nächsten Schritten gebaut sind. */
@Composable
private fun Placeholder(title: String) {
    Box(Modifier.fillMaxSize().padding(24.dp), contentAlignment = Alignment.Center) {
        Text(title, style = MaterialTheme.typography.headlineSmall)
    }
}
