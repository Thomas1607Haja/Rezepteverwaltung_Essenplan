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
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.navigation.NavDestination.Companion.hasRoute
import androidx.navigation.NavDestination.Companion.hierarchy
import androidx.navigation.NavGraph.Companion.findStartDestination
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.currentBackStackEntryAsState
import androidx.navigation.compose.rememberNavController
import io.github.thomas1607haja.rezeptplaner.R
import kotlinx.serialization.Serializable

// Ziele der Navigation. @Serializable macht sie zu typsicheren Routen
// (Navigation Compose); später kommen Ziele mit Parametern hinzu, z. B.
// ein Rezept mit seiner ID.
@Serializable data object MealPlanRoute
@Serializable data object RecipesRoute
@Serializable data object ShoppingRoute
@Serializable data object SettingsRoute

/** Die vier Hauptbereiche in der unteren Leiste, wie in der Weboberfläche. */
private enum class Section(val route: Any, @StringRes val label: Int, @DrawableRes val icon: Int) {
    MealPlan(MealPlanRoute, R.string.nav_meal_plan, R.drawable.ic_nav_calendar),
    Recipes(RecipesRoute, R.string.nav_recipes, R.drawable.ic_nav_book),
    Shopping(ShoppingRoute, R.string.nav_shopping, R.drawable.ic_nav_cart),
    Settings(SettingsRoute, R.string.nav_settings, R.drawable.ic_nav_gear),
}

@Composable
fun RezeptplanerApp() {
    val navController = rememberNavController()
    val backStackEntry by navController.currentBackStackEntryAsState()
    val destination = backStackEntry?.destination

    Scaffold(
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
        NavHost(
            navController = navController,
            startDestination = MealPlanRoute,
            modifier = Modifier.padding(innerPadding),
        ) {
            composable<MealPlanRoute> { Placeholder(stringResource(R.string.nav_meal_plan)) }
            composable<RecipesRoute> { Placeholder(stringResource(R.string.nav_recipes)) }
            composable<ShoppingRoute> { Placeholder(stringResource(R.string.nav_shopping)) }
            composable<SettingsRoute> { Placeholder(stringResource(R.string.nav_settings)) }
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
