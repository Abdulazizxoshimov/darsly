package uz.darsly.mentor.ui.blocklist

import org.junit.Assert.assertEquals
import org.junit.Test
import uz.darsly.mentor.data.api.BlocklistEntry

/**
 * Qora ro'yxat ekranining sof mantiqi.
 *
 * Eng muhim mezon — **ism hech qachon bo'sh qator bo'lib chiqmasin**: ustoz
 * "kimni qaytaryapman?" degan savolga javob ololmasa, tugmani bosolmaydi.
 */
class BlocklistFormatTest {

    private fun entry(
        id: String = "b1",
        identity: String = "guest-7",
        displayName: String = "Alijon",
        createdAt: String? = "2026-07-20T10:00:00Z",
    ) = BlocklistEntry(id = id, identity = identity, displayName = displayName, createdAt = createdAt)

    @Test
    fun `eng yangi ban tepada`() {
        val items = listOf(
            entry(id = "eski", createdAt = "2026-07-01T10:00:00Z"),
            entry(id = "yangi", createdAt = "2026-07-25T10:00:00Z"),
            entry(id = "orta", createdAt = "2026-07-10T10:00:00Z"),
        )
        assertEquals(
            listOf("yangi", "orta", "eski"),
            BlocklistFormat.sortForDisplay(items).map { it.id },
        )
    }

    @Test
    fun `sanasiz yozuv royxat oxirida qoladi`() {
        val items = listOf(
            entry(id = "sanasiz", createdAt = null),
            entry(id = "sanali", createdAt = "2026-07-01T10:00:00Z"),
        )
        assertEquals(
            listOf("sanali", "sanasiz"),
            BlocklistFormat.sortForDisplay(items).map { it.id },
        )
    }

    @Test
    fun `korsatiladigan ism display_name dan olinadi`() {
        assertEquals("Alijon", BlocklistFormat.nameOf(entry(displayName = "Alijon")))
    }

    @Test
    fun `display_name bosh bolsa identity ishlatiladi`() {
        // Eski yozuv yoki nomsiz kirgan mehmon — bo'sh qator ko'rsatilmasin.
        assertEquals("guest-7", BlocklistFormat.nameOf(entry(displayName = "   ")))
    }

    @Test
    fun `ikkalasi ham bosh bolsa ochiq matn`() {
        assertEquals(
            "Noma'lum ishtirokchi",
            BlocklistFormat.nameOf(entry(displayName = "", identity = "")),
        )
    }
}
