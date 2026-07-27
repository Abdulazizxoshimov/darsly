package uz.darsly.mentor.ui.update

import uz.darsly.mentor.data.api.AndroidConfig
import uz.darsly.mentor.util.Semver

/** Yangilanish holati (M42). */
sealed interface UpdateState {
    /** Hech narsa kerak emas (yoki tekshirib bo'lmadi — fail-open). */
    data object None : UpdateState

    /** Yumshoq eslatma — yopib ketish mumkin. */
    data class Optional(val latest: String, val apkUrl: String, val notes: String) : UpdateState

    /** Bloklovchi — ilovadan foydalanib bo'lmaydi. */
    data class Required(val minVersion: String, val apkUrl: String, val notes: String) : UpdateState
}

/**
 * `GET /api/v1/app-config` javobi asosida qaror (M42).
 *
 * **FAIL-OPEN prinsipi:** endpoint javob bermasa (`cfg == null`) yoki versiya
 * qiymati buzuq bo'lsa — [UpdateState.None]. Aks holda serverning bir daqiqalik
 * nosozligi BARCHA ustozlarni darsdan mahrum qilardi. Kill-switch faqat server
 * aniq va tushunarli javob berganda ishlaydi.
 */
object UpdateDecider {

    fun decide(currentVersion: String, cfg: AndroidConfig?): UpdateState {
        if (cfg == null) return UpdateState.None

        // Favqulodda to'xtatish: versiyadan qat'i nazar.
        if (cfg.forceUpdate) {
            return UpdateState.Required(cfg.minVersion, cfg.apkUrl, cfg.releaseNotes)
        }

        // `Semver.compare` null qaytarsa (buzuq qiymat) — bloklamaymiz.
        if (Semver.compare(currentVersion, cfg.minVersion).let { it != null && it < 0 }) {
            return UpdateState.Required(cfg.minVersion, cfg.apkUrl, cfg.releaseNotes)
        }

        if (Semver.compare(currentVersion, cfg.latestVersion).let { it != null && it < 0 }) {
            return UpdateState.Optional(cfg.latestVersion, cfg.apkUrl, cfg.releaseNotes)
        }

        return UpdateState.None
    }
}
