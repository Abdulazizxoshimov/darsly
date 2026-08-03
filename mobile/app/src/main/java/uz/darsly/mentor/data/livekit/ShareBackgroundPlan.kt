package uz.darsly.mentor.data.livekit

/**
 * Ekran ulashish boshlanganda ilova fonga o'tsinmi — **sof** qaror (№25).
 *
 * ## Muammo (haqiqiy yozuvdan, 2026-08-01)
 * Asoschi telefondan dars o'tdi va ekranini ulashdi. Yozuvda o'quvchilar dars
 * materialini emas, **Jonly ilovasining o'zini** ko'rdi: «Ekraningiz
 * ulashilmoqda» ekrani, ishtirokchilar paneli, «Hammani mute», qo'l
 * ko'targanlar ro'yxati. Sababi oddiy: Android BUTUN ekranni ulashadi, ustoz
 * esa ilova ichida qolib ketdi — hech kim unga "endi materialingizni oching"
 * demadi.
 *
 * Zoom buni shunday hal qiladi: ulashish boshlanishi bilan ilova o'zini fonga
 * oladi va foydalanuvchi o'zi ochmoqchi bo'lgan kontentda qoladi.
 *
 * ## Nega alohida, sof funksiya
 * Qaror uchta mustaqil shartdan yig'iladi (ulashish haqiqatan boshlandimi ·
 * bu sessiyada allaqachon bajarilganmi · ustoz sozlamani o'chirganmi) va
 * ularning har biri noto'g'ri bo'lsa oqibat OG'IR: ilova o'zini kutilmaganda
 * fonga tashlaydi. Bunday qarorni `moveTaskToBack` chaqiruvi yonida, Compose
 * ichida ushlab bo'lmaydi — u yerda uni sinab ko'rishning iloji yo'q.
 */
object ShareBackgroundPlan {

    /**
     * @param moveToBack ilova fonga olinsinmi (`Activity.moveTaskToBack`).
     * @param message fonga o'tishdan OLDIN ko'rsatiladigan qisqa xabar
     *   (`null` — ko'rsatiladigan narsa yo'q).
     */
    data class Decision(
        val moveToBack: Boolean,
        val message: String?,
    ) {
        /** Hech narsa qilinmaydi — chaqiruvchi holatni ham yangilamasin. */
        val isEmpty: Boolean get() = !moveToBack && message == null
    }

    val NONE = Decision(moveToBack = false, message = null)

    /**
     * Fonga o'tish xabari.
     *
     * Ikkinchi jumla MAJBURIY: fonga o'tgan ilovaga qaytish yo'li ustoz uchun
     * ravshan bo'lishi kerak, aks holda u dars o'rtasida "ilova yo'qoldi" deb
     * o'ylaydi. Doimiy bildirishnoma (`LessonNotifications`) bosilganda
     * `MainActivity` SINGLE_TOP bilan ochiladi — ya'ni xona ekrani o'sha
     * holatida qaytadi.
     */
    const val BACKGROUND_MESSAGE =
        "Ekran ulashilmoqda — materialingizni oching. " +
            "Qaytish uchun bildirishnomani bosing."

    /**
     * Kamera o'chiq bo'lsa qo'shiladigan ogohlantirish.
     *
     * Nega aynan shu payt: yozuvda ustozning plitkasi **bo'sh avatar** bo'lib
     * qoladi (2026-08-01 yozuvida shunday bo'lgan) va buni ustoz dars tugagach,
     * ya'ni tuzatib bo'lmaydigan paytda ko'radi. Ulashish boshlanishi — oxirgi
     * qulay lahza.
     *
     * Kamera AVTOMATIK yoqilmaydi: `docs/PRODUCT.md` bo'yicha ulashishda
     * kontent butun kadrni egallaydi, kamera esa kichik yoki umuman yo'q —
     * ya'ni bu ustozning tanlovi, ilovaning emas.
     */
    const val CAMERA_OFF_MESSAGE =
        "Kamerangiz o'chiq — yozuvda plitkangiz bo'sh ko'rinadi."

    /**
     * @param sharing ekran ulashish HAQIQATAN boshlandimi (`screenShareOn`).
     * @param alreadyHandled shu dars sessiyasida qaror allaqachon bajarilganmi.
     *   Qayta ulanish/tiklashda ulashish qaytadan yoqiladi va ustoz o'sha payt
     *   ILOVAGA QAYTGAN bo'lishi mumkin — uni ikkinchi marta fonga tashlash
     *   "ilova o'zboshimchalik qilyapti" degani.
     * @param autoBackground ustoz sozlamada fonga o'tishni yoqib qo'yganmi.
     * @param cameraOn kamera treki hozir yoqilganmi.
     */
    fun onShareStarted(
        sharing: Boolean,
        alreadyHandled: Boolean,
        autoBackground: Boolean,
        cameraOn: Boolean,
    ): Decision {
        if (!sharing || alreadyHandled) return NONE

        val message = when {
            autoBackground && cameraOn -> BACKGROUND_MESSAGE
            autoBackground -> "$BACKGROUND_MESSAGE $CAMERA_OFF_MESSAGE"
            // Sozlama o'chiq bo'lsa ham kamera ogohlantirishi qoladi: u fonga
            // o'tish bilan bog'liq emas, yozuv sifati bilan bog'liq.
            !cameraOn -> CAMERA_OFF_MESSAGE
            else -> null
        }
        return Decision(moveToBack = autoBackground, message = message)
    }
}
