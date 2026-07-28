package uz.darsly.mentor.data.livekit

import io.livekit.android.room.network.ReconnectContext
import io.livekit.android.room.network.ReconnectPolicy
import kotlin.time.Duration
import kotlin.time.Duration.Companion.milliseconds
import kotlin.time.Duration.Companion.seconds

/**
 * Qayta ulanish siyosati — **tarmoq almashuvi (Wi-Fi ↔ LTE) uchun sozlangan**.
 *
 * ## Muammo (C-11)
 * Qurilma sinovida yagona yiqilgan mezon: ustoz Wi-Fi'dan mobil internetga
 * o'tganda dars tiklanishi juda uzoq davom etardi. Sabab — SDK'ning umumiy
 * backoff'i "internet yomon" holatiga mo'ljallangan: u kutadi, chunki tez qayta
 * urinish yomon kanalni battar bo'g'adi.
 *
 * Tarmoq ALMASHUVI esa boshqa hodisa: eski interfeys o'lgan, yangisi esa odatda
 * DARHOL tayyor. Bu yerda kutish sof yo'qotish — ustoz jim ekranga qarab turadi.
 *
 * ## Yechim
 * Birinchi urinishlar deyarli darhol (200 ms, 500 ms, 1 s) — bu aynan tarmoq
 * almashuvi holatini qoplaydi. Undan keyin backoff o'sadi: agar uch tez urinish
 * yordam bermagan bo'lsa, muammo interfeys emas, kanal — va u yerda tez urinish
 * zarar keltiradi.
 *
 * Umumiy oyna [MAX_RECONNECT] bilan chegaralangan: undan keyin "qayta ulanmoqda"
 * deb cheksiz turishdan ko'ra rost xabar berish yaxshiroq (ustoz qayta kirishi mumkin).
 *
 * Sof sinf — `ReconnectContext` oddiy ma'lumot obyekti bo'lgani uchun JVM testida
 * to'liq tekshiriladi (loyiha qoidasi: media/tarmoq qarorlari sinovsiz qolmasin).
 */
class LessonReconnectPolicy : ReconnectPolicy {

    override fun getNextRetryDelay(context: ReconnectContext): Duration? {
        if (context.retryCount >= RETRY_DELAYS.size) return null
        if (context.elapsedTime >= MAX_RECONNECT) return null
        return RETRY_DELAYS[context.retryCount]
    }

    companion object {
        /**
         * Urinishlar oralig'i. Dastlabki uchtasi — tarmoq almashuvi uchun;
         * qolganlari — haqiqatan yomon kanal uchun.
         */
        val RETRY_DELAYS: List<Duration> = listOf(
            200.milliseconds,
            500.milliseconds,
            1.seconds,
            2.seconds,
            4.seconds,
            6.seconds,
            8.seconds,
            10.seconds,
        )

        /**
         * Qayta ulanishga ajratiladigan umumiy vaqt.
         *
         * 60 soniya: bundan uzunroq kutish ustoz uchun "osilib qolgan ilova"ga
         * aylanadi; qisqaroq bo'lsa esa sekin 3G'da tiklanish imkoni yo'qoladi.
         */
        val MAX_RECONNECT: Duration = 60.seconds
    }
}
