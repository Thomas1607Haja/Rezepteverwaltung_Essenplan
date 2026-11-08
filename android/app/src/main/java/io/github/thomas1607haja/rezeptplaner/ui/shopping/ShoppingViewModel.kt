package io.github.thomas1607haja.rezeptplaner.ui.shopping

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import io.github.thomas1607haja.rezeptplaner.R
import io.github.thomas1607haja.rezeptplaner.api.SendResult
import io.github.thomas1607haja.rezeptplaner.api.ShoppingEntry
import io.github.thomas1607haja.rezeptplaner.api.ShoppingPreview
import io.github.thomas1607haja.rezeptplaner.api.TodoList
import io.github.thomas1607haja.rezeptplaner.data.Outcome
import io.github.thomas1607haja.rezeptplaner.data.ShoppingRepository
import io.github.thomas1607haja.rezeptplaner.ui.common.Texts
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

/** Zustand der Einkaufsseite. [next]: false = eingekaufte Woche, true = nächster Einkauf. */
data class ShoppingUi(
    val next: Boolean = false,
    val preview: ShoppingPreview? = null,
    val loading: Boolean = true,
    val error: String? = null,
    /** Abgewählte Artikel (key) der Liste bzw. der Differenz. */
    val deselected: Set<String> = emptySet(),
    val deselectedMissing: Set<String> = emptySet(),
    val sending: Boolean = false,
    val result: SendResult? = null,
    val sendError: String? = null,
    /** Keine Bring-Liste gewählt: Auswahl anbieten. */
    val needsList: Boolean = false,
    /** Gesetzt, solange die Auswahl der Bring-Liste offen ist. */
    val lists: List<TodoList>? = null,
    /** Kurze Bestätigung, z. B. nach der Wahl der Bring-Liste. */
    val notice: String? = null,
)

/** Ist für die eingekaufte Woche nichts mehr zu tun, zeigt die Seite gleich den nächsten Einkauf. */
fun nothingLeft(p: ShoppingPreview) = p.items.isEmpty() && p.missing.isEmpty()

/** Warum ein Gericht nicht in die Liste eingeht (leer = es geht ein). */
fun entryReason(entry: ShoppingEntry, today: String): String = when {
    entry.included -> ""
    entry.recipeId == null -> Texts.get(R.string.shopping_reason_no_recipe)
    entry.status == "transferred" -> Texts.get(R.string.shopping_reason_transferred)
    entry.status == "have_ingredients" -> Texts.get(R.string.shopping_reason_have_ingredients)
    entry.date < today -> Texts.get(R.string.shopping_reason_past)
    else -> ""
}

/** „12 Artikel an Bring gesendet, davon 2 bei schon vorhandenen ergänzt.“ */
fun sentSummary(r: SendResult): String {
    val n = r.sent.size
    val merged = r.sent.count { it.merged }
    return if (merged > 0) Texts.plural(R.plurals.shopping_sent_merged, n, "merged" to merged)
    else Texts.plural(R.plurals.shopping_sent, n)
}

/**
 * Fehlt die Bring-Liste? Code no_bring_list (übersetzter Text); ältere
 * Server ohne Code melden den deutschen Text mit „Bring-Liste“.
 */
private fun needsBringList(message: String): Boolean =
    message == Texts.get(R.string.error_no_bring_list) || "bring-liste" in message.lowercase()

/**
 * Einkauf wie in der Weboberfläche (web/src/pages/ShoppingPage.svelte):
 * Zeitraum wählen, Artikel abwählen, an Bring senden, Differenz nachsenden.
 */
class ShoppingViewModel(private val repo: ShoppingRepository) : ViewModel() {
    private val _ui = MutableStateFlow(ShoppingUi())
    val ui: StateFlow<ShoppingUi> = _ui.asStateFlow()

    init {
        load(first = true)
    }

    /** Lädt die Liste neu; [keepResult]: Ergebnis der letzten Übertragung stehen lassen. */
    fun load(first: Boolean = false, keepResult: Boolean = false) {
        _ui.update { it.copy(loading = true, error = null, result = if (keepResult) it.result else null) }
        viewModelScope.launch {
            val next = _ui.value.next
            when (val r = repo.preview(next)) {
                is Outcome.Error -> _ui.update { it.copy(loading = false, error = r.message) }
                is Outcome.Ok -> {
                    if (first && !next && nothingLeft(r.value)) {
                        _ui.update { it.copy(next = true) }
                        load()
                        return@launch
                    }
                    _ui.update {
                        it.copy(loading = false, preview = r.value, deselected = emptySet(), deselectedMissing = emptySet())
                    }
                }
            }
        }
    }

    fun switchPeriod(next: Boolean) {
        if (_ui.value.next == next) return
        _ui.update { it.copy(next = next, preview = null, sendError = null, needsList = false) }
        load()
    }

    fun toggle(key: String, missing: Boolean = false) = _ui.update {
        if (missing) {
            it.copy(deselectedMissing = if (key in it.deselectedMissing) it.deselectedMissing - key else it.deselectedMissing + key)
        } else {
            it.copy(deselected = if (key in it.deselected) it.deselected - key else it.deselected + key)
        }
    }

    fun selectAll(all: Boolean) = _ui.update { s ->
        s.copy(deselected = if (all) emptySet() else s.preview?.items.orEmpty().map { it.key }.toSet())
    }

    fun send(missing: Boolean) {
        val s = _ui.value
        _ui.update { it.copy(sending = true, sendError = null, needsList = false, notice = null) }
        viewModelScope.launch {
            when (val r = repo.send(s.next, if (missing) s.deselectedMissing else s.deselected, missing)) {
                is Outcome.Error -> _ui.update {
                    it.copy(sending = false, sendError = r.message, needsList = needsBringList(r.message))
                }
                is Outcome.Ok -> {
                    _ui.update { it.copy(sending = false, result = r.value) }
                    load(keepResult = true)
                }
            }
        }
    }

    /** Öffnet die Auswahl der Bring-Liste. */
    fun chooseList() {
        viewModelScope.launch {
            when (val r = repo.lists()) {
                is Outcome.Error -> _ui.update { it.copy(sendError = r.message) }
                is Outcome.Ok -> _ui.update {
                    if (r.value.isEmpty()) it.copy(sendError = Texts.get(R.string.shopping_no_todo_lists))
                    else it.copy(lists = r.value)
                }
            }
        }
    }

    fun closeLists() = _ui.update { it.copy(lists = null) }

    fun setList(list: TodoList) {
        _ui.update { it.copy(lists = null) }
        viewModelScope.launch {
            when (val r = repo.setList(list.entityId)) {
                is Outcome.Error -> _ui.update { it.copy(sendError = r.message) }
                is Outcome.Ok -> _ui.update {
                    it.copy(needsList = false, sendError = null, notice = Texts.get(R.string.shopping_list_chosen, "name" to list.name))
                }
            }
        }
    }
}
