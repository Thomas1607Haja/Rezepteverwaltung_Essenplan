package io.github.thomas1607haja.rezeptplaner.share

import io.github.thomas1607haja.rezeptplaner.api.Recipe
import io.github.thomas1607haja.rezeptplaner.ui.formatMinutes
import java.net.URI

// Rezept als PDF (docs/PLANUNG.md, Schritt 10.1): Seitenumbruch, Dateiname
// und Texte ohne Android-Grafik, damit sie sich ohne Gerät testen lassen.
// Das Zeichnen steht in RecipePdf.kt.

/**
 * Ein Abschnitt des PDFs aus Zeilen fester Höhe (in Punkt). Zwischen zwei
 * Zeilen darf die Seite umbrechen, innerhalb einer Zeile nie – so wird
 * keine Textzeile abgeschnitten.
 */
class PdfBlock(
    val rows: List<Float>,
    /** Ganz auf eine Seite, solange er auf eine Seite passt (Zutatengruppe, Schritt). */
    val keepTogether: Boolean = true,
    /** Nicht als Letztes auf einer Seite stehen lassen (Überschriften). */
    val keepWithNext: Boolean = false,
    /** Abstand zum vorigen Block; entfällt oben auf einer Seite. */
    val spaceBefore: Float = 0f,
) {
    val height: Float get() = rows.sum()
}

/** Teil eines Blocks auf einer Seite: Zeilen [from] bis vor [to], Oberkante bei [y]. */
data class PdfSlice(val block: Int, val from: Int, val to: Int, val y: Float)

/**
 * Verteilt die Blöcke auf Seiten mit [pageHeight] Platz. Ein Block, der
 * nicht mehr auf die Seite passt, beginnt auf der nächsten. Nur ein Block,
 * der allein höher als eine Seite ist (oder [PdfBlock.keepTogether] =
 * false hat), wird zwischen zwei Zeilen geteilt; dann bleiben mindestens
 * zwei Zeilen zusammen.
 */
fun paginate(blocks: List<PdfBlock>, pageHeight: Float): List<List<PdfSlice>> {
    val pages = mutableListOf(mutableListOf<PdfSlice>())
    var y = 0f
    blocks.forEachIndexed { i, block ->
        var from = 0
        while (from < block.rows.size) {
            val top = pages.last().isEmpty()
            val gap = if (top || from > 0) 0f else block.spaceBefore
            if (!top && y + gap + needed(blocks, i, from, pageHeight) > pageHeight + EPSILON) {
                pages += mutableListOf<PdfSlice>()
                y = 0f
                continue
            }
            y += gap
            val start = y
            var to = from
            while (to < block.rows.size && y + block.rows[to] <= pageHeight + EPSILON) {
                y += block.rows[to]
                to++
            }
            // Eine einzelne Zeile höher als die Seite kommt trotzdem (abgeschnitten
            // statt Endlosschleife); bei Text und dem verkleinerten Bild nicht möglich.
            if (to == from) y += block.rows[to++]
            pages.last() += PdfSlice(i, from, to, start)
            from = to
            if (from < block.rows.size) {
                pages += mutableListOf<PdfSlice>()
                y = 0f
            }
        }
    }
    return pages
}

private const val EPSILON = 0.01f

/** Platz, der ab Zeile [from] von Block [i] am Stück frei sein muss. */
private fun needed(blocks: List<PdfBlock>, i: Int, from: Int, pageHeight: Float): Float {
    val block = blocks[i]
    if (from > 0) return block.rows[from]
    fun start(b: PdfBlock) = if (b.keepTogether && b.height <= pageHeight) b.height else b.rows.take(2).sum()
    var need = start(block)
    if (block.keepWithNext) blocks.getOrNull(i + 1)?.let { next -> need += next.spaceBefore + start(next) }
    return need
}

/**
 * Dateiname aus dem Titel: „Rezept_Romanesco-Tomaten-Pasta.pdf“. Zeichen,
 * die Windows, Android oder Mailprogramme nicht mögen, werden zu „-“,
 * Leerzeichen zu „_“; Umlaute bleiben.
 */
fun shareFileName(title: String): String {
    val clean = title
        .replace(Regex("[\\\\/:*?\"<>|#%&{}\\p{Cntrl}\\u200B]"), "-")
        .replace(Regex("\\s*-[\\s-]*"), "-") // „Chili / sin“ -> „Chili-sin“
        .trim()
        .replace(Regex("\\s+"), "_")
        .trim('-', '_', '.')
        .take(80)
        .trimEnd('-', '_', '.')
    return if (clean.isEmpty()) "Rezept.pdf" else "Rezept_$clean.pdf"
}

/**
 * Zeilen unter dem Titel: Zeiten, Gang und Kategorien. [courseGroup] ist
 * der Name der Kategoriegruppe „Gang“ (Schritt 9.1), leer, wenn es keine gibt.
 */
fun pdfMeta(recipe: Recipe, courseGroup: String): List<String> = buildList {
    val times = listOfNotNull(
        recipe.prepMinutes?.let { "Vorbereitung ${formatMinutes(it)}" },
        recipe.totalMinutes?.let { "Gesamt ${formatMinutes(it)}" },
    )
    if (times.isNotEmpty()) add(times.joinToString(" · "))
    val (courses, others) = recipe.tags.partition { courseGroup.isNotEmpty() && it.group == courseGroup }
    if (courses.isNotEmpty()) add("Gang: " + courses.joinToString(", ") { it.name })
    if (others.isNotEmpty()) add("Kategorien: " + others.joinToString(", ") { it.name })
}

/** Quelle mit Webadresse, null ohne Quelle. */
fun pdfSource(recipe: Recipe): String? {
    val web = recipe.sourceUrl.trim().takeIf { it.startsWith("https://") || it.startsWith("http://") }
    return when {
        recipe.sourceName.isNotEmpty() && web != null -> "Quelle: ${recipe.sourceName}\n$web"
        recipe.sourceName.isNotEmpty() -> "Quelle: ${recipe.sourceName}"
        web != null -> "Quelle: " + (runCatching { URI(web).host }.getOrNull() ?: web).let { host -> if (host == web) web else "$host\n$web" }
        else -> null
    }
}
