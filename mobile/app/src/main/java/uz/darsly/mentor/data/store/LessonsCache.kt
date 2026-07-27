package uz.darsly.mentor.data.store

import android.content.Context
import android.content.SharedPreferences
import com.squareup.moshi.Moshi
import com.squareup.moshi.Types
import io.livekit.android.util.LKLog
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
 * Shifrlanmagan: dars sarlavhasi maxfiy emas, token esa bu yerga umuman tushmaydi.
 */
interface LessonsCache {
    /** Oxirgi saqlangan ro'yxat; hech qachon saqlanmagan yoki buzilgan bo'lsa `null`. */
    fun read(): CachedLessons?

    /** Muvaffaqiyatli tarmoq javobidan keyin ustiga yozadi. */
    fun write(lessons: List<Lesson>, savedAtMillis: Long)

    /** Logout paytida — boshqa ustozning darslari ko'rinib qolmasin. */
    fun clear()
}

/** Keshdan o'qilgan ro'yxat va u qachon saqlangani (UI "oxirgi yangilanish" uchun). */
data class CachedLessons(
    val lessons: List<Lesson>,
    val savedAtMillis: Long,
)

class PrefsLessonsCache(
    private val prefs: SharedPreferences,
    moshi: Moshi = Moshi.Builder().build(),
) : LessonsCache {

    private val adapter = moshi.adapter<List<Lesson>>(
        Types.newParameterizedType(List::class.java, Lesson::class.java),
    )

    override fun read(): CachedLessons? {
        val json = prefs.getString(KEY_JSON, null)?.takeIf { it.isNotBlank() } ?: return null
        val lessons = runCatching { adapter.fromJson(json) }
            .onFailure {
                // Model o'zgargan yoki fayl buzilgan — keshni tashlab yuboramiz.
                // Ilova crash bo'lmasligi kerak: kesh — qulaylik, majburiyat emas.
                LKLog.w(it) { "darslar keshi o'qilmadi — tozalanadi" }
                clear()
            }
            .getOrNull() ?: return null
        return CachedLessons(lessons, prefs.getLong(KEY_SAVED_AT, 0L))
    }

    override fun write(lessons: List<Lesson>, savedAtMillis: Long) {
        val json = runCatching { adapter.toJson(lessons) }.getOrNull() ?: return
        prefs.edit()
            .putString(KEY_JSON, json)
            .putLong(KEY_SAVED_AT, savedAtMillis)
            .apply()
    }

    override fun clear() {
        prefs.edit().remove(KEY_JSON).remove(KEY_SAVED_AT).apply()
    }

    companion object {
        private const val PREFS_NAME = "darsly_lessons_cache"
        private const val KEY_JSON = "lessons_json"
        private const val KEY_SAVED_AT = "saved_at"

        fun create(context: Context): LessonsCache = PrefsLessonsCache(
            context.applicationContext.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE),
        )
    }
}
