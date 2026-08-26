package uz.darsly.mentor.data.repo

import org.junit.Assert.assertEquals
import org.junit.Test
import uz.darsly.mentor.data.repo.PendingUploadResumer.PendingFile

/**
 * Yetim yozuvlarni tanlash/tartiblash/dedup (M-4).
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
            listOf("lesson-a.mp4"),
            select(PendingFile("lesson-a.mp4", 1000), PendingFile("notes.txt", 1000), PendingFile("x.tmp", 1000)),
        )
    }

    @Test fun emptyFilesAreSkipped() {
        // BUG: 0 baytli fayl — buzuq yozuv; yuklash serverni yaroqsiz fayl bilan
        // to'ldirardi ([RecorderPipeline.isUsableOutput] bilan bir qoida).
        assertEquals(
            listOf("full.mp4"),
            select(PendingFile("empty.mp4", 0), PendingFile("full.mp4", 500)),
        )
    }

    @Test fun orderIsStableByName() {
        // BUG: diskning tasodifiy tartibiga tayanilsa urinishlar har ochilishda
        // boshqacha bo'lib, xatoni takrorlash qiyinlashardi. Nom bo'yicha barqaror.
        assertEquals(
            listOf("a.mp4", "b.mp4", "c.mp4"),
            select(PendingFile("c.mp4", 1), PendingFile("a.mp4", 1), PendingFile("b.mp4", 1)),
        )
    }

    @Test fun sameLessonAppearsOnce() {
        // Fayl nomi = `<lessonId>.mp4`; bir dars ikki marta yuklanmasin.
        // (Amalda bir nom ikki marta bo'lmaydi, lekin dedup himoya sifatida qoladi.)
        assertEquals(
            listOf("lesson-a.mp4"),
            select(PendingFile("lesson-a.mp4", 100), PendingFile("lesson-a.mp4", 200)),
        )
    }

    @Test fun emptyInputYieldsEmpty() {
        assertEquals(emptyList<String>(), PendingUploadResumer.selectUploads(emptyList()).map { it.name })
    }
}
