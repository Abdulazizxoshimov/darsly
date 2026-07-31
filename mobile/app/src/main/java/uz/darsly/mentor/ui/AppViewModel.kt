package uz.darsly.mentor.ui

import androidx.lifecycle.ViewModel
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.StateFlow
import uz.darsly.mentor.data.api.SessionManager
import uz.darsly.mentor.data.repo.NotificationsBadge
import javax.inject.Inject

/**
 * Butun ilovaga tegishli holat: kirilganmi va o'qilmagan bildirishnomalar soni.
 *
 * ## Nega kerak
 *
 * Bu ikki qiymatni `MainActivity` `@Inject` bilan olib, keyin `AppNav`
 * composable'iga uzatish mumkin edi. Lekin `AppNav` — sinfdan TASHQARIDAGI
 * top-level funksiya (Compose'da odatiy) va u Activity maydonlarini ko'rmaydi.
 *
 * Ularni parametr sifatida uzatish ham ishlardi, ammo shunda navigatsiya
 * `Application` hayotiy siklidagi obyektlarga bevosita bog'lanib qolardi.
 * ViewModel esa Compose uchun tabiiy chegara: `hiltViewModel()` bilan olinadi,
 * konfiguratsiya o'zgarishida saqlanadi va testda almashtiriladi.
 */
@HiltViewModel
class AppViewModel @Inject constructor(
    private val session: SessionManager,
    badge: NotificationsBadge,
) : ViewModel() {

    /** `true` — yaroqli sessiya bor. Navigatsiya AYNAN shunga bog'lanadi. */
    val loggedIn: StateFlow<Boolean> = session.loggedIn

    /** Pastki paneldagi nishon uchun o'qilmaganlar soni. */
    val unreadCount: StateFlow<Int> = badge.count

    /**
     * Ilova ochilgandagi boshlang'ich holat.
     *
     * `loggedIn` oqimining birinchi qiymatini kutmasdan, sinxron o'qiladi:
     * navigatsiya boshlang'ich marshrutini TANLASHDAN oldin bilishi kerak,
     * aks holda ustoz bir lahza login ekranini ko'rib, keyin darslarga
     * "sakrab" o'tardi.
     */
    val isLoggedInNow: Boolean get() = session.isLoggedIn
}
