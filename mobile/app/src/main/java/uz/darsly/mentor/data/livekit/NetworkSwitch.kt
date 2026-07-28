package uz.darsly.mentor.data.livekit

/** Qurilma qaysi tarmoq orqali chiqyapti. */
enum class Transport { WIFI, CELLULAR, OTHER, NONE }

/**
 * Tarmoq almashuvi qarori — **sof** mantiq (Android'siz, JVM testida).
 *
 * ## Nega alohida qoida kerak (C-11)
 * LiveKit SDK'sining o'zi tarmoqni kuzatadi va qayta ulanadi. Lekin qurilma
 * sinovida yagona yiqilgan mezon aynan shu bo'ldi: Wi-Fi'dan LTE'ga o'tishda
 * tiklanish juda uzoq davom etdi. Sabab — Android'da eski interfeys **darhol
 * o'lmaydi**: soket ochiq ko'rinadi, paketlar esa ketmaydi ("half-open"
 * ulanish). SDK buni faqat timeout orqali sezadi.
 *
 * Shuning uchun biz TRANSPORT o'zgarishini alohida kuzatamiz: bu hodisa
 * "eski yo'l endi yaroqsiz" degan ANIQ signal va uni timeout kutmasdan
 * qo'llash mumkin.
 *
 * ## Nega har o'zgarishda emas
 * Bir xil transport ichidagi o'zgarish (masalan bir Wi-Fi nuqtadan boshqasiga,
 * yoki uyali tarmoqning ichki qayta ulanishi) odatda WebRTC uchun shaffof —
 * u yerda majburiy uzib-ulash foydadan ko'ra zarar keltiradi (dars uziladi).
 */
object NetworkSwitchPolicy {

    /**
     * Majburiy qayta ulanish kerakmi?
     *
     * @param from oldingi transport (`null` — hali noma'lum, ilova endi ishga tushgan)
     * @param to yangi transport
     * @param connected xona hozir ulangan (yoki ulanmoqda) holatdami
     */
    fun shouldForceReconnect(from: Transport?, to: Transport, connected: Boolean): Boolean {
        // Xonada bo'lmasak qayta ulanadigan narsa yo'q.
        if (!connected) return false
        // Birinchi o'lchov — bu "almashuv" emas, boshlang'ich holat.
        if (from == null) return false
        // Tarmoq butunlay yo'qolgan bo'lsa kutamiz: majburiy urinish shu zahoti
        // yiqiladi va faqat backoff'ni behuda sarflaydi. Tarmoq qaytganda
        // bu funksiya qaytadan chaqiriladi.
        if (to == Transport.NONE) return false
        // Faqat TRANSPORT o'zgarganda (Wi-Fi ↔ LTE) — ayni C-11 stsenariysi.
        return from != to
    }

    /** UI'da ko'rsatiladigan qisqa izoh (`null` — ko'rsatilmaydi). */
    fun label(from: Transport?, to: Transport): String? = when {
        from == null -> null
        to == Transport.NONE -> "Internet yo'q"
        from == to -> null
        to == Transport.CELLULAR -> "Mobil internetga o'tildi"
        to == Transport.WIFI -> "Wi-Fi'ga o'tildi"
        else -> "Tarmoq almashdi"
    }
}
