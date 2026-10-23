package io.github.thomas1607haja.rezeptplaner.data

import io.github.thomas1607haja.rezeptplaner.api.ApiClient
import io.github.thomas1607haja.rezeptplaner.api.ApiException
import io.github.thomas1607haja.rezeptplaner.api.SendResult
import io.github.thomas1607haja.rezeptplaner.api.ShoppingPreview
import io.github.thomas1607haja.rezeptplaner.api.ShoppingSend
import io.github.thomas1607haja.rezeptplaner.api.TodoList
import kotlinx.coroutines.flow.first

/**
 * Einkaufsliste und Senden an Bring – nur online, gerechnet wird auf dem
 * Server (wie in der Weboberfläche). Nach dem Senden gleicht die App ab,
 * damit der Essensplan „an Bring gesendet“ zeigt.
 */
class ShoppingRepository(
    private val config: ConfigRepository,
    private val api: ApiClient,
    private val sync: SyncRepository,
) {
    suspend fun preview(next: Boolean): Outcome<ShoppingPreview> = call { api.shoppingPreview(it, next) }

    /** Sendet die Liste ([missing] = nur die Differenz); abgewählte Artikel gelten danach als erledigt. */
    suspend fun send(next: Boolean, deselected: Set<String>, missing: Boolean): Outcome<SendResult> =
        call { api.sendShopping(it, ShoppingSend(if (next) "next" else "current", deselected.toList(), missing)) }
            .also { if (it is Outcome.Ok) sync.sync(SyncTrigger.Manual) }

    /** To-do-Listen aus Home Assistant (für die Wahl der Bring-Liste). */
    suspend fun lists(): Outcome<List<TodoList>> = when (val r = call { api.homeAssistantStatus(it) }) {
        is Outcome.Error -> r
        is Outcome.Ok -> if (r.value.error.isNotEmpty()) {
            Outcome.Error("Home Assistant nicht erreichbar: ${r.value.error}")
        } else {
            Outcome.Ok(r.value.lists)
        }
    }

    suspend fun setList(entityId: String): Outcome<Unit> = call { api.setBringList(it, entityId) }

    private suspend fun <T> call(action: suspend (ServerConfig) -> T): Outcome<T> {
        val cfg = config.config.first() ?: return Outcome.Error("Nicht verbunden.")
        return try {
            Outcome.Ok(action(cfg))
        } catch (e: ApiException) {
            Outcome.Error(if (e.offline) "Der Einkauf geht nur mit Verbindung zum Rezeptplaner." else e.message ?: "Fehler")
        }
    }
}
