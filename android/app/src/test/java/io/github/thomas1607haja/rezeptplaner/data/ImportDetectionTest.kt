package io.github.thomas1607haja.rezeptplaner.data

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class ImportDetectionTest {
    private fun bytes(s: String) = s.toByteArray(Charsets.UTF_8)

    @Test
    fun recognizesByContent() {
        assertTrue(ImportRepository.isImportFile(byteArrayOf(0x50, 0x4B, 0x03, 0x04, 1, 2)))
        assertTrue(ImportRepository.isImportFile(bytes(Char(0xFEFF).toString() + "  {\n \"format\": \"rezeptplaner-import\", \"version\": 1}")))
        assertFalse(ImportRepository.isImportFile(bytes("{\"name\": \"kein Rezept\"}")))
        // JPEG und Text
        assertFalse(ImportRepository.isImportFile(byteArrayOf(0xFF.toByte(), 0xD8.toByte(), 0xFF.toByte(), 0xE0.toByte())))
        assertFalse(ImportRepository.isImportFile(bytes("https://www.chefkoch.de/rezepte/123")))
        assertFalse(ImportRepository.isImportFile(ByteArray(0)))
    }
}
