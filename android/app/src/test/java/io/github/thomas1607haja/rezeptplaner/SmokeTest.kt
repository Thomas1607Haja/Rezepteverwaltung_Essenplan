package io.github.thomas1607haja.rezeptplaner

import org.junit.Assert.assertEquals
import org.junit.Test

/** Erster Test, damit die Unit-Tests in der GitHub Action von Anfang an laufen. */
class SmokeTest {
    @Test
    fun versionNameHasThreeParts() {
        assertEquals(3, BuildConfig.VERSION_NAME.removeSuffix("-debug").split(".").size)
    }
}
