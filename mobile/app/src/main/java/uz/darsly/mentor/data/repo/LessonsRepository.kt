package uz.darsly.mentor.data.repo

import android.content.Context
import uz.darsly.mentor.data.api.CreateLessonReq
import uz.darsly.mentor.data.api.DarslyApi
import uz.darsly.mentor.data.api.Lesson
import uz.darsly.mentor.data.api.Net
import uz.darsly.mentor.data.store.CachedLessons
import uz.darsly.mentor.data.store.LessonsCache
import uz.darsly.mentor.data.store.PrefsLessonsCache
import java.io.IOException

/**
 * Bir yangilanishning natijasi: darslar + serverdagi umumiy son + qirqilganmi.
 *
 * [truncated] `true` bo'lsa UI buni **aytishi shart** — jimgina kesib tashlash
 * ustozga "hammasi shu" degan yolg'on taassurot beradi.
 */
data class LessonsPage(
    val lessons: List<Lesson>,
    val total: Int,
    val truncated: Boolean,
)

/**
 * Darslar repozitoriysi (M6 · M7).
 *
 * ViewModel endi [Net] ga to'g'ridan-to'g'ri murojaat qilmaydi — tarmoq + kesh
 * qarori shu yerda. Shu tufayli offline xulqi (tarmoq o'ldi → keshdagi ro'yxat)
 * **JVM testida** MockWebServer bilan to'liq sinaladi.
 *
 * Qatlam yo'nalishi: `ui/ → data/repo/ → data/api/ + data/store/`.
 */
class LessonsRepository(
    private val api: DarslyApi,
    private val cache: LessonsCache,
    private val now: () -> Long = System::currentTimeMillis,
) {

    /** Tarmoqqa chiqmasdan, darhol: oxirgi ma'lum ro'yxat (bo'lmasa `null`). */
    fun cached(): CachedLessons? = cache.read()

    /**
     * Serverdan **to'liq** ro'yxat. Muvaffaqiyatda kesh ustiga yoziladi.
     *
     * SAHIFALASH (🟡E): avval faqat birinchi 50 ta olinardi va `total` e'tiborsiz
     * qolardi — 50 tadan ko'p darsi bor ustoz qolganini **jimgina** ko'rmasdi.
     * Endi `total` qoplanguncha sahifalar ketma-ket olinadi.
     *
     * Cheklov OSHKORA: [MAX_PAGES] dan oshsa [LessonsPage.truncated] `true` bo'ladi va
     * UI buni aytadi — jim qirqish "hammasi ko'rsatildi" degan yolg'on taassurot beradi.
     *
     * Xato holida kesh TEGILMAYDI — internet uzilgani ustozning ro'yxatini
     * o'chirish uchun sabab emas.
     */
    suspend fun refresh(pageSize: Int = PAGE_SIZE): Result<LessonsPage> = runCatching {
        val all = mutableListOf<Lesson>()
        var page = 1
        var total = 0
        var truncated = false
        while (true) {
            val env = api.lessons(limit = pageSize, page = page)
            val chunk = env.data.orEmpty()
            all += chunk
            total = env.total
            // Bo'sh sahifa — server aytgan `total` haqiqatdan katta bo'lsa ham to'xtaymiz
            // (aks holda cheksiz sikl bo'lardi).
            if (chunk.isEmpty() || all.size >= total) break
            if (page >= MAX_PAGES) {
                truncated = true
                break
            }
            page++
        }
        cache.write(all, now())
        LessonsPage(lessons = all, total = total, truncated = truncated)
    }

    /**
     * Yangi dars (M7). Server javobidagi `join_slug` darhol ulashishga yaroqli.
     *
     * Kesh oldiga qo'shiladi: yaratgandan keyin internet uzilsa ham dars ro'yxatda
     * qoladi (server `created_at DESC` bo'yicha tartiblaydi — biz ham shunday).
     */
    suspend fun create(req: CreateLessonReq): Result<Lesson> = runCatching {
        val lesson = api.createLesson(req).data
            ?: throw IllegalStateException("Server bo'sh javob qaytardi")
        cache.read()?.let { cached ->
            cache.write(listOf(lesson) + cached.lessons.filterNot { it.id == lesson.id }, now())
        }
        lesson
    }

    // DIQQAT: keshni tozalash bu yerda EMAS — u `DarslyApp.observeLogoutCleanup()` da,
    // `Session.loggedIn == false` signali bo'yicha (🟡B). Bitta egasi bo'lishi kerak,
    // aks holda qattiq logout yo'li e'tibordan chetda qolardi.

    companion object {
        /** Bitta so'rovdagi yozuvlar soni (backend default'i 20, maksimumi kattaroq). */
        const val PAGE_SIZE = 50

        /** Xavfsizlik chegarasi: 10 × 50 = 500 dars. Oshsa foydalanuvchiga aytiladi. */
        const val MAX_PAGES = 10

        /**
         * Tarmoq uzilishimi (offline belgisi uchun) yoki server/API xatosimi.
         *
         * Retrofit `IOException` ni o'ramaydi — ulanmagan holat aynan shu tipda keladi;
         * HTTP status xatolari esa `HttpException` (u `IOException` emas).
         */
        fun isOffline(t: Throwable): Boolean = t is IOException

        fun create(context: Context): LessonsRepository = LessonsRepository(
            api = Net.api,
            cache = PrefsLessonsCache.create(context),
        )
    }
}
