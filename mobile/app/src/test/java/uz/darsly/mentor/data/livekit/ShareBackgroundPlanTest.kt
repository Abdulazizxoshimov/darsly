package uz.darsly.mentor.data.livekit

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * №25 — ulashishda ilovaning o'zini fonga olishi.
 *
 * Bu testlar bitta HAQIQIY nosozlikni qo'riqlaydi: 2026-08-01 yozuvida
 * o'quvchilar dars materialini emas, Jonly interfeysini ko'rgan. Va ular
 * teskari xavfni ham qo'riqlaydi: ilova o'zini KUTILMAGANDA fonga tashlashi
 * (qayta ulanishda, ustoz ilovaga qaytgan paytda) birinchisidan yomonroq.
 */
class ShareBackgroundPlanTest {

    @Test
    fun `ulashish boshlanganda ilova fonga olinadi`() {
        val d = ShareBackgroundPlan.onShareStarted(
            sharing = true,
            alreadyHandled = false,
            autoBackground = true,
            cameraOn = true,
        )
        assertTrue(d.moveToBack)
        assertEquals(ShareBackgroundPlan.BACKGROUND_MESSAGE, d.message)
    }

    @Test
    fun `xabar qaytish yolini aytadi`() {
        // Fonga o'tish o'zi yarim yechim: ustoz qaytish yo'lini bilmasa,
        // dars o'rtasida "ilova yo'qoldi" degan vahima paydo bo'ladi.
        assertTrue(ShareBackgroundPlan.BACKGROUND_MESSAGE.contains("bildirishnoma"))
    }

    @Test
    fun `ulashish boshlanmagan bolsa hech narsa qilinmaydi`() {
        val d = ShareBackgroundPlan.onShareStarted(
            sharing = false,
            alreadyHandled = false,
            autoBackground = true,
            cameraOn = true,
        )
        assertTrue(d.isEmpty)
        assertFalse(d.moveToBack)
    }

    @Test
    fun `IKKINCHI marta fonga olinmaydi — qayta ulanish ustozni quvmasin`() {
        // ⭐ ASOSIY MEZON. Qayta ulanish/tiklashda ulashish qaytadan yoqiladi
        // va `screenShareOn` yana `true` bo'ladi. Ustoz o'sha payt ataylab
        // ilovaga qaytgan bo'lishi mumkin (chatni o'qigan, qo'llarni ko'rgan) —
        // uni yana fonga uloqtirish ilova o'zboshimchaligi bo'lardi.
        val d = ShareBackgroundPlan.onShareStarted(
            sharing = true,
            alreadyHandled = true,
            autoBackground = true,
            cameraOn = true,
        )
        assertTrue(d.isEmpty)
    }

    @Test
    fun `sozlama ochirilgan bolsa fonga otilmaydi`() {
        val d = ShareBackgroundPlan.onShareStarted(
            sharing = true,
            alreadyHandled = false,
            autoBackground = false,
            cameraOn = true,
        )
        assertFalse(d.moveToBack)
        assertNull(d.message)
        assertTrue(d.isEmpty)
    }

    // ─── Kamera holati (yozuvdagi bo'sh plitka) ──────────────────────────────

    @Test
    fun `kamera ochiq bolsa ustoz ogohlantiriladi`() {
        val d = ShareBackgroundPlan.onShareStarted(
            sharing = true,
            alreadyHandled = false,
            autoBackground = true,
            cameraOn = false,
        )
        assertTrue(d.moveToBack)
        assertTrue(d.message!!.contains(ShareBackgroundPlan.CAMERA_OFF_MESSAGE))
    }

    @Test
    fun `kamera ogohlantirishi fonga otishdan MUSTAQIL`() {
        // Sozlama o'chirilgani "yozuv sifati haqida ogohlantirmang" degani emas.
        val d = ShareBackgroundPlan.onShareStarted(
            sharing = true,
            alreadyHandled = false,
            autoBackground = false,
            cameraOn = false,
        )
        assertFalse(d.moveToBack)
        assertEquals(ShareBackgroundPlan.CAMERA_OFF_MESSAGE, d.message)
    }

    @Test
    fun `kamera yoniq bolsa ortiqcha ogohlantirish yoq`() {
        val d = ShareBackgroundPlan.onShareStarted(
            sharing = true,
            alreadyHandled = false,
            autoBackground = true,
            cameraOn = true,
        )
        assertFalse(d.message!!.contains(ShareBackgroundPlan.CAMERA_OFF_MESSAGE))
    }
}
