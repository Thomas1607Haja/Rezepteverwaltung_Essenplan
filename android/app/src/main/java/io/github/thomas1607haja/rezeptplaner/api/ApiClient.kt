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

/** Fehler beim Zugriff auf den Rezeptplaner, mit Meldung für die Anzeige. */
class ApiException(message: String, cause: Throwable? = null) : Exception(message, cause)

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

    private suspend inline fun <reified T> get(config: ServerConfig, path: String): T =
        withContext(Dispatchers.IO) {
            val request = Request.Builder()
                .url(config.url + path)
                .header("Authorization", "Bearer ${config.apiKey}")
                .header("Accept", "application/json")
                .apply {
                    if (config.hasCloudflareAccess) {
                        header("CF-Access-Client-Id", config.cfAccessId)
                        header("CF-Access-Client-Secret", config.cfAccessSecret)
                    }
                }
                .build()
            val body = try {
                client.newCall(request).execute().use { response ->
                    checkStatus(response)
                    response.body.string()
                }
            } catch (e: ApiException) {
                throw e
            } catch (e: UnknownHostException) {
                throw ApiException("Server nicht gefunden – Adresse und Internetverbindung prüfen.", e)
            } catch (e: SSLException) {
                throw ApiException("Sichere Verbindung fehlgeschlagen (Zertifikat).", e)
            } catch (e: IOException) {
                throw ApiException("Server nicht erreichbar – Internetverbindung prüfen.", e)
            }
            try {
                json.decodeFromString<T>(body)
            } catch (e: Exception) {
                throw ApiException("Unter dieser Adresse antwortet kein Rezeptplaner.", e)
            }
        }

    private fun checkStatus(response: Response) {
        if (response.isSuccessful) return
        val location = response.header("Location").orEmpty()
        throw ApiException(
            when {
                response.isRedirect && "cloudflareaccess.com" in location ->
                    "Cloudflare Access verlangt eine Anmeldung – Service Token (Client ID/Secret) fehlt oder ist falsch."
                response.code == 403 && response.header("cf-ray") != null && response.header("content-type")?.startsWith("application/json") != true ->
                    "Cloudflare Access hat die Anfrage abgelehnt – Service Token prüfen."
                response.code == 401 -> "Der API-Schlüssel ist ungültig oder gesperrt. Gerät in der Weboberfläche neu verbinden."
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
