/**
 * ⭐ C-6 — ekran audiosini mikrofondan MUSTAQIL qilish rejasi (sof mantiq).
 *
 * ## Muammo
 * Zoom'da ustoz o'zini mute qilib video ovozini qoldira oladi. Bizda ekran audiosi
 * mikrofon trekiga mikslanadi, shuning uchun mute video ovozini ham o'chirardi.
 *
 * ## QURILMADA O'LCHANGAN IKKI FAKT (2026-07-26, HONOR ABR-LX1 / Android 15)
 *
 * 1. **Bufer callback'i ADM darajasida** (bitta `AudioBufferCallbackDispatcher`,
 *    `audioModule(...)` ga uzatiladi) va telefonda audio kirishi bitta — ya'ni
 *    e'lon qilingan hamma lokal audio trek AYNAN BIR XIL buferni kodlaydi.
 *    "Ekran audiosi uchun mustaqil manba" olishning imkoni yo'q.
 *
 * 2. **Mute holatida e'lon qilingan audio trek obunachiga BERILMAYDI** va uni
 *    keyinroq unmute qilish ham yordam bermaydi: o'lchovda o'quvchi
 *    `SCREEN_SHARE_AUDIO` trekiga umuman obuna bo'lmadi (`OnTrackSubscribed`
 *    chaqirilmadi), mute'dan keyin **jimlik** qoldi.
 *
 * (2) tufayli dastlabki "ikki trekni navbatlashtirish" g'oyasi **rad etildi**:
 * kech qo'shilgan o'quvchi mute'dan keyin hech narsa eshitmaydi.
 *
 * ## ISHLAYDIGAN YECHIM — BITTA TREK, MAZMUNNI BOSHQARISH
 * Mikrofon treki **serverda hech qachon mute qilinmaydi** (shu sababli u har doim
 * obunachilarga yetadi), o'rniga **buferning mazmuni** o'zgaradi:
 *
 * | Ustozning niyati | Mikrofon treki | ADM buferi | O'quvchi eshitadi |
 * |---|---|---|---|
 * | Mikrofon YONIQ | ovozli | mikrofon + ekran ovozi | ustoz + video |
 * | Mikrofon O'CHIQ | ovozli (server) | **mikrofon nollangan** + ekran ovozi | **faqat video** |
 *
 * Ustozning ovozi kodlashdan OLDIN o'chiriladi — ya'ni qurilmadan chiqmaydi.
 * Bu Zoom'ning ko'rinadigan xulqini beradi: mute ovozni to'xtatadi, video ovozi qoladi.
 *
 * ## Ma'lum cheklov (hujjatlashtirilgan, 4-blokda yopiladi)
 * Trek serverda "unmuted" bo'lgani uchun **web'dagi ishtirokchilar ro'yxati** ustozni
 * mute deb ko'rsatmaydi. To'g'ri holat data-channel orqali yuborilishi kerak (M29/D-8).
 * Ilovaning O'Z ekrani rost gapiradi. Ekran ulashilmayotganda esa oddiy, haqiqiy
 * mute ishlatiladi — ya'ni bu chetlanish faqat ulashish davomida amal qiladi.
 */
object ScreenAudioPlan {

    /**
     * @param micWanted ustoz mikrofonni yoqmoqchimi (UI dagi holat)
     * @param screenAudioActive ekran audiosi (capturer) faolmi
     */
    data class Plan(
        /** `microphone` treki serverda ovozli bo'lsinmi. */
        val micTrackLive: Boolean,
        /** ADM buferida mikrofon namunalari o'chirilsinmi. */
        val silenceMicSamples: Boolean,
    ) {
        /** Ustozning ovozi o'quvchilarga bormaydimi (UI shuni ko'rsatadi). */
        val voiceMuted: Boolean get() = !micTrackLive || silenceMicSamples
    }

    fun plan(micWanted: Boolean, screenAudioActive: Boolean): Plan = when {
        // Ekran audiosi yo'q — oddiy, haqiqiy mute (server ham to'g'ri ko'radi).
        !screenAudioActive -> Plan(micTrackLive = micWanted, silenceMicSamples = false)

        // Ekran audiosi bor va mikrofon yoniq — hamma narsa bitta trekdan ketadi.
        micWanted -> Plan(micTrackLive = true, silenceMicSamples = false)

        // Ekran audiosi bor, ustoz mute bosdi: trek ovozli QOLADI (aks holda
        // o'quvchi umuman eshitmaydi — o'lchov bilan tasdiqlangan), lekin mikrofon
        // namunalari nollanadi. Natija: faqat video ovozi.
        else -> Plan(micTrackLive = true, silenceMicSamples = true)
    }
}
