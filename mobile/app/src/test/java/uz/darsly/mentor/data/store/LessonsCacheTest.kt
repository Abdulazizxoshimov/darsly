package uz.darsly.mentor.data.store

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import uz.darsly.mentor.data.api.Lesson

/**
 * M6 · mezon B-4 — offline kesh.
 *
 * [PrefsTokenStore] testidagi kabi: ishlab chiqarish klassining AYNI o'zi sinaladi,
 * faqat `SharedPreferences` o'rniga faqe (Robolectric kerak emas).
 */
class LessonsCacheTest {

    private fun lesson(id: String, title: String = "Algebra") = Lesson(
        id = id,
        title = title,
        status = "scheduled",
        joinSlug = "slug-$id",
        durationMin = 45,
    )

    @Test
    fun `saqlangan royxat ilova qayta ochilganda oqiladi`() {
        val prefs = FakeSharedPreferences()
        PrefsLessonsCache(prefs).write(listOf(lesson("1"), lesson("2")), savedAtMillis = 1234L)

        // Yangi obyekt — "ilova qayta ishga tushdi" holati (xotira keshi yordam bermaydi).
        val cached = PrefsLessonsCache(prefs.reopen()).read()

        assertEquals(listOf("1", "2"), cached!!.lessons.map { it.id })
        assertEquals("slug-1", cached.lessons[0].joinSlug)
        assertEquals(45, cached.lessons[0].durationMin)
        assertEquals(1234L, cached.savedAtMillis)
    }

    @Test
    fun `hech qachon saqlanmagan bolsa null`() {
        assertNull(PrefsLessonsCache(FakeSharedPreferences()).read())
    }

    @Test
    fun `buzilgan JSON ilovani yiqitmaydi va kesh tozalanadi`() {
        val prefs = FakeSharedPreferences()
        prefs.edit().putString("lessons_json", "{bu JSON emas").apply()
        val cache = PrefsLessonsCache(prefs)

        assertNull("buzilgan kesh null qaytarishi kerak", cache.read())
        // Ikkinchi o'qishda ham crash bo'lmasin va kalit o'chgan bo'lsin.
        assertNull(cache.read())
        assertNull(prefs.getString("lessons_json", null))
    }

    @Test
    fun `clear keshni ochiradi`() {
        val prefs = FakeSharedPreferences()
        val cache = PrefsLessonsCache(prefs)
        cache.write(listOf(lesson("1")), savedAtMillis = 1L)
        cache.clear()
        assertNull(cache.read())
    }

    @Test
    fun `bosh royxat ham saqlanadi`() {
        // Ustoz hamma darsni o'chirsa, kesh eski ro'yxatni "tirilib" ko'rsatmasligi kerak.
        val prefs = FakeSharedPreferences()
        val cache = PrefsLessonsCache(prefs)
        cache.write(listOf(lesson("1")), savedAtMillis = 1L)
        cache.write(emptyList(), savedAtMillis = 2L)
        assertTrue(cache.read()!!.lessons.isEmpty())
    }
}
