package io.github.thomas1607haja.rezeptplaner.api

import io.github.thomas1607haja.rezeptplaner.R
import io.github.thomas1607haja.rezeptplaner.data.ServerConfig
import io.github.thomas1607haja.rezeptplaner.ui.common.Texts
import java.io.IOException
import java.net.UnknownHostException
import java.util.concurrent.TimeUnit
import javax.net.ssl.SSLException
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.contentOrNull
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.MultipartBody
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody
import okhttp3.RequestBody.Companion.toRequestBody
import okhttp3.Response

/** Version der Server-Schnittstelle, die diese App kennt (server/internal/system, APIVersion). */
const val SUPPORTED_API_VERSION = 1

/** Antwort von GET /api/v1/system (nur die Felder, die die App braucht). */
@Serializable
data class SystemInfo(
    val version: String,
    @SerialName("api_version") val apiVersion: Int = 0,
)

/**
 * Fehler beim Zugriff auf den Rezeptplaner. [code] ist ein Fehlercode des
 * Servers (server/internal/apierr, Texte in strings_errors.xml) oder ein
 * Code der App ("client_…", Texte in strings_api.xml); [message] liefert
 * den Text in der Sprache der App. [serverText] ist der englische Text des
 * Servers als Rückfall. [offline]: keine Verbindung (Netz weg, Server nicht
 * erreichbar) – dann zeigt die App den gespeicherten Stand mit Hinweis.
 */
class ApiException(
    val code: String,
    val params: Map<String, Any?> = emptyMap(),
    private val serverText: String = "",
    cause: Throwable? = null,
    val offline: Boolean = false,
) : Exception(serverText.ifEmpty { code }, cause) {
    override val message: String
        get() = Texts.errorOrNull(code, params) ?: serverText.ifEmpty { code }
}

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
            throw ApiException("client_server_too_old", mapOf("version" to info.version))
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

    // Essensplan ändern. Geändert wird nur online; danach gleicht die App ab.

    suspend fun createEntry(config: ServerConfig, entry: NewEntry): MealPlanEntry =
        send(config, "POST", "/api/v1/meal-plan", json.encodeToString(entry))

    suspend fun updateEntry(config: ServerConfig, id: Long, changes: EntryChanges): MealPlanEntry =
        send(config, "PUT", "/api/v1/meal-plan/$id", json.encodeToString(changes))

    suspend fun deleteEntry(config: ServerConfig, id: Long) {
        withContext(Dispatchers.IO) { execute(config, "/api/v1/meal-plan/$id", "application/json", "DELETE", null) }
    }

    suspend fun moveEntry(config: ServerConfig, id: Long, move: MoveRequest): MoveResult =
        send(config, "POST", "/api/v1/meal-plan/$id/move", json.encodeToString(move))

    // Menüs (Schritt 9.4, Server 0.16)

    suspend fun createMenu(config: ServerConfig, menu: MenuInput): Menu =
        send(config, "POST", "/api/v1/menus", json.encodeToString(menu))

    suspend fun updateMenu(config: ServerConfig, id: Long, menu: MenuInput): Menu =
        send(config, "PUT", "/api/v1/menus/$id", json.encodeToString(menu))

    /** [withEntries] = false löst das Menü auf, die Gänge bleiben im Plan. */
    suspend fun deleteMenu(config: ServerConfig, id: Long, withEntries: Boolean) {
        withContext(Dispatchers.IO) {
            execute(config, "/api/v1/menus/$id" + if (withEntries) "?with_entries=true" else "", "application/json", "DELETE", null)
        }
    }

    suspend fun moveMenu(config: ServerConfig, id: Long, move: MoveRequest): MoveResult =
        send(config, "POST", "/api/v1/menus/$id/move", json.encodeToString(move))

    suspend fun toMenu(config: ServerConfig, entryId: Long, request: ToMenuRequest): Menu =
        send(config, "POST", "/api/v1/meal-plan/$entryId/to-menu", json.encodeToString(request))

    suspend fun detachEntry(config: ServerConfig, entryId: Long): MealPlanEntry =
        send(config, "POST", "/api/v1/meal-plan/$entryId/detach", "{}")

    // Einkauf und Bring

    /** Einkaufsliste der eingekauften Woche bzw. mit [next] des nächsten Einkaufs. */
    suspend fun statistics(config: ServerConfig, period: String): Statistics =
        get(config, "/api/v1/statistics?period=$period")

    suspend fun shoppingPreview(config: ServerConfig, next: Boolean): ShoppingPreview =
        get(config, "/api/v1/shopping/preview?period=" + if (next) "next" else "current")

    suspend fun sendShopping(config: ServerConfig, request: ShoppingSend): SendResult =
        send(config, "POST", "/api/v1/shopping/send", json.encodeToString(request))

    suspend fun homeAssistantStatus(config: ServerConfig): HomeAssistantStatus = get(config, "/api/v1/homeassistant/status")

    /** Vorschau „Übertragung zurücksetzen“: Gerichte des Zeitraums bzw. nur [entryId]. */
    suspend fun resetPreview(config: ServerConfig, next: Boolean, entryId: Long?): ResetPreview =
        get(config, if (entryId != null) "/api/v1/shopping/reset?entry=$entryId" else "/api/v1/shopping/reset?period=" + if (next) "next" else "current")

    suspend fun resetTransfer(config: ServerConfig, request: ResetRequest): ResetResult =
        send(config, "POST", "/api/v1/shopping/reset", json.encodeToString(request))

    /** Wählt die Bring-Liste (Einstellung bring_entity; die anderen bleiben). */
    suspend fun setBringList(config: ServerConfig, entityId: String) {
        send<HouseholdSettings>(config, "PUT", "/api/v1/settings", json.encodeToString(mapOf("bring_entity" to entityId)))
    }

    /**
     * Lädt Importdateien hoch (multipart, Feld "file"); Ergebnis ist die
     * Vorschau. [files]: Anzeigename und Inhalt je Datei.
     */
    suspend fun uploadImport(config: ServerConfig, files: List<Pair<String, ByteArray>>): ImportPreview {
        val body = MultipartBody.Builder().setType(MultipartBody.FORM).apply {
            for ((name, data) in files) {
                addFormDataPart("file", name, data.toRequestBody("application/octet-stream".toMediaType()))
            }
        }.build()
        return withContext(Dispatchers.IO) {
            val text = execute(config, "/api/v1/import/file", "application/json", "POST", null, body).decodeToString()
            try {
                json.decodeFromString<ImportPreview>(text)
            } catch (e: Exception) {
                throw ApiException("client_unexpected_import", cause = e)
            }
        }
    }

    /** Vorschaubild eines Rezepts der Import-Vorschau. */
    suspend fun importThumbnail(config: ServerConfig, token: String, index: Int): ByteArray =
        withContext(Dispatchers.IO) { execute(config, "/api/v1/import/file/$token/images/$index", "image/jpeg") }

    /** Legt die ausgewählten Rezepte der Vorschau an. */
    suspend fun confirmImport(
        config: ServerConfig,
        token: String,
        selected: List<Int>,
        categories: Map<String, CategoryChoice> = emptyMap(),
    ): ImportResult =
        send(config, "POST", "/api/v1/import/file/$token", json.encodeToString(ImportConfirm(selected, categories)))

    // Rezepte ändern – nur online, danach gleicht die App ab.

    suspend fun createRecipe(config: ServerConfig, input: RecipeInput): Recipe =
        send(config, "POST", "/api/v1/recipes", json.encodeToString(input))

    suspend fun updateRecipe(config: ServerConfig, id: Long, input: RecipeInput): Recipe =
        send(config, "PUT", "/api/v1/recipes/$id", json.encodeToString(input))

    suspend fun deleteRecipe(config: ServerConfig, id: Long) {
        withContext(Dispatchers.IO) { execute(config, "/api/v1/recipes/$id", "application/json", "DELETE") }
    }

    suspend fun createVariant(config: ServerConfig, id: Long): Recipe =
        // POST braucht einen Inhalt; der Server liest ihn nicht.
        send(config, "POST", "/api/v1/recipes/$id/variants", "{}")

    suspend fun setFavorite(config: ServerConfig, id: Long, favorite: Boolean) {
        withContext(Dispatchers.IO) {
            execute(config, "/api/v1/recipes/$id/favorite", "application/json", "PUT", json.encodeToString(mapOf("is_favorite" to favorite)))
        }
    }

    /** Neues Rezeptbild; der Server prüft und verkleinert es. */
    suspend fun setImage(config: ServerConfig, id: Long, data: ByteArray, mimeType: String): Recipe =
        withContext(Dispatchers.IO) {
            val body = data.toRequestBody(mimeType.toMediaType())
            json.decodeFromString<Recipe>(execute(config, "/api/v1/recipes/$id/image", "application/json", "PUT", null, body).decodeToString())
        }

    /** Bild der Quellseite übernehmen (URL-Import). */
    suspend fun fetchImage(config: ServerConfig, id: Long, url: String): Recipe =
        send(config, "POST", "/api/v1/recipes/$id/image/fetch", json.encodeToString(mapOf("url" to url)))

    suspend fun deleteImage(config: ServerConfig, id: Long) {
        withContext(Dispatchers.IO) { execute(config, "/api/v1/recipes/$id/image", "application/json", "DELETE") }
    }

    /** Liest ein Rezept von einer Webseite (Entwurf). */
    suspend fun importUrl(config: ServerConfig, url: String): UrlDraft =
        send(config, "POST", "/api/v1/import/url", json.encodeToString(mapOf("url" to url)))

    /** Zerlegt Zutatenzeilen wie „500 g Mehl“. */
    suspend fun parseIngredients(config: ServerConfig, lines: List<String>): List<Ingredient> =
        send<IngredientsBody>(config, "POST", "/api/v1/import/ingredients", json.encodeToString(mapOf("lines" to lines))).ingredients

    /** Je Name: steht er auf der Vorratsliste? */
    suspend fun pantryMatch(config: ServerConfig, names: List<String>): List<Boolean> =
        send<PantryMatchBody>(config, "POST", "/api/v1/pantry/match", json.encodeToString(mapOf("names" to names))).inPantry

    @Serializable
    private data class IngredientsBody(val ingredients: List<Ingredient> = emptyList())

    @Serializable
    private data class PantryMatchBody(@kotlinx.serialization.SerialName("in_pantry") val inPantry: List<Boolean> = emptyList())

    /** Lädt eine Bilddatei (Name aus Recipe.image bzw. thumbnail). */
    suspend fun image(config: ServerConfig, name: String): ByteArray =
        withContext(Dispatchers.IO) { execute(config, "/api/v1/images/$name", "image/jpeg") }

    private suspend inline fun <reified T> get(config: ServerConfig, path: String): T =
        send(config, "GET", path, null)

    private suspend inline fun <reified T> send(config: ServerConfig, method: String, path: String, body: String?): T =
        withContext(Dispatchers.IO) {
            val text = execute(config, path, "application/json", method, body).decodeToString()
            try {
                json.decodeFromString<T>(text)
            } catch (e: Exception) {
                throw ApiException("client_not_rezeptplaner", cause = e)
            }
        }

    /** Führt eine GET-Anfrage aus und liefert den Inhalt; Fehler als [ApiException]. */
    private fun execute(
        config: ServerConfig,
        path: String,
        accept: String,
        method: String = "GET",
        body: String? = null,
        rawBody: RequestBody? = null,
    ): ByteArray {
        val request = Request.Builder()
            .url(config.url + path)
            .method(method, rawBody ?: body?.toRequestBody("application/json; charset=utf-8".toMediaType()))
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
            throw ApiException("client_host_not_found", cause = e, offline = true)
        } catch (e: SSLException) {
            throw ApiException("client_tls_failed", cause = e)
        } catch (e: IOException) {
            throw ApiException("client_unreachable", cause = e, offline = true)
        }
    }

    private fun checkStatus(response: Response) {
        if (response.isSuccessful) return
        val location = response.header("Location").orEmpty()
        // Anfang der Antwort, um Home Assistant zu erkennen (siehe unten).
        val start = runCatching { response.peekBody(64).string().trim() }.getOrDefault("")
        throw when {
            response.isRedirect && "cloudflareaccess.com" in location -> ApiException("client_cloudflare_login")
            response.code == 403 && response.header("cf-ray") != null && response.header("content-type")?.startsWith("application/json") != true ->
                ApiException("client_cloudflare_denied")
            response.code == 401 -> ApiException("client_key_invalid")
            // So antwortet Home Assistant (aiohttp) auf unbekannte Pfade – die
            // Adresse von Home Assistant statt der des App-Zugangs.
            response.code == 404 && start.startsWith("404: Not Found") -> ApiException("client_home_assistant_address")
            response.code == 404 && !start.startsWith("{") -> ApiException("client_not_rezeptplaner_or_old")
            response.isRedirect -> ApiException("client_redirect", mapOf("location" to location))
            // 502: Home Assistant (Bring) nicht erreichbar, Grund steht in der Antwort.
            response.code == 502 && start.startsWith("{") -> serverError(response)
            response.code in 500..599 -> ApiException("client_server_error", mapOf("status" to response.code))
            // Prüffehler des Servers: {"error": "…", "code": "out_of_range", "params": {…}}
            response.code in 400..499 && start.startsWith("{") -> serverError(response)
            else -> ApiException("client_unexpected_status", mapOf("status" to response.code))
        }
    }

    /** Fehlerantwort des Servers als [ApiException] mit seinem Code. */
    private fun serverError(response: Response): ApiException {
        val text = runCatching { response.peekBody(4096).string() }.getOrDefault("")
        val body = runCatching { json.decodeFromString<ErrorBody>(text) }.getOrNull()
        if (body == null || (body.code.isEmpty() && body.error.isEmpty())) {
            return ApiException("client_rejected", mapOf("status" to response.code))
        }
        val params = body.params.mapValues { (_, v) -> (v as? JsonPrimitive)?.contentOrNull ?: v.toString() }
        // Ältere Server ohne Code: ihr Text (deutsch) wird angezeigt.
        return ApiException(body.code, params, body.error.replaceFirstChar { it.uppercase() })
    }

    @Serializable
    private data class ErrorBody(
        val error: String = "",
        val code: String = "",
        val params: Map<String, JsonElement> = emptyMap(),
    )

    companion object {
        private val json = Json { ignoreUnknownKeys = true; encodeDefaults = true }

        private val defaultClient: OkHttpClient = OkHttpClient.Builder()
            .connectTimeout(15, TimeUnit.SECONDS)
            .readTimeout(30, TimeUnit.SECONDS)
            .build()
    }
}

/** Codes der App für Fehler, die nicht vom Server kommen (strings_api.xml). */
internal val clientErrorTexts: Map<String, Int> = mapOf(
    "client_server_too_old" to R.string.api_server_too_old,
    "client_unexpected_import" to R.string.api_unexpected_import,
    "client_not_rezeptplaner" to R.string.api_not_rezeptplaner,
    "client_host_not_found" to R.string.api_host_not_found,
    "client_tls_failed" to R.string.api_tls_failed,
    "client_unreachable" to R.string.api_unreachable,
    "client_cloudflare_login" to R.string.api_cloudflare_login,
    "client_cloudflare_denied" to R.string.api_cloudflare_denied,
    "client_key_invalid" to R.string.api_key_invalid,
    "client_home_assistant_address" to R.string.api_home_assistant_address,
    "client_not_rezeptplaner_or_old" to R.string.api_not_rezeptplaner_or_old,
    "client_redirect" to R.string.api_redirect,
    "client_server_error" to R.string.api_server_error,
    "client_unexpected_status" to R.string.api_unexpected_status,
    "client_rejected" to R.string.api_rejected,
)
