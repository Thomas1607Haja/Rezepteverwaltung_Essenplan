package io.github.thomas1607haja.rezeptplaner.api

import io.github.thomas1607haja.rezeptplaner.ui.common.Texts
import kotlinx.serialization.EncodeDefault
import kotlinx.serialization.ExperimentalSerializationApi
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.contentOrNull

// Datenmodelle wie in der API des Servers (server/internal/...). Felder,
// die die App nicht braucht, fehlen hier; der JSON-Leser ignoriert sie.

@Serializable
data class Recipe(
    val id: Long,
    val title: String,
    val description: String = "",
    @SerialName("base_servings") val baseServings: Int,
    /** Einheit der Grundmenge, z. B. "Springform 26 cm"; leer = Portionen. */
    @SerialName("yield_unit") val yieldUnit: String = "",
    @SerialName("prep_minutes") val prepMinutes: Int? = null,
    @SerialName("total_minutes") val totalMinutes: Int? = null,
    @SerialName("source_url") val sourceUrl: String = "",
    @SerialName("source_name") val sourceName: String = "",
    val image: String = "",
    val thumbnail: String = "",
    /** "photo", "illustration" (gezeichnet von der Claude-Skill) oder leer. */
    @SerialName("image_kind") val imageKind: String = "",
    @SerialName("is_favorite") val isFavorite: Boolean = false,
    val notes: String = "",
    val ingredients: List<Ingredient> = emptyList(),
    val steps: List<Step> = emptyList(),
    val tags: List<Tag> = emptyList(),
    @SerialName("parent_recipe_id") val parentRecipeId: Long? = null,
    @SerialName("created_at") val createdAt: String = "",
    @SerialName("updated_at") val updatedAt: String = "",
)

/** Gezeichnetes Bild statt Foto (Importformat Version 2). */
val Recipe.isIllustrated: Boolean get() = imageKind == "illustration" && image.isNotEmpty()

@Serializable
data class Ingredient(
    @SerialName("group_name") val groupName: String = "",
    val quantity: Double? = null,
    val unit: String = "",
    val name: String,
    val note: String = "",
    @SerialName("shopping_relevant") val shoppingRelevant: Boolean = true,
    /** Schlüssel für Einkaufsliste und abgewählte Zutaten (vom Server, beim Speichern ignoriert). */
    @OptIn(ExperimentalSerializationApi::class) @EncodeDefault(EncodeDefault.Mode.NEVER) @SerialName("item_key") val itemKey: String = "",
)

@Serializable
data class Step(val text: String)

@Serializable
data class Tag(
    val id: Long,
    val name: String,
    val group: String = "",
    val position: Int = 0,
    /** Fester Schlüssel eines Gangs der Startliste ("main", "dessert" …, Server 0.20); Namen sind Daten. */
    @SerialName("course_key") val courseKey: String = "",
)

@Serializable
data class TagGroup(val name: String, val tags: List<Tag>, val id: Long = 0, val role: String = "")

/** Termine eines Rezepts im Essensplan (für den Statushinweis). */
@Serializable
data class PlanInfo(
    @SerialName("recipe_id") val recipeId: Long,
    @SerialName("next_planned") val nextPlanned: String? = null,
    @SerialName("last_planned") val lastPlanned: String? = null,
    @SerialName("plan_count") val planCount: Int = 0,
)

@Serializable
data class MealPlanEntry(
    val id: Long,
    val date: String,
    val slot: String = "dinner",
    val position: Int = 0,
    @SerialName("recipe_id") val recipeId: Long? = null,
    val recipe: EntryRecipe? = null,
    val servings: Int,
    val status: String = "planned",
    val note: String = "",
    /** Abgewählte Zutaten („habe ich noch“) als item_key; nur für diesen Eintrag. */
    @SerialName("excluded_ingredients") val excludedIngredients: List<String> = emptyList(),
    /** Gang eines Menüs (Schritt 9.4); null = einzelnes Gericht. */
    @SerialName("menu_id") val menuId: Long? = null,
    /** Kategorie der Gang-Gruppe, z. B. Vorspeise; nur im Menü. */
    @SerialName("course_tag_id") val courseTagId: Long? = null,
)

/**
 * Menü im Essensplan (Schritt 9.4): Klammer um normale Einträge mit
 * menu_id. Titel leer = automatisch ([displayTitle] vom Server).
 */
@Serializable
data class Menu(
    val id: Long,
    val date: String,
    val slot: String = "dinner",
    val position: Int = 0,
    val title: String = "",
    @SerialName("display_title") val displayTitle: String = "",
    val note: String = "",
    val servings: Int = 4,
    /** Nur in Antworten von /menus/{id}: die Gänge in Reihenfolge. */
    val entries: List<MealPlanEntry> = emptyList(),
)

/** POST /api/v1/menus bzw. PUT /api/v1/menus/{id}. */
@Serializable
data class MenuInput(
    val date: String = "",
    val slot: String = "dinner",
    val title: String = "",
    val note: String = "",
    val servings: Int? = null,
    /** Beim Ändern: Personen auf alle Gänge übertragen (außer Kuchen & Co.). */
    @SerialName("apply_servings") val applyServings: Boolean = false,
)

/** POST /api/v1/meal-plan/{id}/to-menu („Zu einem Menü erweitern“). */
@Serializable
data class ToMenuRequest(
    val title: String = "",
    val note: String = "",
    @SerialName("course_tag_id") val courseTagId: Long = 0,
)

@Serializable
data class EntryRecipe(
    val title: String,
    val thumbnail: String = "",
    @SerialName("is_favorite") val isFavorite: Boolean = false,
    @SerialName("yield_unit") val yieldUnit: String = "",
)

@Serializable
data class Period(val from: String, val to: String)

@Serializable
data class HouseholdSettings(
    @SerialName("family_size") val familySize: Int = 4,
    @SerialName("shopping_day") val shoppingDay: String = "saturday",
    /** Ab dieser Uhrzeit gilt am Einkaufstag eingekauft ("18:00"). */
    @SerialName("shopping_time") val shoppingTime: String = "00:00",
    /** Kurzvorschau beim Einplanen zeigen (Einstellung der Weboberfläche). */
    @SerialName("plan_preview") val planPreview: Boolean = true,
    /** Slot „Kaffee & Kuchen“ anbieten (Schritt 9.3). */
    @SerialName("coffee_slot") val coffeeSlot: Boolean = true,
)

@Serializable
data class Deletion(val table: String, val id: Long)

/** Antwort von GET /api/v1/sync (server/internal/appsync). */
@Serializable
data class SyncResponse(
    @SerialName("server_time") val serverTime: String,
    val full: Boolean,
    @SerialName("plan_from") val planFrom: String,
    val today: String,
    @SerialName("shopping_period") val shoppingPeriod: Period,
    val recipes: List<Recipe> = emptyList(),
    @SerialName("plan_info") val planInfo: List<PlanInfo> = emptyList(),
    @SerialName("meal_plan_entries") val mealPlanEntries: List<MealPlanEntry> = emptyList(),
    /** Ab Server 0.16 (immer, auch leer); null = Server kennt keine Menüs. */
    val menus: List<Menu>? = null,
    @SerialName("tag_groups") val tagGroups: List<TagGroup> = emptyList(),
    val settings: HouseholdSettings = HouseholdSettings(),
    val deletions: List<Deletion> = emptyList(),
)

// Schreibende Anfragen an den Essensplan (server/internal/mealplan).

/** POST /api/v1/meal-plan; ohne servings gilt die Familiengröße. */
@Serializable
data class NewEntry(
    val date: String,
    val slot: String = "dinner",
    @SerialName("recipe_id") val recipeId: Long? = null,
    val servings: Int? = null,
    val note: String = "",
    @OptIn(ExperimentalSerializationApi::class) @EncodeDefault(EncodeDefault.Mode.NEVER) @SerialName("excluded_ingredients") val excludedIngredients: List<String> = emptyList(),
    /** Als Gang in dieses Menü (Datum und Slot kommen dann vom Menü). */
    @OptIn(ExperimentalSerializationApi::class) @EncodeDefault(EncodeDefault.Mode.NEVER) @SerialName("menu_id") val menuId: Long? = null,
    @OptIn(ExperimentalSerializationApi::class) @EncodeDefault(EncodeDefault.Mode.NEVER) @SerialName("course_tag_id") val courseTagId: Long? = null,
)

/** PUT /api/v1/meal-plan/{id}: ersetzt diese Felder. */
@Serializable
data class EntryChanges(
    @SerialName("recipe_id") val recipeId: Long?,
    val servings: Int,
    val note: String,
    val status: String,
    /** null = unverändert lassen; beim Ersetzen verfällt die Abwahl ohne neue Angabe. */
    @OptIn(ExperimentalSerializationApi::class) @EncodeDefault(EncodeDefault.Mode.NEVER) @SerialName("excluded_ingredients") val excludedIngredients: List<String>? = null,
    /** null = unverändert, 0 = kein Gang. */
    @OptIn(ExperimentalSerializationApi::class) @EncodeDefault(EncodeDefault.Mode.NEVER) @SerialName("course_tag_id") val courseTagId: Long? = null,
)

/** POST /api/v1/meal-plan/{id}/move. */
@Serializable
data class MoveRequest(
    val date: String,
    val slot: String,
    @SerialName("swap_with") val swapWith: Long? = null,
    /** Mit diesem Menü tauschen (Server 0.16); ältere Server kennen das Feld nicht. */
    @OptIn(ExperimentalSerializationApi::class) @EncodeDefault(EncodeDefault.Mode.NEVER) @SerialName("swap_with_menu") val swapWithMenu: Long? = null,
    val confirm: Boolean = false,
    @SerialName("have_ingredients") val haveIngredients: List<Long> = emptyList(),
)

@Serializable
data class MoveWarning(
    val type: String,
    @SerialName("entry_id") val entryId: Long,
    val message: String,
    /** Ab Server 0.20: Parameter zum Code [type] ({title}). */
    val params: Map<String, JsonElement> = emptyMap(),
) {
    /** Text in der Sprache der App (Rückfall: englischer Text des Servers). */
    fun text(): String = Texts.error(type, params.asTextParams(), message)
}

/**
 * Hinweis oder Fehler des Servers als Code mit Parametern (Server 0.20,
 * server/internal/apierr). [text] liefert ihn in der Sprache der App.
 */
@Serializable
data class Note(val code: String = "", val params: Map<String, JsonElement> = emptyMap(), val message: String = "") {
    fun text(): String = Texts.error(code, params.asTextParams(), message)
}

/** JSON-Parameter als Texte für Texts.error. */
fun Map<String, JsonElement>.asTextParams(): Map<String, Any?> =
    mapValues { (_, v) -> (v as? JsonPrimitive)?.contentOrNull ?: v.toString() }

/** Applied = false: Es gibt Warnungen, nichts ist gespeichert. */
@Serializable
data class MoveResult(val applied: Boolean, val warnings: List<MoveWarning> = emptyList())

// Datei-Import (server/internal/fileimport, docs/importformat.md).

/** Antwort auf POST /api/v1/import/file: nur eine Vorschau, gespeichert ist nichts. */
@Serializable
data class ImportPreview(
    /** Zum Bestätigen; leer, wenn nichts importierbar ist. */
    val token: String = "",
    val recipes: List<ImportCandidate> = emptyList(),
    @SerialName("file_errors") val fileErrors: List<ImportFileError> = emptyList(),
    /** Kategorien der Datei, die es nicht gibt – jede einmal (ab Server 0.9.0). */
    @SerialName("unknown_categories") val unknownCategories: List<String> = emptyList(),
    /** Version der Claude-Skill, die die Datei erzeugt hat (ab Server 0.21.0); null = nicht von der Skill. */
    val skill: SkillCheck? = null,
)

/** Skill-Version einer Importdatei und ob es eine neuere gibt (server/internal/fileimport/skill.go). */
@Serializable
data class SkillCheck(
    val version: String = "",
    val current: String = "",
    @SerialName("update_available") val updateAvailable: Boolean = false,
    @SerialName("download_url") val downloadUrl: String = "",
)

/**
 * Was beim Bestätigen mit einer unbekannten Kategorie geschieht: [tagId] =
 * dieser zuordnen, [newInGroup] = in dieser Gruppe neu anlegen. Fehlt der
 * Name in der Anfrage, wird die Kategorie weggelassen.
 */
@Serializable
data class CategoryChoice(
    @SerialName("tag_id") val tagId: Long = 0,
    @SerialName("new_in_group") val newInGroup: Long = 0,
)

/** POST /api/v1/import/file/{token} */
@Serializable
data class ImportConfirm(val selected: List<Int>, val categories: Map<String, CategoryChoice> = emptyMap())

@Serializable
data class ImportCandidate(
    val index: Int,
    val file: String = "",
    val recipe: Recipe,
    /** Vorschaubild unter GET /api/v1/import/file/{token}/images/{index}. */
    val thumbnail: Boolean = false,
    /** Englische Texte (ältere Server: deutsche); zur Anzeige [errorDetails] nehmen. */
    val errors: List<String> = emptyList(),
    val warnings: List<String> = emptyList(),
    /** Ab Server 0.20: Fehler und Hinweise als Codes (Note.text()). */
    @SerialName("error_details") val errorDetails: List<Note> = emptyList(),
    @SerialName("warning_details") val warningDetails: List<Note> = emptyList(),
    val duplicates: List<DuplicateRecipe> = emptyList(),
    @SerialName("unknown_categories") val unknownCategories: List<String> = emptyList(),
) {
    val importable: Boolean get() = errors.isEmpty()
}

@Serializable
data class DuplicateRecipe(val id: Long, val title: String)

@Serializable
data class ImportFileError(val file: String, val error: String, @SerialName("error_detail") val errorDetail: Note? = null)

@Serializable
data class ImportResult(
    val created: List<ImportCreated> = emptyList(),
    val failed: List<ImportFailed> = emptyList(),
    val warnings: List<String> = emptyList(),
    @SerialName("warning_details") val warningDetails: List<Note> = emptyList(),
)

@Serializable
data class ImportCreated(
    val index: Int,
    val id: Long,
    val title: String,
    val warnings: List<String> = emptyList(),
    @SerialName("warning_details") val warningDetails: List<Note> = emptyList(),
)

@Serializable
data class ImportFailed(val index: Int, val title: String = "", val error: String, @SerialName("error_detail") val errorDetail: Note? = null)

// Rezepte ändern (server/internal/recipes, importer).

/** POST/PUT /api/v1/recipes: das ganze Rezept; Kategorien nur mit ID. */
@Serializable
data class RecipeInput(
    val title: String,
    val description: String,
    @SerialName("base_servings") val baseServings: Int,
    // Nur geschickt, wenn gesetzt: ältere Server kennen das Feld nicht.
    @OptIn(ExperimentalSerializationApi::class) @EncodeDefault(EncodeDefault.Mode.NEVER) @SerialName("yield_unit") val yieldUnit: String = "",
    @SerialName("prep_minutes") val prepMinutes: Int?,
    @SerialName("total_minutes") val totalMinutes: Int?,
    @SerialName("source_url") val sourceUrl: String,
    @SerialName("source_name") val sourceName: String,
    @SerialName("is_favorite") val isFavorite: Boolean,
    val notes: String,
    val ingredients: List<Ingredient>,
    val steps: List<Step>,
    val tags: List<TagRef>,
)

@Serializable
data class TagRef(val id: Long)

/** Antwort von POST /api/v1/import/url: Entwurf, gespeichert ist nichts. */
@Serializable
data class UrlDraft(
    val recipe: Recipe,
    @SerialName("image_url") val imageUrl: String = "",
    val duplicates: List<DuplicateRecipe> = emptyList(),
)

// Einkauf (server/internal/shopping): Liste eines Einkaufszeitraums und
// Senden an Bring über Home Assistant.

@Serializable
data class ShoppingPreview(
    val period: Period,
    val today: String,
    val entries: List<ShoppingEntry> = emptyList(),
    val items: List<ShoppingItem> = emptyList(),
    /** Fehlt noch auf der Bring-Liste (Gerichte nach dem Senden geändert). */
    val missing: List<ShoppingItem> = emptyList(),
    /** Zu viel gesendet. */
    val surplus: List<ShoppingItem> = emptyList(),
    /** Eingestellte Bring-Liste (Server 0.18); leere Entität = keine gewählt. */
    val list: TodoList? = null,
)

@Serializable
data class ShoppingEntry(
    val id: Long,
    val date: String,
    val slot: String = "dinner",
    @SerialName("recipe_id") val recipeId: Long? = null,
    val title: String,
    val servings: Int = 0,
    @SerialName("yield_unit") val yieldUnit: String = "",
    val status: String = "",
    /** Die Zutaten stehen auf der Liste. */
    val included: Boolean = false,
)

@Serializable
data class ShoppingItem(
    val key: String,
    val name: String,
    /** Mengenangabe als Text, z. B. "500 g + 2 EL"; so geht sie an Bring. */
    val spec: String = "",
    val recipes: List<String> = emptyList(),
)

@Serializable
data class ShoppingSend(
    val period: String,
    val deselected: List<String>,
    val missing: Boolean = false,
    /** Nur diese Gerichte senden; leer = alle des Zeitraums. */
    val entries: List<Long> = emptyList(),
)

@Serializable
data class SentItem(val name: String, val spec: String = "", val merged: Boolean = false)

@Serializable
data class FailedItem(val name: String, val spec: String = "", val error: String = "", val code: String = "") {
    /** Grund in der Sprache der App (Code ab Server 0.20). */
    fun reason(): String = Texts.error(code, emptyMap(), error)
}

@Serializable
data class SendResult(
    val list: String = "",
    val sent: List<SentItem> = emptyList(),
    val failed: List<FailedItem> = emptyList(),
    val entries: Int = 0,
)

/** To-do-Liste in Home Assistant (Bring). */
@Serializable
data class TodoList(@SerialName("entity_id") val entityId: String = "", val name: String = "")

// Übertragung zurücksetzen (Server 0.18): übertragene Gerichte wieder auf
// „geplant“, auf Wunsch Artikel aus der damaligen Liste entfernen.

@Serializable
data class ResetEntry(
    val id: Long,
    val date: String,
    val title: String,
    /** Listen, an die Artikel dieses Gerichts gingen. */
    val lists: List<TodoList> = emptyList(),
    /** Vor Server 0.18 gesendet: Liste nicht gespeichert. */
    @SerialName("unknown_list") val unknownList: Boolean = false,
    /** Gesendete Artikel. */
    val items: Int = 0,
)

@Serializable
data class ResetPreview(
    val entries: List<ResetEntry> = emptyList(),
    @SerialName("current_list") val currentList: TodoList = TodoList(),
    /** Alle To-do-Listen, zur Auswahl für Gerichte ohne gespeicherte Liste. */
    val lists: List<TodoList> = emptyList(),
    @SerialName("lists_error") val listsError: String = "",
    /** Ab Server 0.20: Code zu [listsError]. */
    @SerialName("lists_error_code") val listsErrorCode: String = "",
)

@Serializable
data class ResetRequest(
    val entries: List<Long>,
    val remove: Boolean,
    @SerialName("fallback_list") val fallbackList: String = "",
)

@Serializable
data class ResetResult(
    val entries: Int = 0,
    val removed: List<String> = emptyList(),
    /** Menge verringert, andere Gerichte brauchen den Artikel noch. */
    val reduced: List<String> = emptyList(),
    /** Nicht mehr offen auf der Liste (schon abgehakt). */
    @SerialName("not_found") val notFound: List<String> = emptyList(),
    val failed: List<FailedItem> = emptyList(),
    /** Liste unbekannt, nicht entfernt. */
    val skipped: List<String> = emptyList(),
)

@Serializable
data class HomeAssistantStatus(val mode: String = "", val lists: List<TodoList> = emptyList(), val error: String = "")

// Statistik (server/internal/recipes/stats.go): Häufigkeiten zählen
// Planeinträge bis einschließlich heute.

@Serializable
data class StatRecipe(
    val id: Long,
    val title: String,
    val thumbnail: String = "",
    val count: Int = 0,
    @SerialName("last_planned") val lastPlanned: String? = null,
    @SerialName("created_at") val createdAt: String = "",
)

@Serializable
data class StatTag(val id: Long, val name: String, val count: Int = 0)

@Serializable
data class StatGroup(val name: String, val tags: List<StatTag> = emptyList())

@Serializable
data class StatNote(val note: String, val count: Int)

/** GET /api/v1/statistics?period=3m|12m|all */
@Serializable
data class Statistics(
    val period: String = "all",
    val from: String? = null,
    val today: String = "",
    val recipes: Int = 0,
    val planned: Int = 0,
    @SerialName("never_planned") val neverPlanned: Int = 0,
    val illustrated: Int = 0,
    val favorites: Int = 0,
    val entries: Int = 0,
    /** Davon Menüs (jedes Gericht zählt trotzdem einzeln). */
    val menus: Int = 0,
    val top: List<StatRecipe> = emptyList(),
    @SerialName("favorites_waiting") val favoritesWaiting: List<StatRecipe> = emptyList(),
    @SerialName("recently_added") val recentlyAdded: List<StatRecipe> = emptyList(),
    val categories: List<StatGroup> = emptyList(),
    @SerialName("without_category") val withoutCategory: Int = 0,
    @SerialName("without_recipe") val withoutRecipe: List<StatNote> = emptyList(),
)
