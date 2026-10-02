package uz.darsly.mentor.service

import android.content.pm.ServiceInfo
import org.junit.Assert.assertEquals
import org.junit.Test

/** H6 va B-2 — foreground servis qarorlari. */
class ForegroundServicePlanTest {

    private val proj = ServiceInfo.FOREGROUND_SERVICE_TYPE_MEDIA_PROJECTION
    private val mic = ServiceInfo.FOREGROUND_SERVICE_TYPE_MICROPHONE
    private val cam = ServiceInfo.FOREGROUND_SERVICE_TYPE_CAMERA

    @Test
    fun `H6 — startForeground yiqilsa jonli sessiya OLDIRILMAYDI`() {
        assertEquals(
            ForegroundServicePlan.OnForegroundFailed.KEEP_RUNNING,
            ForegroundServicePlan.onForegroundFailed(sessionActive = true),
        )
        assertEquals(
            ForegroundServicePlan.OnForegroundFailed.STOP_SELF,
            ForegroundServicePlan.onForegroundFailed(sessionActive = false),
        )
    }

    @Test
    fun `Android 14 — faqat berilgan ruxsat tiplari`() {
        assertEquals(mic, ForegroundServicePlan.serviceTypes(34, withProjection = false, micGranted = true, camGranted = false))
        assertEquals(mic or cam, ForegroundServicePlan.serviceTypes(34, false, true, true))
        assertEquals(proj or mic or cam, ForegroundServicePlan.serviceTypes(34, true, true, true))
        assertEquals(0, ForegroundServicePlan.serviceTypes(34, false, false, false))
    }

    @Test
    fun `Android 10 — mikrofon va kamera bitlari yuborilmaydi`() {
        // Q da MICROPHONE/CAMERA konstantalari tizimga notanish bit bo'lardi.
        assertEquals(proj, ForegroundServicePlan.serviceTypes(29, true, true, true))
        assertEquals(0, ForegroundServicePlan.serviceTypes(29, false, true, true))
    }

    @Test
    fun `Android 8 — tipsiz`() {
        assertEquals(0, ForegroundServicePlan.serviceTypes(26, true, true, true))
    }
}
