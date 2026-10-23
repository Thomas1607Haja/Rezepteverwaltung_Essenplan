package io.github.thomas1607haja.rezeptplaner.data

import io.github.thomas1607haja.rezeptplaner.api.ApiClient
import io.github.thomas1607haja.rezeptplaner.api.ApiException
import io.github.thomas1607haja.rezeptplaner.api.Ingredient
import io.github.thomas1607haja.rezeptplaner.api.Recipe
import io.github.thomas1607haja.rezeptplaner.api.RecipeInput
import io.github.thomas1607haja.rezeptplaner.api.UrlDraft
import kotlinx.coroutines.flow.first

/** Ergebnis einer Rezeptänderung: Wert oder Meldung für die Anzeige. */
sealed interface Outcome<out T> {
    data class Ok<T>(val value: T) : Outcome<T>
    data class Error(val message: String) : Outcome<Nothing>
}

/** Bildänderung beim Speichern. */
sealed interface ImageChange {
    data object Keep : ImageChange
    data object Remove : ImageChange
    data class Upload(val data: ByteArray, val mimeType: String) : ImageChange
    /** Bild der Quellseite (URL-Import) übernehmen. */
    data class FromUrl(val url: String) : ImageChange
}

/**
 * Rezepte anlegen, ändern, löschen, Varianten, Favoriten, URL-Import –
 * alles online direkt beim Server, danach Abgleich (wie PlanRepository).
 */
class RecipeRepository(
    private val config: ConfigRepository,
    private val api: ApiClient,
    private val sync: SyncRepository,
) {
    /** Entwurf aus einer Rezept-URL oder dem Foto-Import; angelegt wird erst beim Speichern. */
    var pendingDraft: UrlDraft? = null

    /** Gerichtfoto zum Entwurf aus dem Foto-Import. */
    var pendingImage: ImageChange.Upload? = null

    suspend fun importUrl(url: String): Outcome<UrlDraft> = call { api.importUrl(it, url) }

    suspend fun parseIngredients(lines: List<String>): Outcome<List<Ingredient>> = call { api.parseIngredients(it, lines) }

    suspend fun pantryMatch(names: List<String>): List<Boolean>? =
        (call { api.pantryMatch(it, names) } as? Outcome.Ok)?.value

    /**
     * Speichert ein Rezept (id null = neu) samt Bildänderung und gleicht
     * danach ab. Schlägt nur das Bild fehl, ist das Rezept trotzdem
     * gespeichert; die Meldung nennt dann das Bild.
     */
    suspend fun save(id: Long?, input: RecipeInput, image: ImageChange): Outcome<Pair<Recipe, String?>> {
        val saved = call { if (id == null) api.createRecipe(it, input) else api.updateRecipe(it, id, input) }
        if (saved !is Outcome.Ok) return saved as Outcome.Error
        val recipe = saved.value
        val imageError = when (image) {
            ImageChange.Keep -> null
            ImageChange.Remove -> (call { api.deleteImage(it, recipe.id) } as? Outcome.Error)?.message
            is ImageChange.Upload -> (call { api.setImage(it, recipe.id, image.data, image.mimeType) } as? Outcome.Error)?.message
            is ImageChange.FromUrl -> (call { api.fetchImage(it, recipe.id, image.url) } as? Outcome.Error)?.message
        }
        sync.sync(SyncTrigger.Manual)
        return Outcome.Ok(recipe to imageError?.let { "Rezept gespeichert, aber das Bild nicht: $it" })
    }

    suspend fun delete(id: Long): Outcome<Unit> = call { api.deleteRecipe(it, id) }.also { if (it is Outcome.Ok) sync.sync(SyncTrigger.Manual) }

    suspend fun createVariant(id: Long): Outcome<Recipe> = call { api.createVariant(it, id) }.also { if (it is Outcome.Ok) sync.sync(SyncTrigger.Manual) }

    suspend fun setFavorite(id: Long, favorite: Boolean): Outcome<Unit> =
        call { api.setFavorite(it, id, favorite) }.also { if (it is Outcome.Ok) sync.sync(SyncTrigger.Manual) }

    private suspend fun <T> call(action: suspend (ServerConfig) -> T): Outcome<T> {
        val cfg = config.config.first() ?: return Outcome.Error("Nicht verbunden.")
        return try {
            Outcome.Ok(action(cfg))
        } catch (e: ApiException) {
            Outcome.Error(if (e.offline) "Ändern geht nur mit Verbindung zum Rezeptplaner." else e.message ?: "Fehler")
        }
    }
}
