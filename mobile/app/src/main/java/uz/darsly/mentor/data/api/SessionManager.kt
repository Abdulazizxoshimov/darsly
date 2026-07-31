package uz.darsly.mentor.data.api

import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import uz.darsly.mentor.data.store.TokenStore
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Sessiya — butun ilova uchun yagona token manbai (M1).
 *
 * Avval bu `object Session` edi: jarayon darajasidagi global singleton, ichida
 * `init(context)` va `installForTest(store)` bilan. Ikkinchisi — DI yo'qligining
 * alomati: global holatni test uchun teshish. Endi u oddiy sinf va saqlagichni
 * konstruktordan oladi, ya'ni testda hech qanday maxsus teshik kerak emas.
 *
 * Jarayon darajasidagi yagonalik yo'qolmadi — uni endi Hilt (`@Singleton`)
 * kafolatlaydi.
 *
 * ## Nega [loggedIn] — HOLAT ([StateFlow]), hodisa emas
 *
 * Avval bu yerda `MutableSharedFlow` (replay=0) bor edi va logout **hodisa**
 * sifatida yuborilardi. Bu jimgina yo'qoladigan poyga edi: refresh
 * muvaffaqiyatsizligi UI kollektori ulanishidan OLDIN sodir bo'lsa, `tryEmit`
 * `true` qaytarardi, lekin qiymat hech kimga yetib bormasdi — tokenlar o'chgan,
 * ekran esa "Darslar"da qolib ketardi va ustoz har so'rovda xato ko'rardi.
 *
 * `StateFlow`da bunday teshik prinsipial ravishda yo'q: uning **joriy qiymati**
 * bor, kech ulangan obunachi ham darhol to'g'ri holatni oladi. `replay = 1` li
 * `SharedFlow` ham signalni saqlagan bo'lardi, lekin u eski logout "hodisasi"ni
 * keyingi login'dan so'ng qayta o'ynashi mumkin edi va uni qo'lda tozalash
 * kerak bo'lardi. Holat esa o'z-o'zidan to'g'ri: login → `true`, logout →
 * `false`, boshqa hisob yuritish yo'q.
 */
@Singleton
class SessionManager @Inject constructor(
    private val store: TokenStore,
) : TokenStore {

    private val _loggedIn = MutableStateFlow(store.read() != null)

    /** `true` — yaroqli sessiya bor. UI navigatsiyasi AYNAN shunga bog'lanadi. */
    val loggedIn: StateFlow<Boolean> = _loggedIn.asStateFlow()

    private val _logoutReason = MutableStateFlow(LogoutReason.UNKNOWN)

    /**
     * NEGA chiqarildik — login ekrani shuni ko'rsatadi.
     *
     * Bu ham HOLAT ([loggedIn] bilan bir sababdan): sessiya tugatilishi va login
     * ekranining paydo bo'lishi orasida bir necha kadr o'tadi, hodisa esa aynan
     * shu oraliqda yo'qolardi va ustoz sababsiz login formasini ko'rardi.
     */
    val logoutReason: StateFlow<LogoutReason> = _logoutReason.asStateFlow()

    override fun read(): TokenPair? = store.read()

    override fun save(pair: TokenPair) {
        store.save(pair)
        // Yangi sessiya — eski sabab endi ahamiyatsiz. Aks holda u keyingi
        // chiqishda qayta ko'rinib, ustozni chalg'itardi.
        _logoutReason.value = LogoutReason.UNKNOWN
        _loggedIn.value = true
    }

    override fun clear() {
        store.clear()
        _loggedIn.value = false
    }

    val accessToken: String? get() = read()?.accessToken
    val refreshToken: String? get() = read()?.refreshToken
    val isLoggedIn: Boolean get() = _loggedIn.value

    /**
     * Toza logout: tokenlarni o'chiradi va [loggedIn] ni `false` qiladi.
     * Refresh muvaffaqiyatsiz bo'lganda [TokenAuthenticator] shuni chaqiradi —
     * shu tufayli keyingi 401 uchun refresh urinishi umuman bo'lmaydi (sikl yo'q).
     *
     * [reason] — login ekranida ko'rsatiladigan sabab. Foydalanuvchi O'ZI
     * chiqqanda [LogoutReason.UNKNOWN] qoladi: u nima qilganini biladi va
     * unga "nega chiqdingiz" deb tushuntirish ortiqcha shovqin bo'lardi.
     */
    fun forceLogout(reason: LogoutReason = LogoutReason.UNKNOWN) {
        _logoutReason.value = reason
        clear()
    }

    /** Sabab ekranda ko'rsatildi — takror chiqmasin. */
    fun logoutReasonShown() {
        _logoutReason.value = LogoutReason.UNKNOWN
    }
}
