package io.github.thomas1607haja.rezeptplaner.ui

import androidx.activity.ComponentActivity
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.test.junit4.createAndroidComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.onRoot
import androidx.compose.ui.test.performScrollToNode
import androidx.compose.ui.test.hasText
import androidx.compose.ui.test.hasScrollToIndexAction
import androidx.compose.ui.unit.Density
import androidx.test.ext.junit.runners.AndroidJUnit4
import com.github.takahirom.roborazzi.captureRoboImage
import io.github.thomas1607haja.rezeptplaner.RezeptplanerApplication
import io.github.thomas1607haja.rezeptplaner.api.EntryRecipe
import io.github.thomas1607haja.rezeptplaner.api.MealPlanEntry
import io.github.thomas1607haja.rezeptplaner.api.Recipe
import io.github.thomas1607haja.rezeptplaner.api.Tag
import io.github.thomas1607haja.rezeptplaner.api.TagGroup
import io.github.thomas1607haja.rezeptplaner.data.Snapshot
import io.github.thomas1607haja.rezeptplaner.ui.plan.MealPlanScreen
import io.github.thomas1607haja.rezeptplaner.ui.plan.PickRecipeScreen
import io.github.thomas1607haja.rezeptplaner.ui.theme.RezeptplanerTheme
import kotlinx.coroutines.runBlocking
import org.junit.Before
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.annotation.Config
import org.robolectric.annotation.GraphicsMode

/**
 * Rezeptauswahl beim Planen mit vielen Schnellauswahlen: Die Rezepte
 * müssen auch mit großer Schrift und im Querformat erreichbar sein.
 * Bildschirmfotos mit `./gradlew recordRoborazziDebug`.
 */
@RunWith(AndroidJUnit4::class)
@GraphicsMode(GraphicsMode.Mode.NATIVE)
@Config(sdk = [36], qualifiers = "w411dp-h891dp-xxhdpi")
class PickerLayoutTest {

    @get:Rule
    val compose = createAndroidComposeRule<ComponentActivity>()

    private val courses = TagGroup(
        "Gang", id = 9, role = "course",
        tags = listOf(Tag(1, "Hauptgericht"), Tag(2, "Vorspeise"), Tag(3, "Nachspeise"), Tag(4, "Kuchen & Torten")),
    )

    @Before
    fun seed() {
        val app = compose.activity.application as RezeptplanerApplication
        val titles = listOf(
            "Romanesco-Tomaten-Pasta", "Bratkartoffeln mit Speck", "Chili con Carne", "Griechischer Bauernsalat",
            "Hähnchen-Curry mit Reis", "Kartoffel-Lauch-Suppe", "Linsen-Dal", "Spaghetti Bolognese",
        )
        val notes = listOf("Bei Oma", "Grillen", "Sushi holen", "Döner", "Kantine", "Brotzeit", "Restaurant am See")
        runBlocking {
            app.localStore.save(
                Snapshot(
                    serverTime = "2026-10-24T12:00:00Z",
                    today = "2026-10-24",
                    planFrom = "2026-10-10",
                    menusSupported = true,
                    tagGroups = listOf(courses),
                    recipes = titles.mapIndexed { i, t -> Recipe(i + 1L, t, baseServings = 4, tags = listOf(Tag(1, "Hauptgericht"))) },
                    entries = notes.mapIndexed { i, n -> MealPlanEntry(100L + i, "2026-10-${10 + i}", servings = 4, note = n) } +
                        MealPlanEntry(200, "2026-10-24", servings = 4, recipeId = 1, recipe = EntryRecipe("Romanesco-Tomaten-Pasta")) +
                        MealPlanEntry(201, "2026-10-25", servings = 4, recipeId = 2, recipe = EntryRecipe("Pizza selbstgemacht")),
                ),
            )
        }
    }

    private fun picker(fontScale: Float = 1f) {
        compose.setContent {
            val d = LocalDensity.current
            CompositionLocalProvider(LocalDensity provides Density(d.density, fontScale)) {
                RezeptplanerTheme { PickRecipeScreen(PickRecipeRoute("2026-10-26"), onDone = {}) }
            }
        }
    }

    private fun shot(name: String) = compose.onRoot().captureRoboImage("build/outputs/roborazzi/$name.png")

    /** Das letzte Rezept muss sich in die Ansicht scrollen lassen. */
    private fun lastRecipeReachable() {
        compose.onNode(hasScrollToIndexAction()).performScrollToNode(hasText("Spaghetti Bolognese"))
        compose.onNodeWithText("Spaghetti Bolognese").assertExists()
    }

    @Test
    fun portrait() {
        picker()
        shot("picker-hoch")
        // Feste Schnellauswahl plus höchstens drei zuletzt verwendete Freitexte.
        compose.onNodeWithText("Kantine").assertExists()
        compose.onNodeWithText("Döner").assertDoesNotExist()
        lastRecipeReachable()
        shot("picker-hoch-gescrollt")
    }

    @Test
    fun largeFont() {
        picker(fontScale = 2f)
        shot("picker-schrift-200")
        lastRecipeReachable()
        shot("picker-schrift-200-gescrollt")
    }

    @Test
    @Config(qualifiers = "w891dp-h411dp-land-xxhdpi")
    fun landscape() {
        picker(fontScale = 1.3f)
        shot("picker-quer")
        lastRecipeReachable()
        shot("picker-quer-gescrollt")
    }

    @Test
    fun mealPlan() {
        compose.setContent { RezeptplanerTheme { MealPlanScreen(onOpenRecipe = { _, _ -> }, onAdd = {}, onReplace = {}) } }
        shot("plan-heute")
    }
}
