package io.github.thomas1607haja.rezeptplaner.api

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

// Datenmodelle wie in der API des Servers (server/internal/...). Felder,
// die die App nicht braucht, fehlen hier; der JSON-Leser ignoriert sie.

@Serializable
data class Recipe(
    val id: Long,
    val title: String,
    val description: String = "",
    @SerialName("base_servings") val baseServings: Int,
    @SerialName("prep_minutes") val prepMinutes: Int? = null,
    @SerialName("total_minutes") val totalMinutes: Int? = null,
    @SerialName("source_url") val sourceUrl: String = "",
    @SerialName("source_name") val sourceName: String = "",
    val image: String = "",
    val thumbnail: String = "",
    @SerialName("is_favorite") val isFavorite: Boolean = false,
    val notes: String = "",
    val ingredients: List<Ingredient> = emptyList(),
    val steps: List<Step> = emptyList(),
    val tags: List<Tag> = emptyList(),
    @SerialName("parent_recipe_id") val parentRecipeId: Long? = null,
    @SerialName("created_at") val createdAt: String = "",
    @SerialName("updated_at") val updatedAt: String = "",
)

@Serializable
data class Ingredient(
    @SerialName("group_name") val groupName: String = "",
    val quantity: Double? = null,
    val unit: String = "",
    val name: String,
    val note: String = "",
    @SerialName("shopping_relevant") val shoppingRelevant: Boolean = true,
)

@Serializable
data class Step(val text: String)

@Serializable
data class Tag(val id: Long, val name: String, val group: String = "", val position: Int = 0)

@Serializable
data class TagGroup(val name: String, val tags: List<Tag>)

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
)

@Serializable
data class EntryRecipe(val title: String, val thumbnail: String = "", @SerialName("is_favorite") val isFavorite: Boolean = false)

@Serializable
data class Period(val from: String, val to: String)

@Serializable
data class HouseholdSettings(
    @SerialName("family_size") val familySize: Int = 4,
    @SerialName("shopping_day") val shoppingDay: String = "saturday",
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
)

/** PUT /api/v1/meal-plan/{id}: ersetzt diese Felder. */
@Serializable
data class EntryChanges(
    @SerialName("recipe_id") val recipeId: Long?,
    val servings: Int,
    val note: String,
    val status: String,
)

/** POST /api/v1/meal-plan/{id}/move. */
@Serializable
data class MoveRequest(
    val date: String,
    val slot: String,
    @SerialName("swap_with") val swapWith: Long? = null,
    val confirm: Boolean = false,
    @SerialName("have_ingredients") val haveIngredients: List<Long> = emptyList(),
)

@Serializable
data class MoveWarning(val type: String, @SerialName("entry_id") val entryId: Long, val message: String)

/** Applied = false: Es gibt Warnungen, nichts ist gespeichert. */
@Serializable
data class MoveResult(val applied: Boolean, val warnings: List<MoveWarning> = emptyList())
