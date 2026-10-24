package io.github.thomas1607haja.rezeptplaner.ui

import androidx.annotation.DrawableRes
import androidx.annotation.StringRes
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
import io.github.thomas1607haja.rezeptplaner.ui.edit.RecipeEditScreen
import io.github.thomas1607haja.rezeptplaner.ui.edit.UrlImportScreen
import io.github.thomas1607haja.rezeptplaner.ui.photo.PhotoImportScreen
import io.github.thomas1607haja.rezeptplaner.ui.shopping.ShoppingScreen
import androidx.compose.runtime.rememberCoroutineScope
import kotlinx.coroutines.launch
import io.github.thomas1607haja.rezeptplaner.sync.SyncWorker
import io.github.thomas1607haja.rezeptplaner.ui.common.LocalSnackbar
import io.github.thomas1607haja.rezeptplaner.ui.plan.MealPlanScreen
import io.github.thomas1607haja.rezeptplaner.ui.plan.PickRecipeScreen
import io.github.thomas1607haja.rezeptplaner.ui.recipe.RecipeScreen
import io.github.thomas1607haja.rezeptplaner.ui.recipes.RecipeFilter
import io.github.thomas1607haja.rezeptplaner.ui.recipes.RecipesScreen
import io.github.thomas1607haja.rezeptplaner.R
import io.github.thomas1607haja.rezeptplaner.RezeptplanerApplication
import io.github.thomas1607haja.rezeptplaner.data.ServerConfig
import io.github.thomas1607haja.rezeptplaner.ui.settings.SettingsScreen
import io.github.thomas1607haja.rezeptplaner.ui.stats.StatisticsScreen
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
@Serializable data object StatisticsRoute
@Serializable data object SettingsRoute

/**
 * Rezeptauswahl beim Planen: für [date]/[slot] einplanen oder – mit
 * [replaceEntryId] > 0 – das Gericht dieses Eintrags ersetzen.
 */
@Serializable data class PickRecipeRoute(val date: String, val slot: String = "dinner", val replaceEntryId: Long = 0)

/** Rezept bearbeiten bzw. anlegen (id 0); fromDraft: Entwurf aus dem URL-Import. */
@Serializable data class EditRecipeRoute(val id: Long = 0, val fromDraft: Boolean = false, val freshVariant: Boolean = false)

/** Rezept aus einer Webadresse (geteilt, eingegeben oder per QR-Code mit [scan]). */
@Serializable data class UrlImportRoute(val url: String = "", val scan: Boolean = false)

/** Rezept aus Fotos (Texterkennung auf dem Handy). */
@Serializable data object PhotoImportRoute

/** Datei-Import (Teilen, „Öffnen mit“, Dateiauswahl). */
@Serializable data object ImportRoute

/**
 * Kochansicht. Aus dem Essensplan mit [entryId] (Portionen, Tag und Notiz
 * des Eintrags), aus dem Einkauf mit [planServings] > 0.
 */
@Serializable data class RecipeRoute(val id: Long, val planServings: Int = 0, val entryId: Long = 0)

/** Die Hauptbereiche in der unteren Leiste, wie in der Weboberfläche. */
private enum class Section(val route: Any, @StringRes val label: Int, @DrawableRes val icon: Int) {
    MealPlan(MealPlanRoute, R.string.nav_meal_plan, R.drawable.ic_nav_calendar),
    Recipes(RecipesRoute, R.string.nav_recipes, R.drawable.ic_nav_book),
    Shopping(ShoppingRoute, R.string.nav_shopping, R.drawable.ic_nav_cart),
    Statistics(StatisticsRoute, R.string.nav_statistics, R.drawable.ic_nav_chart),
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
    sharedUrl: String? = null,
    onUrlOpened: () -> Unit = {},
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
            sharedUrl = sharedUrl,
            onUrlOpened = onUrlOpened,
        )
    }
}

/** App mit unterer Leiste; internal für die Oberflächentests. */
@Composable
internal fun MainScaffold(
    onReconnect: () -> Unit,
    openImport: Boolean = false,
    onImportOpened: () -> Unit = {},
    sharedUrl: String? = null,
    onUrlOpened: () -> Unit = {},
) {
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
    // Geteilte Rezept-URL: gleich einlesen.
    LaunchedEffect(sharedUrl) {
        if (sharedUrl != null) {
            navController.navigate(UrlImportRoute(sharedUrl))
            onUrlOpened()
        }
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
                        // Fünf Bereiche: kleinere Schrift, damit „Einstellungen“ in eine Zeile passt.
                        label = { Text(stringResource(section.label), style = MaterialTheme.typography.labelSmall, maxLines = 1, softWrap = false) },
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
                    onOpenRecipe = { id, entryId -> navController.navigate(RecipeRoute(id, entryId = entryId)) },
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
                    onNew = { navController.navigate(EditRecipeRoute()) },
                    onFromUrl = { navController.navigate(UrlImportRoute()) },
                    onFromQr = { navController.navigate(UrlImportRoute(scan = true)) },
                    onFromPhoto = { navController.navigate(PhotoImportRoute) },
                )
            }
            composable<PhotoImportRoute> {
                PhotoImportScreen(
                    onDraft = {
                        navController.navigate(EditRecipeRoute(fromDraft = true)) {
                            popUpTo<PhotoImportRoute> { inclusive = true }
                        }
                    },
                    onClose = { navController.popBackStack() },
                )
            }
            composable<UrlImportRoute> { entry ->
                val route = entry.toRoute<UrlImportRoute>()
                UrlImportScreen(
                    initialUrl = route.url,
                    scan = route.scan,
                    onDraft = {
                        navController.navigate(EditRecipeRoute(fromDraft = true)) {
                            popUpTo<UrlImportRoute> { inclusive = true }
                        }
                    },
                    onClose = { navController.popBackStack() },
                )
            }
            composable<EditRecipeRoute> { entry ->
                val route = entry.toRoute<EditRecipeRoute>()
                val snackbar = LocalSnackbar.current
                val scope = rememberCoroutineScope()
                RecipeEditScreen(
                    recipeId = route.id,
                    fromDraft = route.fromDraft,
                    freshVariant = route.freshVariant,
                    onSaved = { id, warning ->
                        // Kochansicht des gespeicherten Rezepts statt der Bearbeitung.
                        navController.popBackStack()
                        if (route.id == 0L || route.freshVariant) navController.navigate(RecipeRoute(id)) { launchSingleTop = true }
                        scope.launch { snackbar.showSnackbar(warning ?: "Gespeichert") }
                    },
                    onCancel = { navController.popBackStack() },
                    onOpenRecipe = { id -> navController.navigate(RecipeRoute(id)) },
                    onDeleted = {
                        // Verworfene Variante: zurück zum Original. Sonst vor die (gelöschte)
                        // Kochansicht, egal über welchen Weg sie geöffnet wurde.
                        if (route.freshVariant || !navController.popBackStack<RecipeRoute>(inclusive = true)) navController.popBackStack()
                    },
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
                    entryId = route.entryId,
                    onBack = { navController.popBackStack() },
                    onTag = { tagId ->
                        // Wie im Web: Rezeptübersicht nur mit dieser Kategorie. Die Kochansicht
                        // wird dabei verlassen (nicht gespeichert), damit der Essensplan-Reiter
                        // danach wieder den Plan zeigt.
                        app.recipeFilter.value = RecipeFilter(tagIds = setOf(tagId), sort = app.recipeFilter.value.sort)
                        navController.navigate(RecipesRoute) {
                            popUpTo(navController.graph.findStartDestination().id)
                            launchSingleTop = true
                        }
                    },
                    onOpen = { id -> navController.navigate(RecipeRoute(id)) },
                    onEdit = { navController.navigate(EditRecipeRoute(route.id)) },
                    onVariantCreated = { id -> navController.navigate(EditRecipeRoute(id, freshVariant = true)) },
                )
            }
            composable<ShoppingRoute> {
                ShoppingScreen(onOpenRecipe = { id, servings -> navController.navigate(RecipeRoute(id, servings)) })
            }
            composable<StatisticsRoute> {
                StatisticsScreen(
                    onRecipes = { filter ->
                        app.recipeFilter.value = filter.copy(sort = app.recipeFilter.value.sort)
                        navController.navigate(RecipesRoute) {
                            popUpTo(navController.graph.findStartDestination().id)
                            launchSingleTop = true
                        }
                    },
                    onOpen = { id -> navController.navigate(RecipeRoute(id)) },
                )
            }
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
