package uz.darsly.mentor.ui.lessons

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/** Dars holatiga qarab qaysi amal ochiq (yakunlangan dars xonani ochmasin). */
class LessonActionsTest {

    @Test
    fun `yakunlangan dars xonani ochmaydi`() {
        // Server `lesson is not active` (400) qaytaradi — interfeys qila
        // olmaydigan ishni taklif qilmasligi kerak.
        assertFalse(LessonActions.canOpenRoom("ended"))
        assertEquals(LessonActions.Primary.RECORDINGS, LessonActions.primary("ended"))
        assertEquals("Yozuvlar", LessonActions.label(LessonActions.primary("ended")))
    }

    @Test
    fun `bekor qilingan darsda ochiladigan narsa yoq`() {
        assertFalse(LessonActions.canOpenRoom("cancelled"))
        assertEquals(LessonActions.Primary.NONE, LessonActions.primary("cancelled"))
    }

    @Test
    fun `jonli va rejalashtirilgan dars xonani ochadi`() {
        assertTrue(LessonActions.canOpenRoom("live"))
        assertTrue(LessonActions.canOpenRoom("scheduled"))
        assertEquals("Davom etish", LessonActions.label(LessonActions.primary("live")))
        assertEquals("Boshlash", LessonActions.label(LessonActions.primary("scheduled")))
    }

    @Test
    fun `notanish holat darsni bloklamaydi`() {
        // Backend yangi status qo'shsa (masalan "paused") ustoz xonaga kira
        // olmay qolmasin: noma'lum qiymat "boshlash" deb qaraladi.
        assertTrue(LessonActions.canOpenRoom("paused"))
        assertEquals(LessonActions.Primary.START, LessonActions.primary(""))
    }

    @Test
    fun `tugagan va bekor qilingan dars tahrirlanmaydi`() {
        // Tahrirlash faqat tugamagan darsda: sozlamalar sessiyaga tegishli,
        // sessiya tugagan bo'lsa o'zgartirish ma'nosiz (2026-08-15).
        assertFalse(LessonActions.isEditable("ended"))
        assertFalse(LessonActions.isEditable("cancelled"))
        assertTrue(LessonActions.isEditable("scheduled"))
        assertTrue(LessonActions.isEditable("live"))
    }
}
