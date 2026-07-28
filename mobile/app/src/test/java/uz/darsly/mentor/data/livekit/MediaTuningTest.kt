package uz.darsly.mentor.data.livekit

import io.livekit.android.room.track.ScreenSharePresets
import livekit.org.webrtc.RtpParameters
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * Past internet uchun media sozlamalari.
 *
 * Bu testlar "kod shundaymi?" degan savolga emas, **mahsulot qaroriga** javob
 * beradi: zaif tarmoqdagi o'quvchi darsni ko'ra oladimi. Ular buzilsa —
 * foydalanuvchi buni faqat yomon internetli hududda sezadi, ya'ni biz emas,
 * o'qituvchi va o'quvchi to'laydi.
 */
class MediaTuningTest {

    // ─── Ekran ulashish: past internet uchun qatlam ──────────────────────────

    @Test
    fun `ekran ulashishda simulcast YOQILGAN`() {
        // ⭐ ASOSIY MEZON. Simulcast o'chiq bo'lsa ekran BITTA qatlamda ketadi
        // (server jurnalida tasdiqlangan: `SCREEN_SHARE layers: [HIGH 720p]`),
        // ya'ni 1.5 Mbit/s ni ko'tara olmagan o'quvchi ekranni UMUMAN ko'rmaydi.
        assertTrue(
            "Ekran ulashishda simulcast o'chirilsa, zaif internetli o'quvchi past sifatga tusha olmaydi",
            MediaTuning.screenSharePublish().simulcast,
        )
    }

    @Test
    fun `past qatlam haqiqatan past — 200 kbps atrofida`() {
        // Qatlam "bor" bo'lishi yetarli emas: u zaif kanalga SIG'ISHI kerak.
        // 200 kbps — kuchsiz 3G'da ham o'tadigan chegara.
        val low = MediaTuning.SCREEN_LOW.encoding
        assertEquals(200_000, low.maxBitrate)
        assertTrue("past qatlam yuqori qatlamdan sezilarli arzon bo'lsin",
            low.maxBitrate * 5 <= MediaTuning.SCREEN_HIGH.encoding.maxBitrate)
    }

    @Test
    fun `past qatlam royxatga kiritilgan`() {
        val layers = MediaTuning.screenSharePublish().simulcastLayers
        assertNotNull("qatlamlar oshkora berilishi kerak — SDK o'zi tanlasa nazorat yo'qoladi", layers)
        assertTrue(MediaTuning.SCREEN_LOW in layers!!)
    }

    @Test
    fun `yuqori qatlam TEGILMAGAN — yaxshi internetda sifat pasaymaydi`() {
        // Simulcast qo'shish yaxshi kanaldagi o'quvchining sifatini
        // PASAYTIRMASLIGI kerak, aks holda muammoni ko'chirgan bo'lardik.
        assertEquals(ScreenSharePresets.H720_FPS15, MediaTuning.SCREEN_HIGH)
        assertEquals(
            ScreenSharePresets.H720_FPS15.encoding.maxBitrate,
            MediaTuning.screenSharePublish().videoEncoding!!.maxBitrate,
        )
    }

    @Test
    fun `kanal torayganda OLCHAM saqlanadi, fps qurbon boladi`() {
        // Matn uchun yagona to'g'ri tanlov. `BALANCED` yoki `MAINTAIN_FRAMERATE`
        // bo'lsa slayd xiralashadi va dars ma'nosini yo'qotadi.
        assertEquals(
            RtpParameters.DegradationPreference.MAINTAIN_RESOLUTION,
            MediaTuning.screenSharePublish().degradationPreference,
        )
    }

    // ─── Ovoz: oxirigacha yashaydigan oqim ───────────────────────────────────

    @Test
    fun `ovozda RED va DTX yoqilgan`() {
        // Bular SDK default'i ham, lekin OSHKORA qotirilgan: kelajakda kimdir
        // `AudioTrackPublishDefaults` ni boshqa sabab bilan o'rnatsa, ular
        // jimgina o'chib ketardi va buni faqat yomon tarmoqda sezilardi.
        val audio = MediaTuning.audioPublish()
        assertTrue("RED — paket yo'qolganda ovoz uzilmasligi uchun", audio.red)
        assertTrue("DTX — jim paytda kanal ekran uchun bo'shasin", audio.dtx)
    }

    @Test
    fun `ovoz bitrate ekran oqimidan ancha kichik`() {
        // Ovozni "tejash" vasvasasi bor, lekin u umumiy oqimning kichik qismi:
        // uni yarmiga tushirish sezilarli tezlik bermaydi, sifatni esa (va
        // MAJBURIY yozuvni) buzadi. Bu nisbat qarorni asoslaydi.
        val audioBps = MediaTuning.audioPublish().audioBitrate ?: 48_000
        assertTrue(
            "ovoz ekran oqimining kichik ulushi bo'lishi kerak",
            audioBps * 10 < MediaTuning.SCREEN_HIGH.encoding.maxBitrate,
        )
    }
}
