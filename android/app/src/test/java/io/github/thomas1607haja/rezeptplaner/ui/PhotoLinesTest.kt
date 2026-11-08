package io.github.thomas1607haja.rezeptplaner.ui

import io.github.thomas1607haja.rezeptplaner.ui.photo.LineKind
import io.github.thomas1607haja.rezeptplaner.ui.photo.LineKind.Ignore
import io.github.thomas1607haja.rezeptplaner.ui.photo.LineKind.Ingredient
import io.github.thomas1607haja.rezeptplaner.ui.photo.LineKind.Step
import io.github.thomas1607haja.rezeptplaner.ui.photo.LineKind.Title
import io.github.thomas1607haja.rezeptplaner.ui.photo.OcrLine
import io.github.thomas1607haja.rezeptplaner.ui.photo.buildDraft
import io.github.thomas1607haja.rezeptplaner.ui.photo.guessKinds
import org.junit.Assert.assertEquals
import org.junit.Test

class PhotoLinesTest {
    /** Zeilen einer Seite; jede Gruppe ist ein Absatz der Texterkennung. */
    private fun page(vararg blocks: List<String>, page: Int = 0) =
        blocks.flatMapIndexed { b, lines -> lines.map { OcrLine(it, page, b) } }

    @Test
    fun headingsSwitchSections() {
        val lines = page(
            listOf("Linsensuppe mit Speck"),
            listOf("Zutaten für 4 Personen"),
            listOf("250 g Linsen", "1 Zwiebel", "Salz"),
            listOf("Zubereitung"),
            listOf("Zwiebel würfeln und in Öl", "andünsten."),
            listOf("Linsen dazugeben und 30 Mi-", "nuten köcheln."),
            listOf("Seite 42"),
        )
        val kinds = guessKinds(lines)
        assertEquals(
            listOf(Title, Ignore, Ingredient, Ingredient, Ingredient, Ignore, Step, Step, Step, Step, Ignore),
            kinds,
        )
        val draft = buildDraft(lines, kinds)
        assertEquals("Linsensuppe mit Speck", draft.title)
        assertEquals(listOf("250 g Linsen", "1 Zwiebel", "Salz"), draft.ingredientLines)
        assertEquals(
            listOf("Zwiebel würfeln und in Öl andünsten.", "Linsen dazugeben und 30 Minuten köcheln."),
            draft.steps,
        )
    }

    @Test
    fun withoutHeadingsQuantityAndLengthDecide() {
        val lines = page(
            listOf("Gnocchi-Pfanne"),
            listOf("400 g Gnocchi", "½ Bund Basilikum", "etwas Parmesan"),
            listOf("1. Gnocchi in einer Pfanne goldbraun braten, dabei öfter wenden."),
            listOf("2. Basilikum zupfen und mit Parmesan darüber geben."),
        )
        val kinds = guessKinds(lines)
        assertEquals(listOf(Title, Ingredient, Ingredient, Ingredient, Step, Step), kinds)
        assertEquals(
            listOf("Gnocchi in einer Pfanne goldbraun braten, dabei öfter wenden.", "Basilikum zupfen und mit Parmesan darüber geben."),
            buildDraft(lines, kinds).steps,
        )
    }

    @Test
    fun numberedStepsInOneBlockAreSplitAndNumbersOnlyIgnored() {
        val lines = page(
            listOf("Zubereitung"),
            listOf("1", "Ofen auf 200 Grad vorheizen.", "2", "10 Minuten backen."),
        )
        val kinds = guessKinds(lines)
        assertEquals(listOf(Ignore, Ignore, Step, Ignore, Step), kinds)
        // „10 Minuten“ ist keine Schrittnummer.
        assertEquals(listOf("Ofen auf 200 Grad vorheizen.", "10 Minuten backen."), buildDraft(lines, kinds).steps)
    }

    @Test
    fun userChoiceWinsAndPagesStartNewSteps() {
        val lines = page(listOf("Teig kneten und"), page = 0) + page(listOf("ruhen lassen."), page = 1)
        val kinds = listOf(LineKind.Step, LineKind.Step)
        // Verschiedene Seiten sind verschiedene Absätze.
        assertEquals(listOf("Teig kneten und", "ruhen lassen."), buildDraft(lines, kinds).steps)
        assertEquals("", buildDraft(lines, listOf(Ignore, Ignore)).title)
    }

    @Test
    fun headingsInOtherLanguages() {
        // Titel, Zutaten-Überschrift, zwei Zutaten, Zubereitungs-Überschrift, Schritt.
        val recipes = listOf(
            listOf("Pancakes", "Ingredients", "flour", "milk", "Method", "Mix everything"),
            listOf("Crêpes", "Ingrédients pour 4 personnes", "farine", "lait", "Préparation", "Tout mélanger"),
            listOf("Crêpes", "INGREDIENTS", "farine", "lait", "Étapes :", "Tout mélanger"),
            listOf("Tortitas", "Ingredientes", "harina", "leche", "Elaboración:", "Mezclar todo"),
            listOf("Tortitas", "Ingredientes", "harina", "leche", "Pasos", "Mezclar todo"),
            listOf("Frittelle", "Ingredienti per 4 persone", "farina", "latte", "Procedimento", "Mescolare tutto"),
            listOf("Frittelle", "Ingredienti", "farina", "latte", "Preparazione", "Mescolare tutto"),
            listOf("Pancakes", "For 4 servings", "flour", "milk", "Instructions", "Mix everything"),
        )
        for (r in recipes) {
            val lines = page(listOf(r[0]), listOf(r[1]), listOf(r[2], r[3]), listOf(r[4]), listOf(r[5]))
            assertEquals(r.toString(), listOf(Title, Ignore, Ingredient, Ingredient, Ignore, Step), guessKinds(lines))
        }
    }

    @Test
    fun timeInHeaderIsNoStepsHeading() {
        // „Préparation : 15 min“ steht oft im Kopf – keine Überschrift für die Schritte.
        val lines = page(
            listOf("Quiche lorraine"),
            listOf("Préparation : 15 min"),
            listOf("Ingrédients"),
            listOf("200 g de lardons"),
            listOf("Étapes"),
            listOf("Cuire la pâte."),
        )
        assertEquals(listOf(Title, Ignore, Ignore, Ingredient, Ignore, Step), guessKinds(lines))
    }

    @Test
    fun stepWordsInOtherLanguages() {
        val lines = page(
            listOf("Instructions"),
            listOf("Step 1", "Preheat the oven.", "Step 2", "Bake for 10 minutes."),
        )
        val kinds = guessKinds(lines)
        assertEquals(listOf(Ignore, Ignore, Step, Ignore, Step), kinds)
        assertEquals(listOf("Preheat the oven.", "Bake for 10 minutes."), buildDraft(lines, kinds).steps)
        // „Étape 2 : …“ am Zeilenanfang beginnt einen neuen Schritt und wird entfernt.
        val fr = page(listOf("Étape 1 : Cuire la pâte dix minutes.", "Étape 2 : Ajouter les lardons."))
        assertEquals(listOf("Cuire la pâte dix minutes.", "Ajouter les lardons."), buildDraft(fr, listOf(Step, Step)).steps)
    }
}
