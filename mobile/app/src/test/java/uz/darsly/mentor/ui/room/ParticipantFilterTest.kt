package uz.darsly.mentor.ui.room

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * Ishtirokchilar panelidagi qidiruv (ommaviy dars — 200+ kishi).
 *
 * Mezon: **ustoz aniq bir odamni bir necha harf yozib topa olishi**. Shu sabab
 * moslik ichki (familiya bo'yicha ham ishlaydi) va katta-kichik harfga befarq.
 */
class ParticipantFilterTest {

    private fun p(name: String, identity: String = name) = RosterEntry(
        identity = identity,
        name = name,
        audioMuted = true,
        handRaised = false,
    )

    private val roster = listOf(
        p("Alijon Valiyev"),
        p("Malika Karimova"),
        p("Vali Nazarov"),
        p("aziz"),
    )

    @Test
    fun `bosh sorov royxatni ozgartirmaydi`() {
        assertEquals(roster, ParticipantFilter.roster(roster, ""))
    }

    @Test
    fun `faqat boshliqdan iborat sorov royxatni ozgartirmaydi`() {
        // Maydonni tozalash "hech kim topilmadi" holatiga tushirmasligi kerak.
        assertEquals(roster, ParticipantFilter.roster(roster, "   "))
    }

    @Test
    fun `katta-kichik harf farqsiz`() {
        // Ism kichik harf bilan yozilgan, so'rov katta — baribir topiladi.
        assertEquals(listOf("aziz"), ParticipantFilter.roster(roster, "AZIZ").map { it.name })
        assertEquals(listOf("Malika Karimova"), ParticipantFilter.roster(roster, "MALIKA").map { it.name })
    }

    @Test
    fun `qisqa sorov ichki mosliklarni ham qaytaradi`() {
        // "az" → "aziz" va "NAZarov". Bu ATAYLAB shunday: ustoz ismning
        // o'rtasidan boshlab yozishi mumkin (masalan familiya bo'yicha).
        // Ro'yxat qisqargani muhim, aniq bitta natija emas.
        assertEquals(
            listOf("Vali Nazarov", "aziz"),
            ParticipantFilter.roster(roster, "az").map { it.name },
        )
    }

    @Test
    fun `familiya bolagi boyicha ham topadi`() {
        // "vali" — Alijon VALIyev (ichki moslik) va Vali Nazarov (bosh moslik).
        val found = ParticipantFilter.roster(roster, "vali").map { it.name }
        assertEquals(listOf("Alijon Valiyev", "Vali Nazarov"), found)
    }

    @Test
    fun `sorov atrofidagi boshliq hisobga olinmaydi`() {
        assertEquals(listOf("Malika Karimova"), ParticipantFilter.roster(roster, "  malika ").map { it.name })
    }

    @Test
    fun `topilmasa bosh royxat`() {
        assertTrue(ParticipantFilter.roster(roster, "zzz").isEmpty())
    }

    @Test
    fun `tartib saqlanadi`() {
        // Ro'yxat filtrlashdan keyin qayta tartiblanmaydi — ustoz uchun qatorlar
        // sakrab ketishi (noto'g'ri odamni bosish) eng yomon natija.
        val found = ParticipantFilter.roster(roster, "a").map { it.name }
        assertEquals(roster.map { it.name }, found)
    }

    // ─── Maydon qachon ko'rinadi ──────────────────────────────────────────────

    @Test
    fun `kichik royxatda qidiruv maydoni korsatilmaydi`() {
        assertFalse(ParticipantFilter.shouldShowSearch(0))
        assertFalse(ParticipantFilter.shouldShowSearch(9))
    }

    @Test
    fun `on va undan kop ishtirokchida maydon chiqadi`() {
        assertTrue(ParticipantFilter.shouldShowSearch(10))
        assertTrue(ParticipantFilter.shouldShowSearch(300))
    }

    @Test
    fun `generik filtr ixtiyoriy tur bilan ishlaydi`() {
        val hands = listOf("Aziz" to "i1", "Bekzod" to "i2")
        assertEquals(
            listOf("Bekzod" to "i2"),
            ParticipantFilter.filter(hands, "bek") { it.first },
        )
    }
}
