package uz.darsly.mentor.ui.room

import org.junit.Assert.assertEquals
import org.junit.Assert.assertSame
import org.junit.Test

/**
 * Xona chat ro'yxati (M-1: chat state / error handling).
 *
 * Ro'yxat ikki manbadan to'ladi (optimistik qo'shish + server echo'si). Bu
 * yerdagi testlar aynan darsda ko'rinadigan xatolarni ushlaydi: bir xabar ikki
 * marta chiqishi, o'z xabaridan "o'qilmagan" sanog'i ko'tarilishi, ro'yxatning
 * cheksiz o'sishi.
 */
class ChatLogTest {

    private fun msg(id: String, self: Boolean = false) =
        ChatMessageUi(id = id, name = "u", body = "b$id", self = self, toIdentity = null)

    @Test fun duplicateIdIsNotAddedTwice() {
        // BUG: o'z xabarimizni optimistik qo'shamiz, keyin server echo'si ham
        // AYNI ID bilan keladi — dedup bo'lmasa xabar ikki marta ko'rinardi.
        val start = listOf(msg("a"))
        val res = ChatLog.add(start, msg("a"))
        assertSame("dublikat → ayni ro'yxat", start, res.chat)
        assertEquals(0, res.unreadDelta)
    }

    @Test fun othersMessageIncrementsUnread() {
        val res = ChatLog.add(emptyList(), msg("a", self = false))
        assertEquals(1, res.unreadDelta)
    }

    @Test fun ownMessageDoesNotIncrementUnread() {
        // BUG: o'z yuborgan xabaridan "o'qilmagan" ko'tarilsa chat rozetkasi
        // yolg'on son ko'rsatardi.
        val res = ChatLog.add(emptyList(), msg("a", self = true))
        assertEquals(0, res.unreadDelta)
        assertEquals(listOf("a"), res.chat.map { it.id })
    }

    @Test fun listIsCappedAtMax() {
        // BUG: 90 daqiqalik darsda chat cheksiz o'ssa Compose render'i og'irlashadi.
        var chat = emptyList<ChatMessageUi>()
        for (i in 1..(ChatLog.MAX + 50)) chat = ChatLog.add(chat, msg("m$i")).chat
        assertEquals(ChatLog.MAX, chat.size)
        // Eng yangilari qoladi (oxiridan kesiladi).
        assertEquals("m${ChatLog.MAX + 50}", chat.last().id)
    }

    @Test fun deleteRemovesById() {
        val chat = listOf(msg("a"), msg("b"), msg("c"))
        assertEquals(listOf("a", "c"), ChatLog.delete(chat, "b").map { it.id })
    }

    @Test fun deleteMissingIdReturnsSameList() {
        // O'chirilmagan ID uchun AYNI ro'yxat — ortiqcha rekompozitsiya yo'q.
        val chat = listOf(msg("a"))
        assertSame(chat, ChatLog.delete(chat, "yoq"))
    }
}
