package uz.darsly.mentor.data.livekit

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * C-6 — ekran audiosi mikrofondan mustaqilligi ([ScreenAudioPlan]).
 *
 * Dizayn **qurilma o'lchovidan keyin** o'zgardi: "ikki trekni navbatlashtirish"
 * ishlamadi, chunki mute holatida e'lon qilingan trek obunachiga umuman berilmaydi
 * (o'lchov: o'quvchi `SCREEN_SHARE_AUDIO` ga obuna bo'lmadi, mute'dan keyin jimlik).
 * Endi bitta trek ishlatiladi, mazmuni boshqariladi.
 *
 * Har bir xatoning narxi:
 *  · trek mute qilinsa → o'quvchi **hech narsa eshitmaydi** (aynan shu xato o'lchovda ushlandi);
 *  · nollash ishlamasa → ustozning ovozi mute paytida **sizib chiqadi** (maxfiylik);
 *  · nollash mikrofon yoniqda yoqilsa → ustoz gapiradi, lekin **hech kim eshitmaydi**.
 */
class ScreenAudioPlanTest {

    @Test
    fun `ekran audiosi yoq bolsa haqiqiy mute ishlatiladi`() {
        // Ulashish yo'q — chetlanishga hojat yo'q, server holati ham rost bo'ladi.
        val off = ScreenAudioPlan.plan(micWanted = false, screenAudioActive = false)
        assertFalse(off.micTrackLive)
        assertFalse(off.silenceMicSamples)
        assertTrue(off.voiceMuted)

        val on = ScreenAudioPlan.plan(micWanted = true, screenAudioActive = false)
        assertTrue(on.micTrackLive)
        assertFalse(on.silenceMicSamples)
        assertFalse(on.voiceMuted)
    }

    @Test
    fun `mikrofon yoniqda ustoz va video birga ketadi`() {
        val p = ScreenAudioPlan.plan(micWanted = true, screenAudioActive = true)
        assertTrue(p.micTrackLive)
        assertFalse("ustoz gapiryapti — nollash mumkin emas", p.silenceMicSamples)
        assertFalse(p.voiceMuted)
    }

    @Test
    fun `mute bosilganda trek OVOZLI qoladi lekin mikrofon nollanadi`() {
        // ⭐ C-6 ning mohiyati va o'lchovdan olingan dars: trekni mute qilish
        // o'quvchini butunlay jim qoldiradi, shuning uchun trek ovozli qoladi.
        val p = ScreenAudioPlan.plan(micWanted = false, screenAudioActive = true)
        assertTrue("trek mute qilinsa o'quvchi video ovozini ham eshitmaydi", p.micTrackLive)
        assertTrue("ustoz ovozi qurilmadan chiqmasligi kerak", p.silenceMicSamples)
        assertTrue("UI uchun: ovoz ketmayapti", p.voiceMuted)
    }

    @Test
    fun `nollash faqat ustoz mute bosganda yoqiladi`() {
        listOf(true, false).forEach { mic ->
            listOf(true, false).forEach { screen ->
                val p = ScreenAudioPlan.plan(micWanted = mic, screenAudioActive = screen)
                if (p.silenceMicSamples) {
                    assertFalse("mic=$mic screen=$screen — niyat mute bo'lishi kerak", mic)
                }
            }
        }
    }

    @Test
    fun `ustoz mute bosgan holatda ovoz hech qachon ketmaydi`() {
        // Invariant: niyat "mute" bo'lsa, qaysi yo'l bilan bo'lmasin ovoz to'xtaydi.
        listOf(true, false).forEach { screen ->
            val p = ScreenAudioPlan.plan(micWanted = false, screenAudioActive = screen)
            assertTrue("screen=$screen", p.voiceMuted)
        }
    }
}
