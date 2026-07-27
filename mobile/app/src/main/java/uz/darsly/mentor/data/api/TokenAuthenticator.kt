package uz.darsly.mentor.data.api

import kotlinx.coroutines.runBlocking
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import okhttp3.Authenticator
import okhttp3.Request
import okhttp3.Response
import okhttp3.Route
import uz.darsly.mentor.data.store.TokenStore
import java.io.IOException

/**
 * Jim token yangilash — **single-flight** (M2).
 *
 * ## Nega [Authenticator], [okhttp3.Interceptor] emas
 * OkHttp `Authenticator`ni AYNAN 401 javobida chaqiradi va qaytarilgan so'rovni
 * o'zi qayta yuboradi — original so'rovning **body'si, metodi va header'lari
 * saqlanadi** (Interceptor'da body bir marta o'qilgach qayta o'ynatish qo'lda
 * qilinadi va `RequestBody` streaming bo'lsa umuman iloji yo'q). Bundan tashqari
 * `Response.priorResponse` zanjiri qayta urinishlar sonini sanashning tayyor,
 * ishonchli usulini beradi va 401'dan boshqa trafikka umuman aralashmaydi.
 *
 * Yagona narxi — `authenticate()` **bloklovchi** funksiya, shuning uchun [Mutex]
 * `runBlocking` ichida ushlanadi. Bu xavfsiz: OkHttp bu metodni hech qachon main
 * thread'da chaqirmaydi (o'z dispatcher/ulanish oqimida), va refresh so'rovi
 * ALOHIDA, authenticator'siz klient orqali ketadi — ya'ni na rekursiya, na
 * dispatcher tiqilishi bo'ladi.
 *
 * ## Single-flight kafolati
 * Bir vaqtda N ta 401 kelsa ham serverga **BITTA** `POST /auth/refresh` ketadi:
 *  1. Hammasi [mutex] navbatiga turadi.
 *  2. Birinchisi yangilaydi va yangi juftlikni [store] ga yozadi.
 *  3. Qolganlari qulfni olgach ko'radi: saqlangan access token ularning **eskirgan**
 *     tokenidan farq qiladi → refresh YUBORMAYDI, shu yangi token bilan qayta uradi.
 *
 * Backend'da 60 soniyalik grace oynasi bor (BE-2), lekin u klient xatosini
 * oqlamaydi — bu yerda poyga prinsipial ravishda yo'q qilingan.
 *
 * ## Cheksiz sikl himoyasi
 *  · `priorResponse != null` → bu so'rov allaqachon bir marta qayta urinilgan → to'xtash.
 *  · Refresh 401/403 bersa → [onHardLogout] (tokenlar o'chadi) → keyingi 401'da
 *    `store.read() == null` → refresh umuman yuborilmaydi.
 *  · `/auth/login` va `/auth/refresh` yo'llari butunlay chetlab o'tiladi.
 */
class TokenAuthenticator(
    private val store: TokenStore,
    private val refreshApi: AuthRefreshApi,
    private val onHardLogout: () -> Unit,
) : Authenticator {

    private val mutex = Mutex()

    override fun authenticate(route: Route?, response: Response): Request? {
        // 1) Auth endpointlariga aralashmaymiz (login xatosi refreshni qo'zg'atmasin).
        val path = response.request.url.encodedPath
        if (SKIP_PATHS.any { path.endsWith(it) }) return null

        // 2) Faqat BIR marta qayta urinamiz.
        if (response.priorResponse != null) return null

        // 3) Tokensiz ketgan so'rovni "yangilash" ma'nosiz.
        val stale = response.request.header(HEADER)?.removePrefix(PREFIX)?.takeIf { it.isNotBlank() }
            ?: return null

        val fresh = runBlocking { freshToken(stale) } ?: return null

        return response.request.newBuilder()
            .header(HEADER, PREFIX + fresh)
            .build()
    }

    /**
     * Yangi access token qaytaradi yoki `null` (qayta urinmaslik kerak).
     * Butun tanasi [mutex] ostida — bu single-flight'ning o'zagi.
     */
    private suspend fun freshToken(stale: String): String? = mutex.withLock {
        val current = store.read() ?: return@withLock null

        // Boshqa oqim allaqachon yangilagan — qayta so'ramaymiz.
        //
        // ONGLI MUROSA: saqlangan token `stale`dan farq qilsa uni tekshirmasdan
        // ishlatamiz. Nazariy chekka holat — bitta access TTL ichida IKKI marta
        // rotatsiya bo'lsa (masalan boshqa oqim endigina yangilagan, lekin biz uni
        // olishdan oldin u ham eskirgan) qayta urinish yana 401 beradi. Bu holda
        // `priorResponse` qorovuli ishlaydi: ikkinchi marta urinilmaydi, 401 UI'ga
        // chiqadi va foydalanuvchi amalni takrorlaydi. Muqobil yechim — har safar
        // tokenni tekshirish uchun qo'shimcha so'rov yuborish — har bir 401 ga
        // kechikish qo'shardi va SIKL xavfini tug'dirardi. Xavfsizroq varianti
        // tanlangan: hech qachon sikl yo'q, eng yomoni bitta so'rov qayta bosiladi.
        if (current.accessToken != stale) return@withLock current.accessToken

        val refreshToken = current.refreshToken.takeIf { it.isNotBlank() } ?: run {
            onHardLogout()
            return@withLock null
        }

        val resp = try {
            refreshApi.refresh(RefreshReq(refreshToken)).execute()
        } catch (e: IOException) {
            // Tarmoq uzilishi — sessiyani O'LDIRMAYMIZ. Ustozni dars o'rtasida
            // internet sekinlashgani uchun tizimdan chiqarib yuborish mumkin emas.
            //
            // Lekin `null` qaytarib qo'ya olmaymiz: u holda chaqiruvchiga ASL 401
            // yetib borardi va UI "Sessiya tugadi — qaytadan kiring" deb yozardi,
            // holbuki foydalanuvchi chiqarilmagan va aybdor — tarmoq. Shuning uchun
            // `IOException` uzatiladi: `ApiErrors.humanError` uni tarmoq xatosi deb
            // taniydi va "Serverga ulanib bo'lmadi" matnini beradi.
            throw IOException("token yangilanmadi: tarmoqqa ulanib bo'lmadi", e)
        }

        val pair = resp.body()?.data
        if (!resp.isSuccessful || pair == null) {
            resp.errorBody()?.close()
            // 4xx = refresh token haqiqatan yaroqsiz → toza logout.
            // 5xx = server nosozligi → keyinroq qayta urinsin, sessiya saqlanadi.
            if (resp.code() in 400..499) onHardLogout()
            return@withLock null
        }

        store.save(pair)
        pair.accessToken
    }

    private companion object {
        const val HEADER = "Authorization"
        const val PREFIX = "Bearer "
        val SKIP_PATHS = listOf("/auth/login", "/auth/refresh", "/auth/register")
    }
}
