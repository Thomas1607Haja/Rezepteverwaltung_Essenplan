package io.github.thomas1607haja.rezeptplaner.data

import io.github.thomas1607haja.rezeptplaner.api.ApiClient
import io.github.thomas1607haja.rezeptplaner.api.ApiException
import io.github.thomas1607haja.rezeptplaner.api.EntryChanges
import io.github.thomas1607haja.rezeptplaner.api.MealPlanEntry
import io.github.thomas1607haja.rezeptplaner.api.Menu
import io.github.thomas1607haja.rezeptplaner.api.MenuInput
import io.github.thomas1607haja.rezeptplaner.api.MoveRequest
import io.github.thomas1607haja.rezeptplaner.api.MoveResult
import io.github.thomas1607haja.rezeptplaner.api.NewEntry
import io.github.thomas1607haja.rezeptplaner.api.ToMenuRequest
import kotlinx.coroutines.flow.first

/** Ergebnis einer Änderung: null = gespeichert, sonst die Meldung für die Anzeige. */
typealias PlanError = String?

/**
 * Ergebnis der Kurzvorschau beim Einplanen (Schritt 8.E): Portionen,
 * abgewählte Zutaten (item_key) oder „Alles vorhanden“.
 */
data class PlanChoice(val servings: Int, val excluded: List<String>, val allAvailable: Boolean)

/**
 * Ändert den Essensplan. Jede Änderung geht direkt an den Server (die App
 * hat keine Warteschlange und damit keine Konflikte); danach gleicht sie
 * ab, damit der Plan auf dem Handy stimmt. Siehe docs/PLANUNG.md, Abschnitt 7.
 */
class PlanRepository(
    private val config: ConfigRepository,
    private val api: ApiClient,
    private val sync: SyncRepository,
) {
    suspend fun add(date: String, slot: String, recipeId: Long, choice: PlanChoice? = null): PlanError = write {
        val entry = api.createEntry(
            it,
            NewEntry(
                date = date, slot = slot, recipeId = recipeId, servings = choice?.servings,
                excludedIngredients = choice?.excluded.orEmpty(),
            ),
        )
        // Anlegen kennt nur geplante Einträge; „Alles vorhanden“ danach setzen.
        if (choice?.allAvailable == true) {
            api.updateEntry(it, entry.id, EntryChanges(recipeId, entry.servings, entry.note, "have_ingredients", emptyList()))
        }
    }

    /** Eintrag ohne Rezept, z. B. „Essen gehen“; der Text steht in note. */
    suspend fun addNote(date: String, slot: String, note: String): PlanError =
        write { api.createEntry(it, NewEntry(date = date, slot = slot, note = note.trim())) }

    /** Macht aus [entry] einen Eintrag ohne Rezept mit dem Text [note]. */
    suspend fun replaceWithNote(entry: MealPlanEntry, note: String): PlanError = write {
        val status = if (entry.status == "have_ingredients") "planned" else entry.status
        api.updateEntry(it, entry.id, EntryChanges(null, entry.servings, note.trim(), status))
    }

    /**
     * Ersetzt das Gericht eines Eintrags durch [recipeId]. Ohne [choice]
     * bleiben die Portionen, und die alte Abwahl verfällt auf dem Server.
     */
    suspend fun replace(entry: MealPlanEntry, recipeId: Long, choice: PlanChoice? = null): PlanError = write {
        var status = if (entry.status == "have_ingredients") "planned" else entry.status
        if (choice?.allAvailable == true && status != "transferred") status = "have_ingredients"
        api.updateEntry(
            it, entry.id,
            EntryChanges(
                recipeId = recipeId,
                servings = choice?.servings ?: entry.servings,
                // Bei einem Eintrag ohne Rezept war die Notiz der Titel.
                note = if (entry.recipeId != null) entry.note else "",
                status = status,
                excludedIngredients = choice?.excluded,
            ),
        )
    }

    /** „Zutaten vorhanden …“ im Menü des Eintrags: Abwahl und Portionen speichern. */
    suspend fun setChoice(entry: MealPlanEntry, choice: PlanChoice): PlanError = write {
        val status = when {
            entry.status == "transferred" -> "transferred"
            choice.allAvailable -> "have_ingredients"
            else -> "planned"
        }
        api.updateEntry(it, entry.id, EntryChanges(entry.recipeId, choice.servings, entry.note, status, choice.excluded))
    }

    suspend fun update(entry: MealPlanEntry, servings: Int, note: String, haveIngredients: Boolean): PlanError = write {
        val status = when {
            entry.status == "transferred" -> "transferred"
            haveIngredients -> "have_ingredients"
            else -> "planned"
        }
        api.updateEntry(it, entry.id, EntryChanges(entry.recipeId, servings, note.trim(), status))
    }

    suspend fun delete(entry: MealPlanEntry): PlanError = write { api.deleteEntry(it, entry.id) }

    /**
     * Verschiebt [entry] an [date]/[slot] (ans Ende) oder tauscht mit
     * [swapWith]. Ohne [confirm] kommen bei Warnungen nur diese zurück und
     * nichts ist gespeichert.
     */
    suspend fun move(
        entry: MealPlanEntry,
        date: String,
        slot: String,
        swapWith: Long?,
        confirm: Boolean,
        haveIngredients: List<Long> = emptyList(),
        swapWithMenu: Long? = null,
    ): Pair<MoveResult?, PlanError> {
        var result: MoveResult? = null
        val error = write {
            result = api.moveEntry(it, entry.id, MoveRequest(date, slot, swapWith, swapWithMenu, confirm, haveIngredients))
        }
        return result to error
    }

    // Menüs (Schritt 9.4). Ein Menü ist eine Klammer um normale Einträge;
    // verschoben wird es immer als Block.

    /** Leeres Menü anlegen; die Gänge kommen danach über [addCourse]. */
    suspend fun createMenu(date: String, slot: String): Pair<Menu?, PlanError> {
        var menu: Menu? = null
        val error = write { menu = api.createMenu(it, MenuInput(date = date, slot = slot)) }
        return menu to error
    }

    /** Gericht ([recipeId]) oder Text ([note]) als Gang ins Menü; Portionen vom Menü. */
    suspend fun addCourse(menu: Menu, recipeId: Long?, note: String, courseTagId: Long?, choice: PlanChoice? = null): PlanError = write {
        val entry = api.createEntry(
            it,
            NewEntry(
                date = menu.date, slot = menu.slot, recipeId = recipeId, servings = choice?.servings, note = note.trim(),
                excludedIngredients = choice?.excluded.orEmpty(), menuId = menu.id, courseTagId = courseTagId?.takeIf { id -> id > 0 },
            ),
        )
        if (choice?.allAvailable == true) {
            api.updateEntry(it, entry.id, EntryChanges(recipeId, entry.servings, entry.note, "have_ingredients", emptyList()))
        }
    }

    /** Titel (leer = automatisch), Notiz, Personen; [applyServings] überträgt sie auf die Gänge. */
    suspend fun updateMenu(menu: Menu, title: String, note: String, servings: Int, applyServings: Boolean): PlanError = write {
        api.updateMenu(it, menu.id, MenuInput(title = title.trim(), note = note.trim(), servings = servings, applyServings = applyServings))
    }

    /** [withEntries] = false: auflösen, die Gänge bleiben als einzelne Gerichte. */
    suspend fun deleteMenu(menu: Menu, withEntries: Boolean): PlanError = write { api.deleteMenu(it, menu.id, withEntries) }

    /** „Zu einem Menü erweitern“: [entry] wird der erste Gang. */
    suspend fun toMenu(entry: MealPlanEntry, courseTagId: Long): Pair<Menu?, PlanError> {
        var menu: Menu? = null
        val error = write { menu = api.toMenu(it, entry.id, ToMenuRequest(courseTagId = courseTagId)) }
        return menu to error
    }

    /** „Aus Menü lösen“: der Gang bleibt als einzelnes Gericht stehen. */
    suspend fun detach(entry: MealPlanEntry): PlanError = write { api.detachEntry(it, entry.id) }

    /** Gang eines Eintrags ändern; 0 = kein Gang. */
    suspend fun setCourse(entry: MealPlanEntry, courseTagId: Long): PlanError = write {
        api.updateEntry(it, entry.id, EntryChanges(entry.recipeId, entry.servings, entry.note, entry.status, courseTagId = courseTagId))
    }

    /** „Zutaten vorhanden“ fürs ganze Menü: je Gang Abwahl bzw. alles vorhanden. */
    suspend fun setMenuChoices(choices: Map<MealPlanEntry, PlanChoice>): PlanError = write {
        for ((entry, choice) in choices) {
            val status = when {
                entry.status == "transferred" -> "transferred"
                choice.allAvailable -> "have_ingredients"
                else -> "planned"
            }
            api.updateEntry(it, entry.id, EntryChanges(entry.recipeId, entry.servings, entry.note, status, choice.excluded))
        }
    }

    /** Wie [move], für das ganze Menü; [swapWithMenu] tauscht mit einem anderen Menü. */
    suspend fun moveMenu(
        menu: Menu,
        date: String,
        slot: String,
        swapWith: Long?,
        swapWithMenu: Long?,
        confirm: Boolean,
        haveIngredients: List<Long> = emptyList(),
    ): Pair<MoveResult?, PlanError> {
        var result: MoveResult? = null
        val error = write {
            result = api.moveMenu(it, menu.id, MoveRequest(date, slot, swapWith, swapWithMenu, confirm, haveIngredients))
        }
        return result to error
    }

    private suspend fun write(action: suspend (ServerConfig) -> Unit): PlanError {
        val cfg = config.config.first() ?: return "Nicht verbunden."
        return try {
            action(cfg)
            sync.sync(SyncTrigger.Manual)
            null
        } catch (e: ApiException) {
            if (e.offline) "Ändern geht nur mit Verbindung zum Rezeptplaner." else e.message
        }
    }
}
