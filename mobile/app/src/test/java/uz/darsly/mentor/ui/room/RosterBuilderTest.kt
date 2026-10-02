package uz.darsly.mentor.ui.room

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * Moderatsiya paneli ro'yxatini yig'ish (M-1: participant list derivation).
 *
 * Bu qoidalar jimgina buzilib ustozga zarar yetkazadigan turdan: `me` filtri
 * tushib qolsa ustoz o'zini chiqarib yuborishi mumkin; qo'l belgisi noto'g'ri
 * bo'lsa navbat panelda ko'rinmaydi.
 */
class RosterBuilderTest {

    private data class P(val id: String, val nm: String = "", val muted: Boolean = false)

    private fun build(items: List<P>, me: String?, raised: Set<String>) =
        RosterBuilder.build(
            items = items, me = me, raised = raised,
            identity = { it.id }, name = { it.nm }, audioMuted = { it.muted },
        )

    @Test fun mentorSelfIsFilteredOut() {
        // BUG: ustoz ro'yxatda qolsa o'ziga "mute"/"chiqarish" tugmasi chiqadi —
        // bir bosishda o'zini darsdan chiqarib yuborishi mumkin.
        val roster = build(listOf(P("ustoz"), P("ali"), P("vali")), me = "ustoz", raised = emptySet())
        assertEquals(listOf("ali", "vali"), roster.map { it.identity })
    }

    @Test fun raisedHandsAreMarked() {
        val roster = build(listOf(P("ali"), P("vali")), me = null, raised = setOf("vali"))
        assertFalse(roster.first { it.identity == "ali" }.handRaised)
        assertTrue(roster.first { it.identity == "vali" }.handRaised)
    }

    @Test fun blankNameFallsBackToIdentity() {
        // BUG: ismsiz o'quvchi bo'sh qator bo'lib ko'rinardi — ustoz kimni
        // mute qilayotganini bilmasdi.
        val roster = build(listOf(P("ali", nm = "  ")), me = null, raised = emptySet())
        assertEquals("ali", roster.single().name)
    }

    @Test fun mutedStateIsCarried() {
        val roster = build(listOf(P("ali", muted = true)), me = null, raised = emptySet())
        assertTrue(roster.single().audioMuted)
    }

    @Test fun nullMeKeepsEveryone() {
        // Ustoz identity'si hali ma'lum bo'lmasa (token kelmagan) hech kim
        // filtrlanib qolmasin.
        val roster = build(listOf(P("a"), P("b")), me = null, raised = emptySet())
        assertEquals(2, roster.size)
    }
}
