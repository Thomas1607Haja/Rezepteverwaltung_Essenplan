package io.github.thomas1607haja.rezeptplaner.share

import android.graphics.Bitmap
import android.graphics.BitmapFactory
import android.graphics.Canvas
import android.graphics.Paint
import android.graphics.Path
import android.graphics.Rect
import android.graphics.RectF
import android.graphics.Typeface
import android.graphics.pdf.PdfDocument
import android.graphics.text.LineBreaker
import android.text.Layout
import android.text.StaticLayout
import android.text.TextPaint
import androidx.core.graphics.createBitmap
import androidx.core.graphics.withClip
import androidx.core.graphics.withTranslation
import io.github.thomas1607haja.rezeptplaner.api.Recipe
import io.github.thomas1607haja.rezeptplaner.ui.amountText
import io.github.thomas1607haja.rezeptplaner.ui.formatQuantity
import io.github.thomas1607haja.rezeptplaner.ui.wrapTitle
import java.io.File
import java.io.OutputStream
import java.util.Locale
import kotlin.math.max

/** Was ins PDF kommt. [image]: Datei des Rezeptbilds, null = ohne Bild. */
class PdfInput(
    val recipe: Recipe,
    val image: File?,
    val withNotes: Boolean,
    /** Titel des Ursprungsrezepts, wenn das Rezept eine Variante ist. */
    val parentTitle: String? = null,
    /** Name der Kategoriegruppe „Gang“ (leer, wenn es keine gibt). */
    val courseGroup: String = "",
)

/**
 * Zeichnet ein Rezept als PDF im Format DIN A4 mit der eingebauten
 * PdfDocument-Schnittstelle (ohne Zusatzbibliothek). Die Schrift ist die
 * Systemschrift; sie wird eingebettet, damit Umlaute, ß und Brüche (½)
 * überall richtig erscheinen. Mengen immer für die Grundportionen des
 * Rezepts. Gibt die Zahl der Seiten zurück.
 */
fun writeRecipePdf(input: PdfInput, out: OutputStream): Int {
    val image = loadImage(input.image)
    var blocks = RecipePdfBlocks(input, image, IMAGE_HEIGHT).build()
    var pages = paginate(blocks.map { it.block }, CONTENT_BOTTOM - MARGIN)
    // Stünde auf der letzten Seite nur ein kleiner Rest (etwa die Quelle),
    // wird das Bild so viel niedriger, dass eine Seite weniger reicht.
    // Oben auf der Seite entfällt der Abstand des ersten Blocks; zurück auf
    // der vorigen Seite braucht er ihn wieder.
    val rest = pages.last().maxOf { s -> s.y + blocks[s.block].block.rows.subList(s.from, s.to).sum() } +
        blocks[pages.last().first().block].block.spaceBefore
    if (image != null && pages.size > 1 && rest <= IMAGE_HEIGHT - MIN_IMAGE_HEIGHT) {
        val smaller = RecipePdfBlocks(input, image, IMAGE_HEIGHT - rest - 4f).build()
        val fewer = paginate(smaller.map { it.block }, CONTENT_BOTTOM - MARGIN)
        if (fewer.size < pages.size) {
            blocks = smaller
            pages = fewer
        }
    }
    val doc = PdfDocument()
    try {
        pages.forEachIndexed { index, slices ->
            val page = doc.startPage(PdfDocument.PageInfo.Builder(PAGE_WIDTH, PAGE_HEIGHT, index + 1).create())
            val canvas = page.canvas
            slices.forEach { slice ->
                val block = blocks[slice.block]
                canvas.withTranslation(MARGIN, MARGIN + slice.y) {
                    for (row in slice.from until slice.to) {
                        block.draw(this, row)
                        translate(0f, block.block.rows[row])
                    }
                }
            }
            drawFooter(canvas, index + 1, pages.size)
            doc.finishPage(page)
        }
        doc.writeTo(out)
    } finally {
        doc.close()
        image?.recycle()
    }
    return pages.size
}

// Maße in Punkt (1/72 Zoll): A4 = 595 × 842, Rand etwa 15 mm.
private const val PAGE_WIDTH = 595
private const val PAGE_HEIGHT = 842
private const val MARGIN = 42f
private const val CONTENT_WIDTH = PAGE_WIDTH - 2 * MARGIN
private const val FOOTER_HEIGHT = 26f
private const val CONTENT_BOTTOM = PAGE_HEIGHT - MARGIN - FOOTER_HEIGHT

// Bild: 16:9 über die volle Breite, 800 Pixel breit. PdfDocument speichert
// Bilder verlustfrei; die kleine Pixelzahl hält die Datei unter 1 MB.
private const val IMAGE_PIXELS_WIDE = 800
private const val IMAGE_PIXELS_HIGH = 450
private const val IMAGE_HEIGHT = CONTENT_WIDTH * IMAGE_PIXELS_HIGH / IMAGE_PIXELS_WIDE
/** Niedriger wird das Bild nicht, um eine Seite zu sparen (etwa 3:1). */
private const val MIN_IMAGE_HEIGHT = 170f

// Farben der App (ui/theme/Theme.kt, helles Schema): druckfreundlich,
// nur Text und feine Linien auf Weiß.
private const val INK = 0xFF1F1D1A.toInt()
private const val MUTED = 0xFF6B665E.toInt()
private const val ACCENT = 0xFFC2552D.toInt()
private const val LINE = 0xFFE2DFD8.toInt()

private const val AMOUNT_WIDTH = 82f
private const val COLUMN_GAP = 12f
private const val NUMBER_WIDTH = 24f

/** Ein Block mit der Zeichenanweisung je Zeile (Ursprung = linke obere Ecke der Zeile). */
private class DrawBlock(val block: PdfBlock, val draw: (Canvas, Int) -> Unit)

private fun paint(size: Float, color: Int = INK, weight: Int = 400) = TextPaint(Paint.ANTI_ALIAS_FLAG).apply {
    textSize = size
    this.color = color
    typeface = Typeface.create(Typeface.DEFAULT, weight, false)
    textLocale = Locale.GERMANY // Silbentrennung nach deutschen Regeln
}

private fun layout(text: CharSequence, paint: TextPaint, width: Float, align: Layout.Alignment = Layout.Alignment.ALIGN_NORMAL) =
    StaticLayout.Builder.obtain(text, 0, text.length, paint, width.toInt())
        .setAlignment(align)
        .setLineSpacing(0f, 1.15f)
        .setIncludePad(false)
        .setBreakStrategy(LineBreaker.BREAK_STRATEGY_HIGH_QUALITY)
        .setHyphenationFrequency(Layout.HYPHENATION_FREQUENCY_NORMAL)
        .build()

private fun Layout.lineHeight(line: Int) = (getLineBottom(line) - getLineTop(line)).toFloat()

/** Zeichnet nur Zeile [line] des Layouts an den Ursprung. */
private fun Layout.drawLine(canvas: Canvas, line: Int) {
    canvas.withClip(0f, 0f, width.toFloat(), lineHeight(line)) {
        withTranslation(0f, -getLineTop(line).toFloat()) { draw(this) }
    }
}

/** [imageHeight]: Höhe des Bilds im PDF; ist sie kleiner als 16:9, zeigt es einen mittigen Streifen. */
private class RecipePdfBlocks(private val input: PdfInput, private val image: Bitmap?, private val imageHeight: Float) {
    private val recipe = input.recipe
    private val blocks = mutableListOf<DrawBlock>()

    private val title = paint(22f, weight = 700)
    private val heading = paint(14f, ACCENT, 600)
    private val group = paint(11f, weight = 600)
    private val body = paint(10.5f)
    private val bodyBold = paint(10.5f, weight = 600)
    private val number = paint(10.5f, ACCENT, 700)
    private val meta = paint(9.5f, MUTED)
    private val variant = paint(9.5f, ACCENT)
    private val line = Paint(Paint.ANTI_ALIAS_FLAG).apply { color = LINE; strokeWidth = 0.6f }
    private val accentLine = Paint(Paint.ANTI_ALIAS_FLAG).apply { color = ACCENT; strokeWidth = 0.8f }

    fun build(): List<DrawBlock> {
        // Titel, „Variante von …“, Zeiten/Gang/Kategorien bleiben zusammen.
        val head = layout(wrapTitle(recipe.title), title, CONTENT_WIDTH)
        val parent = input.parentTitle?.let { layout("Variante von $it", variant, CONTENT_WIDTH) }
        val metaLines = pdfMeta(recipe, input.courseGroup)
        val metaLayout = metaLines.takeIf { it.isNotEmpty() }?.let { layout(it.joinToString("\n"), meta, CONTENT_WIDTH) }
        textBlock(head, keepWithNext = true)
        parent?.let { textBlock(it, spaceBefore = 2f, keepWithNext = true) }
        metaLayout?.let { textBlock(it, spaceBefore = 6f) }

        image?.let { bitmap ->
            val clip = Path().apply { addRoundRect(RectF(0f, 0f, CONTENT_WIDTH, imageHeight), 6f, 6f, Path.Direction.CW) }
            val visible = (bitmap.height * imageHeight / IMAGE_HEIGHT).toInt()
            val source = Rect(0, (bitmap.height - visible) / 2, bitmap.width, (bitmap.height + visible) / 2)
            val filter = Paint(Paint.FILTER_BITMAP_FLAG)
            add(PdfBlock(listOf(imageHeight), spaceBefore = 14f)) { canvas, _ ->
                canvas.withClip(clip) { drawBitmap(bitmap, source, RectF(0f, 0f, CONTENT_WIDTH, imageHeight), filter) }
            }
        }

        if (recipe.description.isNotEmpty()) {
            textBlock(layout(recipe.description, body, CONTENT_WIDTH), spaceBefore = 12f, keepTogether = false)
        }

        if (recipe.ingredients.isNotEmpty()) {
            sectionHeading("Zutaten", "für " + amountText(recipe.baseServings, recipe.yieldUnit))
            ingredients()
        }
        if (recipe.steps.isNotEmpty()) {
            sectionHeading("Zubereitung")
            recipe.steps.forEachIndexed { i, step -> step(i + 1, step.text) }
        }
        if (input.withNotes && recipe.notes.isNotBlank()) {
            sectionHeading("Notizen")
            textBlock(layout(recipe.notes.trim(), body, CONTENT_WIDTH), spaceBefore = 6f, keepTogether = false)
        }
        pdfSource(recipe)?.let { textBlock(layout(it, meta, CONTENT_WIDTH), spaceBefore = 20f) }
        return blocks
    }

    private fun add(block: PdfBlock, draw: (Canvas, Int) -> Unit) {
        blocks += DrawBlock(block, draw)
    }

    /** Text, zeilenweise teilbar. */
    private fun textBlock(layout: StaticLayout, spaceBefore: Float = 0f, keepTogether: Boolean = true, keepWithNext: Boolean = false) {
        val rows = (0 until layout.lineCount).map(layout::lineHeight)
        add(PdfBlock(rows, keepTogether, keepWithNext, spaceBefore)) { canvas, row -> layout.drawLine(canvas, row) }
    }

    /** „Zutaten  für 4 Portionen“ mit Linie in der Akzentfarbe darunter. */
    private fun sectionHeading(text: String, extra: String = "") {
        val left = layout(text, heading, CONTENT_WIDTH)
        val right = extra.takeIf { it.isNotEmpty() }?.let { layout(it, meta, CONTENT_WIDTH, Layout.Alignment.ALIGN_OPPOSITE) }
        val height = left.height + 6f
        add(PdfBlock(listOf(height), keepWithNext = true, spaceBefore = 22f)) { canvas, _ ->
            left.draw(canvas)
            right?.let {
                // Grundlinie wie die Überschrift.
                canvas.withTranslation(0f, (left.getLineBaseline(0) - it.getLineBaseline(0)).toFloat()) { it.draw(this) }
            }
            canvas.drawLine(0f, height - 2f, CONTENT_WIDTH, height - 2f, accentLine)
        }
    }

    /**
     * Zutaten: je Gruppe ein Block, der möglichst auf einer Seite bleibt;
     * Zutaten ohne Gruppe dürfen zwischen zwei Zutaten umbrechen.
     */
    private fun ingredients() {
        val groups = mutableListOf<Pair<String, MutableList<Int>>>()
        recipe.ingredients.forEachIndexed { i, ing ->
            if (groups.isEmpty() || groups.last().first != ing.groupName) groups += ing.groupName to mutableListOf()
            groups.last().second += i
        }
        groups.forEachIndexed { index, (name, items) ->
            val rows = mutableListOf<Float>()
            val draws = mutableListOf<(Canvas) -> Unit>()
            if (name.isNotEmpty()) {
                val l = layout(name, group, CONTENT_WIDTH)
                val top = if (index == 0) 4f else 10f
                rows += top + l.height + 2f
                draws += { canvas -> canvas.withTranslation(0f, top) { l.draw(this) } }
            }
            items.forEach { i ->
                val ing = recipe.ingredients[i]
                val amountText = listOf(formatQuantity(ing.quantity), ing.unit).filter { it.isNotEmpty() }.joinToString(" ")
                val amount = layout(amountText, bodyBold, AMOUNT_WIDTH, Layout.Alignment.ALIGN_OPPOSITE)
                val label = ing.name + if (ing.note.isNotEmpty()) ", ${ing.note}" else ""
                val nameLayout = layout(label, body, CONTENT_WIDTH - AMOUNT_WIDTH - COLUMN_GAP)
                val pad = 3.5f
                val height = max(amount.height, nameLayout.height) + 2 * pad
                rows += height
                draws += { canvas ->
                    canvas.withTranslation(0f, pad) {
                        amount.draw(this)
                        withTranslation(AMOUNT_WIDTH + COLUMN_GAP, 0f) { nameLayout.draw(this) }
                    }
                    canvas.drawLine(0f, height, CONTENT_WIDTH, height, line)
                }
            }
            add(PdfBlock(rows, keepTogether = name.isNotEmpty(), spaceBefore = if (index == 0) 4f else 0f)) { canvas, row ->
                draws[row](canvas)
            }
        }
    }

    /** Ein Schritt: Nummer in der Akzentfarbe, Text daneben; bleibt möglichst zusammen. */
    private fun step(n: Int, text: String) {
        val textLayout = layout(text, body, CONTENT_WIDTH - NUMBER_WIDTH)
        val num = layout("$n.", number, NUMBER_WIDTH)
        val rows = (0 until textLayout.lineCount).map(textLayout::lineHeight)
        add(PdfBlock(rows, spaceBefore = if (n == 1) 8f else 7f)) { canvas, row ->
            if (row == 0) num.drawLine(canvas, 0)
            canvas.withTranslation(NUMBER_WIDTH, 0f) { textLayout.drawLine(this, row) }
        }
    }
}

/** „Erstellt mit Rezeptplaner“ links, Seitenzahl rechts, feine Linie darüber. */
private fun drawFooter(canvas: Canvas, page: Int, pages: Int) {
    val small = paint(8f, MUTED)
    val y = PAGE_HEIGHT - MARGIN
    canvas.drawLine(MARGIN, y - 14f, PAGE_WIDTH - MARGIN, y - 14f, Paint().apply { color = LINE; strokeWidth = 0.5f })
    canvas.drawText("Erstellt mit Rezeptplaner", MARGIN, y, small)
    val label = "Seite $page von $pages"
    canvas.drawText(label, PAGE_WIDTH - MARGIN - small.measureText(label), y, small)
}

/**
 * Lädt das Rezeptbild verkleinert auf 800 × 450 Pixel (mittiger
 * Ausschnitt im Format 16:9). null, wenn die Datei fehlt oder kaputt ist.
 */
private fun loadImage(file: File?): Bitmap? {
    if (file == null || !file.exists()) return null
    return runCatching {
        val bounds = BitmapFactory.Options().apply { inJustDecodeBounds = true }
        BitmapFactory.decodeFile(file.path, bounds)
        if (bounds.outWidth <= 0 || bounds.outHeight <= 0) return null
        var sample = 1
        while (bounds.outWidth / (sample * 2) >= IMAGE_PIXELS_WIDE && bounds.outHeight / (sample * 2) >= IMAGE_PIXELS_HIGH) sample *= 2
        val source = BitmapFactory.decodeFile(file.path, BitmapFactory.Options().apply { inSampleSize = sample }) ?: return null
        // Mittiger Ausschnitt 16:9.
        val wanted = IMAGE_PIXELS_WIDE.toFloat() / IMAGE_PIXELS_HIGH
        val crop = if (source.width.toFloat() / source.height > wanted) {
            val w = (source.height * wanted).toInt()
            Rect((source.width - w) / 2, 0, (source.width + w) / 2, source.height)
        } else {
            val h = (source.width / wanted).toInt()
            Rect(0, (source.height - h) / 2, source.width, (source.height + h) / 2)
        }
        val target = createBitmap(IMAGE_PIXELS_WIDE, IMAGE_PIXELS_HIGH)
        // Ohne Alphakanal: sonst legt das PDF zusätzlich eine Maske an.
        target.setHasAlpha(false)
        Canvas(target).drawBitmap(source, crop, Rect(0, 0, IMAGE_PIXELS_WIDE, IMAGE_PIXELS_HIGH), Paint(Paint.FILTER_BITMAP_FLAG))
        source.recycle()
        target
    }.getOrNull()
}
