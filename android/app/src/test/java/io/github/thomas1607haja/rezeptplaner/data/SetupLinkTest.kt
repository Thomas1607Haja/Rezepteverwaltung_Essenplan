package io.github.thomas1607haja.rezeptplaner.data

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Assert.fail
import org.junit.Test

class SetupLinkTest {
    private val key = "rp_5r13s154siEAjGyfI9GC0ah52Z20_LXDK8f0soq-NDg"

    // So erzeugt die Weboberfläche den Link (web/src/lib/appSetup.js, URLSearchParams).
    private fun link(vararg params: Pair<String, String>) =
        "rezeptplaner://setup?" + params.joinToString("&") { (k, v) ->
            k + "=" + java.net.URLEncoder.encode(v, Charsets.UTF_8)
        }

    @Test
    fun parsesLinkFromWebInterface() {
        val config = SetupLink.parse(link("url" to "https://rezepte.example.org", "key" to key))
        assertEquals("https://rezepte.example.org", config.url)
        assertEquals(key, config.apiKey)
        assertFalse(config.hasCloudflareAccess)
        assertEquals("rezepte.example.org", config.host)
    }

    @Test
    fun parsesCloudflareAccess() {
        val config = SetupLink.parse(
            link("url" to "https://rezepte.example.org/", "key" to key, "cf_id" to "abc.access", "cf_secret" to "s+e/c=r"),
        )
        assertEquals("https://rezepte.example.org", config.url)
        assertTrue(config.hasCloudflareAccess)
        assertEquals("s+e/c=r", config.cfAccessSecret)
    }

    @Test
    fun rejectsInvalidInput() {
        val bad = listOf(
            "https://rezepte.example.org",
            "rezeptplaner://andere?url=https%3A%2F%2Fa.de&key=$key",
            link("url" to "http://rezepte.example.org", "key" to key),
            link("url" to "https://user:pw@rezepte.example.org", "key" to key),
            link("url" to "https://rezepte.example.org", "key" to "falsch"),
            link("url" to "https://rezepte.example.org", "key" to key, "cf_id" to "nur-id"),
            "kein link",
        )
        for (text in bad) {
            try {
                SetupLink.parse(text)
                fail("angenommen: $text")
            } catch (_: InvalidSetupException) {
            }
        }
    }

    @Test
    fun keyIsNotInToString() {
        val config = SetupLink.validate("https://a.example.org", key)
        assertFalse(config.toString().contains(key))
    }

    @Test
    fun localHttpOnlyWhenAllowed() {
        val text = link("url" to "http://10.0.2.2:8100", "key" to key)
        assertEquals("http://10.0.2.2:8100", SetupLink.parse(text, allowLocalHttp = true).url)
        try {
            SetupLink.parse(text)
            fail("http ohne Freigabe angenommen")
        } catch (_: InvalidSetupException) {
        }
        try {
            SetupLink.parse(link("url" to "http://rezepte.example.org", "key" to key), allowLocalHttp = true)
            fail("http für fremde Adresse angenommen")
        } catch (_: InvalidSetupException) {
        }
    }
}
