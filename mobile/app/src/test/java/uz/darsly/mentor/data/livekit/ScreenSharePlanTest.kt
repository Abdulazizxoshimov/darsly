package uz.darsly.mentor.data.livekit

import org.junit.Assert.assertEquals
import org.junit.Test
import uz.darsly.mentor.data.livekit.ScreenSharePlan.Action

/** Uzilishdan keyin ekran ulashishni tiklash qarori (C-11 · 3-gipoteza). */
class ScreenSharePlanTest {

    private val ANDROID_13 = 33
    private val ANDROID_15 = 35

    @Test
    fun `ustoz ulashmagan bolsa tiklanmaydi`() {
        // Eng muhim shox: qayta ulanish HAR SAFAR ulashishni yoqib yuborsa,
        // ustoz o'zi to'xtatgan ulashish orqasidan qaytib kelardi — maxfiylik.
        assertEquals(
            Action.NONE,
            ScreenSharePlan.afterReconnect(
                wanted = false,
                sharingNow = false,
                hasToken = true,
                sdkInt = ANDROID_13,
            ),
        )
    }

    @Test
    fun `ulashish davom etayotgan bolsa tegilmaydi`() {
        // SDK o'z qayta ulanishida trekni saqlab qolgan bo'lishi mumkin —
        // bunda qayta boshlash ishlayotgan oqimni uzardi.
        assertEquals(
            Action.NONE,
            ScreenSharePlan.afterReconnect(
                wanted = true,
                sharingNow = true,
                hasToken = true,
                sdkInt = ANDROID_13,
            ),
        )
    }

    @Test
    fun `eski Android da jimgina tiklanadi`() {
        assertEquals(
            Action.REUSE_TOKEN,
            ScreenSharePlan.afterReconnect(
                wanted = true,
                sharingNow = false,
                hasToken = true,
                sdkInt = ANDROID_13,
            ),
        )
    }

    @Test
    fun `Android 14 va yuqorida rozilik qayta soraladi`() {
        // Platforma cheklovi: token qayta ishlatilsa SecurityException.
        assertEquals(
            Action.ASK_CONSENT,
            ScreenSharePlan.afterReconnect(
                wanted = true,
                sharingNow = false,
                hasToken = true,
                sdkInt = ScreenSharePlan.CONSENT_PER_SESSION_SDK,
            ),
        )
        assertEquals(
            Action.ASK_CONSENT,
            ScreenSharePlan.afterReconnect(
                wanted = true,
                sharingNow = false,
                hasToken = true,
                sdkInt = ANDROID_15,
            ),
        )
    }

    @Test
    fun `token yoq bolsa eski Android da ham rozilik soraladi`() {
        assertEquals(
            Action.ASK_CONSENT,
            ScreenSharePlan.afterReconnect(
                wanted = true,
                sharingNow = false,
                hasToken = false,
                sdkInt = ANDROID_13,
            ),
        )
    }
}
