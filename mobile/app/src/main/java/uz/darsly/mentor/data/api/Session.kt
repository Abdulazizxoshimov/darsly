package uz.darsly.mentor.data.api

import android.content.Context
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import uz.darsly.mentor.data.store.InMemoryTokenStore
import uz.darsly.mentor.data.store.SecureTokenStore
import uz.darsly.mentor.data.store.TokenStore

/**
 * Sessiya — butun ilova uchun yagona token manbai (M1).
 *
 * O'zi ham [TokenStore], ichidagi haqiqiy saqlagichni [init] almashtiradi. Shu sabab
 * [Net] ni yaratishda "hozircha xotirada" saqlagich ushlanib qolish xavfi yo'q:
 * hamma joy shu obyektga murojaat qiladi.
 *
 * ## Nega [loggedIn] — HOLAT ([StateFlow]), hodisa emas
 * Avval bu yerda `MutableSharedFlow` (replay=0) bor edi va logout **hodisa** sifatida
 * yuborilardi. Bu jimgina yo'qoladigan poyga edi: agar refresh muvaffaqiyatsizligi UI
 * kollektori ulanishidan OLDIN sodir bo'lsa, `tryEmit` `true` qaytarardi, lekin qiymat
 * hech kimga yetib bormasdi — tokenlar o'chgan, ekran esa "Darslar"da qolib ketardi va
 * ustoz har so'rovda xato ko'rardi.
 *
 * `StateFlow`da bunday teshik prinsipial ravishda yo'q: uning **joriy qiymati** bor,
 * kech ulangan obunachi ham darhol to'g'ri holatni oladi. `replay = 1` li `SharedFlow`
 * ham signalni saqlagan bo'lardi, lekin u eski logout "hodisasi"ni keyingi loginдан
 * so'ng qayta o'ynashi mumkin edi va uni qo'lda tozalash kerak bo'lardi. Holat esa
 * o'z-o'zidan to'g'ri: login → `true`, logout → `false`, boshqa hisob yuritish yo'q.
 */
object Session : TokenStore {

    @Volatile
    private var delegate: TokenStore = InMemoryTokenStore()

    private val _loggedIn = MutableStateFlow(false)

    /** `true` — yaroqli sessiya bor. UI navigatsiyasi AYNAN shunga bog'lanadi. */
    val loggedIn: StateFlow<Boolean> = _loggedIn.asStateFlow()

    /** `Application.onCreate` da bir marta — shifrlangan saqlagichni ulaydi. */
    fun init(context: Context) {
        delegate = SecureTokenStore.create(context.applicationContext)
        _loggedIn.value = delegate.read() != null
    }

    /** FAQAT test uchun: saqlagichni almashtirish. */
    fun installForTest(store: TokenStore) {
        delegate = store
        _loggedIn.value = store.read() != null
    }

    override fun read(): TokenPair? = delegate.read()

    override fun save(pair: TokenPair) {
        delegate.save(pair)
        _loggedIn.value = true
    }

    override fun clear() {
        delegate.clear()
        _loggedIn.value = false
    }

    val accessToken: String? get() = read()?.accessToken
    val refreshToken: String? get() = read()?.refreshToken
    val isLoggedIn: Boolean get() = _loggedIn.value

    /**
     * Toza logout: tokenlarni o'chiradi va [loggedIn] ni `false` qiladi.
     * Refresh muvaffaqiyatsiz bo'lganda [TokenAuthenticator] shuni chaqiradi —
     * shu tufayli keyingi 401 uchun refresh urinishi umuman bo'lmaydi (sikl yo'q).
     */
    fun forceLogout() = clear()
}
