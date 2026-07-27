package uz.darsly.mentor.data.livekit

import android.os.Build
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * B-4 · B-5 — ekran audiosi qoidalari.
 *
 * ENG MUHIM MEZON: **UI yolg'on gapirmasligi kerak.** Avval mikrofon o'chirilganda
 * ham "Ekran audiosi: yoniq" deb turardi — ustoz video qo'yib, o'quvchilar
 * hech narsa eshitmayotganini bilmasdi. Endi holat va sabab bir joyda hisoblanadi.
 */
class ScreenAudioPolicyTest {

    private companion object {
        const val ANDROID_9 = Build.VERSION_CODES.P // 28
        const val ANDROID_10 = Build.VERSION_CODES.Q // 29
        const val ANDROID_15 = 35
    }

    @Test
    fun `hamma shart bajarilsa audio oqadi`() {
        assertTrue(
            ScreenAudioPolicy.canCapture(
                sdkInt = ANDROID_15,
                hasRecordAudio = true,
                sharing = true,
                micOn = true,
            ),
        )
    }

    @Test
    fun `C6 dan keyin mikrofon ochirilsa ham audio oqadi`() {
        // ⭐ Mustaqil trek (Zoom xulqi): mute ekran ovozini to'xtatmaydi.
        assertEquals(
            ScreenAudioPolicy.Blocked.NONE,
            ScreenAudioPolicy.blockedBy(
                sdkInt = ANDROID_15,
                hasRecordAudio = true,
                sharing = true,
                micOn = false,
                independentTrack = true,
            ),
        )
    }

    @Test
    fun `zaxira yolda mikrofon ochirilsa audio oqmaydi va sababi aytiladi`() {
        // `INDEPENDENT_TRACK = false` holati: eski, mikrofonga bog'liq xulq.
        val blocked = ScreenAudioPolicy.blockedBy(
            sdkInt = ANDROID_15,
            hasRecordAudio = true,
            sharing = true,
            micOn = false,
            independentTrack = false,
        )
        assertEquals(ScreenAudioPolicy.Blocked.MIC_OFF, blocked)
        assertEquals(
            "o'chiq — mikrofon o'chirilgan",
            ScreenAudioPolicy.statusLabel(active = false, blocked = blocked),
        )
    }

    @Test
    fun `Android 9 va pastda prinsipial ishlamaydi`() {
        // SDK `ScreenAudioCapturer` `@RequiresApi(Q)`; minSdk 26 bo'lgani uchun
        // 8_0/8_1/9 qurilmalar bor va ular crash bo'lmasligi kerak (mezon C-7).
        assertEquals(
            ScreenAudioPolicy.Blocked.OLD_ANDROID,
            ScreenAudioPolicy.blockedBy(ANDROID_9, hasRecordAudio = true, sharing = true, micOn = true),
        )
        assertFalse(ScreenAudioPolicy.canCapture(ANDROID_9, true, true, true))
        // Eski Android — mustaqil trekdan qat'i nazar to'siq.
        assertFalse(ScreenAudioPolicy.canCapture(ANDROID_9, true, true, true, independentTrack = true))
        assertEquals(
            "bu Android versiyasida ishlamaydi",
            ScreenAudioPolicy.statusLabel(false, ScreenAudioPolicy.Blocked.OLD_ANDROID),
        )
    }

    @Test
    fun `Android 10 chegara qiymati ishlaydi`() {
        assertTrue(ScreenAudioPolicy.canCapture(ANDROID_10, true, sharing = true, micOn = true))
    }

    @Test
    fun `ruxsat yoq bolsa audio oqmaydi`() {
        assertEquals(
            ScreenAudioPolicy.Blocked.NO_PERMISSION,
            ScreenAudioPolicy.blockedBy(ANDROID_15, hasRecordAudio = false, sharing = true, micOn = true),
        )
    }

    @Test
    fun `ekran ulashilmasa audio manbasi yoq`() {
        assertEquals(
            ScreenAudioPolicy.Blocked.NOT_SHARING,
            ScreenAudioPolicy.blockedBy(ANDROID_15, hasRecordAudio = true, sharing = false, micOn = true),
        )
        // Bu holat uchun sabab yozilmaydi — ustoz hali ulashishni boshlamagan.
        assertEquals(
            "o'chiq",
            ScreenAudioPolicy.statusLabel(false, ScreenAudioPolicy.Blocked.NOT_SHARING),
        )
    }

    @Test
    fun `sabablar tartibi eng qatiy tosiqdan boshlanadi`() {
        // Bir vaqtda bir necha to'siq bo'lsa, foydalanuvchiga TUZATILMAYDIGANI
        // (Android versiyasi) birinchi aytiladi — u mikrofonni yoqib ham hal qilmaydi.
        assertEquals(
            ScreenAudioPolicy.Blocked.OLD_ANDROID,
            ScreenAudioPolicy.blockedBy(ANDROID_9, hasRecordAudio = false, sharing = false, micOn = false),
        )
    }

    @Test
    fun `active bolsa sabab ehtiborsiz qoldiriladi`() {
        // Audio haqiqatan oqyapti — holat bayrog'i ustun.
        assertEquals(
            "yoniq",
            ScreenAudioPolicy.statusLabel(active = true, blocked = ScreenAudioPolicy.Blocked.MIC_OFF),
        )
    }
}
