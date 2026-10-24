package io.github.thomas1607haja.rezeptplaner.data

import io.github.thomas1607haja.rezeptplaner.api.Deletion
import io.github.thomas1607haja.rezeptplaner.api.EntryChanges
import io.github.thomas1607haja.rezeptplaner.api.MealPlanEntry
import io.github.thomas1607haja.rezeptplaner.api.Menu
import io.github.thomas1607haja.rezeptplaner.api.MoveRequest
import io.github.thomas1607haja.rezeptplaner.api.NewEntry
import io.github.thomas1607haja.rezeptplaner.api.Period
import io.github.thomas1607haja.rezeptplaner.api.SyncResponse
import io.github.thomas1607haja.rezeptplaner.ui.plan.PlanItem
import io.github.thomas1607haja.rezeptplaner.ui.plan.planItems
import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

// Menüs (Schritt 9.4): Abgleich, Anzeige im Plan, Anfragen an ältere Server.
class MenuMergeTest {
    private val url = "https://rezepte.example.org"

    private fun response(full: Boolean, menus: List<Menu>?, entries: List<MealPlanEntry> = emptyList(), deletions: List<Deletion> = emptyList()) =
        SyncResponse(
            serverTime = "2026-10-23T10:00:00.000Z", full = full, planFrom = "2026-10-09", today = "2026-10-23",
            shoppingPeriod = Period("2026-10-17", "2026-10-23"),
            mealPlanEntries = entries, menus = menus, deletions = deletions,
        )

    private fun entry(id: Long, position: Int, menuId: Long? = null, slot: String = "dinner") =
        MealPlanEntry(id = id, date = "2026-10-24", slot = slot, position = position, recipeId = 1, servings = 4, menuId = menuId)

    @Test
    fun mergeKeepsAndDeletesMenus() {
        val first = Snapshot().merge(response(true, listOf(Menu(1, "2026-10-24"), Menu(2, "2026-10-25"))), url)
        assertTrue(first.menusSupported)
        assertEquals(listOf(1L, 2L), first.menus.map { it.id })
        val second = first.merge(
            response(false, listOf(Menu(1, "2026-10-24", title = "Festessen")), deletions = listOf(Deletion("meal_plan_menus", 2))),
            url,
        )
        assertEquals(listOf("Festessen"), second.menus.map { it.title })
    }

    @Test
    fun olderServerHasNoMenus() {
        val json = Json { ignoreUnknownKeys = true }
        val r = json.decodeFromString<SyncResponse>(
            """{"server_time":"x","full":true,"plan_from":"2026-10-09","today":"2026-10-23","shopping_period":{"from":"a","to":"b"}}""",
        )
        assertFalse(Snapshot().merge(r, url).menusSupported)
    }

    @Test
    fun coursesStayInTheirMenu() {
        val menus = listOf(Menu(7, "2026-10-24", position = 1))
        val items = planItems(
            listOf(entry(1, 0), entry(2, 0, menuId = 7), entry(3, 1, menuId = 7), entry(4, 2), entry(5, 0, slot = "lunch"), entry(6, 0, menuId = 99)),
            menus,
        )
        // Mittag zuerst; Gang 6 gehört zu einem unbekannten Menü und steht einzeln.
        assertEquals(listOf("e5", "e1", "e6", "m7", "e4"), items.map { it.key })
        assertEquals(listOf(2L, 3L), (items[3] as PlanItem.MenuBlock).courses.map { it.id })
    }

    @Test
    fun newFieldsOnlyWhenSet() {
        // Ältere Server lehnen unbekannte Felder ab.
        val json = Json { encodeDefaults = true }
        assertFalse("menu_id" in json.encodeToString(NewEntry(date = "2026-10-24", recipeId = 1)))
        assertFalse("course_tag_id" in json.encodeToString(EntryChanges(1, 4, "", "planned")))
        assertFalse("swap_with_menu" in json.encodeToString(MoveRequest("2026-10-24", "dinner")))
        assertTrue("\"menu_id\":3" in json.encodeToString(NewEntry(date = "", menuId = 3)))
    }
}
