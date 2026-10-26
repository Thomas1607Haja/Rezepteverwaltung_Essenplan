package io.github.thomas1607haja.rezeptplaner.share

import io.github.thomas1607haja.rezeptplaner.api.Recipe
import io.github.thomas1607haja.rezeptplaner.api.Tag
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class PdfLayoutTest {

    private fun rows(vararg h: Float) = h.toList()

    /** Jede Zeile jedes Blocks steht genau einmal im Ergebnis, in Reihenfolge. */
    private fun assertComplete(blocks: List<PdfBlock>, pages: List<List<PdfSlice>>) {
        val placed = pages.flatten().flatMap { s -> (s.from until s.to).map { s.block to it } }
        val expected = blocks.flatMapIndexed { i, b -> b.rows.indices.map { i to it } }
        assertEquals(expected, placed)
    }

    /** Nichts ragt über den unteren Rand. */
    private fun assertFits(blocks: List<PdfBlock>, pages: List<List<PdfSlice>>, height: Float) {
        pages.flatten().forEach { s ->
            val bottom = s.y + blocks[s.block].rows.subList(s.from, s.to).sum()
            assertTrue("Block ${s.block} endet bei $bottom", bottom <= height + 0.01f)
        }
    }

    @Test
    fun shortRecipeOnOnePage() {
        val blocks = listOf(PdfBlock(rows(30f)), PdfBlock(rows(10f, 10f), spaceBefore = 5f))
        val pages = paginate(blocks, 100f)
        assertEquals(1, pages.size)
        assertEquals(PdfSlice(1, 0, 2, 35f), pages[0][1])
    }

    @Test
    fun blockThatDoesNotFitMovesToNextPage() {
        // Zweiter Block (Zutatengruppe) passt nicht mehr ganz: kommt ganz auf Seite 2.
        val blocks = listOf(PdfBlock(rows(70f)), PdfBlock(rows(10f, 10f, 10f, 10f), spaceBefore = 5f))
        val pages = paginate(blocks, 100f)
        assertEquals(2, pages.size)
        // Oben auf der Seite entfällt der Abstand.
        assertEquals(listOf(PdfSlice(1, 0, 4, 0f)), pages[1])
    }

    @Test
    fun headingStaysWithNextBlock() {
        val blocks = listOf(
            PdfBlock(rows(80f)),
            PdfBlock(rows(10f), keepWithNext = true), // „Zubereitung“
            PdfBlock(rows(12f, 12f)), // Schritt 1
        )
        val pages = paginate(blocks, 100f)
        assertEquals(2, pages.size)
        assertEquals(listOf(1, 2), pages[1].map { it.block })
    }

    @Test
    fun blockLongerThanAPageIsSplitBetweenRows() {
        val tall = PdfBlock(List(25) { 10f })
        val blocks = listOf(PdfBlock(rows(40f)), tall)
        val pages = paginate(blocks, 100f)
        assertComplete(blocks, pages)
        assertFits(blocks, pages, 100f)
        assertEquals(PdfSlice(1, 0, 6, 40f), pages[0][1])
        assertEquals(PdfSlice(1, 6, 16, 0f), pages[1][0])
    }

    @Test
    fun splittableBlockKeepsAtLeastTwoRows() {
        // Zutaten ohne Gruppe: nur eine Zeile hätte Platz – also gleich neue Seite.
        val blocks = listOf(PdfBlock(rows(85f)), PdfBlock(List(5) { 10f }, keepTogether = false))
        val pages = paginate(blocks, 100f)
        assertEquals(listOf(PdfSlice(1, 0, 5, 0f)), pages[1])
        // Mit Platz für zwei Zeilen wird geteilt.
        val split = paginate(listOf(PdfBlock(rows(78f)), PdfBlock(List(5) { 10f }, keepTogether = false)), 100f)
        assertEquals(PdfSlice(1, 0, 2, 78f), split[0][1])
        assertEquals(PdfSlice(1, 2, 5, 0f), split[1][0])
    }

    @Test
    fun longRecipeManyPages() {
        // Kopf, Bild, drei Zutatengruppen, 40 Schritte unterschiedlicher Länge.
        val blocks = buildList {
            add(PdfBlock(rows(30f, 30f), keepWithNext = true))
            add(PdfBlock(rows(287f), spaceBefore = 14f))
            add(PdfBlock(rows(20f), keepWithNext = true, spaceBefore = 22f))
            repeat(3) { add(PdfBlock(listOf(22f) + List(12) { 19f })) }
            add(PdfBlock(rows(20f), keepWithNext = true, spaceBefore = 22f))
            repeat(40) { i -> add(PdfBlock(List(1 + i % 6) { 14f }, spaceBefore = 7f)) }
        }
        val pages = paginate(blocks, 734f)
        assertTrue(pages.size >= 4)
        assertComplete(blocks, pages)
        assertFits(blocks, pages, 734f)
        // Kein Block, der auf eine Seite passt, ist geteilt.
        assertTrue(pages.flatten().all { it.from == 0 && it.to == blocks[it.block].rows.size })
        // Keine Überschrift unten auf einer Seite.
        pages.dropLast(1).forEach { page -> assertTrue(!blocks[page.last().block].keepWithNext) }
    }

    @Test
    fun fileNames() {
        assertEquals("Rezept_Romanesco-Tomaten-Pasta.pdf", shareFileName("Romanesco-Tomaten-Pasta"))
        assertEquals("Rezept_Omas_Käsekuchen_mit_Öl.pdf", shareFileName("Omas Käsekuchen mit Öl"))
        assertEquals("Rezept_Chili-sin-Carne.pdf", shareFileName("Chili / sin: Carne?"))
        assertEquals("Rezept_A-B.pdf", shareFileName("  \"A\" | <B>  "))
        assertEquals("Rezept.pdf", shareFileName("???"))
        assertTrue(shareFileName("x".repeat(200)).length <= "Rezept_.pdf".length + 80)
    }

    private fun recipe(vararg tags: Tag) = Recipe(id = 1, title = "T", baseServings = 4, tags = tags.toList())

    @Test
    fun metaWithCourseAndCategories() {
        val r = recipe(Tag(1, "Hauptgericht", "Gang"), Tag(2, "Pasta", "Art"), Tag(3, "Vegetarisch", "Ernährung"))
            .copy(prepMinutes = 20, totalMinutes = 75)
        assertEquals(
            listOf("Vorbereitung 20 Min. · Gesamt 1 Std. 15 Min.", "Gang: Hauptgericht", "Kategorien: Pasta, Vegetarisch"),
            pdfMeta(r, "Gang"),
        )
        // Ohne Gruppe „Gang“ sind alle Kategorien gleich.
        assertEquals(listOf("Kategorien: Hauptgericht"), pdfMeta(recipe(Tag(1, "Hauptgericht", "Gang")), ""))
        assertEquals(emptyList<String>(), pdfMeta(recipe(), "Gang"))
    }

    @Test
    fun source() {
        val r = recipe()
        assertNull(pdfSource(r))
        assertEquals("Quelle: Chefkoch\nhttps://www.chefkoch.de/r/1", pdfSource(r.copy(sourceName = "Chefkoch", sourceUrl = "https://www.chefkoch.de/r/1")))
        assertEquals("Quelle: www.chefkoch.de\nhttps://www.chefkoch.de/r/1", pdfSource(r.copy(sourceUrl = "https://www.chefkoch.de/r/1")))
        assertEquals("Quelle: Omas Kochbuch", pdfSource(r.copy(sourceName = "Omas Kochbuch", sourceUrl = "S. 12")))
    }
}
