package io.github.thomas1607haja.rezeptplaner.data

import kotlinx.serialization.Serializable
import java.net.URI

/**
 * Verbindung zum Rezeptplaner: Adresse des App-Zugangs (Port 8100, meist
 * über einen Cloudflare-Tunnel), API-Schlüssel des Geräts und optional ein
 * Service Token von Cloudflare Access. Siehe docs/APP-ZUGANG.md.
 */
@Serializable
data class ServerConfig(
    val url: String,
    val apiKey: String,
    val cfAccessId: String = "",
    val cfAccessSecret: String = "",
) {
    val hasCloudflareAccess: Boolean get() = cfAccessId.isNotBlank() && cfAccessSecret.isNotBlank()

    /** Nur der Hostname, z. B. für die Anzeige „verbunden mit rezepte.example.org“. */
    val host: String get() = runCatching { URI(url).host }.getOrNull() ?: url

    // Der Schlüssel gehört in kein Protokoll.
    override fun toString() = "ServerConfig(url=$url, apiKey=***, cfAccess=$hasCloudflareAccess)"
}

/** Fehler beim Lesen eines Einrichtungslinks oder einer Eingabe. */
class InvalidSetupException(message: String) : Exception(message)

/**
 * Liest und prüft die Angaben für eine Verbindung – aus dem QR-Code der
 * Weboberfläche oder aus der Eingabe von Hand.
 */
object SetupLink {
    const val SCHEME = "rezeptplaner"
    const val HOST = "setup"

    /** Format der API-Schlüssel (server/internal/apikeys): "rp_" + 43 Zeichen Base64url. */
    private val keyPattern = Regex("^rp_[A-Za-z0-9_-]{20,100}$")

    /**
     * Liest einen Link der Form
     * `rezeptplaner://setup?url=…&key=…[&cf_id=…&cf_secret=…]`
     * (erzeugt in web/src/lib/appSetup.js).
     */
    fun parse(text: String, allowLocalHttp: Boolean = false): ServerConfig {
        val uri = runCatching { URI(text.trim()) }.getOrNull()
        if (uri == null || uri.scheme != SCHEME || uri.host != HOST) {
            throw InvalidSetupException("Das ist kein QR-Code zum Verbinden des Rezeptplaners.")
        }
        val params = parseQuery(uri.rawQuery.orEmpty())
        return validate(
            url = params["url"].orEmpty(),
            apiKey = params["key"].orEmpty(),
            cfAccessId = params["cf_id"].orEmpty(),
            cfAccessSecret = params["cf_secret"].orEmpty(),
            allowLocalHttp = allowLocalHttp,
        )
    }

    /**
     * Prüft die Angaben und liefert sie bereinigt zurück. [allowLocalHttp]
     * (nur Debug-Build) erlaubt http für den Testserver auf dem
     * Entwicklungsrechner (10.0.2.2 aus dem Emulator).
     */
    fun validate(
        url: String,
        apiKey: String,
        cfAccessId: String = "",
        cfAccessSecret: String = "",
        allowLocalHttp: Boolean = false,
    ): ServerConfig {
        val cleanUrl = url.trim().trimEnd('/')
        val parsed = runCatching { URI(cleanUrl) }.getOrNull()
        val localHttp = allowLocalHttp && parsed?.scheme == "http" && parsed.host in setOf("10.0.2.2", "localhost")
        // Nur https: Der API-Schlüssel darf nicht unverschlüsselt durchs Netz.
        if (parsed == null || (parsed.scheme != "https" && !localHttp) || parsed.host.isNullOrEmpty() ||
            parsed.rawQuery != null || parsed.rawFragment != null || parsed.rawUserInfo != null
        ) {
            throw InvalidSetupException("Die Adresse muss mit https:// beginnen, z. B. https://rezepte.example.org.")
        }
        val key = apiKey.trim()
        if (!keyPattern.matches(key)) {
            throw InvalidSetupException("Der API-Schlüssel ist ungültig (er beginnt mit rp_).")
        }
        val id = cfAccessId.trim()
        val secret = cfAccessSecret.trim()
        if (id.isEmpty() != secret.isEmpty()) {
            throw InvalidSetupException("Für Cloudflare Access werden Client ID und Client Secret gebraucht.")
        }
        return ServerConfig(url = cleanUrl, apiKey = key, cfAccessId = id, cfAccessSecret = secret)
    }

    private fun parseQuery(query: String): Map<String, String> =
        query.split('&').filter { it.isNotEmpty() }.associate { pair ->
            val (name, value) = pair.split('=', limit = 2).let { it[0] to it.getOrElse(1) { "" } }
            decode(name) to decode(value)
        }

    // Die Variante mit Charset-Objekt gibt es erst ab Android 13 (minSdk ist 29).
    private fun decode(s: String): String = java.net.URLDecoder.decode(s, "UTF-8")
}
