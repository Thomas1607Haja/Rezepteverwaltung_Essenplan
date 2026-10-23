package io.github.thomas1607haja.rezeptplaner.api

import io.github.thomas1607haja.rezeptplaner.data.ServerConfig
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.Response
import java.io.IOException
import java.net.UnknownHostException
import java.util.concurrent.TimeUnit
import javax.net.ssl.SSLException

/** Version der Server-Schnittstelle, die diese App kennt (server/internal/system, APIVersion). */
const val SUPPORTED_API_VERSION = 1

/** Antwort von GET /api/v1/system (nur die Felder, die die App braucht). */
@Serializable
data class SystemInfo(
    val version: String,
    @SerialName("api_version") val apiVersion: Int = 0,
)

/**
 * Fehler beim Zugriff auf den Rezeptplaner, mit Meldung für die Anzeige.
 * [offline]: keine Verbindung (Netz weg, Server nicht erreichbar) – dann
 * zeigt die App den gespeicherten Stand mit Hinweis.
 */
class ApiException(message: String, cause: Throwable? = null, val offline: Boolean = false) : Exception(message, cause)

/**
 * Zugriff auf die REST-API des Rezeptplaners über den App-Zugang. Jede
 * Anfrage trägt den API-Schlüssel und – falls eingerichtet – das Service
 * Token von Cloudflare Access.
 */
class ApiClient(baseClient: OkHttpClient = defaultClient) {

    // Weiterleitungen nicht folgen: Cloudflare Access leitet ohne gültiges
    // Token auf eine Anmeldeseite um; das soll als klarer Fehler ankommen
    // und nicht als unlesbares HTML.
    private val client = baseClient.newBuilder().followRedirects(false).followSslRedirects(false).build()

    /** Prüft die Verbindung: Server erreichbar, Schlüssel gültig, Schnittstelle passend. */
    suspend fun system(config: ServerConfig): SystemInfo {
        val info: SystemInfo = get(config, "/api/v1/system")
        if (info.apiVersion < SUPPORTED_API_VERSION) {
            throw ApiException(
                "Der Rezeptplaner auf dem Server ist zu alt (Version ${info.version}). " +
                    "Bitte in Home Assistant aktualisieren.",
            )
        }
        return info
    }

    /**
     * Änderungen seit [since] (server_time des letzten Abgleichs; null =
     * alles), siehe server/internal/appsync.
     */
    suspend fun sync(config: ServerConfig, since: String?): SyncResponse {
        val query = since?.let { "?since=" + java.net.URLEncoder.encode(it, "UTF-8") }.orEmpty()
        return get(config, "/api/v1/sync$query")
    }

    /** Lädt eine Bilddatei (Name aus Recipe.image bzw. thumbnail). */
    suspend fun image(config: ServerConfig, name: String): ByteArray =
        withContext(Dispatchers.IO) { execute(config, "/api/v1/images/$name", "image/jpeg") }

    private suspend inline fun <reified T> get(config: ServerConfig, path: String): T =
        withContext(Dispatchers.IO) {
            val body = execute(config, path, "application/json").decodeToString()
            try {
                json.decodeFromString<T>(body)
            } catch (e: Exception) {
                throw ApiException("Unter dieser Adresse antwortet kein Rezeptplaner.", e)
            }
        }

    /** Führt eine GET-Anfrage aus und liefert den Inhalt; Fehler als [ApiException]. */
    private fun execute(config: ServerConfig, path: String, accept: String): ByteArray {
        val request = Request.Builder()
            .url(config.url + path)
            .header("Authorization", "Bearer ${config.apiKey}")
            .header("Accept", accept)
            .apply {
                if (config.hasCloudflareAccess) {
                    header("CF-Access-Client-Id", config.cfAccessId)
                    header("CF-Access-Client-Secret", config.cfAccessSecret)
                }
            }
            .build()
        return try {
            client.newCall(request).execute().use { response ->
                checkStatus(response)
                response.body.bytes()
            }
        } catch (e: ApiException) {
            throw e
        } catch (e: UnknownHostException) {
            throw ApiException("Server nicht gefunden – Adresse und Internetverbindung prüfen.", e, offline = true)
        } catch (e: SSLException) {
            throw ApiException("Sichere Verbindung fehlgeschlagen (Zertifikat).", e)
        } catch (e: IOException) {
            throw ApiException("Server nicht erreichbar – Internetverbindung prüfen.", e, offline = true)
        }
    }

    private fun checkStatus(response: Response) {
        if (response.isSuccessful) return
        val location = response.header("Location").orEmpty()
        // Anfang der Antwort, um Home Assistant zu erkennen (siehe unten).
        val start = runCatching { response.peekBody(64).string().trim() }.getOrDefault("")
        throw ApiException(
            when {
                response.isRedirect && "cloudflareaccess.com" in location ->
                    "Cloudflare Access verlangt eine Anmeldung – Service Token (Client ID/Secret) fehlt oder ist falsch."
                response.code == 403 && response.header("cf-ray") != null && response.header("content-type")?.startsWith("application/json") != true ->
                    "Cloudflare Access hat die Anfrage abgelehnt – Service Token prüfen."
                response.code == 401 -> "Der API-Schlüssel ist ungültig oder gesperrt. Gerät in der Weboberfläche neu verbinden."
                // So antwortet Home Assistant (aiohttp) auf unbekannte Pfade – die
                // Adresse von Home Assistant statt der des App-Zugangs.
                response.code == 404 && start.startsWith("404: Not Found") ->
                    "Diese Adresse führt zu Home Assistant, nicht zum App-Zugang des Rezeptplaners. " +
                        "In der Weboberfläche die Adresse für die App prüfen (eigener Hostname, Port 8100)."
                response.code == 404 -> "Unter dieser Adresse antwortet kein Rezeptplaner (oder er ist zu alt)."
                response.isRedirect -> "Der Server leitet um ($location) – Adresse prüfen."
                response.code in 500..599 -> "Fehler auf dem Server (${response.code}). Später erneut versuchen."
                else -> "Unerwartete Antwort vom Server (${response.code})."
            },
        )
    }

    companion object {
        private val json = Json { ignoreUnknownKeys = true }

        private val defaultClient: OkHttpClient = OkHttpClient.Builder()
            .connectTimeout(15, TimeUnit.SECONDS)
            .readTimeout(30, TimeUnit.SECONDS)
            .build()
    }
}
