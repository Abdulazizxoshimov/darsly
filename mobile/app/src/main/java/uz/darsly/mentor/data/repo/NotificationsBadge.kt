package uz.darsly.mentor.data.repo

import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update

/**
 * O'qilmagan bildirishnomalar soni — pastki paneldagi nishon uchun.
 *
 * ## Nega jarayon darajasida (singleton), ViewModel'da emas
 * Nishon **navigatsiya panelida** turadi, ya'ni bildirishnomalar ekrani ochiq
 * bo'lmaganda ham ko'rinadi. ViewModel'ga bog'lansa ekran ochilmaguncha
 * hisoblanmasdi — ya'ni ustoz nishon sababli ekranni ochardi, nishon esa faqat
 * ekran ochilganda paydo bo'lardi. Klassik "tovuq va tuxum".
 *
 * ## Kim yozadi
 *  · `DarslyApp` — kirishda boshlang'ich qiymatni serverdan oladi va
 *    real-time `notification` hodisasida [increment] qiladi;
 *  · `NotificationsViewModel` — ro'yxat holatidan ANIQ qiymatni [set] qiladi
 *    (server haqiqati), ya'ni oshirilgan taxmin uning ustiga yozilib to'g'rilanadi;
 *  · chiqishda [clear] — boshqa ustozning soni ko'rinib qolmasin.
 */
object NotificationsBadge {

    private val _count = MutableStateFlow(0)
    val count: StateFlow<Int> = _count.asStateFlow()

    /** Aniq qiymat (server yoki to'liq ro'yxatdan hisoblangan). Manfiy qiymat nolga tushadi. */
    fun set(value: Int) {
        _count.value = value.coerceAtLeast(0)
    }

    /** Jonli xabar keldi — ekran ochiq bo'lmasa ham son o'sadi. */
    fun increment() = _count.update { it + 1 }

    /** Logout (sabab nima bo'lishidan qat'i nazar). */
    fun clear() {
        _count.value = 0
    }
}
