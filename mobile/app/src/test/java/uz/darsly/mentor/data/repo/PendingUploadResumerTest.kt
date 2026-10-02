package uz.darsly.mentor.data.repo

import org.junit.Assert.assertEquals
import org.junit.Test

/**
 * Yetim yozuv fayli bilan nima qilish qarori (S4 bilan yangilangan).
 *
 * Bu — ma'lumot yo'qotmaslik va maxfiylik o'rtasidagi chegara: `recording`
 * yuklanadi; `ready`/`failed`/`expired` (server yakuniy qaror bergan) o'chadi;
 * noma'lum holat saqlanadi, LEKIN 7 kundan keyin — o'quvchilar yozuvi
 * telefonda abadiy yotmasin.
 */
class PendingUploadResumerTest {

    private val day = 24L * 60 * 60 * 1000

    @Test fun recordingUploads() {
        assertEquals(PendingUploadResumer.Action.UPLOAD, PendingUploadResumer.decide("recording"))
        // Yosh ahamiyatsiz — hali yuklanmagan fayl doim yuklanadi.
        assertEquals(PendingUploadResumer.Action.UPLOAD, PendingUploadResumer.decide("recording", ageMs = 30 * day))
    }

    @Test fun readyDeletesOrphanFile() {
        assertEquals(PendingUploadResumer.Action.DELETE, PendingUploadResumer.decide("ready"))
    }

    @Test fun serverGaveUpDeletes() {
        // S4: `failed`/`expired` — server hech qachon qabul qilmaydi; fayl faqat joy va xavf.
        assertEquals(PendingUploadResumer.Action.DELETE, PendingUploadResumer.decide("failed"))
        assertEquals(PendingUploadResumer.Action.DELETE, PendingUploadResumer.decide("expired"))
    }

    @Test fun unknownStatusesAreKeptWhileYoung() {
        for (s in listOf("processing", "archived", "unknown", "", null)) {
            assertEquals("status=$s → SKIP", PendingUploadResumer.Action.SKIP, PendingUploadResumer.decide(s, ageMs = 6 * day))
        }
    }

    @Test fun unknownStatusesAgeOut() {
        for (s in listOf("processing", "archived", "unknown", "", null)) {
            assertEquals("status=$s eskirgan → DELETE", PendingUploadResumer.Action.DELETE, PendingUploadResumer.decide(s, ageMs = 8 * day))
        }
        assertEquals(PendingUploadResumer.Action.SKIP, PendingUploadResumer.decide(null, ageMs = PendingUploadResumer.MAX_AGE_MS))
    }
}
