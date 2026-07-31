package uz.darsly.mentor.di

import android.content.Context
import com.squareup.moshi.Moshi
import dagger.hilt.android.qualifiers.ApplicationContext
import dagger.Module
import dagger.Provides
import dagger.hilt.InstallIn
import dagger.hilt.components.SingletonComponent
import okhttp3.Interceptor
import okhttp3.OkHttpClient
import okhttp3.logging.HttpLoggingInterceptor
import retrofit2.Retrofit
import retrofit2.converter.moshi.MoshiConverterFactory
import uz.darsly.mentor.BuildConfig
import uz.darsly.mentor.data.api.AuthRefreshApi
import uz.darsly.mentor.data.api.DarslyApi
import uz.darsly.mentor.data.api.SessionManager
import uz.darsly.mentor.data.api.TokenAuthenticator
import uz.darsly.mentor.util.DevServer
import java.util.concurrent.TimeUnit
import javax.inject.Qualifier
import javax.inject.Singleton

/**
 * Faqat `POST /auth/refresh` uchun mo'ljallangan klient/Retrofit.
 *
 * Alohida belgilanishi SHART: agar refresh so'rovi ham [TokenAuthenticator]
 * ulangan klientdan ketsa va u 401 qaytarsa, authenticator yana refresh
 * chaqirardi — cheksiz rekursiya.
 */
@Qualifier
@Retention(AnnotationRetention.BINARY)
annotation class Bare

/**
 * Tarmoq qatlami.
 *
 * Avval bularning hammasi `object Net` ichida edi va uning o'zida
 * `TODO(R2): Hilt yoki AppContainer bilan almashtirish` izohi turardi.
 * Global bo'lgani uchun ViewModel testida boshqa `baseUrl` (MockWebServer)
 * berish imkoni yo'q edi — shuning uchun ViewModel'lar umuman sinalmagan.
 */
@Module
@InstallIn(SingletonComponent::class)
object NetworkModule {

    @Provides
    @Singleton
    fun moshi(): Moshi = Moshi.Builder().build()

    /** BASIC — parol/token log'ga tushmasligi uchun BODY ishlatilmaydi. */
    private fun logging() = HttpLoggingInterceptor().apply {
        level = HttpLoggingInterceptor.Level.BASIC
    }

    private fun OkHttpClient.Builder.commonTimeouts() = apply {
        connectTimeout(15, TimeUnit.SECONDS)
        readTimeout(30, TimeUnit.SECONDS)
    }

    /**
     * DEV: server manzilini so'rov paytida almashtirish (`DevServer`).
     * Release'da override doim `null` — interceptor no-op bo'lib qoladi.
     * Retrofit `baseUrl` yaratishda qotib qolgani uchun almashtirish shu
     * qatlamda: restart talab qilinmaydi.
     */
    private fun devServerOverride(ctx: Context) = Interceptor { chain ->
        val target = DevServer.override(ctx)
            ?: return@Interceptor chain.proceed(chain.request())
        val url = chain.request().url.newBuilder()
            .scheme(target.scheme)
            .host(target.host)
            .port(target.port)
            .build()
        chain.proceed(chain.request().newBuilder().url(url).build())
    }

    @Provides
    @Singleton
    @Bare
    fun bareClient(@ApplicationContext ctx: Context): OkHttpClient = OkHttpClient.Builder()
        .addInterceptor(devServerOverride(ctx))
        .apply { if (BuildConfig.DEBUG) addInterceptor(logging()) }
        .commonTimeouts()
        .build()

    @Provides
    @Singleton
    @Bare
    fun bareRetrofit(@Bare client: OkHttpClient, moshi: Moshi): Retrofit =
        retrofit(client, moshi)

    @Provides
    @Singleton
    fun refreshApi(@Bare retrofit: Retrofit): AuthRefreshApi =
        retrofit.create(AuthRefreshApi::class.java)

    @Provides
    @Singleton
    fun okHttp(
        @ApplicationContext ctx: Context,
        session: SessionManager,
        refreshApi: AuthRefreshApi,
    ): OkHttpClient {
        // Har so'rovga `Authorization: Bearer <access>` (token bo'lsa).
        val auth = Interceptor { chain ->
            val token = session.accessToken
            val req = if (token.isNullOrBlank()) {
                chain.request()
            } else {
                chain.request().newBuilder().header("Authorization", "Bearer $token").build()
            }
            chain.proceed(req)
        }
        return OkHttpClient.Builder()
            .addInterceptor(devServerOverride(ctx))
            .addInterceptor(auth)
            .authenticator(
                TokenAuthenticator(
                    store = session,
                    refreshApi = refreshApi,
                    // Refresh o'lgan → toza logout. UI `SessionManager.loggedIn`
                    // holatini kuzatib login ekraniga qaytadi; qayta refresh
                    // urinishi bo'lmaydi (sikl yo'q). Sabab ham uzatiladi:
                    // «boshqa qurilmada kirildi» ni umumiy «sessiya tugadi» dan
                    // ajratadigan yagona joy shu.
                    onHardLogout = { reason -> session.forceLogout(reason) },
                ),
            )
            .apply { if (BuildConfig.DEBUG) addInterceptor(logging()) }
            .commonTimeouts()
            .build()
    }

    @Provides
    @Singleton
    fun api(client: OkHttpClient, moshi: Moshi): DarslyApi =
        retrofit(client, moshi).create(DarslyApi::class.java)

    private fun retrofit(client: OkHttpClient, moshi: Moshi): Retrofit = Retrofit.Builder()
        .baseUrl(BuildConfig.API_BASE_URL.trimEnd('/') + "/")
        .client(client)
        .addConverterFactory(MoshiConverterFactory.create(moshi))
        .build()
}
