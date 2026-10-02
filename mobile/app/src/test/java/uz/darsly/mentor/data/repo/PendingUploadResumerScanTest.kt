package uz.darsly.mentor.data.repo

import org.junit.Assert.assertEquals
import org.junit.Test
import uz.darsly.mentor.data.repo.PendingUploadResumer.PendingFile

/**
 * Yetim yozuvlarni tanlash/tartiblash/dedup (M-4, H4 nomlash bilan).
 *
 * `decide(status)` (yuklash/o'chirish/tegmaslik) allaqachon sinalgan
 * ([PendingUploadResumerTest]); bu — undan OLDINGI qadam: diskdagi qaysi fayllar
 * umuman nomzod bo'ladi. Bu "yozuvni yo'qotmaslik" xavfsizlik to'rining birinchi
 * bo'g'ini — noto'g'ri filtr buzuq faylni yuklab yoki yaroqli faylni tashlab
 * yuborardi.
 */
class PendingUploadResumerScanTest {

    private fun select(vararg files: PendingFile) =
        PendingUploadResumer.selectUploads(files.toList()).map { it.name }

    @Test fun onlyMp4FilesAreCandidates() {
        // Boshqa fayllar (masalan yarim yozilgan `.tmp`) yuklanmaydi.
        assertEquals(
            listOf("lesson-a_1.mp4"),
            select(PendingFile("lesson-a_1.mp4", 1000), PendingFile("notes.txt", 1000), PendingFile("x.tmp", 1000)),
        )
    }

    @Test fun emptyFilesAreSkipped() {
        // BUG: 0 baytli fayl — buzuq yozuv; yuklash serverni yaroqsiz fayl bilan
        // to'ldirardi ([RecorderPipeline.isUsableOutput] bilan bir qoida).
        assertEquals(
            listOf("full_1.mp4"),
            select(PendingFile("empty_1.mp4", 0), PendingFile("full_1.mp4", 500)),
        )
    }

    @Test fun orderIsStableByLesson() {
        // BUG: diskning tasodifiy tartibiga tayanilsa urinishlar har ochilishda
        // boshqacha bo'lib, xatoni takrorlash qiyinlashardi. Dars bo'yicha barqaror.
        assertEquals(
            listOf("a_1.mp4", "b_1.mp4", "c_1.mp4"),
            select(PendingFile("c_1.mp4", 1), PendingFile("a_1.mp4", 1), PendingFile("b_1.mp4", 1)),
        )
    }

    @Test fun sameLessonAppearsOnceNewestFirst() {
        // H4: bir darsda bir necha segment bo'lishi mumkin (`<lessonId>_<ms>.mp4`).
        // Bitta dars BIR marta yuklanadi — eng yangi segment tanlanadi.
        assertEquals(
            listOf("lesson-a_200.mp4"),
            select(PendingFile("lesson-a_100.mp4", 100), PendingFile("lesson-a_200.mp4", 200)),
        )
    }

    @Test fun legacyNamesStillUpload() {
        // Yangilanishdan oldingi `<lessonId>.mp4` ham nomzod.
        assertEquals(listOf("lesson-a.mp4"), select(PendingFile("lesson-a.mp4", 100)))
    }

    @Test fun emptyInputYieldsEmpty() {
        assertEquals(emptyList<String>(), PendingUploadResumer.selectUploads(emptyList()).map { it.name })
    }
}
