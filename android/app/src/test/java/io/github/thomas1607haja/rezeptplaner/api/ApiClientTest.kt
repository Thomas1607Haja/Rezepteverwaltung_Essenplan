package io.github.thomas1607haja.rezeptplaner.api

import io.github.thomas1607haja.rezeptplaner.data.ServerConfig
import kotlinx.coroutines.test.runTest
import mockwebserver3.MockResponse
import mockwebserver3.MockWebServer
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

class ApiClientTest {
    private lateinit var server: MockWebServer
    private val api = ApiClient()

    @Before
    fun start() {
        server = MockWebServer()
        server.start()
    }

    @After
    fun stop() {
        server.close()
    }

    // MockWebServer spricht http; die Prüfung auf https steckt in SetupLink.
    private fun config(cf: Boolean = false) = ServerConfig(
        url = server.url("/").toString().trimEnd('/'),
        apiKey = "rp_test",
        cfAccessId = if (cf) "id.access" else "",
        cfAccessSecret = if (cf) "geheim" else "",
    )

    private suspend fun errorFor(response: MockResponse): String {
        server.enqueue(response)
        return try {
            api.system(config())
            "kein Fehler"
        } catch (e: ApiException) {
            (e as? ApiException)?.code ?: e.message.orEmpty()
        }
    }

    @Test
    fun sendsKeyAndCloudflareHeaders() = runTest {
        server.enqueue(MockResponse.Builder().body("""{"version":"0.6.1","api_version":1,"go_version":"x"}""").build())
        val info = api.system(config(cf = true))
        assertEquals("0.6.1", info.version)

        val request = server.takeRequest()
        assertEquals("/api/v1/system", request.url.encodedPath)
        assertEquals("Bearer rp_test", request.headers["Authorization"])
        assertEquals("id.access", request.headers["CF-Access-Client-Id"])
        assertEquals("geheim", request.headers["CF-Access-Client-Secret"])
    }

    @Test
    fun explainsErrors() = runTest {
        assertTrue(errorFor(MockResponse.Builder().code(401).build()).contains("client_key_invalid"))
        assertTrue(errorFor(MockResponse.Builder().code(404).build()).contains("client_not_rezeptplaner_or_old"))
        assertTrue(errorFor(MockResponse.Builder().code(404).body("404: Not Found").build()).contains("client_home_assistant_address"))
        assertTrue(
            errorFor(
                MockResponse.Builder().code(302)
                    .addHeader("Location", "https://team.cloudflareaccess.com/cdn-cgi/access/login").build(),
            ).contains("client_cloudflare_login"),
        )
        assertTrue(
            errorFor(MockResponse.Builder().code(403).addHeader("cf-ray", "abc").body("<html>Forbidden</html>").build())
                .contains("client_cloudflare_denied"),
        )
        assertTrue(errorFor(MockResponse.Builder().body("<html>Home Assistant</html>").build()).contains("client_not_rezeptplaner"))
        // Server vor 0.6.0 kennt api_version nicht.
        assertTrue(errorFor(MockResponse.Builder().body("""{"version":"0.5.1"}""").build()).contains("client_server_too_old"))
    }

    // Übertragung zurücksetzen (Server 0.18): Antwort wie vom Server, Anfrage mit Ersatzliste.
    @Test
    fun resetTransfer() = runTest {
        server.enqueue(
            MockResponse.Builder().body(
                """{"entries":[{"id":1,"date":"2026-11-07","slot":"dinner","recipe_id":1,"title":"Gulasch","servings":4,
                "menu_title":"","yield_unit":"","status":"transferred","included":false,"lists":[],"unknown_list":true,"items":3}],
                "current_list":{"entity_id":"todo.familie","name":"Einkauf Familie"},
                "lists":[{"entity_id":"todo.test","name":"Test"},{"entity_id":"todo.familie","name":"Einkauf Familie"}]}""",
            ).build(),
        )
        val preview = api.resetPreview(config(), next = false, entryId = 1)
        val first = server.takeRequest()
        assertEquals("/api/v1/shopping/reset", first.url.encodedPath)
        assertEquals("1", first.url.queryParameter("entry"))
        assertTrue(preview.entries.single().unknownList)
        assertEquals("Einkauf Familie", preview.currentList.name)
        assertEquals(2, preview.lists.size)

        server.enqueue(MockResponse.Builder().body("""{"entries":1,"removed":["Eier"],"reduced":[],"not_found":[],"failed":[],"skipped":[]}""").build())
        val result = api.resetTransfer(config(), ResetRequest(listOf(1), remove = true, fallbackList = "todo.test"))
        assertEquals(listOf("Eier"), result.removed)
        val request = server.takeRequest()
        assertEquals("/api/v1/shopping/reset", request.url.encodedPath)
        assertEquals("""{"entries":[1],"remove":true,"fallback_list":"todo.test"}""", request.body?.utf8())
    }
}
