package uz.darsly.mentor.util

import java.net.URI

/**
 * Serverdan kelgan havolani tizimga (`ACTION_VIEW`) berishdan oldingi
 * darvoza — **sof** (JVM testida).
 *
 * ## Nega (S2)
 * `apk_url` (yangilanish dialogi, shu jumladan YOPIB BO'LMAYDIGAN majburiy
 * dialog) va chat fayl havolalari tekshirilmasdan `ACTION_VIEW` ga ketardi.
 * Server javobi buzilsa yoki oraliqda almashtirilsa `intent://`, `file://`,
 * `content://` yoki begona hostdagi APK ochilardi — majburiy yangilanish
 * dialogida ustozning boshqa iloji ham yo'q edi.
 *
 * Qoida: faqat `https` (debug'da lokal server uchun `http` ham) va, talab
 * qilinsa, host [allowedHosts] ichida.
 */
object SafeLinks {

    /**
     * @param allowedHosts bo'sh — host tekshirilmaydi (masalan presigned MinIO
     *   havolasi API hostidan boshqa subdomenda bo'ladi); bo'sh emas — host
     *   aynan shulardan biri bo'lishi shart (registrsiz).
     * @param allowCleartext `http` ga ruxsat (faqat debug build).
     */
    fun isAllowed(url: String, allowedHosts: Set<String> = emptySet(), allowCleartext: Boolean = false): Boolean {
        val uri = runCatching { URI(url.trim()) }.getOrNull() ?: return false
        val scheme = uri.scheme?.lowercase() ?: return false
        if (scheme != "https" && !(allowCleartext && scheme == "http")) return false
        val host = uri.host?.lowercase()?.takeIf { it.isNotBlank() } ?: return false
        if (allowedHosts.isEmpty()) return true
        return allowedHosts.any { it.equals(host, ignoreCase = true) }
    }

    /** Bazaviy manzil(lar)dan hostlarni yig'adi — yaroqsiz/bo'sh qiymat tashlanadi. */
    fun hostsOf(vararg baseUrls: String): Set<String> =
        baseUrls.mapNotNull { runCatching { URI(it.trim()).host?.lowercase() }.getOrNull() }
            .filter { it.isNotBlank() }
            .toSet()
}
