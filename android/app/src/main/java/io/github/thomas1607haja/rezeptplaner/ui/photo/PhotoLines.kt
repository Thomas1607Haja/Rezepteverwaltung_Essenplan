package io.github.thomas1607haja.rezeptplaner.ui.photo

/** Eine erkannte Textzeile; [block] ist der Absatz der Texterkennung auf Seite [page]. */
data class OcrLine(val text: String, val page: Int, val block: Int)

/** Wozu eine Zeile gehört. */
enum class LineKind(val label: String) {
    Title("Titel"),
    Ingredient("Zutat"),
    Step("Schritt"),
    Ignore("Ignorieren"),
}

/** Ergebnis der Zuordnung, bevor der Server die Zutaten zerlegt. */
data class PhotoDraft(val title: String, val ingredientLines: List<String>, val steps: List<String>)

private val ingredientsHeading = Regex("""^(zutaten(liste)?|einkaufsliste|das brauchst du( noch)?|für \d+ (personen|portionen))\b.*""", RegexOption.IGNORE_CASE)
private val stepsHeading = Regex("""^(zubereitung|so geht'?s|anleitung|und so wird'?s gemacht|arbeitsschritte)\b.*""", RegexOption.IGNORE_CASE)
private val quantityStart = Regex("""^(\d|[½¼¾⅓⅔⅛]|eine?\b|etwas\b|evtl\.|prise\b|n\.\s?b\.|nach belieben)""", RegexOption.IGNORE_CASE)
// „1.“, „2)“, „Schritt 3“ – nicht „200 g“ oder „1,5 l“.
private val stepNumber = Regex("""^\s*(?:schritt\s*\d{1,2}\s*[.):]?|\d{1,2}\s*[.):])\s*""", RegexOption.IGNORE_CASE)
private val stepNumberOnly = Regex("""^\s*(?:schritt\s*)?\d{1,2}\s*[.):]?\s*$""", RegexOption.IGNORE_CASE)
private val noise = Regex("""^(www\.|https?://|©|seite \d+|\d+$|[^\p{L}\d]*$)""", RegexOption.IGNORE_CASE)

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
