package uz.darsly.mentor.ui.lessons

import uz.darsly.mentor.data.api.CreateLessonReq
import java.time.Instant
import java.time.ZoneId
import java.time.ZoneOffset
import java.time.format.DateTimeFormatter

/**
 * Dars yaratish formasining **sof** mantiqi (M7 · mezon B-7).
 *
 * Chegaralar `backend/internal/entity/lesson.go` dagi `validate` teglaridan aynan
 * ko'chirilgan. Nega klientda takrorlanadi: server xatosi inglizcha va maydon
 * darajasida keladi ("title: must be at least 2 characters") — ustoz uni tushunmaydi.
 * Bu yerda xato **o'zbekcha va maydon yonida**, tarmoqqa chiqmasdan ko'rsatiladi.
 *
 * Compose'dan ajratilgan: JVM testida to'liq sinaladi.
 */
object LessonForm {

    const val TITLE_MIN = 2
    const val TITLE_MAX = 255
    const val DESCRIPTION_MAX = 2000
    const val DURATION_MIN = 5
    const val DURATION_MAX = 1440
    const val DURATION_DEFAULT = 60
    const val PASSCODE_MIN = 4
    const val PASSCODE_MAX = 20

    /** Formadagi xom kiritma (matn maydonlari — satr, chunki foydalanuvchi shunday yozadi). */
    data class Input(
        val title: String = "",
        val description: String = "",
        /** Tanlangan sana+vaqt (epoch millis) yoki `null` — "vaqti belgilanmagan". */
        val scheduledAtMillis: Long? = null,
        val duration: String = DURATION_DEFAULT.toString(),
        val passcode: String = "",
        val waitingRoom: Boolean = false,
        val recording: Boolean = false,
    )

    data class Errors(
        val title: String? = null,
        val description: String? = null,
        val duration: String? = null,
        val passcode: String? = null,
        /** Bloklamaydi — faqat ogohlantiradi (o'tgan vaqt ba'zan ataylab tanlanadi). */
        val scheduleWarning: String? = null,
    ) {
        /** Yaratish tugmasi faqat shu `true` bo'lganda ishlaydi. */
        val isValid: Boolean
            get() = title == null && description == null && duration == null && passcode == null
    }

    fun validate(input: Input, nowMillis: Long = System.currentTimeMillis()): Errors {
        val title = input.title.trim()
        val description = input.description.trim()
        val duration = input.duration.trim()
        val passcode = input.passcode.trim()

        return Errors(
            title = when {
                title.length < TITLE_MIN -> "Sarlavha kamida $TITLE_MIN belgidan iborat bo'lsin"
                title.length > TITLE_MAX -> "Sarlavha $TITLE_MAX belgidan oshmasin"
                else -> null
            },
            description = if (description.length > DESCRIPTION_MAX) {
                "Tavsif $DESCRIPTION_MAX belgidan oshmasin"
            } else {
                null
            },
            duration = when {
                // Bo'sh qoldirilsa server default'i (60 daqiqa) ishlatiladi.
                duration.isEmpty() -> null
                duration.toIntOrNull() == null -> "Davomiylik faqat raqamlardan iborat bo'lsin"
                duration.toInt() < DURATION_MIN || duration.toInt() > DURATION_MAX ->
                    "Davomiylik $DURATION_MIN–$DURATION_MAX daqiqa oralig'ida bo'lsin"
                else -> null
            },
            passcode = when {
                passcode.isEmpty() -> null
                passcode.length < PASSCODE_MIN || passcode.length > PASSCODE_MAX ->
                    "Parol $PASSCODE_MIN–$PASSCODE_MAX belgidan iborat bo'lsin"
                else -> null
            },
            scheduleWarning = input.scheduledAtMillis
                ?.takeIf { it < nowMillis - PAST_TOLERANCE_MILLIS }
                ?.let { "Tanlangan vaqt allaqachon o'tgan" },
        )
    }

    /**
     * Formani so'rovga aylantiradi. FAQAT [Errors.isValid] bo'lganda chaqiriladi.
     *
     * Bo'sh matnlar `null` bo'lib ketadi — Moshi ularni JSON'ga qo'shmaydi, ya'ni
     * server "bo'sh satr" o'rniga "berilmagan" deb ko'radi (`omitempty` validatsiyasi).
     */
    fun toRequest(input: Input): CreateLessonReq = CreateLessonReq(
        title = input.title.trim(),
        description = input.description.trim().takeIf { it.isNotEmpty() },
        scheduledAt = input.scheduledAtMillis?.let { rfc3339Utc(it) },
        durationMin = input.duration.trim().toIntOrNull() ?: DURATION_DEFAULT,
        passcode = input.passcode.trim().takeIf { it.isNotEmpty() },
        isRecordingEnabled = input.recording,
        isWaitingRoomEnabled = input.waitingRoom,
    )

    /**
     * Epoch millis → `2026-07-27T14:30:00Z`.
     *
     * Go `time.RFC3339` soniyalarni TALAB qiladi, shuning uchun shabloni aniq
     * belgilaymiz (`Instant.toString()` ga tayanmaymiz).
     */
    fun rfc3339Utc(millis: Long): String =
        RFC3339.format(Instant.ofEpochMilli(millis).atOffset(ZoneOffset.UTC))

    /**
     * Sana tanlagich + vaqt tanlagich natijalarini bitta lahzaga qo'shadi.
     *
     * DIQQAT: Material3 `DatePicker` sanani **UTC yarim tunida** qaytaradi. To'g'ridan-
     * to'g'ri soat qo'shilsa Toshkent (UTC+5) da dars 5 soat surilib ketardi —
     * shuning uchun avval UTC'dan **kalendar sanasi** olinadi, keyin u qurilma
     * mintaqasidagi soat bilan birlashtiriladi.
     */
    fun combineDateAndTime(
        dateUtcMillis: Long,
        hour: Int,
        minute: Int,
        zone: ZoneId = ZoneId.systemDefault(),
    ): Long = Instant.ofEpochMilli(dateUtcMillis)
        .atZone(ZoneOffset.UTC)
        .toLocalDate()
        .atTime(hour, minute)
        .atZone(zone)
        .toInstant()
        .toEpochMilli()

    /** Bir daqiqalik "o'tgan vaqt" bag'rikengligi — tanlash paytidagi kechikish uchun. */
    private const val PAST_TOLERANCE_MILLIS = 60_000L

    private val RFC3339: DateTimeFormatter =
        DateTimeFormatter.ofPattern("yyyy-MM-dd'T'HH:mm:ss'Z'")
}
