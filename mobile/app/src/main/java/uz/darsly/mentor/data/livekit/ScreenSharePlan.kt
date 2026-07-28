package uz.darsly.mentor.data.livekit

/**
 * Uzilishdan keyin ekran ulashishni tiklash qarori — **sof** mantiq (C-11 · 3-gipoteza).
 *
 * ## Muammo
 * Tarmoq almashganda majburiy qayta ulanish `room.disconnect()` bilan boshlanadi.
 * LiveKit shunda mahalliy treklarni to'xtatadi, ekran treki esa `MediaProjection`
 * ustida turadi — ya'ni proyeksiya ham o'ladi. Dars tiklanadi, LEKIN o'quvchilar
 * qora ekranga qarab qoladi va ustoz (u odatda PDF/GeoGebra ichida) buni sezmaydi.
 *
 * ## Nega "shunchaki qaytadan boshlash" ishlamaydi
 * Android 14 (API 34) dan boshlab **har bir yozib olish sessiyasi uchun yangi
 * rozilik** talab qilinadi: eski `createScreenCaptureIntent()` natijasini qayta
 * ishlatish `SecurityException` beradi. Ya'ni 14+ da avtomatik tiklashning
 * PRINSIPIAL imkoni yo'q — foydalanuvchi bir marta bosishi SHART.
 *
 * Shuning uchun qaror ikki xil: eski Android'da jimgina tiklaymiz, yangisida
 * ustozga bir bosishlik taklif ko'rsatamiz (va u fonda bo'lsa — bildirishnoma).
 */
object ScreenSharePlan {

    enum class Action {
        /** Hech narsa qilinmaydi (ulashish so'ralmagan yoki allaqachon ketmoqda). */
        NONE,

        /** Saqlangan rozilik bilan jimgina qayta boshlanadi (Android 13 va pastda). */
        REUSE_TOKEN,

        /** Ustozdan bir bosishlik rozilik so'raladi (Android 14+). */
        ASK_CONSENT,
    }

    /** Android 14 — rozilik har sessiya uchun qayta so'raladi. */
    const val CONSENT_PER_SESSION_SDK = 34

    /**
     * @param wanted ustoz ulashishni YOQQAN va o'zi to'xtatmagan
     * @param sharingNow hozir ulashish ketmoqdami (uzilish ekran trekiga tegmagan bo'lishi mumkin)
     * @param hasToken oldingi rozilik natijasi saqlanganmi
     * @param sdkInt qurilma Android versiyasi (`Build.VERSION.SDK_INT`)
     */
    fun afterReconnect(
        wanted: Boolean,
        sharingNow: Boolean,
        hasToken: Boolean,
        sdkInt: Int,
    ): Action = when {
        // Ustoz ulashmagan yoki o'zi to'xtatgan — tiklash "yordam" emas, bosqinchilik.
        !wanted -> Action.NONE
        // Uzilish ekran trekini o'ldirmagan (SDK o'z ichida tiklagan) — tegmaymiz.
        sharingNow -> Action.NONE
        !hasToken -> Action.ASK_CONSENT
        sdkInt >= CONSENT_PER_SESSION_SDK -> Action.ASK_CONSENT
        else -> Action.REUSE_TOKEN
    }
}
