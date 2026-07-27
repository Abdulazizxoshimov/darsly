package uz.darsly.mentor.data.api

import com.squareup.moshi.Moshi
import okhttp3.Interceptor
import okhttp3.OkHttpClient
import okhttp3.logging.HttpLoggingInterceptor
import retrofit2.Retrofit
import retrofit2.converter.moshi.MoshiConverterFactory
import uz.darsly.mentor.BuildConfig
import java.util.concurrent.TimeUnit

/**
 * Retrofit/OkHttp yig'uvchisi. Bitta global instans.
 * TODO(R2): Hilt yoki qo'lda yasalgan AppContainer bilan almashtirish.
 *
 * IKKI klient bor:
 *  · [client] — oddiy so'rovlar; `Authorization` qo'shadi va 401'da
 *    [TokenAuthenticator] orqali single-flight refresh qiladi.
 *  · [bareClient] — FAQAT `POST /auth/refresh` uchun; authenticator'siz,
 *    shuning uchun refresh 401 bersa rekursiya bo'lmaydi.
 */
object Net {

    private val moshi: Moshi = Moshi.Builder().build()

    private fun logging(): HttpLoggingInterceptor = HttpLoggingInterceptor().apply {
        // BASIC — parol/token log'ga tushmasligi uchun BODY ishlatilmaydi.
        level = HttpLoggingInterceptor.Level.BASIC
    }

    /** Har so'rovga `Authorization: Bearer <access>` qo'shadi (token bo'lsa). */
    private val authInterceptor = Interceptor { chain ->
        val token = Session.accessToken
        val req = if (token.isNullOrBlank()) {
            chain.request()
        } else {
            chain.request().newBuilder()
                .header("Authorization", "Bearer $token")
                .build()
        }
        chain.proceed(req)
    }

    private val bareClient: OkHttpClient = OkHttpClient.Builder()
        .apply { if (BuildConfig.DEBUG) addInterceptor(logging()) }
        .connectTimeout(15, TimeUnit.SECONDS)
        .readTimeout(30, TimeUnit.SECONDS)
        .build()

    private fun retrofit(client: OkHttpClient): Retrofit = Retrofit.Builder()
        .baseUrl(BuildConfig.API_BASE_URL.trimEnd('/') + "/")
        .client(client)
        .addConverterFactory(MoshiConverterFactory.create(moshi))
        .build()

    private val refreshApi: AuthRefreshApi =
        retrofit(bareClient).create(AuthRefreshApi::class.java)

    private val client: OkHttpClient = OkHttpClient.Builder()
        .addInterceptor(authInterceptor)
        .authenticator(
            TokenAuthenticator(
                store = Session,
                refreshApi = refreshApi,
                // Refresh o'lgan → toza logout. UI `Session.loggedIn` holatini
                // kuzatib login ekraniga qaytadi; qayta refresh urinishi bo'lmaydi.
                onHardLogout = { Session.forceLogout() },
            ),
        )
        .apply { if (BuildConfig.DEBUG) addInterceptor(logging()) }
        .connectTimeout(15, TimeUnit.SECONDS)
        .readTimeout(30, TimeUnit.SECONDS)
        .build()

    val api: DarslyApi = retrofit(client).create(DarslyApi::class.java)

    /** @see ApiErrors.humanError — eski chaqiruvlar uchun qisqartma. */
    fun humanError(t: Throwable): String = ApiErrors.humanError(t)
}
