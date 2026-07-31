package uz.darsly.mentor.data.store

import android.content.Context
import android.content.SharedPreferences
import com.squareup.moshi.Moshi
import com.squareup.moshi.Types
import io.livekit.android.util.LKLog
import kotlinx.coroutines.CoroutineDispatcher
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import uz.darsly.mentor.data.api.Lesson

/**
 * Darslar ro'yxatining **offline keshi** (M6 · mezon B-4).
 *
 * NEGA KERAK: ustoz darsni ko'pincha yo'lda, liftda yoki maktab yerto'lasida ochadi.
 * Internet bo'lmasa bo'sh ekran o'rniga oxirgi ma'lum ro'yxat ko'rsatiladi va
 * yuqorida "internet yo'q" belgisi turadi.
 *
 * NEGA `SharedPreferences` (DataStore emas): ro'yxat kichik (o'nlab yozuv), bitta
 * kalit ostida JSON sifatida yotadi, va bu interfeys [PrefsTokenStore] bilan bir xil
 * naqshda — ya'ni [uz.darsly.mentor.data.store.FakeSharedPreferences] bilan **ishlab
 * chiqarish klassining aynan o'zi** JVM testida sinaladi (Robolectric kerak emas).
 *
 * Kesh SHIFRLANGAN (`EncryptedSharedPreferences`) — sababi [PrefsLessonsCache.create] da.
 *
 * ## Nega metodlar `suspend`
 *
 * Ular DISKKA tegadi: `SharedPreferences` birinchi murojaatda butun XML'ni
 * o'qib parse qiladi, ustiga shifr ochish va Moshi parse qo'shiladi. Avval
 * bular asosiy oqimdan chaqirilardi (`LessonsViewModel.start()` →
 * `repo.cached()`) — ya'ni ko'p darsli ustozda ANR xavfi bor edi va u
 * kesh shifrlangach yanada oshdi.
 */
interface LessonsCache {
    /** Oxirgi saqlangan ro'yxat; hech qachon saqlanmagan yoki buzilgan bo'lsa `null`. */
    suspend fun read(): CachedLessons?

    /** Muvaffaqiyatli tarmoq javobidan keyin ustiga yozadi. */
    suspend fun write(lessons: List<Lesson>, savedAtMillis: Long)

    /** Logout paytida — boshqa ustozning darslari ko'rinib qolmasin. */
    suspend fun clear()
}

/** Keshdan o'qilgan ro'yxat va u qachon saqlangani (UI "oxirgi yangilanish" uchun). */
data class CachedLessons(
    val lessons: List<Lesson>,
    val savedAtMillis: Long,
)

class PrefsLessonsCache(
    private val prefs: SharedPreferences,
    moshi: Moshi = Moshi.Builder().build(),
    /** Test uchun almashtiriladi (`UnconfinedTestDispatcher`). */
    private val io: CoroutineDispatcher = Dispatchers.IO,
) : LessonsCache {

    private val adapter = moshi.adapter<List<Lesson>>(
        Types.newParameterizedType(List::class.java, Lesson::class.java),
    )

    override suspend fun read(): CachedLessons? = withContext(io) {
        val json = prefs.getString(KEY_JSON, null)?.takeIf { it.isNotBlank() }
            ?: return@withContext null
        val lessons = runCatching { adapter.fromJson(json) }
            .onFailure {
                // Model o'zgargan yoki fayl buzilgan — keshni tashlab yuboramiz.
                // Ilova crash bo'lmasligi kerak: kesh — qulaylik, majburiyat emas.
                LKLog.w(it) { "darslar keshi o'qilmadi — tozalanadi" }
                clear()
            }
            .getOrNull() ?: return@withContext null
        CachedLessons(lessons, prefs.getLong(KEY_SAVED_AT, 0L))
    }

    override suspend fun write(lessons: List<Lesson>, savedAtMillis: Long) = withContext(io) {
        val json = runCatching { adapter.toJson(lessons) }.getOrNull() ?: return@withContext
        prefs.edit()
            .putString(KEY_JSON, json)
            .putLong(KEY_SAVED_AT, savedAtMillis)
            .apply()
    }

    override suspend fun clear() = withContext(io) {
        prefs.edit().remove(KEY_JSON).remove(KEY_SAVED_AT).apply()
    }

    companion object {
        private const val PREFS_NAME = "darsly_lessons_cache"
        private const val KEY_JSON = "lessons_json"
        private const val KEY_SAVED_AT = "saved_at"

        /**
         * Kesh SHIFRLANGAN saqlagichda (maxfiylik).
         *
         * # Nega
         *
         * Keshda dars sarlavhalari va **`join_slug`** ochiq matnda yotardi.
         * `join_slug` — darsga kirish kaliti: uni bilgan har kim (kutish xonasi
         * yoqilmagan bo'lsa) darsga qo'shila oladi. Root'langan qurilmada,
         * qurilma zaxira nusxasida yoki boshqa ilova orqali fayl o'qilsa,
         * o'quvchilarning kelajakdagi darslariga kirish havolalari sizardi.
         *
         * Token allaqachon shu yo'l bilan saqlanadi (`SecureTokenStore`) —
         * kirish kaliti bo'lgan ma'lumot esa undan kam himoyalangan bo'lishi
         * mantiqsiz edi.
         *
         * Shifrlash ochilmasa (OEM Keystore nosozligi) kesh UMUMAN yozilmaydi
         * (`NoopLessonsCache`): ilova ishlaydi, faqat oflayn ro'yxat bo'lmaydi.
         * Bu shifrlanmagan holatga qaytishdan xavfsizroq.
         */
        fun create(context: Context): LessonsCache {
            val ctx = context.applicationContext
            val prefs = SecureTokenStore.openEncryptedPrefs(ctx, PREFS_NAME, firstAttempt = true)
                ?: run {
                    SecureTokenStore.wipePrefs(ctx, PREFS_NAME)
                    SecureTokenStore.openEncryptedPrefs(ctx, PREFS_NAME, firstAttempt = false)
                }
            return if (prefs != null) PrefsLessonsCache(prefs) else NoopLessonsCache
        }
    }
}

/**
 * Kesh ishlamaydigan variant — shifrlangan saqlagich ochilmaganda.
 *
 * "Shifrlanmagan holatga qaytish" ATAYLAB tanlanmadi: kesh qulaylik,
 * `join_slug` esa kirish kaliti. Qulaylik uchun maxfiylikni almashtirish
 * noto'g'ri savdo bo'lardi.
 */
object NoopLessonsCache : LessonsCache {
    override suspend fun read(): CachedLessons? = null
    override suspend fun write(lessons: List<Lesson>, savedAtMillis: Long) = Unit
    override suspend fun clear() = Unit
}
