package uz.darsly.mentor.data.livekit

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import kotlin.math.abs

/**
 * Yozuv tuvaliga joylash hisobi. Bu — foydalanuvchi ko'radigan yagona haqiqat:
 * xato bo'lsa video siljib qoladi yoki xroma buzilib rangli chiziqlar chiqadi.
 */
class LetterboxFitTest {

    private val W = 1280
    private val H = 720

    @Test fun landscapeSameAspectFillsCanvas() {
        // 16:9 manba (masalan planshet doskasi) tuvalni to'liq to'ldiradi — chegara yo'q.
        assertEquals(LetterboxFit.Rect(0, 0, 1280, 720), LetterboxFit.fit(1920, 1080, W, H))
    }

    @Test fun portraitGetsSideBars() {
        // Tik telefon (20:9): balandlik to'liq, yon tomonda qora chegara, markazda.
        val r = LetterboxFit.fit(1080, 2400, W, H)
        assertEquals(720, r.h)
        assertEquals(0, r.y)
        assertTrue("kenglik tuvaldan kichik", r.w < W)
        assertTrue("chap chegara musbat", r.x > 0)
        assertEquals("markazda (chap = o'ng)", r.x, W - r.w - r.x)
    }

    @Test fun widerThanCanvasGetsTopBottomBars() {
        // Tuvaldan kengroq manba: kenglik to'liq, tepa/pastda chegara.
        val r = LetterboxFit.fit(2400, 1080, W, H)
        assertEquals(1280, r.w)
        assertEquals(0, r.x)
        assertTrue(r.h < H)
        assertTrue(r.y > 0)
        assertEquals("markazda (tepa = past)", r.y, H - r.h - r.y)
    }

    @Test fun the572x1280SampleBecomesLandscapeFile() {
        // Haqiqiy nosoz namuna (ustoz shikoyat qilgan tik yozuv). Endi fayl
        // gorizontal tuvalga tushadi — pleyerda to'liq ekranga ochiladi.
        val r = LetterboxFit.fit(572, 1280, W, H)
        assertEquals(720, r.h)
        assertTrue("tik manba → yon chegara", r.w < W)
        assertTrue(r.w in 2..W)
    }

    @Test fun aspectRatioPreserved() {
        val r = LetterboxFit.fit(1080, 2400, W, H)
        val srcAspect = 1080.0 / 2400.0
        val dstAspect = r.w.toDouble() / r.h
        assertTrue("nisbat ~saqlangan", abs(srcAspect - dstAspect) < 0.02)
    }

    @Test fun allValuesEvenAndInside() {
        val cases = listOf(
            572 to 1280, 1000 to 1001, 333 to 777, 101 to 99, 1920 to 1080, 2 to 5000,
        )
        for ((sw, sh) in cases) {
            val r = LetterboxFit.fit(sw, sh, W, H)
            val tag = "${sw}x$sh"
            assertEquals("$tag: x juft", 0, r.x % 2)
            assertEquals("$tag: y juft", 0, r.y % 2)
            assertEquals("$tag: w juft", 0, r.w % 2)
            assertEquals("$tag: h juft", 0, r.h % 2)
            assertTrue("$tag: gorizontal chegarada", r.x + r.w <= W)
            assertTrue("$tag: vertikal chegarada", r.y + r.h <= H)
            assertTrue("$tag: musbat o'lcham", r.w >= 2 && r.h >= 2)
            assertTrue("$tag: musbat offset", r.x >= 0 && r.y >= 0)
        }
    }

    @Test fun unknownSizeReturnsFullCanvas() {
        assertEquals(LetterboxFit.Rect(0, 0, W, H), LetterboxFit.fit(0, 0, W, H))
        assertEquals(LetterboxFit.Rect(0, 0, W, H), LetterboxFit.fit(-1, 100, W, H))
    }
}
