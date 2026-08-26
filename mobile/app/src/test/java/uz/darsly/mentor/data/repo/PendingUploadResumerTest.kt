package uz.darsly.mentor.data.repo

import org.junit.Assert.assertEquals
import org.junit.Test

/**
 * Yetim yozuv fayli bilan nima qilish qarori. Bu — ma'lumot yo'qotmaslikning
 * yagona qopqog'i: `recording` yuklanishi, `ready` tozalanishi, qolgani esa
 * TEGILMASLIGI shart (aks holda hali yuklanmagan fayl o'chib ketishi mumkin).
 */
class PendingUploadResumerTest {

    @Test fun recordingUploads() {
        assertEquals(PendingUploadResumer.Action.UPLOAD, PendingUploadResumer.decide("recording"))
    }

    @Test fun readyDeletesOrphanFile() {
        assertEquals(PendingUploadResumer.Action.DELETE, PendingUploadResumer.decide("ready"))
    }

    @Test fun otherStatusesAreLeftUntouched() {
        // processing/failed/archived/expired/null/xato — faylga TEGMAYMIZ (saqlanadi).
        for (s in listOf("processing", "failed", "archived", "expired", "unknown", "", null)) {
            assertEquals("status=$s → SKIP", PendingUploadResumer.Action.SKIP, PendingUploadResumer.decide(s))
        }
    }
}
