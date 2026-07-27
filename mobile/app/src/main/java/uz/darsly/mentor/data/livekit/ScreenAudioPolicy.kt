package uz.darsly.mentor.data.livekit

import android.os.Build

/**
 * Ekran audiosi qachon ISHLASHI MUMKINligi — sof qoidalar (B-4 · B-5).
 *
 * ## Nega alohida
 * UI "Ekran audiosi: yoniq" deb yozardi, aslida esa mikrofon o'chirilgan bo'lsa ovoz
 * o'tmasdi — **interfeys yolg'on gapirardi**. Yolg'onning sababi: holat bayrog'i bilan
 * haqiqiy shart-sharoit turli joylarda edi. Endi qoida bitta joyda va JVM testida
 * sinaladi; [LessonSession] va UI ikkalasi ham shunga tayanadi.
 *
 * ## C-6 dan keyin: mikrofon endi TO'SIQ EMAS
 * Ekran audiosi alohida trekka chiqarildi ([ScreenAudioPlan]), shuning uchun ustoz
 * mute bosganda ham ovoz o'tadi. [INDEPENDENT_TRACK] — yagona kalit: agar qurilma
 * sinovi ikki-trek yondashuvini rad etsa, uni `false` qilish eski (mikrofonga
 * bog'liq) xulqni va unga mos rost matnlarni qaytaradi.
 */
object ScreenAudioPolicy {

    /**
     * Ekran audiosi mikrofondan mustaqilmi (C-6 · ikki trek navbatlashtirish).
     *
     * `false` qilinsa: eski xulq — ekran audiosi mikrofon trekiga mikslanadi va
     * mikrofon o'chirilganda [Blocked.MIC_OFF] bilan to'xtaydi. Bu zaxira yo'l
     * ataylab saqlanadi: qurilmada ikki-trek ishlamasa bitta konstanta bilan
     * qaytish mumkin (kod va testlar ikkalasini ham qoplaydi).
     */
    const val INDEPENDENT_TRACK = true

    /** Ekran audiosi nima uchun ishlamayotgani (UI shu matnni ko'rsatadi). */
    enum class Blocked {
        /** To'siq yo'q — audio oqishi mumkin. */
        NONE,

        /** Android 9 va pastda `ScreenAudioCapturer` mavjud emas (`@RequiresApi(Q)`). */
        OLD_ANDROID,

        /** `RECORD_AUDIO` berilmagan. */
        NO_PERMISSION,

        /** Ekran ulashilmayotgan bo'lsa audio manbasi ham yo'q. */
        NOT_SHARING,

        /**
         * Mikrofon o'chirilgan.
         *
         * FAQAT [INDEPENDENT_TRACK] `false` bo'lganda (zaxira yo'l) ishlatiladi:
         * o'sha rejimda audio mikrofon trekiga mikslanadi va mute uni to'xtatadi.
         */
        MIC_OFF,
    }

    fun blockedBy(
        sdkInt: Int = Build.VERSION.SDK_INT,
        hasRecordAudio: Boolean,
        sharing: Boolean,
        micOn: Boolean,
        /** Ataylab parametr: zaxira yo'l ham testlar ostida qolishi uchun. */
        independentTrack: Boolean = INDEPENDENT_TRACK,
    ): Blocked = when {
        sdkInt < Build.VERSION_CODES.Q -> Blocked.OLD_ANDROID
        !hasRecordAudio -> Blocked.NO_PERMISSION
        !sharing -> Blocked.NOT_SHARING
        // Mustaqil trekda mikrofon holati ahamiyatsiz — Zoom'dagi kabi.
        !independentTrack && !micOn -> Blocked.MIC_OFF
        else -> Blocked.NONE
    }

    fun canCapture(
        sdkInt: Int = Build.VERSION.SDK_INT,
        hasRecordAudio: Boolean,
        sharing: Boolean,
        micOn: Boolean,
        independentTrack: Boolean = INDEPENDENT_TRACK,
    ): Boolean =
        blockedBy(sdkInt, hasRecordAudio, sharing, micOn, independentTrack) == Blocked.NONE

    /**
     * Xona ekranidagi yozuv. `active` — audio HAQIQATAN oqyaptimi (`LessonSession`
     * dagi capturer holati), `blocked` — oqmasa sababi.
     *
     * Sabab ko'rsatilishi muhim: "o'chiq" deb qo'yish ustozga nima qilish kerakligini
     * aytmaydi, "mikrofon o'chirilgan" esa aytadi.
     */
    fun statusLabel(active: Boolean, blocked: Blocked): String = when {
        active -> "yoniq"
        blocked == Blocked.MIC_OFF -> "o'chiq — mikrofon o'chirilgan"
        blocked == Blocked.OLD_ANDROID -> "bu Android versiyasida ishlamaydi"
        blocked == Blocked.NO_PERMISSION -> "o'chiq — mikrofon ruxsati yo'q"
        blocked == Blocked.NOT_SHARING -> "o'chiq"
        else -> "o'chiq"
    }
}
