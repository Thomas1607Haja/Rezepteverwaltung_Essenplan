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
            e.message.orEmpty()
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
        assertTrue(errorFor(MockResponse.Builder().code(401).build()).contains("ungültig oder gesperrt"))
        assertTrue(errorFor(MockResponse.Builder().code(404).build()).contains("kein Rezeptplaner"))
        assertTrue(errorFor(MockResponse.Builder().code(404).body("404: Not Found").build()).contains("Home Assistant"))
        assertTrue(
            errorFor(
                MockResponse.Builder().code(302)
                    .addHeader("Location", "https://team.cloudflareaccess.com/cdn-cgi/access/login").build(),
            ).contains("Cloudflare Access"),
        )
        assertTrue(
            errorFor(MockResponse.Builder().code(403).addHeader("cf-ray", "abc").body("<html>Forbidden</html>").build())
                .contains("Cloudflare Access"),
        )
        assertTrue(errorFor(MockResponse.Builder().body("<html>Home Assistant</html>").build()).contains("kein Rezeptplaner"))
        // Server vor 0.6.0 kennt api_version nicht.
        assertTrue(errorFor(MockResponse.Builder().body("""{"version":"0.5.1"}""").build()).contains("zu alt"))
    }
}
