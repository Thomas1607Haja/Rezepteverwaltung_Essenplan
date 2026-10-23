package io.github.thomas1607haja.rezeptplaner.ui

import androidx.activity.ComponentActivity
import androidx.compose.runtime.collectAsState
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.SemanticsProperties
import androidx.compose.ui.test.SemanticsMatcher
import androidx.compose.ui.test.assertIsNotSelected
import androidx.compose.ui.test.hasText
import androidx.compose.ui.test.assertIsSelected
import androidx.compose.ui.test.junit4.createAndroidComposeRule
import androidx.compose.ui.test.onNodeWithText
import androidx.compose.ui.test.onRoot
import androidx.compose.ui.test.performClick
import androidx.test.ext.junit.runners.AndroidJUnit4
import com.github.takahirom.roborazzi.captureRoboImage
import io.github.thomas1607haja.rezeptplaner.RezeptplanerApplication
import io.github.thomas1607haja.rezeptplaner.ui.setup.SetupScreen
import io.github.thomas1607haja.rezeptplaner.ui.setup.SetupViewModel
import io.github.thomas1607haja.rezeptplaner.ui.theme.RezeptplanerTheme
import org.junit.Rule
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.annotation.Config
import org.robolectric.annotation.GraphicsMode

/**
 * Oberflächentests ohne Gerät (Robolectric). Mit
 * `./gradlew recordRoborazziDebug` entstehen zusätzlich Bildschirmfotos
 * unter app/build/outputs/roborazzi/ – zum Ansehen, nicht zum Vergleichen.
 */
@RunWith(AndroidJUnit4::class)
@GraphicsMode(GraphicsMode.Mode.NATIVE)
@Config(sdk = [36], qualifiers = "w411dp-h891dp-xxhdpi") // etwa Galaxy A54
class ScreensTest {

    @get:Rule
    val compose = createAndroidComposeRule<ComponentActivity>()

    /** Reiter der unteren Leiste (der Seitentitel heißt oft genauso). */
    private fun tab(label: String) = compose.onNode(
        hasText(label) and SemanticsMatcher.expectValue(SemanticsProperties.Role, Role.Tab),
    )

    private fun shot(name: String) = compose.onRoot().captureRoboImage("build/outputs/roborazzi/$name.png")

    @Test
    fun backFromOtherSectionReturnsToMealPlan() {
        compose.setContent { RezeptplanerTheme { MainScaffold(onReconnect = {}) } }
        shot("01-essensplan")

        tab("Rezepte").performClick()
        tab("Rezepte").assertIsSelected()

        compose.runOnUiThread { compose.activity.onBackPressedDispatcher.onBackPressed() }
        tab("Essensplan").assertIsSelected()
        tab("Rezepte").assertIsNotSelected()
    }

    @Test
    fun settings() {
        compose.setContent { RezeptplanerTheme { MainScaffold(onReconnect = {}) } }
        tab("Einstellungen").performClick()
        compose.onNodeWithText("Verbindung testen").assertExists()
        shot("02-einstellungen")
    }

    @Test
    fun setupSteps() {
        val app = compose.activity.application as RezeptplanerApplication
        val viewModel = SetupViewModel(app.configRepository, app.apiClient)
        compose.setContent {
            RezeptplanerTheme {
                val state = viewModel.state.collectAsState().value
                SetupScreen(viewModel, state, onFinished = {}, onCancel = null)
            }
        }
        shot("03-einrichtung-start")

        compose.onNodeWithText("Von Hand eingeben").performClick()
        shot("04-einrichtung-von-hand")

        compose.runOnUiThread {
            viewModel.onLink(
                "rezeptplaner://setup?url=https%3A%2F%2Frezepte.example.org" +
                    "&key=rp_5r13s154siEAjGyfI9GC0ah52Z20_LXDK8f0soq-NDg",
            )
        }
        compose.onNodeWithText("rezepte.example.org").assertExists()
        shot("05-einrichtung-bestaetigen")

        compose.runOnUiThread { viewModel.onLink("https://irgendwas.example.org") }
        compose.onNodeWithText("Das ist kein QR-Code zum Verbinden des Rezeptplaners.").assertExists()
    }

    @Test
    @Config(qualifiers = "+night")
    fun darkMode() {
        compose.setContent { RezeptplanerTheme { MainScaffold(onReconnect = {}) } }
        shot("06-dunkel")
    }
}
