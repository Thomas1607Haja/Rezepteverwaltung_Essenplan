package io.github.thomas1607haja.rezeptplaner.ui.photo

import androidx.annotation.StringRes
import io.github.thomas1607haja.rezeptplaner.R

/** Eine erkannte Textzeile; [block] ist der Absatz der Texterkennung auf Seite [page]. */
data class OcrLine(val text: String, val page: Int, val block: Int)

/** Wozu eine Zeile gehört; [label] ist der angezeigte Name (strings_photo.xml). */
enum class LineKind(@StringRes val label: Int) {
    Title(R.string.photo_kind_title),
    Ingredient(R.string.photo_kind_ingredient),
    Step(R.string.photo_kind_step),
    Ignore(R.string.photo_kind_ignore),
}

/** Ergebnis der Zuordnung, bevor der Server die Zutaten zerlegt. */
data class PhotoDraft(val title: String, val ingredientLines: List<String>, val steps: List<String>)

// Erkennungsmerkmale, keine Anzeige: Rezepte können in jeder Sprache
// fotografiert werden – unabhängig von der Sprache der App. Deshalb stehen
// hier Wörter aus Deutsch, Englisch, Französisch, Spanisch und Italienisch
// nebeneinander (Akzente optional, die Texterkennung verliert sie oft).
private val ingredientsHeading = Regex(
    """^(zutaten(liste)?|einkaufsliste|das brauchst du( noch)?|für \d+ (personen|portionen)""" +
        """|ingr[eé]dients?|ingredientes|ingredienti""" +
        """|for \d+ (servings|people|persons)|pour \d+ (personnes|portions)""" +
        """|para \d+ (personas|raciones|porciones)|per \d+ (persone|porzioni))\b.*""",
    RegexOption.IGNORE_CASE,
)
// Deutsche Überschriften dürfen weiteren Text haben; die anderen nur allein
// stehen (auch mit Doppelpunkt) – „Préparation : 15 min“ oder „Preparation
// time 20 min“ sind Zeitangaben im Kopf, keine Überschrift.
private val stepsHeading = Regex(
    """^(?:(zubereitung|so geht'?s|anleitung|und so wird'?s gemacht|arbeitsschritte)\b.*""" +
        """|(preparation|steps|method|instructions|directions""" +
        """|pr[eé]paration|[eé]tapes""" +
        """|preparaci[oó]n|elaboraci[oó]n|pasos|modo de preparaci[oó]n""" +
        """|preparazione|procedimento)\s*:?)$""",
    RegexOption.IGNORE_CASE,
)
private val quantityStart = Regex("""^(\d|[½¼¾⅓⅔⅛]|eine?\b|etwas\b|evtl\.|prise\b|n\.\s?b\.|nach belieben)""", RegexOption.IGNORE_CASE)
// Wort vor einer Schrittnummer: „Schritt 3“, „Step 3“, „Étape 3“, „Paso 3“, „Passaggio 3“.
private const val STEP_WORD = """(?:schritt|step|[eé]tape|paso|passaggio)"""
// „1.“, „2)“, „Schritt 3“ – nicht „200 g“ oder „1,5 l“.
private val stepNumber = Regex("""^\s*(?:$STEP_WORD\s*\d{1,2}\s*[.):]?|\d{1,2}\s*[.):])\s*""", RegexOption.IGNORE_CASE)
private val stepNumberOnly = Regex("""^\s*(?:$STEP_WORD\s*)?\d{1,2}\s*[.):]?\s*$""", RegexOption.IGNORE_CASE)
private val noise = Regex("""^(www\.|https?://|©|(seite|page|p[aá]gina) \d+|\d+$|[^\p{L}\d]*$)""", RegexOption.IGNORE_CASE)

/**
 * Schlägt für jede Zeile eine Art vor. Überschriften wie „Zutaten“ und
 * „Zubereitung“ schalten den Abschnitt um; ohne Überschriften entscheiden
 * Mengenangabe am Anfang (Zutat) und Länge bzw. Satzende (Schritt). Die
 * erste kurze Zeile vor der ersten Zutat wird der Titel. Der Nutzer
 * korrigiert danach per Antippen.
 */
fun guessKinds(lines: List<OcrLine>): List<LineKind> {
    var section: LineKind? = null
    var titleSet = false
    var seenIngredient = false
    return lines.map { line ->
        val t = line.text.trim()
        val kind = when {
            t.length < 2 || noise.containsMatchIn(t) || stepNumberOnly.matches(t) -> LineKind.Ignore
            ingredientsHeading.matches(t) -> { section = LineKind.Ingredient; LineKind.Ignore }
            stepsHeading.matches(t) -> { section = LineKind.Step; LineKind.Ignore }
            section != null -> section!!
            stepNumber.containsMatchIn(t) && t.length > 8 -> LineKind.Step
            quantityStart.containsMatchIn(t) && t.length <= 45 -> LineKind.Ingredient
            !titleSet && !seenIngredient && t.length <= 60 && !t.endsWith(".") -> LineKind.Title
            t.length > 45 || t.endsWith(".") -> LineKind.Step
            seenIngredient -> LineKind.Ingredient
            else -> LineKind.Ignore
        }
        if (kind == LineKind.Title) titleSet = true
        if (kind == LineKind.Ingredient) seenIngredient = true
        kind
    }
}

/**
 * Setzt Titel, Zutatenzeilen und Schritte zusammen. Ein Schritt ist ein
 * Absatz der Texterkennung; eine Nummer am Zeilenanfang („2.“) beginnt
 * ebenfalls einen neuen Schritt und wird entfernt. Am Zeilenende getrennte
 * Wörter („Zwie-“ / „beln“) werden wieder zusammengefügt.
 */
fun buildDraft(lines: List<OcrLine>, kinds: List<LineKind>): PhotoDraft {
    val title = lines.indices.filter { kinds[it] == LineKind.Title }.joinToString(" ") { lines[it].text.trim() }
    val ingredients = lines.indices.filter { kinds[it] == LineKind.Ingredient }.map { lines[it].text.trim() }.filter { it.isNotEmpty() }

    val steps = mutableListOf<String>()
    var current = StringBuilder()
    var lastBlock: Pair<Int, Int>? = null
    var lastIndex = -2
    fun finish() {
        if (current.isNotBlank()) steps += current.toString().trim()
        current = StringBuilder()
    }
    for (i in lines.indices) {
        if (kinds[i] != LineKind.Step) {
            // Eine reine Schrittnummer („3“) zwischen zwei Absätzen trennt auch.
            if (stepNumberOnly.matches(lines[i].text)) finish()
            continue
        }
        val block = lines[i].page to lines[i].block
        var text = lines[i].text.trim()
        val numbered = stepNumber.find(text)
        if (numbered != null) text = text.substring(numbered.range.last + 1).trim()
        // Neuer Schritt: nummeriert, anderer Absatz oder eine andere Zeile lag dazwischen.
        if (numbered != null || block != lastBlock || i != lastIndex + 1) finish()
        if (current.isEmpty()) {
            current.append(text)
        } else if (current.endsWith("-") && text.firstOrNull()?.isLowerCase() == true) {
            current.setLength(current.length - 1) // Trennstrich weg
            current.append(text)
        } else {
            current.append(' ').append(text)
        }
        lastBlock = block
        lastIndex = i
    }
    finish()
    return PhotoDraft(title, ingredients, steps)
}
