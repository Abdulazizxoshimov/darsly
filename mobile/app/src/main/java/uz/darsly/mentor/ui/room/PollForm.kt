package uz.darsly.mentor.ui.room

import uz.darsly.mentor.data.api.CreatePollReq
import uz.darsly.mentor.data.api.Poll

/**
 * So'rovnoma formasi va holat qoidalari — **sof** mantiq (JVM testida).
 *
 * ## Nega Compose'dan ajratilgan
 * Bu yerdagi qarorlarning aksariyati "tugma ko'rinsinmi" degan savolga javob
 * beradi va ular NOZIK: `mentor_only` so'rovnomada «E'lon qilish» bosilsa
 * server 400 beradi, ya'ni noto'g'ri ko'rsatilgan tugma ustozni boshi berk
 * ko'chaga olib borardi. Bunday qoidalarni ekran ichida saqlash ularni
 * sinovsiz qoldirardi.
 *
 * ## Chegaralar
 * `entity.CreatePollReq` bilan bir xil: savol 1..500, variantlar 2..10 ta,
 * har biri 1..200. Klientda takrorlanishi — ustoz xatoni serverga bormasdan,
 * o'zbekcha ko'rishi uchun (`LessonForm` bilan bir naqsh).
 */
object PollForm {

    const val MIN_OPTIONS = 2
    const val MAX_OPTIONS = 10
    const val MAX_QUESTION = 500
    const val MAX_OPTION = 200

    /** `entity.PollResultsMentorOnly` — natijani FAQAT mentor ko'radi (default). */
    const val MENTOR_ONLY = "mentor_only"

    /** `entity.PollResultsPublic` — mentor «E'lon qilish» bosgach o'quvchi ham ko'radi. */
    const val PUBLIC = "public"

    /**
     * Bo'sh variantlarni tashlaydi va chetdagi bo'shliqlarni oladi.
     *
     * Formada doim kamida ikkita bo'sh maydon turadi (ustoz uchtasini
     * to'ldirib, to'rtinchisini bo'sh qoldirishi normal holat) — shuning uchun
     * bo'sh qatorlar xato emas, ular shunchaki hisobga olinmaydi.
     */
    fun cleanOptions(options: List<String>): List<String> =
        options.map { it.trim() }.filter { it.isNotEmpty() }

    /** `null` — savol yaroqli. */
    fun questionError(question: String): String? {
        val q = question.trim()
        return when {
            q.isEmpty() -> "Savolni yozing"
            q.length > MAX_QUESTION -> "Savol $MAX_QUESTION belgidan oshmasligi kerak"
            else -> null
        }
    }

    /** `null` — variantlar yaroqli. */
    fun optionsError(options: List<String>): String? {
        val clean = cleanOptions(options)
        return when {
            clean.size < MIN_OPTIONS -> "Kamida $MIN_OPTIONS ta variant kerak"
            clean.size > MAX_OPTIONS -> "Ko'pi bilan $MAX_OPTIONS ta variant bo'ladi"
            clean.any { it.length > MAX_OPTION } -> "Variant $MAX_OPTION belgidan oshmasligi kerak"
            // Bir xil variantlar natijani ma'nosiz qiladi: ovozlar ikkiga
            // bo'linadi va diagramma yolg'on ko'rsatadi.
            clean.map { it.lowercase() }.toSet().size != clean.size -> "Variantlar takrorlanmasin"
            else -> null
        }
    }

    fun canSubmit(question: String, options: List<String>): Boolean =
        questionError(question) == null && optionsError(options) == null

    /** Yaroqli forma → so'rov. Chaqirishdan oldin [canSubmit] tekshirilgan bo'lsin. */
    fun request(question: String, options: List<String>, publicResults: Boolean): CreatePollReq =
        CreatePollReq(
            question = question.trim(),
            options = cleanOptions(options),
            resultsVisibility = if (publicResults) PUBLIC else MENTOR_ONLY,
        )

    // ── Holat qoidalari ──────────────────────────────────────────────────────

    /**
     * «E'lon qilish» tugmasi ko'rinsinmi.
     *
     * `mentor_only` da YO'Q: rejim yaratishda tanlangan va o'zgarmas, server
     * bunday so'rovda 400 beradi. Allaqachon e'lon qilinganda ham yo'q —
     * takroriy bosish idempotent bo'lsa-da, ustozga "yana nima qilay" degan
     * savol tug'dirardi.
     */
    fun canPublish(poll: Poll): Boolean =
        poll.resultsVisibility == PUBLIC && poll.resultsPublishedAt.isNullOrBlank()

    fun isPublished(poll: Poll): Boolean = !poll.resultsPublishedAt.isNullOrBlank()

    /** Faqat ochiq so'rovnomani yopish mumkin. */
    fun canClose(poll: Poll): Boolean = poll.isActive

    /**
     * Natija kimga ko'rinishi haqidagi bir qatorlik izoh — kartochkada doim
     * turadi. Ustoz «E'lon qilish» tugmasi NEGA yo'qligini shundan biladi.
     */
    fun visibilityHint(poll: Poll): String = when {
        poll.resultsVisibility != PUBLIC -> "Natijani faqat siz ko'rasiz"
        isPublished(poll) -> "Natija o'quvchilarga e'lon qilindi"
        else -> "Natija o'quvchilarga «E'lon qilish» bosilgach ko'rinadi"
    }

    fun statusLabel(poll: Poll): String = if (poll.isActive) "Ochiq" else "Yopilgan"

    /**
     * Ulush foizi (0..100). Umumiy nol bo'lsa 0 — "0 ovoz" holatida
     * diagramma to'la ko'rinib qolmasligi kerak.
     */
    fun percent(count: Int, total: Int): Int =
        if (total <= 0 || count <= 0) 0 else Math.round(count * 100f / total)

    /**
     * Ro'yxat tartibi: avval OCHIQ so'rovnomalar (ustoz ular bilan ishlayapti),
     * ichida esa eng yangisi birinchi.
     */
    fun sortForDisplay(polls: List<Poll>): List<Poll> =
        polls.sortedWith(
            compareByDescending<Poll> { it.isActive }
                .thenByDescending { uz.darsly.mentor.util.LessonFormat.epochOrNull(it.createdAt) ?: 0L },
        )
}
