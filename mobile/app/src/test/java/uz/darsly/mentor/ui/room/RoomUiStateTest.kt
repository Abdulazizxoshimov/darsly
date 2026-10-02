package uz.darsly.mentor.ui.room

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import uz.darsly.mentor.data.livekit.ScreenAudioPolicy

/**
 * 🟢L — tizim "Orqaga" tugmasi qachon tasdiq so'rashi kerak.
 *
 * Ikki xato ham qimmat:
 *  · **Kam so'rash** → tasodifiy "Orqaga" jonli darsni uzadi (ustoz sinf oldida qoladi).
 *  · **Ko'p so'rash** → xonaga kirmagan ustoz har chiqishda ortiqcha oyna ko'radi.
 */
class RoomUiStateTest {

    /** Sinov qurilmasi: HONOR ABR-LX1, Android 15 (API 35). */
    private companion object { const val API_35 = 35 }

    @Test
    fun `ulangan darsda tasdiq soraladi`() {
        assertTrue(RoomUiState(connState = ConnState.CONNECTED).lessonActive)
    }

    @Test
    fun `ulanish jarayonida ham tasdiq soraladi`() {
        // Yarim ochilgan sessiya + tirik foreground servis qolib ketmasin.
        assertTrue(RoomUiState(connecting = true).lessonActive)
    }

    @Test
    fun `qayta ulanishda tasdiq soraladi`() {
        // Internet vaqtincha uzilgan — dars TUGAMAGAN.
        assertTrue(RoomUiState(connState = ConnState.RECONNECTING).lessonActive)
    }

    @Test
    fun `ekran ulashilayotgan bolsa albatta tasdiq soraladi`() {
        // Eng qimmat holat: ekran yozilyapti (maxfiylik) — hatto holat matni
        // kutilmagan qiymatda bo'lsa ham so'raymiz.
        assertTrue(RoomUiState(connState = ConnState.DISCONNECTED, screenOn = true).lessonActive)
    }

    // ── B-4: "Ekran audiosi" qatori rost gapirishi ────────────────────────────

    @Test
    fun `C6 dan keyin mikrofon ochiq bolsa ham ekran audiosi yoniq deb korsatiladi`() {
        // Mustaqil trek: ustoz mute bosdi, video ovozi esa o'tishda davom etadi —
        // UI "mikrofon o'chirilgan" deb YOLG'ON to'siq ko'rsatmasligi kerak.
        val state = RoomUiState(
            connState = ConnState.CONNECTED,
            screenOn = true,
            micOn = false,
            screenAudioOn = true,
        ).withAudioReason(sdkInt = API_35)
        assertEquals("yoniq", state.screenAudioLabel)
        assertEquals(ScreenAudioPolicy.Blocked.NONE, state.screenAudioBlocked)
    }

    @Test
    fun `mikrofon va ulashish yoniq bolganda yoniq deb korsatiladi`() {
        val state = RoomUiState(
            connState = ConnState.CONNECTED,
            screenOn = true,
            micOn = true,
            screenAudioOn = true,
        ).withAudioReason(sdkInt = API_35)
        assertEquals("yoniq", state.screenAudioLabel)
    }

    @Test
    fun `ulashish boshlanmaganda qoshimcha sabab yozilmaydi`() {
        val state = RoomUiState(connState = ConnState.CONNECTED, micOn = true).withAudioReason(sdkInt = API_35)
        assertEquals("o'chiq", state.screenAudioLabel)
    }

    @Test
    fun `xonaga ulanmagan holda tasdiq soralmaydi`() {
        assertFalse(RoomUiState().lessonActive)
        assertFalse(RoomUiState(connState = ConnState.DISCONNECTED).lessonActive)
        // Ulanish xatosidan keyin ham: ustoz shunchaki orqaga qaytadi.
        assertFalse(RoomUiState(connState = ConnState.DISCONNECTED, error = "Ulanmadi").lessonActive)
        assertFalse(RoomUiState(micDenied = true).lessonActive)
    }
}
